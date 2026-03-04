package planner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/planner"
)

type mockTableSizer struct {
	sizes map[string]int64
}

func (m *mockTableSizer) TableSizeBytes(_ context.Context, name string) (int64, error) {
	if size, ok := m.sizes[name]; ok {
		return size, nil
	}

	return -1, nil
}

func testMigrations() []migration.Migration {
	return []migration.Migration{
		{Version: "001", Name: "create_users", UpSQL: "CREATE TABLE users (id serial);"},
		{Version: "002", Name: "add_index", UpSQL: "CREATE INDEX idx_users_email ON users (email);"},
		{Version: "003", Name: "add_column", UpSQL: "ALTER TABLE users ADD COLUMN name text;"},
	}
}

func testResults(migrations []migration.Migration) []analyzer.AnalysisResult {
	return []analyzer.AnalysisResult{
		{Migration: &migrations[0], Findings: nil, MaxSeverity: analyzer.Safe},
		{
			Migration:   &migrations[1],
			MaxSeverity: analyzer.High,
			Findings: []analyzer.Finding{
				{
					Rule:     "create-index-not-concurrent",
					Severity: analyzer.High,
					Table:    "users",
					LockType: "SHARE",
				},
			},
		},
		{Migration: &migrations[2], Findings: nil, MaxSeverity: analyzer.Safe},
	}
}

func TestBuildPlan_allPending(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.Len(t, plan.Steps, 3)
	assert.Equal(t, 3, plan.TotalPending)

	for _, step := range plan.Steps {
		assert.Equal(t, planner.StatusPending, step.Status)
	}
}

func TestBuildPlan_mixedStatus(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{"001": true},
	})

	require.NoError(t, err)
	assert.Equal(t, planner.StatusApplied, plan.Steps[0].Status)
	assert.Equal(t, planner.StatusPending, plan.Steps[1].Status)
	assert.Equal(t, planner.StatusPending, plan.Steps[2].Status)
	assert.Equal(t, 2, plan.TotalPending)
}

func TestBuildPlan_noFindings(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "create_users", UpSQL: "CREATE TABLE users (id serial);"},
	}
	results := []analyzer.AnalysisResult{
		{Migration: &migrations[0], Findings: nil, MaxSeverity: analyzer.Safe},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.Empty(t, plan.Steps[0].Findings)
	assert.Empty(t, plan.Steps[0].Impacts)
	assert.Equal(t, analyzer.Safe, plan.Steps[0].MaxSeverity())
}

func TestBuildPlan_highRiskCounting(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, plan.HighRiskCount)
	assert.Equal(t, 0, plan.CriticalCount)
}

func TestBuildPlan_criticalRiskCounting(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "drop_users", UpSQL: "DROP TABLE users;"},
	}
	results := []analyzer.AnalysisResult{
		{
			Migration:   &migrations[0],
			MaxSeverity: analyzer.Critical,
			Findings: []analyzer.Finding{
				{
					Rule:     "drop-table",
					Severity: analyzer.Critical,
					Table:    "users",
					LockType: "ACCESS EXCLUSIVE",
				},
			},
		},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, plan.HighRiskCount)
	assert.Equal(t, 1, plan.CriticalCount)
}

func TestBuildPlan_concurrentMigration_runInTxFalse(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "add_index", UpSQL: "CREATE INDEX CONCURRENTLY idx ON users (email);"},
	}
	results := []analyzer.AnalysisResult{
		{Migration: &migrations[0], MaxSeverity: analyzer.Safe},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.False(t, plan.Steps[0].RunInTx)
}

func TestBuildPlan_regularMigration_runInTxTrue(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "create_users", UpSQL: "CREATE TABLE users (id serial);"},
	}
	results := []analyzer.AnalysisResult{
		{Migration: &migrations[0], MaxSeverity: analyzer.Safe},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.True(t, plan.Steps[0].RunInTx)
}

func TestBuildPlan_withTableSizer(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)
	sizer := &mockTableSizer{sizes: map[string]int64{"users": sizeLarge}}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
		Sizer:      sizer,
	})

	require.NoError(t, err)
	require.Len(t, plan.Steps[1].Impacts, 1)
	assert.Equal(t, sizeLarge, plan.Steps[1].Impacts[0].TableSizeBytes)
	assert.Equal(t, planner.DurationOver5min, plan.Steps[1].Impacts[0].EstimatedLockDuration)
}

