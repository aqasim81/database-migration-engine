package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// AlterColumnTypeRule detects ALTER COLUMN TYPE which causes a full table rewrite (R-4).
type AlterColumnTypeRule struct{}

// NewAlterColumnTypeRule creates a new AlterColumnTypeRule.
func NewAlterColumnTypeRule() *AlterColumnTypeRule { return &AlterColumnTypeRule{} }

// ID returns the rule identifier.
func (r *AlterColumnTypeRule) ID() string { return "alter-column-type" }

// Check examines a statement for ALTER COLUMN TYPE.
func (r *AlterColumnTypeRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	node, ok := stmt.Stmt.Node.(*pg_query.Node_AlterTableStmt)
	if !ok {
		return nil
	}

	alt := node.AlterTableStmt
	var findings []analyzer.Finding

	for _, cmdNode := range alt.Cmds {
		cmd, ok := cmdNode.Node.(*pg_query.Node_AlterTableCmd)
		if !ok {
			continue
		}

		if cmd.AlterTableCmd.Subtype != pg_query.AlterTableType_AT_AlterColumnType {
			continue
		}

		findings = append(findings, analyzer.Finding{
			Rule:       r.ID(),
			Severity:   analyzer.High,
			Table:      analyzer.TableName(alt.Relation),
			Message:    "ALTER COLUMN TYPE rewrites the entire table while holding an ACCESS EXCLUSIVE lock",
			Suggestion: "Use a staged approach: add new column, backfill data, swap columns, drop old column",
			LockType:   "ACCESS EXCLUSIVE",
			StmtIndex:  ctx.StmtIndex,
		})
	}

	return findings
}
