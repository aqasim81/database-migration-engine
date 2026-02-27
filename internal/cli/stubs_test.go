package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRunApply_printsNotImplemented(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "apply: not yet implemented")
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
