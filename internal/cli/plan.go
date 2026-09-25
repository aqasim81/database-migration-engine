package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/planner"
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

	a := newDefaultAnalyzer(cfg.TargetPGVersion)

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
	if cfg.DatabaseURL == "" {
		return make(map[string]bool), nil, nil, nil
	}

	pool, err := connectDB(ctx, cfg, out)
	if err != nil {
		return nil, nil, nil, err
	}

	applied, err = appliedVersions(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}

	return applied, planner.NewPgTableSizer(pool), pool.Close, nil
}

// stepRiskAndLock returns the plain risk label and estimated lock duration for a plan step.
// Returns ("-", "-") when the step is not pending or has no findings.
func stepRiskAndLock(step *planner.MigrationStep) (risk, estLock string) {
	if step.Status != planner.StatusPending || len(step.Findings) == 0 {
		return "-", "-"
	}

	return step.MaxSeverity().String(), maxLockDuration(step.Impacts)
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
		risk, estLock := stepRiskAndLock(&step)
		if risk != "-" {
			risk = colorSeverity(step.MaxSeverity())
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
		risk, estLock := stepRiskAndLock(&step)

		s := PlanJSONStep{
			Version:       step.Migration.Version,
			Name:          step.Migration.Name,
			Status:        step.Status,
			Risk:          risk,
			EstimatedLock: estLock,
			RunInTx:       step.RunInTx,
		}

		for i := range step.Findings {
			s.Findings = append(s.Findings, findingToJSON(&step.Findings[i]))
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
