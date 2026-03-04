package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// Output format constants.
const (
	FormatText          = "text"
	FormatJSON          = "json"
	FormatGitHubActions = "github-actions"
)

// Migration status labels used across status output.
const (
	statusApplied  = "applied"
	statusPending  = "pending"
	statusMismatch = "mismatch"
)

// Lipgloss severity styles.
var ( //nolint:gochecknoglobals // constant style definitions
	styleSafe     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))            // green
	styleLow      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))            // cyan
	styleMedium   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))            // yellow
	styleHigh     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))            // red
	styleCritical = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")) // bright red bold
	styleWarning  = styleMedium                                                    // yellow (same as medium)
)

// colorSeverity returns the severity label styled with lipgloss.
func colorSeverity(s analyzer.Severity) string {
	label := s.String()

	switch s {
	case analyzer.Safe:
		return styleSafe.Render(label)
	case analyzer.Low:
		return styleLow.Render(label)
	case analyzer.Medium:
		return styleMedium.Render(label)
	case analyzer.High:
		return styleHigh.Render(label)
	case analyzer.Critical:
		return styleCritical.Render(label)
	default:
		return label
	}
}

// colorStatus returns a styled status label.
func colorStatus(status string) string {
	switch status {
	case statusApplied:
		return styleSafe.Render(status)
	case statusPending:
		return styleWarning.Render(status)
	case statusMismatch:
		return styleCritical.Render(status)
	default:
		return status
	}
}

// getFormat reads the --format flag from the command, falling back to the config value.
func getFormat(cmd *cobra.Command) string {
	if cmd.Flags().Lookup("format") != nil {
		f, _ := cmd.Flags().GetString("format")
		if f != "" {
			return f
		}
	}

	return AppConfig.Format
}

// printJSON marshals v as indented JSON and writes it to out.
func printJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}

	return nil
}

// AnalyzeJSONOutput is the top-level JSON structure for analyze output.
type AnalyzeJSONOutput struct {
	Migrations    []AnalyzeJSONMigration `json:"migrations"`
	TotalFindings int                    `json:"total_findings"`
	HasHighOrCrit bool                   `json:"has_high_or_critical"`
}

// AnalyzeJSONMigration represents a single migration in analyze JSON output.
type AnalyzeJSONMigration struct {
	Version  string               `json:"version"`
	Name     string               `json:"name"`
	FilePath string               `json:"file_path,omitempty"`
	Findings []AnalyzeJSONFinding `json:"findings"`
}

// AnalyzeJSONFinding represents a single finding in analyze JSON output.
type AnalyzeJSONFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	Table      string `json:"table"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Statement  string `json:"statement,omitempty"`
	LockType   string `json:"lock_type,omitempty"`
}

// PlanJSONOutput is the top-level JSON structure for plan output.
type PlanJSONOutput struct {
	Steps         []PlanJSONStep `json:"steps"`
	TotalPending  int            `json:"total_pending"`
	HighRiskCount int            `json:"high_risk_count"`
	CriticalCount int            `json:"critical_count"`
}

// PlanJSONStep represents a single step in plan JSON output.
type PlanJSONStep struct {
	Version       string               `json:"version"`
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	Risk          string               `json:"risk"`
	EstimatedLock string               `json:"estimated_lock"`
	RunInTx       bool                 `json:"run_in_tx"`
	Findings      []AnalyzeJSONFinding `json:"findings,omitempty"`
}

// StatusJSONOutput is the top-level JSON structure for status output.
type StatusJSONOutput struct {
	Migrations []StatusJSONMigration `json:"migrations"`
	Total      int                   `json:"total"`
	Applied    int                   `json:"applied"`
	Pending    int                   `json:"pending"`
	Mismatches int                   `json:"mismatches"`
}

// StatusJSONMigration represents a single migration in status JSON output.
type StatusJSONMigration struct {
	Version       string `json:"version"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	AppliedAt     string `json:"applied_at,omitempty"`
	DurationMs    int    `json:"duration_ms,omitempty"`
	ChecksumMatch *bool  `json:"checksum_match,omitempty"`
}

// formatGitHubAnnotation formats a finding as a GitHub Actions annotation.
// Uses ::error for HIGH/CRITICAL and ::warning for lower severities.
func formatGitHubAnnotation(filePath string, finding *analyzer.Finding) string {
	level := "warning"
	if finding.Severity >= analyzer.High {
		level = "error"
	}

	if filePath != "" {
		return fmt.Sprintf("::%s file=%s::[%s] %s", level, filePath, finding.Severity, finding.Message)
	}

	return fmt.Sprintf("::%s::[%s] %s", level, finding.Severity, finding.Message)
}
