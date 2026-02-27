package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

func TestSeverity_String_allLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity analyzer.Severity
		expected string
	}{
		{analyzer.Safe, "SAFE"},
		{analyzer.Low, "LOW"},
		{analyzer.Medium, "MEDIUM"},
		{analyzer.High, "HIGH"},
		{analyzer.Critical, "CRITICAL"},
		{analyzer.Severity(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestSeverity_Color_allLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity analyzer.Severity
		expected string
		name     string
	}{
		{analyzer.Safe, "\033[32m", "Safe_green"},
		{analyzer.Low, "\033[36m", "Low_cyan"},
		{analyzer.Medium, "\033[33m", "Medium_yellow"},
		{analyzer.High, "\033[31m", "High_red"},
		{analyzer.Critical, "\033[91m", "Critical_brightRed"},
		{analyzer.Severity(99), "\033[0m", "Unknown_reset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.severity.Color())
		})
	}
}
