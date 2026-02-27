package analyzer

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/migration"
)

// Rule is the interface that all danger detection rules must implement.
type Rule interface {
	// ID returns a unique kebab-case identifier for this rule.
	ID() string
	// Check examines a single parsed statement and returns any findings.
	Check(stmt *pg_query.RawStmt, ctx *RuleContext) []Finding
}

// RuleContext provides contextual information to rules during analysis.
type RuleContext struct {
	Migration       *migration.Migration
	TargetPGVersion int
	StmtIndex       int
	SQL             string // The full migration SQL (for extracting statement text)
}

// Registry holds a collection of rules.
type Registry struct {
	rules []Rule
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a rule to the registry.
func (r *Registry) Register(rule Rule) {
	r.rules = append(r.rules, rule)
}

// Rules returns all registered rules.
func (r *Registry) Rules() []Rule {
	return r.rules
}

// TableName extracts a qualified table name from a RangeVar.
func TableName(rv *pg_query.RangeVar) string {
	if rv == nil {
		return "<unknown>"
	}

	if rv.Schemaname != "" {
		return rv.Schemaname + "." + rv.Relname
	}

	return rv.Relname
}

// ExtractStmtSQL extracts the SQL text for a specific statement from the full SQL string.
func ExtractStmtSQL(stmts []*pg_query.RawStmt, idx int, fullSQL string) string {
	if idx < 0 || idx >= len(stmts) {
		return ""
	}

	start := int(stmts[idx].StmtLocation)

	var end int
	if idx+1 < len(stmts) {
		end = int(stmts[idx+1].StmtLocation)
	} else {
		end = len(fullSQL)
	}

	if start > len(fullSQL) || end > len(fullSQL) || start >= end {
		return ""
	}

	return strings.TrimSpace(fullSQL[start:end])
}
