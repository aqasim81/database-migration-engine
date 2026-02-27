package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

const version = "0.1.0"

// AppConfig holds the loaded configuration, set during PersistentPreRunE.
var AppConfig *config.Config //nolint:gochecknoglobals // standard Cobra pattern for shared config

// rootCmd is the base command for the migrate CLI.
var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:     "migrate",
	Version: version,
	Short:   "Zero-downtime PostgreSQL schema migration CLI",
	Long: `migrate analyzes SQL migrations using the real PostgreSQL parser,
detects dangerous DDL operations that cause table locks and outages,
suggests safe alternatives, and executes migrations with rollback capability.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return loadConfig(cmd)
	},
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	rootCmd.PersistentFlags().String("config", "migrate.yml", "path to configuration file")
	rootCmd.PersistentFlags().String("database-url", "", "PostgreSQL connection string")
	rootCmd.PersistentFlags().String("migrations-dir", "", "path to migration files")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")
}

// Execute runs the root command. Called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig loads configuration with precedence: flag > env > file.
func loadConfig(cmd *cobra.Command) error {
	configPath, _ := cmd.Flags().GetString("config")
	allowMissing := !cmd.Flags().Changed("config")

	cfg, err := config.Load(configPath, allowMissing)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	config.MergeEnv(cfg)
	mergeFlags(cmd, cfg)

	AppConfig = cfg

	return nil
}

// mergeFlags overrides config with explicitly-set CLI flags.
func mergeFlags(cmd *cobra.Command, cfg *config.Config) {
	if cmd.Flags().Changed("database-url") {
		cfg.DatabaseURL, _ = cmd.Flags().GetString("database-url")
	}

	if cmd.Flags().Changed("migrations-dir") {
		cfg.MigrationsDir, _ = cmd.Flags().GetString("migrations-dir")
	}
}
