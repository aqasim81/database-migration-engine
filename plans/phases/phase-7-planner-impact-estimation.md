# Phase 7: Planner + Impact Estimation

> **Status:** Pending
> **Commit:** `feat: add migration planner with impact estimation`

## Goal

Show a detailed execution plan with risk assessment and estimated impact before running `apply`.

---

## Step 7.1: Impact Estimation

**File:** `internal/planner/impact.go`

```go
type Impact struct {
    EstimatedLockDuration string   // "< 1s", "1-10s", "10s-5min", "> 5min"
    LockType              string   // "ACCESS EXCLUSIVE", "SHARE", etc.
    AffectedTable         string
    RequiresFullRewrite   bool
    RequiresFullScan      bool
    CanRunConcurrently    bool     // True for CREATE INDEX CONCURRENTLY
    TableSizeBytes        int64    // -1 if unknown (no DB connection)
}

func EstimateImpact(finding analyzer.Finding, tableSize int64) Impact
```

Heuristics:
- `CREATE INDEX` on table > 1GB → "> 5min"
- `ALTER COLUMN TYPE` → always "10s-5min" (requires rewrite)
- `ADD COLUMN` with volatile default → scales with table size
- Safe operations → "< 1s"

## Step 7.2: Planner

**File:** `internal/planner/planner.go`

```go
type MigrationStep struct {
    Migration   *migration.Migration
    Findings    []analyzer.Finding
    Impacts     []Impact
    Status      string   // "pending", "applied", "failed"
    RunInTx     bool     // false for CREATE INDEX CONCURRENTLY
}

type Plan struct {
    Steps         []MigrationStep
    TotalPending  int
    HighRiskCount int
    CriticalCount int
}

func BuildPlan(migrations []migration.Migration, results []analyzer.AnalysisResult, applied map[string]bool) *Plan
```

- Optionally accepts a database pool to query `pg_class` for table sizes:
  ```sql
  SELECT pg_total_relation_size(c.oid) FROM pg_class c WHERE c.relname = $1
  ```

## Step 7.3: Wire Plan Command

**File:** `internal/cli/plan.go` (update from stub)

- Load migrations, run analyzer
- Optionally connect to DB (if `--database-url` provided) for table sizes and applied status
- Build plan
- Display as a formatted table:
  ```
  Migration Plan
  ══════════════════════════════════════════════════

  #  Version  Name              Status    Risk       Est. Lock
  ─  ───────  ────              ──────    ────       ─────────
  1  V001     create_users      applied   -          -
  2  V002     add_email_index   pending   HIGH       > 5min
  3  V003     add_column_def    pending   SAFE       < 1s

  Summary: 2 pending, 1 HIGH risk
  ```
- `--pending-only` flag: only show pending migrations

## Step 7.4: Verify & Commit

- `make test-unit` — planner tests pass