func TestBuildPlan_nilSizer_usesUnknownSize(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	require.Len(t, plan.Steps[1].Impacts, 1)
	assert.Equal(t, int64(-1), plan.Steps[1].Impacts[0].TableSizeBytes)
}

func TestBuildPlan_emptyMigrations(t *testing.T) {
	t.Parallel()

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: nil,
		Results:    nil,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.Empty(t, plan.Steps)
	assert.Equal(t, 0, plan.TotalPending)
}

func TestPendingOnly_filtersApplied(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{"001": true},
	})
	require.NoError(t, err)

	filtered := planner.PendingOnly(plan)

	assert.Len(t, filtered.Steps, 2)
	assert.Equal(t, 2, filtered.TotalPending)

	for _, step := range filtered.Steps {
		assert.Equal(t, planner.StatusPending, step.Status)
	}
}

func TestPendingOnly_emptyPlan(t *testing.T) {
	t.Parallel()

	plan := &planner.Plan{}

	filtered := planner.PendingOnly(plan)

	assert.Empty(t, filtered.Steps)
	assert.Equal(t, 0, filtered.TotalPending)
}

func TestPendingOnly_preservesRiskCounts(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{"001": true},
	})
	require.NoError(t, err)

	filtered := planner.PendingOnly(plan)

	assert.Equal(t, 1, filtered.HighRiskCount)
}

func TestMigrationStep_MaxSeverity_returnsHighest(t *testing.T) {
	t.Parallel()

	step := planner.MigrationStep{
		Findings: []analyzer.Finding{
			{Severity: analyzer.Low},
			{Severity: analyzer.High},
			{Severity: analyzer.Medium},
		},
	}

	assert.Equal(t, analyzer.High, step.MaxSeverity())
}

func TestMigrationStep_MaxSeverity_noFindings(t *testing.T) {
	t.Parallel()

	step := planner.MigrationStep{}

	assert.Equal(t, analyzer.Safe, step.MaxSeverity())
}

func TestBuildPlan_dropIndexConcurrently_runInTxFalse(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "drop_index", UpSQL: "DROP INDEX CONCURRENTLY idx_users_email;"},
	}
	results := []analyzer.AnalysisResult{
		{Migration: &migrations[0], MaxSeverity: analyzer.Safe},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.False(t, plan.Steps[0].RunInTx)
}

func TestBuildPlan_noConcurrentlyKeyword_skipsParser(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "simple", UpSQL: "CREATE TABLE test (id int);"},
	}
	results := []analyzer.AnalysisResult{
		{Migration: &migrations[0], MaxSeverity: analyzer.Safe},
	}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
	})

	require.NoError(t, err)
	assert.True(t, plan.Steps[0].RunInTx)
}

func TestBuildPlan_sizerError_fallsBackToUnknown(t *testing.T) {
	t.Parallel()

	migrations := testMigrations()
	results := testResults(migrations)
	sizer := &errorTableSizer{}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
		Sizer:      sizer,
	})

	require.NoError(t, err)
	require.Len(t, plan.Steps[1].Impacts, 1)
	assert.Equal(t, int64(-1), plan.Steps[1].Impacts[0].TableSizeBytes)
}

type errorTableSizer struct{}

func (e *errorTableSizer) TableSizeBytes(_ context.Context, _ string) (int64, error) {
	return -1, errors.New("connection error")
}

func TestBuildPlan_emptyTableName_usesUnknownSize(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "custom", UpSQL: "SELECT 1;"},
	}
	results := []analyzer.AnalysisResult{
		{
			Migration:   &migrations[0],
			MaxSeverity: analyzer.Low,
			Findings: []analyzer.Finding{
				{Rule: "custom-rule", Severity: analyzer.Low, Table: ""},
			},
		},
	}

	sizer := &mockTableSizer{sizes: map[string]int64{"users": 100}}

	plan, err := planner.BuildPlan(context.Background(), &planner.BuildPlanParams{
		Migrations: migrations,
		Results:    results,
		Applied:    map[string]bool{},
		Sizer:      sizer,
	})

	require.NoError(t, err)
	require.Len(t, plan.Steps[0].Impacts, 1)
	assert.Equal(t, int64(-1), plan.Steps[0].Impacts[0].TableSizeBytes)
}
