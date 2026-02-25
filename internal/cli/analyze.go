package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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

func runAnalyze(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "analyze: not yet implemented")

	return nil
}
