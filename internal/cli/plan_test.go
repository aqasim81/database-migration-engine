package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/planner"
)

func TestRunPlan_withTestdata(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		MigrationsDir:   "./testdata/migrations",
		TargetPGVersion: 14,
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runPlan(cmd, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Migration Plan")
	assert.Contains(t, output, "pending")
	assert.Contains(t, output, "Summary:")
}

func TestRunPlan_emptyDir(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		MigrationsDir:   t.TempDir(),
		TargetPGVersion: 14,
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runPlan(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No migration files found")
}

func TestRunPlan_invalidDir(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		MigrationsDir:   "/nonexistent/path",
		TargetPGVersion: 14,
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runPlan(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading migrations")
}

func TestRunPlan_pendingOnly(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	AppConfig = &config.Config{
		MigrationsDir:   "./testdata/migrations",
		TargetPGVersion: 14,
	}

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.Flags().Bool("pending-only", false, "")
	require.NoError(t, cmd.Flags().Set("pending-only", "true"))

	err := runPlan(cmd, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Migration Plan")
	assert.Contains(t, output, "pending")
	assert.NotContains(t, output, "applied")
}

func TestPrintPlan_formatsCorrectly(t *testing.T) {
	t.Parallel()

	plan := &planner.Plan{
		Steps: []planner.MigrationStep{
			{
				Migration: &migration.Migration{Version: "001", Name: "create_users"},
				Status:    planner.StatusApplied,
				RunInTx:   true,
			},
			{
				Migration: &migration.Migration{Version: "002", Name: "add_email_index"},
				Status:    planner.StatusPending,
				Findings: []analyzer.Finding{
					{Severity: analyzer.High, Rule: "create-index-not-concurrent"},
				},
				Impacts: []planner.Impact{
					{EstimatedLockDuration: planner.DurationOver5min},
				},
				RunInTx: true,
			},
			{
				Migration: &migration.Migration{Version: "003", Name: "add_column"},
				Status:    planner.StatusPending,
				RunInTx:   true,
			},
		},
		TotalPending:  2,
		HighRiskCount: 1,
	}

	buf := new(bytes.Buffer)
	printPlan(buf, plan)
	output := buf.String()

	assert.Contains(t, output, "Migration Plan")
	assert.Contains(t, output, "001")
	assert.Contains(t, output, "create_users")
	assert.Contains(t, output, "applied")
	assert.Contains(t, output, "002")
	assert.Contains(t, output, "HIGH")
	assert.Contains(t, output, "> 5min")
	assert.Contains(t, output, "2 pending")
	assert.Contains(t, output, "1 HIGH risk")
}

func TestPrintPlan_emptyPlan(t *testing.T) {
	t.Parallel()

	plan := &planner.Plan{}

	buf := new(bytes.Buffer)
	printPlan(buf, plan)
	output := buf.String()

	assert.Contains(t, output, "Migration Plan")
	assert.Contains(t, output, "No migrations found")
	assert.Contains(t, output, "Summary:")
	assert.Contains(t, output, "0 pending")
}

func TestPrintPlanSummary_pendingOnly(t *testing.T) {
	t.Parallel()

	plan := &planner.Plan{TotalPending: 3}

	buf := new(bytes.Buffer)
	printPlanSummary(buf, plan)

	assert.Equal(t, "Summary: 3 pending\n", buf.String())
}

func TestPrintPlanSummary_withHighAndCritical(t *testing.T) {
	t.Parallel()

	plan := &planner.Plan{
		TotalPending:  5,
		HighRiskCount: 2,
		CriticalCount: 1,
	}

	buf := new(bytes.Buffer)
	printPlanSummary(buf, plan)

	assert.Contains(t, buf.String(), "5 pending")
	assert.Contains(t, buf.String(), "2 HIGH risk")
	assert.Contains(t, buf.String(), "1 CRITICAL risk")
}

func TestMaxLockDuration_returnsHighest(t *testing.T) {
	t.Parallel()

	impacts := []planner.Impact{
		{EstimatedLockDuration: planner.Duration1to10s},
		{EstimatedLockDuration: planner.DurationOver5min},
		{EstimatedLockDuration: planner.DurationUnder1s},
	}

	assert.Equal(t, planner.DurationOver5min, maxLockDuration(impacts))
}

func TestMaxLockDuration_empty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-", maxLockDuration(nil))
}

func TestTruncateName_short(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "short", truncateName("short", 22))
}

func TestTruncateName_long(t *testing.T) {
	t.Parallel()

	result := truncateName("this_is_a_very_long_migration_name", 22)
	assert.Len(t, result, 22)
	assert.Contains(t, result, "...")
}
