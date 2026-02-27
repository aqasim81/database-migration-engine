package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestMergeFlags_databaseURL_overridesConfig(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cmd := &cobra.Command{}
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	require.NoError(t, cmd.Flags().Set("database-url", "postgres://test:5432/db"))

	mergeFlags(cmd, cfg)
	assert.Equal(t, "postgres://test:5432/db", cfg.DatabaseURL)
}

func TestMergeFlags_migrationsDir_overridesConfig(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cmd := &cobra.Command{}
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	require.NoError(t, cmd.Flags().Set("migrations-dir", "/custom/migrations"))

	mergeFlags(cmd, cfg)
	assert.Equal(t, "/custom/migrations", cfg.MigrationsDir)
}

func TestMergeFlags_unchangedFlags_preserveConfig(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.DatabaseURL = "postgres://original:5432/db"
	cfg.MigrationsDir = "/original/dir"

	cmd := &cobra.Command{}
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	mergeFlags(cmd, cfg)
	assert.Equal(t, "postgres://original:5432/db", cfg.DatabaseURL)
	assert.Equal(t, "/original/dir", cfg.MigrationsDir)
}

func TestLoadConfig_missingFile_usesDefaults(t *testing.T) { // not parallel: mutates global AppConfig
	old := AppConfig
	t.Cleanup(func() { AppConfig = old })

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "nonexistent.yml", "")
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	err := loadConfig(cmd)
	require.NoError(t, err)
	require.NotNil(t, AppConfig)
	assert.Equal(t, config.DefaultMigrationsDir, AppConfig.MigrationsDir)
	assert.Equal(t, config.DefaultTargetPGVersion, AppConfig.TargetPGVersion)
}

func TestLoadConfig_validFile_loadsValues(t *testing.T) { // not parallel: mutates global AppConfig
	old := AppConfig
	t.Cleanup(func() { AppConfig = old })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yml")

	yamlContent := "migrations_dir: /from/yaml\ntarget_pg_version: 15\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(yamlContent), 0o600))

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	require.NoError(t, cmd.Flags().Set("config", cfgPath))

	err := loadConfig(cmd)
	require.NoError(t, err)
	require.NotNil(t, AppConfig)
	assert.Equal(t, "/from/yaml", AppConfig.MigrationsDir)
	assert.Equal(t, 15, AppConfig.TargetPGVersion)
}

func TestLoadConfig_invalidFile_returnsError(t *testing.T) { // not parallel: mutates global AppConfig
	old := AppConfig
	t.Cleanup(func() { AppConfig = old })

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad-config.yml")

	require.NoError(t, os.WriteFile(cfgPath, []byte("target_pg_version: [unclosed"), 0o600))

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().String("migrations-dir", "", "")

	require.NoError(t, cmd.Flags().Set("config", cfgPath))

	err := loadConfig(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading configuration")
}
