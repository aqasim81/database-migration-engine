package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Fprintln(cmd.OutOrStdout(), "apply: not yet implemented")

	return nil
}
