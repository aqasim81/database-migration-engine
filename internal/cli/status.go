package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

var statusCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "status",
	Short: "Show migration status",
	Long: `Display the current migration status showing applied and pending
migrations. Requires a database connection.`,
	RunE: runStatus,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	statusCmd.Flags().String("format", "text", "output format (text, json)")
	rootCmd.AddCommand(statusCmd)
}

// statusEntry holds the cross-referenced status of a single migration.
type statusEntry struct {
	Version       string
	Name          string
	Status        string // statusApplied, statusPending, statusMismatch
	AppliedAt     time.Time
	DurationMs    int
	ChecksumMatch *bool // nil if not applied
}

func runStatus(cmd *cobra.Command, _ []string) error {
	cfg := AppConfig

	if cfg.DatabaseURL == "" {
		return errDatabaseURLRequired
	}

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

	t := tracker.New(pool)
	if err = t.EnsureTable(ctx); err != nil {
		return fmt.Errorf("ensuring migrations table: %w", err)
	}

	entries, err := buildStatusEntries(ctx, t, sorted)
	if err != nil {
		return err
	}

	format := getFormat(cmd)

	switch format {
	case FormatJSON:
		return printStatusJSON(cmd.OutOrStdout(), entries)
	default:
		printStatusText(cmd.OutOrStdout(), entries)
		return nil
	}
}

func buildStatusEntries(
	ctx context.Context,
	t *tracker.Tracker,
	sorted []migration.Migration,
) ([]statusEntry, error) {
	applied, err := t.GetApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting applied migrations: %w", err)
	}

	appliedMap := make(map[string]*tracker.AppliedMigration, len(applied))
	for i := range applied {
		appliedMap[applied[i].Version] = &applied[i]
	}

	entries := make([]statusEntry, 0, len(sorted))

	for i := range sorted {
		m := &sorted[i]
		entry := statusEntry{
			Version: m.Version,
			Name:    m.Name,
			Status:  statusPending,
		}

		if am, ok := appliedMap[m.Version]; ok {
			entry.Status = statusApplied
			entry.AppliedAt = am.AppliedAt
			entry.DurationMs = am.DurationMs

			match := am.Checksum == m.Checksum
			entry.ChecksumMatch = &match

			if !match {
				entry.Status = statusMismatch
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func printStatusText(out io.Writer, entries []statusEntry) {
	fmt.Fprintln(out, "Migration Status")
	fmt.Fprintln(out, "==================================================")
	fmt.Fprintln(out)

	if len(entries) == 0 {
		fmt.Fprintln(out, "No migrations found.")
		printStatusSummary(out, entries)

		return
	}

	fmt.Fprintf(out, "%-10s %-22s %-12s %-22s %s\n",
		"Version", "Name", "Status", "Applied At", "Duration")
	fmt.Fprintf(out, "%-10s %-22s %-12s %-22s %s\n",
		"-------", "----", "------", "----------", "--------")

	var mismatches []statusEntry

	for _, e := range entries {
		appliedAt := "-"
		duration := "-"

		if e.Status != statusPending {
			appliedAt = e.AppliedAt.Format("2006-01-02 15:04:05")
			duration = formatDurationMs(e.DurationMs)
		}

		fmt.Fprintf(out, "%-10s %-22s %-12s %-22s %s\n",
			e.Version,
			truncateName(e.Name, 22), //nolint:mnd // column width
			colorStatus(e.Status),
			appliedAt,
			duration,
		)

		if e.Status == statusMismatch {
			mismatches = append(mismatches, e)
		}
	}

	fmt.Fprintln(out)

	for _, e := range mismatches {
		fmt.Fprintf(out, "WARNING: %s checksum mismatch! File was modified after applying.\n", e.Version)
	}

	printStatusSummary(out, entries)
}

func printStatusSummary(out io.Writer, entries []statusEntry) {
	applied := 0
	pending := 0
	mismatches := 0

	for _, e := range entries {
		switch e.Status {
		case statusApplied:
			applied++
		case statusPending:
			pending++
		case statusMismatch:
			mismatches++
		}
	}

	fmt.Fprintf(out, "Summary: %d applied, %d pending", applied, pending)

	if mismatches > 0 {
		fmt.Fprintf(out, ", %d checksum mismatch(es)", mismatches)
	}

	fmt.Fprintln(out)
}

func printStatusJSON(out io.Writer, entries []statusEntry) error {
	output := StatusJSONOutput{
		Total:      len(entries),
		Migrations: make([]StatusJSONMigration, 0, len(entries)),
	}

	for _, e := range entries {
		m := StatusJSONMigration{
			Version: e.Version,
			Name:    e.Name,
			Status:  e.Status,
		}

		if e.Status != statusPending {
			m.AppliedAt = e.AppliedAt.Format(time.RFC3339)
			m.DurationMs = e.DurationMs
			m.ChecksumMatch = e.ChecksumMatch
		}

		output.Migrations = append(output.Migrations, m)

		switch e.Status {
		case statusApplied:
			output.Applied++
		case statusPending:
			output.Pending++
		case statusMismatch:
			output.Mismatches++
		}
	}

	return printJSON(out, output)
}

func formatDurationMs(ms int) string {
	if ms < 1000 { //nolint:mnd // 1000ms = 1s
		return fmt.Sprintf("%dms", ms)
	}

	return fmt.Sprintf("%.1fs", float64(ms)/1000) //nolint:mnd // convert ms to seconds
}

func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	return name[:maxLen-3] + "..."
}
