package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestRunApply_noDatabaseURL_returnsError(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{MigrationsDir: "./testdata/migrations"}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errDatabaseURLRequired)
}

func TestRunRollback_noDatabaseURL_returnsError(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{MigrationsDir: "./testdata/migrations"}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runRollback(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errDatabaseURLRequired)
}

func TestRunRollback_noMigrations_printsMessage(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		DatabaseURL:   "postgres://test:test@localhost/test",
		MigrationsDir: t.TempDir(),
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runRollback(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No migration files found")
}
