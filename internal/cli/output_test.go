package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestColorSeverity_allLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity analyzer.Severity
		contains string
	}{
		{analyzer.Safe, "SAFE"},
		{analyzer.Low, "LOW"},
		{analyzer.Medium, "MEDIUM"},
		{analyzer.High, "HIGH"},
		{analyzer.Critical, "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			t.Parallel()
			result := colorSeverity(tt.severity)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestColorStatus_allStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   string
		contains string
	}{
		{"applied", "applied"},
		{"pending", "pending"},
		{"mismatch", "mismatch"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			result := colorStatus(tt.status)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestGetFormat_fromFlag(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	setTestAppConfig(t, &config.Config{Format: "text"})

	cmd := &cobra.Command{}
	cmd.Flags().String("format", "text", "")
	require.NoError(t, cmd.Flags().Set("format", "json"))

	assert.Equal(t, "json", getFormat(cmd))
}

func TestGetFormat_fallsBackToConfig(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	setTestAppConfig(t, &config.Config{Format: "json"})

	cmd := &cobra.Command{}

	assert.Equal(t, "json", getFormat(cmd))
}

func TestGetFormat_defaultText(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	setTestAppConfig(t, &config.Config{Format: "text"})

	cmd := &cobra.Command{}
	cmd.Flags().String("format", "", "")

	assert.Equal(t, "text", getFormat(cmd))
}

func TestPrintJSON_validOutput(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	data := map[string]string{"key": "value"}
	err := printJSON(buf, data)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"key": "value"`)
}

func TestFormatGitHubAnnotation_highSeverity(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Severity: analyzer.High,
		Message:  "Index creation locks table",
	}

	result := formatGitHubAnnotation("migrations/001.up.sql", finding)
	assert.Equal(t, "::error file=migrations/001.up.sql::[HIGH] Index creation locks table", result)
}

func TestFormatGitHubAnnotation_lowSeverity(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Severity: analyzer.Low,
		Message:  "Minor concern",
	}

	result := formatGitHubAnnotation("migrations/002.up.sql", finding)
	assert.Equal(t, "::warning file=migrations/002.up.sql::[LOW] Minor concern", result)
}

func TestFormatGitHubAnnotation_criticalSeverity(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Severity: analyzer.Critical,
		Message:  "Data loss risk",
	}

	result := formatGitHubAnnotation("migrations/003.up.sql", finding)
	assert.Equal(t, "::error file=migrations/003.up.sql::[CRITICAL] Data loss risk", result)
}

func TestFormatGitHubAnnotation_noFilePath(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Severity: analyzer.Medium,
		Message:  "Test message",
	}

	result := formatGitHubAnnotation("", finding)
	assert.Equal(t, "::warning::[MEDIUM] Test message", result)
}
