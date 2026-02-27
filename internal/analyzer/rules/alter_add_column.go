package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

const pgVersionSafeNonVolatileDefault = 11

// AddColumnRule detects ADD COLUMN with dangerous DEFAULT values (R-2).
type AddColumnRule struct{}

// NewAddColumnRule creates a new AddColumnRule.
func NewAddColumnRule() *AddColumnRule { return &AddColumnRule{} }

// ID returns the rule identifier.
func (r *AddColumnRule) ID() string { return "add-column-volatile-default" }

// Check examines a statement for ADD COLUMN with volatile DEFAULT.
func (r *AddColumnRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
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

		if cmd.AlterTableCmd.Subtype != pg_query.AlterTableType_AT_AddColumn {
			continue
		}

		finding := r.checkAddColumn(cmd.AlterTableCmd, alt.Relation, ctx)
		if finding != nil {
			findings = append(findings, *finding)
		}
	}

	return findings
}

func (r *AddColumnRule) checkAddColumn(
	cmd *pg_query.AlterTableCmd,
	relation *pg_query.RangeVar,
	ctx *analyzer.RuleContext,
) *analyzer.Finding {
	if cmd.Def == nil {
		return nil
	}

	colDefNode, ok := cmd.Def.Node.(*pg_query.Node_ColumnDef)
	if !ok {
		return nil
	}

	colDef := colDefNode.ColumnDef

	defaultExpr := extractDefaultExpr(colDef)
	if defaultExpr == nil {
		return nil // no DEFAULT — safe
	}

	if ctx.TargetPGVersion >= pgVersionSafeNonVolatileDefault && !isVolatileDefault(defaultExpr) {
		return nil // PG 11+ with non-volatile default — safe
	}

	msg := "ADD COLUMN with volatile DEFAULT rewrites the entire table"
	if ctx.TargetPGVersion < pgVersionSafeNonVolatileDefault {
		msg = "ADD COLUMN with DEFAULT rewrites the entire table on PG < 11"
	}

	return &analyzer.Finding{
		Rule:       r.ID(),
		Severity:   analyzer.High,
		Table:      analyzer.TableName(relation),
		Message:    msg,
		Suggestion: "Add column without DEFAULT, then backfill in batches",
		LockType:   "ACCESS EXCLUSIVE",
		StmtIndex:  ctx.StmtIndex,
	}
}

// extractDefaultExpr finds the DEFAULT expression from a ColumnDef.
// In pg_query_go v6, DEFAULT is stored as a CONSTR_DEFAULT constraint
// in the Constraints list, with the expression in RawExpr.
func extractDefaultExpr(colDef *pg_query.ColumnDef) *pg_query.Node {
	for _, c := range colDef.Constraints {
		cn, ok := c.Node.(*pg_query.Node_Constraint)
		if !ok {
			continue
		}

		if cn.Constraint.Contype == pg_query.ConstrType_CONSTR_DEFAULT {
			return cn.Constraint.RawExpr
		}
	}

	return nil
}

// isVolatileDefault determines whether a DEFAULT expression is volatile.
// Constants and type casts of constants are non-volatile; everything else
// (including function calls like now(), gen_random_uuid()) is assumed volatile.
func isVolatileDefault(node *pg_query.Node) bool {
	if node == nil {
		return false
	}

	switch n := node.Node.(type) {
	case *pg_query.Node_AConst:
		_ = n
		return false // literal constant — non-volatile
	case *pg_query.Node_TypeCast:
		if n.TypeCast.Arg != nil {
			if _, ok := n.TypeCast.Arg.Node.(*pg_query.Node_AConst); ok {
				return false // type cast of a constant — non-volatile
			}
		}

		return true
	default:
		return true // FuncCall or anything else — assume volatile
	}
}
