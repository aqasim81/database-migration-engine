package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "status",
	Short: "Show migration status",
	Long: `Display the current migration status showing applied and pending
migrations.`,
	RunE: runStatus,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	statusCmd.Flags().String("format", "text", "output format (text, json)")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "status: not yet implemented")

	return nil
}
