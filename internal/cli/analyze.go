package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/migration"
)

var analyzeCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "analyze [migration-dir]",
	Short: "Analyze migrations for dangerous operations",
	Long: `Analyze SQL migration files for dangerous DDL operations that could
cause table locks, downtime, or data loss. Reports findings with severity
levels and suggests safe alternatives.`,
	RunE: runAnalyze,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	analyzeCmd.Flags().String("format", "text", "output format (text, json, github-actions)")
	analyzeCmd.Flags().Bool("fail-on-high", false, "exit with non-zero code if high/critical findings exist")
	rootCmd.AddCommand(analyzeCmd)
}

// errHighSeverityFindings is returned when --fail-on-high is set and high/critical findings exist.
var errHighSeverityFindings = errors.New("high or critical severity findings detected")

func runAnalyze(cmd *cobra.Command, args []string) error {
	dir := AppConfig.MigrationsDir
	if len(args) > 0 {
		dir = args[0]
	}

	migrations, err := migration.LoadFromDir(dir)
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No migration files found.")
		return nil
	}

	sorted := migration.Sort(migrations)

	a := analyzer.New(
		analyzer.WithRegistry(rules.NewDefaultRegistry()),
		analyzer.WithPGVersion(AppConfig.TargetPGVersion),
	)

	results, err := a.AnalyzeAll(sorted)
	if err != nil {
		return fmt.Errorf("analyzing migrations: %w", err)
	}

	hasHighOrCritical := printAnalysisResults(cmd, results)

	failOnHigh, _ := cmd.Flags().GetBool("fail-on-high")
	if failOnHigh && hasHighOrCritical {
		return errHighSeverityFindings
	}

	return nil
}

func printAnalysisResults(cmd *cobra.Command, results []analyzer.AnalysisResult) bool {
	out := cmd.OutOrStdout()
	totalFindings := 0
	hasHighOrCritical := false

	for _, r := range results {
		if len(r.Findings) == 0 {
			continue
		}

		fmt.Fprintf(out, "\n=== %s_%s ===\n", r.Migration.Version, r.Migration.Name)

		for _, f := range r.Findings {
			fmt.Fprintf(out, "  [%s] %s\n", f.Severity, f.Message)
			fmt.Fprintf(out, "    Table: %s\n", f.Table)
			fmt.Fprintf(out, "    Rule:  %s\n", f.Rule)

			if f.Statement != "" {
				fmt.Fprintf(out, "    SQL:   %s\n", f.Statement)
			}

			fmt.Fprintf(out, "    Fix:   %s\n\n", f.Suggestion)
		}

		totalFindings += len(r.Findings)

		if r.HasHighOrCritical() {
			hasHighOrCritical = true
		}
	}

	if totalFindings == 0 {
		fmt.Fprintln(out, "No dangerous operations detected.")
	} else {
		fmt.Fprintf(out, "Found %d finding(s) across %d migration(s).\n", totalFindings, countMigrationsWithFindings(results))
	}

	return hasHighOrCritical
}

func countMigrationsWithFindings(results []analyzer.AnalysisResult) int {
	count := 0

	for _, r := range results {
		if len(r.Findings) > 0 {
			count++
		}
	}

	return count
}
