package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// CreateIndexRule detects non-concurrent CREATE INDEX statements (R-1).
type CreateIndexRule struct{}

// NewCreateIndexRule creates a new CreateIndexRule.
func NewCreateIndexRule() *CreateIndexRule { return &CreateIndexRule{} }

// ID returns the rule identifier.
func (r *CreateIndexRule) ID() string { return "create-index-not-concurrent" }

// Check examines a statement for non-concurrent CREATE INDEX.
func (r *CreateIndexRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	node, ok := stmt.Stmt.Node.(*pg_query.Node_IndexStmt)
	if !ok {
		return nil
	}

	idx := node.IndexStmt
	if idx.Concurrent {
		return nil
	}

	return []analyzer.Finding{{
		Rule:       r.ID(),
		Severity:   analyzer.High,
		Table:      analyzer.TableName(idx.Relation),
		Message:    "CREATE INDEX without CONCURRENTLY locks the table for writes",
		Suggestion: "Use CREATE INDEX CONCURRENTLY to avoid blocking writes during index creation",
		LockType:   "SHARE",
		StmtIndex:  ctx.StmtIndex,
	}}
}
