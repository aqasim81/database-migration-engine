package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestRunPlan_printsNotImplemented(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runPlan(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "plan: not yet implemented")
}

func TestRunApply_noDatabaseURL_returnsError(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{MigrationsDir: "./testdata/migrations"}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errDatabaseURLRequired)
}

func TestRunStatus_printsNotImplemented(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runStatus(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "status: not yet implemented")
}

func TestRunRollback_printsNotImplemented(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runRollback(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "rollback: not yet implemented")
}
