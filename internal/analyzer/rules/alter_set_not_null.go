package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

const pgVersionSafeSetNotNull = 12

// SetNotNullRule detects SET NOT NULL which requires a full table scan (R-5).
type SetNotNullRule struct{}

// NewSetNotNullRule creates a new SetNotNullRule.
func NewSetNotNullRule() *SetNotNullRule { return &SetNotNullRule{} }

// ID returns the rule identifier.
func (r *SetNotNullRule) ID() string { return "set-not-null" }

// Check examines a statement for SET NOT NULL.
func (r *SetNotNullRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
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

		if cmd.AlterTableCmd.Subtype != pg_query.AlterTableType_AT_SetNotNull {
			continue
		}

		severity := analyzer.High
		suggestion := "Requires full table scan. Consider application-level enforcement instead."

		if ctx.TargetPGVersion >= pgVersionSafeSetNotNull {
			severity = analyzer.Medium
			suggestion = "First add CHECK (col IS NOT NULL) NOT VALID, then VALIDATE CONSTRAINT, then SET NOT NULL"
		}

		findings = append(findings, analyzer.Finding{
			Rule:       r.ID(),
			Severity:   severity,
			Table:      analyzer.TableName(alt.Relation),
			Message:    "SET NOT NULL requires a full table scan to verify no NULL values exist",
			Suggestion: suggestion,
			LockType:   "ACCESS EXCLUSIVE",
			StmtIndex:  ctx.StmtIndex,
		})
	}

	return findings
}
