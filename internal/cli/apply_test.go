package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestLoadAndSortMigrations_validDir_returnsSorted(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	sorted, err := loadAndSortMigrations("./testdata/migrations", buf)

	require.NoError(t, err)
	require.NotNil(t, sorted)
	require.Len(t, sorted, 2)
	assert.Equal(t, "001", sorted[0].Version)
	assert.Equal(t, "002", sorted[1].Version)
}

func TestLoadAndSortMigrations_emptyDir_returnsNil(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	sorted, err := loadAndSortMigrations(t.TempDir(), buf)

	require.NoError(t, err)
	assert.Nil(t, sorted)
	assert.Contains(t, buf.String(), "No migration files found")
}

func TestLoadAndSortMigrations_invalidDir_returnsError(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	_, err := loadAndSortMigrations("/nonexistent/path", buf)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading migrations")
}

func TestCheckDangerousMigrations_safeSQL_returnsFalse(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	safe := sorted[:1]
	blocked, err := checkDangerousMigrations(cmd, safe, cfg)

	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestCheckDangerousMigrations_dangerousSQL_returnsTrue(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	blocked, err := checkDangerousMigrations(cmd, sorted, cfg)

	require.NoError(t, err)
	assert.True(t, blocked)
}

// Tests below write to the global AppConfig — they must NOT be parallel.

func TestRunApply_noMigrations_printsMessage(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		DatabaseURL:   "postgres://test:test@localhost/test",
		MigrationsDir: t.TempDir(),
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No migration files found")
}

func TestRunApply_dangerousMigrations_blocked(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		DatabaseURL:      "postgres://test:test@localhost/test",
		MigrationsDir:    "./testdata/migrations",
		TargetPGVersion:  14,
		LockTimeout:      5000000000,
		StatementTimeout: 30000000000,
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, errDangerousMigrations)
}
