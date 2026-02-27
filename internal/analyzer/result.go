package analyzer

import "github.com/aqasim81/database-migration-engine/internal/migration"

// Finding represents a single dangerous pattern detected in a migration.
type Finding struct {
	Rule       string   // Rule ID (e.g., "create-index-not-concurrent")
	Severity   Severity // Danger level
	Table      string   // Affected table name
	Statement  string   // The SQL statement text (truncated for display)
	Message    string   // Human-readable description of the danger
	Suggestion string   // Safe alternative approach
	LockType   string   // PostgreSQL lock type acquired (e.g., "ACCESS EXCLUSIVE")
	StmtIndex  int      // Index in the migration's statement list (0-based)
}

// AnalysisResult holds all findings for a single migration.
type AnalysisResult struct {
	Migration   *migration.Migration
	Findings    []Finding
	MaxSeverity Severity // Highest severity across all findings
}

// HasHighOrCritical returns true if any finding is High or Critical severity.
func (r *AnalysisResult) HasHighOrCritical() bool {
	return r.MaxSeverity >= High
}

// TruncateSQL truncates a SQL string to maxLen characters for display.
func TruncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen-3] + "..."
}
