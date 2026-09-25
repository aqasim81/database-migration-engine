package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func boolPtr(b bool) *bool { return &b }

func testStatusEntries() []statusEntry {
	return []statusEntry{
		{
			Version:       "001",
			Name:          "create_users",
			Status:        statusApplied,
			AppliedAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			DurationMs:    45,
			ChecksumMatch: boolPtr(true),
		},
		{
			Version: "002",
			Name:    "add_email_index",
			Status:  statusPending,
		},
		{
			Version:       "003",
			Name:          "add_column",
			Status:        statusMismatch,
			AppliedAt:     time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC),
			DurationMs:    2300,
			ChecksumMatch: boolPtr(false),
		},
	}
}

func TestPrintStatusText_formatsCorrectly(t *testing.T) {
	t.Parallel()

	entries := testStatusEntries()
	buf := new(bytes.Buffer)

	printStatusText(buf, entries)
	output := buf.String()

	assert.Contains(t, output, "Migration Status")
	assert.Contains(t, output, "001")
	assert.Contains(t, output, "create_users")
	assert.Contains(t, output, "applied")
	assert.Contains(t, output, "2024-01-15 10:30:00")
	assert.Contains(t, output, "45ms")
	assert.Contains(t, output, "002")
	assert.Contains(t, output, "pending")
	assert.Contains(t, output, "003")
	assert.Contains(t, output, "mismatch")
	assert.Contains(t, output, "2.3s")
	assert.Contains(t, output, "WARNING: 003 checksum mismatch")
	assert.Contains(t, output, "1 applied")
	assert.Contains(t, output, "1 pending")
	assert.Contains(t, output, "1 checksum mismatch")
}

func TestPrintStatusText_empty(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	printStatusText(buf, nil)
	output := buf.String()

	assert.Contains(t, output, "Migration Status")
	assert.Contains(t, output, "No migrations found.")
	assert.Contains(t, output, "0 applied, 0 pending")
}

func TestPrintStatusJSON_validStructure(t *testing.T) {
	t.Parallel()

	entries := testStatusEntries()
	buf := new(bytes.Buffer)

	err := printStatusJSON(buf, entries)
	require.NoError(t, err)

	var output StatusJSONOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 1, output.Applied)
	assert.Equal(t, 1, output.Pending)
	assert.Equal(t, 1, output.Mismatches)
	assert.Len(t, output.Migrations, 3)

	assert.Equal(t, "001", output.Migrations[0].Version)
	assert.Equal(t, "applied", output.Migrations[0].Status)
	assert.NotEmpty(t, output.Migrations[0].AppliedAt)
	require.NotNil(t, output.Migrations[0].ChecksumMatch)
	assert.True(t, *output.Migrations[0].ChecksumMatch)

	assert.Equal(t, "002", output.Migrations[1].Version)
	assert.Equal(t, "pending", output.Migrations[1].Status)
	assert.Empty(t, output.Migrations[1].AppliedAt)
	assert.Nil(t, output.Migrations[1].ChecksumMatch)

	assert.Equal(t, "003", output.Migrations[2].Version)
	assert.Equal(t, "mismatch", output.Migrations[2].Status)
	require.NotNil(t, output.Migrations[2].ChecksumMatch)
	assert.False(t, *output.Migrations[2].ChecksumMatch)
}

func TestPrintStatusSummary_allApplied(t *testing.T) {
	t.Parallel()

	entries := []statusEntry{
		{Status: statusApplied},
		{Status: statusApplied},
	}

	buf := new(bytes.Buffer)
	printStatusSummary(buf, entries)

	assert.Equal(t, "Summary: 2 applied, 0 pending\n", buf.String())
}

func TestPrintStatusSummary_withMismatches(t *testing.T) {
	t.Parallel()

	entries := []statusEntry{
		{Status: statusApplied},
		{Status: statusMismatch},
		{Status: statusPending},
	}

	buf := new(bytes.Buffer)
	printStatusSummary(buf, entries)

	output := buf.String()
	assert.Contains(t, output, "1 applied")
	assert.Contains(t, output, "1 pending")
	assert.Contains(t, output, "1 checksum mismatch")
}

func TestFormatDurationMs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ms       int
		expected string
	}{
		{45, "45ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{2300, "2.3s"},
		{10500, "10.5s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatDurationMs(tt.ms))
		})
	}
}

func TestRunStatus_noDatabaseURL_returnsError(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	setTestAppConfig(t, &config.Config{MigrationsDir: "./testdata/migrations"})

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runStatus(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errDatabaseURLRequired)
}

func TestPrintStatusText_coloredStatus_keepsColumnsAligned(t *testing.T) { //nolint:paralleltest // forceColor changes global lipgloss state
	forceColor(t)

	buf := new(bytes.Buffer)
	printStatusText(buf, testStatusEntries())
	output := buf.String()

	require.Contains(t, output, "\x1b[", "precondition: status cell is colored")

	header := tableRow(t, output, "Applied At")
	row := tableRow(t, output, "create_users")
	assert.Equal(t, strings.Index(header, "Applied At"), strings.Index(row, "2024-01-15 10:30:00"),
		"applied-at column misaligned:\n%s\n%s", header, row)
}
