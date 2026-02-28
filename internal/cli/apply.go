package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/database"
	"github.com/aqasim81/database-migration-engine/internal/executor"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

// errDangerousMigrations is returned when apply is blocked by high/critical findings.
var errDangerousMigrations = errors.New("apply aborted: dangerous migrations detected (use --force to override)")

// errDatabaseURLRequired is returned when no database URL is configured.
var errDatabaseURLRequired = errors.New( //nolint:gochecknoglobals // sentinel error
	"database URL is required (set --database-url, MIGRATE_DATABASE_URL, or database_url in config)",
)

var applyCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "apply",
	Short: "Apply pending migrations",
	Long: `Apply pending database migrations with configurable lock and
statement timeouts. Supports dry-run mode and force execution.`,
	RunE: runApply,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	applyCmd.Flags().Bool("dry-run", false, "show what would be applied without executing")
	applyCmd.Flags().Bool("force", false, "skip safety checks and confirmation prompts")
	applyCmd.Flags().Duration("lock-timeout", 0, "override lock timeout (e.g., 10s, 1m)")
	applyCmd.Flags().Duration("statement-timeout", 0, "override statement timeout (e.g., 30s, 5m)")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, _ []string) error {
	cfg := AppConfig

	if cfg.DatabaseURL == "" {
		return errDatabaseURLRequired
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	lockTimeout := cfg.LockTimeout
	if cmd.Flags().Changed("lock-timeout") {
		lockTimeout, _ = cmd.Flags().GetDuration("lock-timeout")
	}

	stmtTimeout := cfg.StatementTimeout
	if cmd.Flags().Changed("statement-timeout") {
		stmtTimeout, _ = cmd.Flags().GetDuration("statement-timeout")
	}

	sorted, err := loadAndSortMigrations(cfg.MigrationsDir, cmd.OutOrStdout())
	if err != nil || sorted == nil {
		return err
	}

	if !force && !dryRun {
		if blocked, analyzeErr := checkDangerousMigrations(cmd, sorted, cfg); analyzeErr != nil {
			return analyzeErr
		} else if blocked {
			return errDangerousMigrations
		}
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	pool, err := connectDB(ctx, cfg, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	defer pool.Close()

	return executeMigrations(ctx, cmd.OutOrStdout(), pool, sorted, applyOpts{
		lockTimeout: lockTimeout,
		stmtTimeout: stmtTimeout,
		dryRun:      dryRun,
	})
}

type applyOpts struct {
	lockTimeout time.Duration
	stmtTimeout time.Duration
	dryRun      bool
}

func loadAndSortMigrations(dir string, out io.Writer) ([]migration.Migration, error) {
	migrations, err := migration.LoadFromDir(dir)
	if err != nil {
		return nil, fmt.Errorf("loading migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Fprintln(out, "No migration files found.")
		return nil, nil //nolint:nilnil // nil,nil signals "no migrations, no error"
	}

	return migration.Sort(migrations), nil
}

func connectDB(ctx context.Context, cfg *config.Config, out io.Writer) (*pgxpool.Pool, error) {
	fmt.Fprintf(out, "Connecting to %s\n", config.RedactURL(cfg.DatabaseURL))

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return pool, nil
}

func executeMigrations(
	ctx context.Context,
	out io.Writer,
	pool *pgxpool.Pool,
	sorted []migration.Migration,
	opts applyOpts,
) error {
	t := tracker.New(pool)

	applied := 0
	skipped := 0

	exec := executor.New(pool, t,
		executor.WithLockTimeout(opts.lockTimeout),
		executor.WithStatementTimeout(opts.stmtTimeout),
		executor.WithDryRun(opts.dryRun),
		executor.WithProgressCallback(func(event executor.ProgressEvent) {
			switch event.Status {
			case executor.StatusStarting:
				fmt.Fprintf(out, "  Applying %s_%s ... ", event.Migration.Version, event.Migration.Name)
			case executor.StatusCompleted:
				fmt.Fprintf(out, "done (%s)\n", event.Duration.Truncate(time.Millisecond))
				applied++
			case executor.StatusSkipped:
				skipped++
			case executor.StatusFailed:
				fmt.Fprintf(out, "FAILED\n")
				fmt.Fprintf(out, "    Error: %v\n", event.Error)
			}
		}),
	)

	if opts.dryRun {
		fmt.Fprintln(out, "\n--- DRY RUN (no changes will be made) ---")
	}

	if err := exec.Apply(ctx, sorted); err != nil {
		return err
	}

	if opts.dryRun {
		fmt.Fprintf(out, "\nDry run complete: %d migration(s) would be applied, %d already applied.\n",
			len(sorted)-skipped, skipped)
	} else {
		fmt.Fprintf(out, "\nApply complete: %d applied, %d skipped.\n", applied, skipped)
	}

	return nil
}

// checkDangerousMigrations runs the analyzer and returns true if
// HIGH/CRITICAL findings were found (blocking apply).
func checkDangerousMigrations(cmd *cobra.Command, sorted []migration.Migration, cfg *config.Config) (bool, error) {
	a := analyzer.New(
		analyzer.WithRegistry(rules.NewDefaultRegistry()),
		analyzer.WithPGVersion(cfg.TargetPGVersion),
	)

	results, err := a.AnalyzeAll(sorted)
	if err != nil {
		return false, fmt.Errorf("analyzing migrations: %w", err)
	}

	hasHighOrCritical := printAnalysisResults(cmd, results)

	return hasHighOrCritical, nil
}
