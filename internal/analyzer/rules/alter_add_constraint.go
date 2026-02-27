package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// AddConstraintRule detects ADD CONSTRAINT without NOT VALID for CHECK and FK constraints (R-3).
type AddConstraintRule struct{}

// NewAddConstraintRule creates a new AddConstraintRule.
func NewAddConstraintRule() *AddConstraintRule { return &AddConstraintRule{} }

// ID returns the rule identifier.
func (r *AddConstraintRule) ID() string { return "add-constraint-without-not-valid" }

// Check examines a statement for ADD CONSTRAINT without NOT VALID.
func (r *AddConstraintRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
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

		if cmd.AlterTableCmd.Subtype != pg_query.AlterTableType_AT_AddConstraint {
			continue
		}

		if cmd.AlterTableCmd.Def == nil {
			continue
		}

		constraintNode, ok := cmd.AlterTableCmd.Def.Node.(*pg_query.Node_Constraint)
		if !ok {
			continue
		}

		constraint := constraintNode.Constraint

		// Only flag CHECK and FOREIGN KEY constraints
		if constraint.Contype != pg_query.ConstrType_CONSTR_CHECK &&
			constraint.Contype != pg_query.ConstrType_CONSTR_FOREIGN {
			continue
		}

		if constraint.SkipValidation {
			continue // Has NOT VALID — safe
		}

		findings = append(findings, analyzer.Finding{
			Rule:       r.ID(),
			Severity:   analyzer.High,
			Table:      analyzer.TableName(alt.Relation),
			Message:    "ADD CONSTRAINT without NOT VALID scans the entire table while holding a lock",
			Suggestion: "Add with NOT VALID, then VALIDATE CONSTRAINT in a separate statement",
			LockType:   "ACCESS EXCLUSIVE",
			StmtIndex:  ctx.StmtIndex,
		})
	}

	return findings
}
