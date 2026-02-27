package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/migration"
)

// setupTestConfig sets AppConfig for the duration of the test and restores it on cleanup.
func setupTestConfig(t *testing.T, migrationsDir string) {
	t.Helper()

	old := AppConfig
	AppConfig = &config.Config{
		MigrationsDir:   migrationsDir,
		TargetPGVersion: config.DefaultTargetPGVersion,
	}

	t.Cleanup(func() { AppConfig = old })
}

// newAnalyzeCmd creates a fresh cobra.Command wired to runAnalyze with a captured output buffer.
func newAnalyzeCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{
		Use:  "analyze [migration-dir]",
		RunE: runAnalyze,
	}
	cmd.Flags().Bool("fail-on-high", false, "exit with non-zero code if high/critical findings exist")
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	return cmd, buf
}

func TestCountMigrationsWithFindings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		results  []analyzer.AnalysisResult
		expected int
	}{
		{
			name:     "empty results",
			results:  nil,
			expected: 0,
		},
		{
			name: "no findings",
			results: []analyzer.AnalysisResult{
				{Migration: &migration.Migration{Version: "001"}, Findings: nil},
			},
			expected: 0,
		},
		{
			name: "one with findings",
			results: []analyzer.AnalysisResult{
				{Migration: &migration.Migration{Version: "001"}, Findings: nil},
				{Migration: &migration.Migration{Version: "002"}, Findings: []analyzer.Finding{{Rule: "test"}}},
			},
			expected: 1,
		},
		{
			name: "all with findings",
			results: []analyzer.AnalysisResult{
				{Migration: &migration.Migration{Version: "001"}, Findings: []analyzer.Finding{{Rule: "a"}}},
				{Migration: &migration.Migration{Version: "002"}, Findings: []analyzer.Finding{{Rule: "b"}}},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, countMigrationsWithFindings(tt.results))
		})
	}
}

func TestPrintAnalysisResults_noFindings_printsNoDangers(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	results := []analyzer.AnalysisResult{
		{Migration: &migration.Migration{Version: "001", Name: "safe"}, Findings: nil},
	}

	hasHigh := printAnalysisResults(cmd, results)
	assert.False(t, hasHigh)
	assert.Contains(t, buf.String(), "No dangerous operations detected.")
}

func TestPrintAnalysisResults_withFindings_formatsOutput(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	results := []analyzer.AnalysisResult{
		{
			Migration:   &migration.Migration{Version: "001", Name: "dangerous"},
			MaxSeverity: analyzer.High,
			Findings: []analyzer.Finding{
				{
					Rule:       "create-index-not-concurrent",
					Severity:   analyzer.High,
					Table:      "users",
					Statement:  "CREATE INDEX idx ON users (email)",
					Message:    "Index creation locks table",
					Suggestion: "Use CREATE INDEX CONCURRENTLY",
				},
			},
		},
	}

	hasHigh := printAnalysisResults(cmd, results)
	assert.True(t, hasHigh)

	output := buf.String()
	assert.Contains(t, output, "=== 001_dangerous ===")
	assert.Contains(t, output, "[HIGH]")
	assert.Contains(t, output, "Table: users")
	assert.Contains(t, output, "Rule:  create-index-not-concurrent")
	assert.Contains(t, output, "SQL:   CREATE INDEX idx ON users (email)")
	assert.Contains(t, output, "Fix:   Use CREATE INDEX CONCURRENTLY")
	assert.Contains(t, output, "Found 1 finding(s) across 1 migration(s).")
}

func TestPrintAnalysisResults_lowSeverityOnly_returnsFalse(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	results := []analyzer.AnalysisResult{
		{
			Migration:   &migration.Migration{Version: "001", Name: "mild"},
			MaxSeverity: analyzer.Low,
			Findings: []analyzer.Finding{
				{Rule: "test-rule", Severity: analyzer.Low, Message: "minor concern"},
			},
		},
	}

	hasHigh := printAnalysisResults(cmd, results)
	assert.False(t, hasHigh)
	assert.Contains(t, buf.String(), "Found 1 finding(s)")
}

func TestPrintAnalysisResults_noStatement_skipsSQL(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	results := []analyzer.AnalysisResult{
		{
			Migration:   &migration.Migration{Version: "001", Name: "test"},
			MaxSeverity: analyzer.Medium,
			Findings: []analyzer.Finding{
				{Rule: "test-rule", Severity: analyzer.Medium, Message: "test", Statement: ""},
			},
		},
	}

	printAnalysisResults(cmd, results)
	assert.NotContains(t, buf.String(), "SQL:")
}

func TestRunAnalyze_withTestdata_producesOutput(t *testing.T) { // not parallel: mutates global AppConfig
	dir := filepath.Join("testdata", "migrations")
	setupTestConfig(t, dir)

	cmd, buf := newAnalyzeCmd(t)
	cmd.SetArgs([]string{dir})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "finding(s)")
}

func TestRunAnalyze_emptyDir_printsNoMigrations(t *testing.T) { // not parallel: mutates global AppConfig
	dir := t.TempDir()
	setupTestConfig(t, dir)

	cmd, buf := newAnalyzeCmd(t)
	cmd.SetArgs([]string{dir})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No migration files found.")
}

func TestRunAnalyze_invalidDir_returnsError(t *testing.T) { // not parallel: mutates global AppConfig
	dir := "/nonexistent/path/to/migrations"
	setupTestConfig(t, dir)

	cmd, _ := newAnalyzeCmd(t)
	cmd.SetArgs([]string{dir})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading migrations")
}

func TestRunAnalyze_failOnHigh_returnsError(t *testing.T) { // not parallel: mutates global AppConfig
	dir := filepath.Join("testdata", "migrations")
	setupTestConfig(t, dir)

	cmd, _ := newAnalyzeCmd(t)
	cmd.SetArgs([]string{"--fail-on-high", dir})

	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errHighSeverityFindings)
}

func TestRunAnalyze_usesConfigDir_whenNoArgs(t *testing.T) { // not parallel: mutates global AppConfig
	dir := filepath.Join("testdata", "migrations")
	setupTestConfig(t, dir)

	cmd, buf := newAnalyzeCmd(t)

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "finding(s)")
}
