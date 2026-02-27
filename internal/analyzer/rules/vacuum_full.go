package rules

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

// VacuumFullRule detects VACUUM FULL statements (R-8).
type VacuumFullRule struct{}

// NewVacuumFullRule creates a new VacuumFullRule.
func NewVacuumFullRule() *VacuumFullRule { return &VacuumFullRule{} }

// ID returns the rule identifier.
func (r *VacuumFullRule) ID() string { return "vacuum-full" }

// Check examines a statement for VACUUM FULL.
func (r *VacuumFullRule) Check(stmt *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	node, ok := stmt.Stmt.Node.(*pg_query.Node_VacuumStmt)
	if !ok {
		return nil
	}

	vacuum := node.VacuumStmt
	if !isVacuumFull(vacuum) {
		return nil
	}

	tableName := extractVacuumTable(vacuum)

	return []analyzer.Finding{{
		Rule:       r.ID(),
		Severity:   analyzer.High,
		Table:      tableName,
		Message:    "VACUUM FULL rewrites the entire table and holds an ACCESS EXCLUSIVE lock",
		Suggestion: "Use regular VACUUM instead, which does not block reads or writes",
		LockType:   "ACCESS EXCLUSIVE",
		StmtIndex:  ctx.StmtIndex,
	}}
}

func isVacuumFull(v *pg_query.VacuumStmt) bool {
	for _, opt := range v.Options {
		de, ok := opt.Node.(*pg_query.Node_DefElem)
		if !ok {
			continue
		}

		if de.DefElem.Defname == "full" {
			return true
		}
	}

	return false
}

func extractVacuumTable(v *pg_query.VacuumStmt) string {
	for _, rel := range v.Rels {
		vr, ok := rel.Node.(*pg_query.Node_VacuumRelation)
		if !ok {
			continue
		}

		if vr.VacuumRelation.Relation != nil {
			return analyzer.TableName(vr.VacuumRelation.Relation)
		}
	}

	return "<all tables>"
}
