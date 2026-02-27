package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// LockTableRule detects explicit LOCK TABLE statements (R-9).
type LockTableRule struct{}

// NewLockTableRule creates a new LockTableRule.
func NewLockTableRule() *LockTableRule { return &LockTableRule{} }

// ID returns the rule identifier.
func (r *LockTableRule) ID() string { return "lock-table" }

// Check examines a statement for explicit LOCK TABLE.
func (r *LockTableRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	node, ok := stmt.Stmt.Node.(*pg_query.Node_LockStmt)
	if !ok {
		return nil
	}

	lock := node.LockStmt
	var findings []analyzer.Finding

	for _, rel := range lock.Relations {
		rv, ok := rel.Node.(*pg_query.Node_RangeVar)
		if !ok {
			continue
		}

		findings = append(findings, analyzer.Finding{
			Rule:       r.ID(),
			Severity:   analyzer.High,
			Table:      analyzer.TableName(rv.RangeVar),
			Message:    "Explicit LOCK TABLE can block other queries and cause downtime",
			Suggestion: "Avoid explicit table locks. Let PostgreSQL manage locking through normal operations",
			LockType:   "EXPLICIT",
			StmtIndex:  ctx.StmtIndex,
		})
	}

	return findings
}
