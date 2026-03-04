package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/planner"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

var planCmd = &cobra.Command{ //nolint:gochecknoglobals // standard Cobra pattern
	Use:   "plan",
	Short: "Show execution plan for pending migrations",
	Long: `Display the execution plan for pending migrations including
analysis results, estimated impact, and execution order.`,
	RunE: runPlan,
}

func init() { //nolint:gochecknoinits // standard Cobra pattern for flag registration
	planCmd.Flags().Bool("pending-only", false, "show only pending migrations")
	planCmd.Flags().String("format", "text", "output format (text, json)")
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, _ []string) error {
	cfg := AppConfig

	sorted, err := loadAndSortMigrations(cfg.MigrationsDir, cmd.OutOrStdout())
	if err != nil || sorted == nil {
		return err
	}

	a := analyzer.New(
		analyzer.WithRegistry(rules.NewDefaultRegistry()),
		analyzer.WithPGVersion(cfg.TargetPGVersion),
	)

	results, err := a.AnalyzeAll(sorted)
	if err != nil {
		return fmt.Errorf("analyzing migrations: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	applied, sizer, cleanup, err := loadAppliedState(ctx, cfg, cmd.OutOrStdout())
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	plan, err := planner.BuildPlan(ctx, &planner.BuildPlanParams{
		Migrations: sorted,
		Results:    results,
		Applied:    applied,
		Sizer:      sizer,
	})
	if err != nil {
		return fmt.Errorf("building plan: %w", err)
	}

	pendingOnly, _ := cmd.Flags().GetBool("pending-only")
	if pendingOnly {
		plan = planner.PendingOnly(plan)
	}

	format := getFormat(cmd)

	switch format {
	case FormatJSON:
		return printPlanJSON(cmd.OutOrStdout(), plan)
	default:
		printPlan(cmd.OutOrStdout(), plan)
		return nil
	}
}

// loadAppliedState optionally connects to the database to get applied migrations
// and a table sizer. Returns a cleanup function to close the pool.
func loadAppliedState(
	ctx context.Context,
	cfg *config.Config,
	out io.Writer,
) (applied map[string]bool, sizer planner.TableSizer, cleanup func(), err error) {
	applied = make(map[string]bool)

	if cfg.DatabaseURL == "" {
		return applied, nil, nil, nil
	}

	pool, err := connectDB(ctx, cfg, out)
	if err != nil {
		return nil, nil, nil, err
	}

	t := tracker.New(pool)
	if err = t.EnsureTable(ctx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("ensuring migrations table: %w", err)
	}

	appliedList, err := t.GetApplied(ctx)
	if err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("getting applied migrations: %w", err)
	}

	for _, am := range appliedList {
		applied[am.Version] = true
	}

	return applied, planner.NewPgTableSizer(pool), pool.Close, nil
}

func printPlan(out io.Writer, plan *planner.Plan) {
	fmt.Fprintln(out, "Migration Plan")
	fmt.Fprintln(out, "==================================================")
	fmt.Fprintln(out)

	if len(plan.Steps) == 0 {
		fmt.Fprintln(out, "No migrations found.")
		fmt.Fprintln(out)
		printPlanSummary(out, plan)

		return
	}

	fmt.Fprintf(out, "%-4s %-10s %-22s %-10s %-10s %s\n",
		"#", "Version", "Name", "Status", "Risk", "Est. Lock")
	fmt.Fprintf(out, "%-4s %-10s %-22s %-10s %-10s %s\n",
		"--", "-------", "----", "------", "----", "---------")

	for i, step := range plan.Steps {
		risk := "-"
		estLock := "-"

		if step.Status == planner.StatusPending && len(step.Findings) > 0 {
			risk = colorSeverity(step.MaxSeverity())
			estLock = maxLockDuration(step.Impacts)
		}

		fmt.Fprintf(out, "%-4d %-10s %-22s %-10s %-10s %s\n",
			i+1,
			step.Migration.Version,
			analyzer.TruncateSQL(step.Migration.Name, 22), //nolint:mnd // column width
			step.Status,
			risk,
			estLock,
		)
	}

	fmt.Fprintln(out)
	printPlanSummary(out, plan)
}

func printPlanJSON(out io.Writer, plan *planner.Plan) error {
	output := PlanJSONOutput{
		TotalPending:  plan.TotalPending,
		HighRiskCount: plan.HighRiskCount,
		CriticalCount: plan.CriticalCount,
		Steps:         make([]PlanJSONStep, 0, len(plan.Steps)),
	}

	for _, step := range plan.Steps {
		risk := "-"
		estLock := "-"

		if step.Status == planner.StatusPending && len(step.Findings) > 0 {
			risk = step.MaxSeverity().String()
			estLock = maxLockDuration(step.Impacts)
		}

		s := PlanJSONStep{
			Version:       step.Migration.Version,
			Name:          step.Migration.Name,
			Status:        step.Status,
			Risk:          risk,
			EstimatedLock: estLock,
			RunInTx:       step.RunInTx,
		}

		for _, f := range step.Findings {
			s.Findings = append(s.Findings, AnalyzeJSONFinding{
				Rule:       f.Rule,
				Severity:   f.Severity.String(),
				Table:      f.Table,
				Message:    f.Message,
				Suggestion: f.Suggestion,
				Statement:  f.Statement,
				LockType:   f.LockType,
			})
		}

		output.Steps = append(output.Steps, s)
	}

	return printJSON(out, output)
}

func printPlanSummary(out io.Writer, plan *planner.Plan) {
	fmt.Fprintf(out, "Summary: %d pending", plan.TotalPending)

	if plan.HighRiskCount > 0 {
		fmt.Fprintf(out, ", %d HIGH risk", plan.HighRiskCount)
	}

	if plan.CriticalCount > 0 {
		fmt.Fprintf(out, ", %d CRITICAL risk", plan.CriticalCount)
	}

	fmt.Fprintln(out)
}

//nolint:gochecknoglobals // constant lookup table for duration ordering
var durationOrder = map[string]int{
	planner.DurationUnder1s:   0,
	planner.Duration1to10s:    1,
	planner.Duration10sTo5min: 2,
	planner.DurationOver5min:  3,
}

func maxLockDuration(impacts []planner.Impact) string {
	if len(impacts) == 0 {
		return "-"
	}

	maxDur := impacts[0].EstimatedLockDuration
	for _, imp := range impacts[1:] {
		if durationOrder[imp.EstimatedLockDuration] > durationOrder[maxDur] {
			maxDur = imp.EstimatedLockDuration
		}
	}

	return maxDur
}
