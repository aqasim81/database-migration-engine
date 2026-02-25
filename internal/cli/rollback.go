package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Fprintln(cmd.OutOrStdout(), "rollback: not yet implemented")

	return nil
}
