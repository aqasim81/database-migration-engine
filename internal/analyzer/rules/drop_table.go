package rules

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// DropTableRule detects DROP TABLE and TRUNCATE statements (R-6).
type DropTableRule struct{}

// NewDropTableRule creates a new DropTableRule.
func NewDropTableRule() *DropTableRule { return &DropTableRule{} }

// ID returns the rule identifier.
func (r *DropTableRule) ID() string { return "drop-table" }

// Check examines a statement for DROP TABLE or TRUNCATE.
func (r *DropTableRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	switch node := stmt.Stmt.Node.(type) {
	case *pg_query.Node_DropStmt:
		return r.checkDrop(node.DropStmt, ctx)
	case *pg_query.Node_TruncateStmt:
		return r.checkTruncate(node.TruncateStmt, ctx)
	default:
		return nil
	}
}

func (r *DropTableRule) checkDrop(drop *pg_query.DropStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	if drop == nil || drop.RemoveType != pg_query.ObjectType_OBJECT_TABLE {
		return nil
	}

	tables := extractDropTableNames(drop)
	msg := "DROP TABLE is irreversible and will permanently delete all data"

	if drop.MissingOk {
		msg = "DROP TABLE IF EXISTS is irreversible and will permanently delete all data"
	}

	return []analyzer.Finding{{
		Rule:       r.ID(),
		Severity:   analyzer.Critical,
		Table:      strings.Join(tables, ", "),
		Message:    msg,
		Suggestion: "Ensure you have a backup and that no application code references this table",
		LockType:   "ACCESS EXCLUSIVE",
		StmtIndex:  ctx.StmtIndex,
	}}
}

func (r *DropTableRule) checkTruncate(trunc *pg_query.TruncateStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	if trunc == nil {
		return nil
	}

	var tables []string

	for _, rel := range trunc.Relations {
		rv, ok := rel.Node.(*pg_query.Node_RangeVar)
		if !ok {
			continue
		}

		tables = append(tables, analyzer.TableName(rv.RangeVar))
	}

	return []analyzer.Finding{{
		Rule:       r.ID(),
		Severity:   analyzer.Critical,
		Table:      strings.Join(tables, ", "),
		Message:    "TRUNCATE removes all data from the table and is difficult to reverse",
		Suggestion: "Ensure you have a backup before truncating production tables",
		LockType:   "ACCESS EXCLUSIVE",
		StmtIndex:  ctx.StmtIndex,
	}}
}

func extractDropTableNames(drop *pg_query.DropStmt) []string {
	var tables []string

	for _, obj := range drop.Objects {
		listNode, ok := obj.Node.(*pg_query.Node_List)
		if !ok {
			continue
		}

		var parts []string

		for _, item := range listNode.List.Items {
			if s, ok := item.Node.(*pg_query.Node_String_); ok {
				parts = append(parts, s.String_.Sval)
			}
		}

		if len(parts) > 0 {
			tables = append(tables, strings.Join(parts, "."))
		}
	}

	return tables
}
