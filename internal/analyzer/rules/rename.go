package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// RenameRule detects RENAME TABLE and RENAME COLUMN statements (R-10).
type RenameRule struct{}

// NewRenameRule creates a new RenameRule.
func NewRenameRule() *RenameRule { return &RenameRule{} }

// ID returns the rule identifier.
func (r *RenameRule) ID() string { return "rename" }

// Check examines a statement for RENAME TABLE or RENAME COLUMN.
func (r *RenameRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	node, ok := stmt.Stmt.Node.(*pg_query.Node_RenameStmt)
	if !ok {
		return nil
	}

	rename := node.RenameStmt
	if rename == nil {
		return nil
	}

	if rename.RenameType == pg_query.ObjectType_OBJECT_TABLE {
		return []analyzer.Finding{{
			Rule:       r.ID(),
			Severity:   analyzer.Medium,
			Table:      analyzer.TableName(rename.Relation),
			Message:    "RENAME TABLE breaks application code that references the old name",
			Suggestion: "Use a staged approach: add new name (view), update app code, remove old name",
			LockType:   "ACCESS EXCLUSIVE",
			StmtIndex:  ctx.StmtIndex,
		}}
	}

	if rename.RenameType == pg_query.ObjectType_OBJECT_COLUMN {
		return []analyzer.Finding{{
			Rule:       r.ID(),
			Severity:   analyzer.Medium,
			Table:      analyzer.TableName(rename.Relation),
			Message:    "RENAME COLUMN breaks application code that references the old column name",
			Suggestion: "Use a staged approach: add new column, backfill, update app code, drop old column",
			LockType:   "ACCESS EXCLUSIVE",
			StmtIndex:  ctx.StmtIndex,
		}}
	}

	return nil // RENAME INDEX, etc. — safe
}
