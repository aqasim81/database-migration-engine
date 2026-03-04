package planner

import (
	"context"
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

// Migration step status constants.
const (
	StatusPending = "pending"
	StatusApplied = "applied"
)

// TableSizer retrieves table size information from a database.
type TableSizer interface {
	TableSizeBytes(ctx context.Context, tableName string) (int64, error)
}

// MigrationStep represents one migration in the execution plan.
type MigrationStep struct {
	Migration *migration.Migration
	Findings  []analyzer.Finding
	Impacts   []Impact
	Status    string // "pending" or "applied"
	RunInTx   bool   // false for CREATE INDEX CONCURRENTLY migrations
}

// MaxSeverity returns the highest severity across all findings in this step.
func (s *MigrationStep) MaxSeverity() analyzer.Severity {
	maxSev := analyzer.Safe
	for _, f := range s.Findings {
		if f.Severity > maxSev {
			maxSev = f.Severity
		}
	}

	return maxSev
}

// Plan holds the complete execution plan for a set of migrations.
type Plan struct {
	Steps         []MigrationStep
	TotalPending  int
	HighRiskCount int
	CriticalCount int
}

// BuildPlanParams holds the inputs for BuildPlan.
type BuildPlanParams struct {
	Migrations []migration.Migration
	Results    []analyzer.AnalysisResult
	Applied    map[string]bool // version -> true if applied
	Sizer      TableSizer      // nil if no DB connection
	Ctx        context.Context // nil uses context.Background()
}

// BuildPlan creates an execution plan from migrations, analysis results,
// and applied status.
func BuildPlan(params *BuildPlanParams) (*Plan, error) {
	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	resultMap := buildResultMap(params.Results)
	plan := &Plan{}

	for i := range params.Migrations {
		m := &params.Migrations[i]

		step, err := buildStep(ctx, m, resultMap[m.Version], params.Applied, params.Sizer)
		if err != nil {
			return nil, fmt.Errorf("building plan step for %s: %w", m.Version, err)
		}

		plan.Steps = append(plan.Steps, *step)
		updatePlanCounts(plan, step)
	}

	return plan, nil
}

// PendingOnly returns a new Plan containing only pending steps.
func PendingOnly(p *Plan) *Plan {
	filtered := &Plan{}

	for _, s := range p.Steps {
		if s.Status != StatusPending {
			continue
		}

		filtered.Steps = append(filtered.Steps, s)
		filtered.TotalPending++
		updateRiskCounts(filtered, &s)
	}

	return filtered
}

func updatePlanCounts(plan *Plan, step *MigrationStep) {
	if step.Status == StatusPending {
		plan.TotalPending++
	}

	updateRiskCounts(plan, step)
}

func updateRiskCounts(plan *Plan, step *MigrationStep) {
	switch step.MaxSeverity() {
	case analyzer.High:
		plan.HighRiskCount++
	case analyzer.Critical:
		plan.CriticalCount++
	case analyzer.Safe, analyzer.Low, analyzer.Medium:
		// No risk counting needed for lower severities.
	}
}

func buildResultMap(results []analyzer.AnalysisResult) map[string]*analyzer.AnalysisResult {
	m := make(map[string]*analyzer.AnalysisResult, len(results))
	for i := range results {
		m[results[i].Migration.Version] = &results[i]
	}

	return m
}

func buildStep(
	ctx context.Context,
	m *migration.Migration,
	result *analyzer.AnalysisResult,
	applied map[string]bool,
	sizer TableSizer,
) (*MigrationStep, error) {
	step := &MigrationStep{
		Migration: m,
		Status:    StatusPending,
		RunInTx:   true,
	}

	if applied[m.Version] {
		step.Status = StatusApplied
	}

	if result != nil {
		step.Findings = result.Findings
		step.Impacts = buildImpacts(ctx, result.Findings, sizer)
	}

	concurrent, err := hasConcurrentOp(m.UpSQL)
	if err != nil {
		return nil, fmt.Errorf("detecting concurrent operations: %w", err)
	}

	if concurrent {
		step.RunInTx = false
	}

	return step, nil
}

func buildImpacts(ctx context.Context, findings []analyzer.Finding, sizer TableSizer) []Impact {
	impacts := make([]Impact, len(findings))
	for i := range findings {
		tableSize := lookupTableSize(ctx, sizer, findings[i].Table)
		impacts[i] = EstimateImpact(&findings[i], tableSize)
	}

	return impacts
}

func lookupTableSize(ctx context.Context, sizer TableSizer, tableName string) int64 {
	if sizer == nil || tableName == "" {
		return sizeUnknown
	}

	size, err := sizer.TableSizeBytes(ctx, tableName)
	if err != nil {
		return sizeUnknown
	}

	return size
}

// hasConcurrentOp checks if SQL contains CREATE/DROP INDEX CONCURRENTLY.
// Deliberate duplication of executor.containsConcurrentOp to avoid cross-package coupling.
func hasConcurrentOp(sql string) (bool, error) {
	if !strings.Contains(strings.ToUpper(sql), "CONCURRENTLY") {
		return false, nil
	}

	result, err := parser.Parse(sql)
	if err != nil {
		return false, fmt.Errorf("parsing SQL for concurrent detection: %w", err)
	}

	for _, stmt := range result.Stmts {
		switch node := stmt.Stmt.Node.(type) {
		case *pg_query.Node_IndexStmt:
			if node.IndexStmt != nil && node.IndexStmt.Concurrent {
				return true, nil
			}
		case *pg_query.Node_DropStmt:
			if node.DropStmt != nil && node.DropStmt.Concurrent {
				return true, nil
			}
		}
	}

	return false, nil
}
