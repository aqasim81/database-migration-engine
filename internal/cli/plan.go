package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "plan",
	Short: "Show execution plan for pending migrations",
	Long: `Display the execution plan for pending migrations including
analysis results, estimated impact, and execution order.`,
	RunE: runPlan,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	planCmd.Flags().Bool("pending-only", false, "show only pending migrations")
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "plan: not yet implemented")

	return nil
}
