package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
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

	a := newDefaultAnalyzer(AppConfig.TargetPGVersion)

	results, err := a.AnalyzeAll(sorted)
	if err != nil {
		return fmt.Errorf("analyzing migrations: %w", err)
	}

	hasHighOrCritical, err := outputAnalysisResults(cmd, results)
	if err != nil {
		return err
	}

	failOnHigh, _ := cmd.Flags().GetBool("fail-on-high")
	if failOnHigh && hasHighOrCritical {
		return errHighSeverityFindings
	}

	return nil
}

func outputAnalysisResults(cmd *cobra.Command, results []analyzer.AnalysisResult) (bool, error) {
	format := getFormat(cmd)

	switch format {
	case FormatJSON:
		return printAnalysisJSON(cmd.OutOrStdout(), results)
	case FormatGitHubActions:
		return printAnalysisGitHub(cmd.OutOrStdout(), results), nil
	default:
		return printAnalysisText(cmd.OutOrStdout(), results), nil
	}
}

func printAnalysisText(out io.Writer, results []analyzer.AnalysisResult) bool {
	totalFindings := 0
	hasHighOrCritical := false

	for _, r := range results {
		if len(r.Findings) == 0 {
			continue
		}

		fmt.Fprintf(out, "\n=== %s_%s ===\n", r.Migration.Version, r.Migration.Name)

		for _, f := range r.Findings {
			fmt.Fprintf(out, "  [%s] %s\n", colorSeverity(f.Severity), f.Message)
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

func printAnalysisJSON(out io.Writer, results []analyzer.AnalysisResult) (bool, error) {
	output := AnalyzeJSONOutput{
		Migrations: make([]AnalyzeJSONMigration, 0, len(results)),
	}

	for _, r := range results {
		m := AnalyzeJSONMigration{
			Version:  r.Migration.Version,
			Name:     r.Migration.Name,
			FilePath: r.Migration.FilePath,
			Findings: make([]AnalyzeJSONFinding, 0, len(r.Findings)),
		}

		for i := range r.Findings {
			m.Findings = append(m.Findings, findingToJSON(&r.Findings[i]))
		}

		output.Migrations = append(output.Migrations, m)
		output.TotalFindings += len(r.Findings)

		if r.HasHighOrCritical() {
			output.HasHighOrCrit = true
		}
	}

	return output.HasHighOrCrit, printJSON(out, output)
}

func printAnalysisGitHub(out io.Writer, results []analyzer.AnalysisResult) bool {
	hasHighOrCritical := false

	for _, r := range results {
		for i := range r.Findings {
			fmt.Fprintln(out, formatGitHubAnnotation(r.Migration.FilePath, &r.Findings[i]))
		}

		if r.HasHighOrCritical() {
			hasHighOrCritical = true
		}
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
