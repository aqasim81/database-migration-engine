package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/executor"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

var rollbackCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "rollback",
	Short: "Roll back applied migrations",
	Long: `Roll back one or more previously applied migrations using their
down migration files.`,
	RunE: runRollback,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	rollbackCmd.Flags().Int("steps", 1, "number of migrations to roll back")
	rollbackCmd.Flags().String("target", "", "roll back to a specific migration version")
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, _ []string) error {
	cfg := AppConfig

	if cfg.DatabaseURL == "" {
		return errDatabaseURLRequired
	}

	steps, _ := cmd.Flags().GetInt("steps")
	target, _ := cmd.Flags().GetString("target")

	sorted, err := loadAndSortMigrations(cfg.MigrationsDir, cmd.OutOrStdout())
	if err != nil || sorted == nil {
		return err
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

	return executeRollback(ctx, cmd.OutOrStdout(), pool, sorted, rollbackOpts{
		steps:            steps,
		target:           target,
		lockTimeout:      cfg.LockTimeout,
		statementTimeout: cfg.StatementTimeout,
	})
}

type rollbackOpts struct {
	steps            int
	target           string
	lockTimeout      time.Duration
	statementTimeout time.Duration
}

func executeRollback(
	ctx context.Context,
	out io.Writer,
	pool *pgxpool.Pool,
	sorted []migration.Migration,
	opts rollbackOpts,
) error {
	t := tracker.New(pool)

	rolledBack := 0

	exec := executor.New(pool, t,
		executor.WithLockTimeout(opts.lockTimeout),
		executor.WithStatementTimeout(opts.statementTimeout),
		executor.WithProgressCallback(func(event executor.ProgressEvent) {
			switch event.Status {
			case executor.StatusRollingBack:
				fmt.Fprintf(out, "  Rolling back %s_%s ... ",
					event.Migration.Version, event.Migration.Name)
			case executor.StatusCompleted:
				fmt.Fprintf(out, "done (%s)\n",
					event.Duration.Truncate(time.Millisecond))
				rolledBack++
			case executor.StatusFailed:
				fmt.Fprintf(out, "FAILED\n")
				fmt.Fprintf(out, "    Error: %v\n", event.Error)
			case executor.StatusSkipped:
				// dry-run placeholder
			}
		}),
	)

	var err error

	if opts.target != "" {
		err = exec.RollbackToVersion(ctx, sorted, opts.target)
	} else {
		err = exec.Rollback(ctx, sorted, opts.steps)
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\nRollback complete: %d migration(s) rolled back.\n", rolledBack)

	return nil
}
