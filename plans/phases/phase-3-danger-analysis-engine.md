# Phase 3: Danger Analysis Engine (Core Value)

> **Status:** Pending
> **Commit:** `feat: implement danger analysis engine with 9 detection rules`

## Goal

The heart of the tool — detect dangerous DDL operations and provide actionable suggestions.

---

## Step 3.1: Severity Enum

**File:** `internal/analyzer/severity.go`

```go
type Severity int

const (
    Safe Severity = iota
    Low
    Medium
    High
    Critical
)

func (s Severity) String() string    // "SAFE", "LOW", "MEDIUM", "HIGH", "CRITICAL"
func (s Severity) Color() string     // Terminal color codes for output
```

## Step 3.2: Analysis Result Types

**File:** `internal/analyzer/result.go`

```go
type Finding struct {
    Rule        string   // Rule ID (e.g., "create-index-not-concurrent")
    Severity    Severity
    Table       string   // Affected table name
    Statement   string   // The SQL statement (truncated for display)
    Message     string   // Human-readable description of the danger
    Suggestion  string   // Safe alternative approach
    LockType    string   // e.g., "ACCESS EXCLUSIVE"
    StmtIndex   int      // Index in the migration's statement list
}

type AnalysisResult struct {
    Migration  *migration.Migration
    Findings   []Finding
    MaxSeverity Severity   // Highest severity across all findings
}
```

## Step 3.3: Rule Interface + Registry

**File:** `internal/analyzer/rules.go`

```go
type Rule interface {
    // ID returns a unique identifier (e.g., "create-index-not-concurrent")
    ID() string
    // Check examines a single parsed statement and returns findings
    Check(stmt *pg_query.RawStmt, ctx *RuleContext) []Finding
}

type RuleContext struct {
    Migration       *migration.Migration
    TargetPGVersion int
    StmtIndex       int
}

type Registry struct {
    rules []Rule
}

func NewDefaultRegistry() *Registry  // Returns registry with all 9+ rules registered
func (r *Registry) Register(rule Rule)
func (r *Registry) Rules() []Rule
```

## Step 3.4: Analyzer Orchestrator

**File:** `internal/analyzer/analyzer.go`

```go
type Analyzer struct {
    registry *Registry
    parser   func(string) (*parser.ParseResult, error)  // injectable for testing
    pgVersion int
}

func New(opts ...Option) *Analyzer
func (a *Analyzer) Analyze(m *migration.Migration) (*AnalysisResult, error)
func (a *Analyzer) AnalyzeAll(migrations []migration.Migration) ([]AnalysisResult, error)
```

Flow:
1. Parse the migration SQL using `parser.Parse()`
2. For each statement in the parse result, run every registered rule
3. Collect all findings, compute max severity
4. Return `AnalysisResult`

**File:** `internal/analyzer/analyzer_test.go`

- Analyze a safe migration → 0 findings, max severity SAFE
- Analyze a migration with known danger → correct finding count and severity
- Analyze multiple migrations → results per migration

## Step 3.5: Rule Implementations

Each rule file follows the same pattern:
1. Define a struct implementing `Rule`
2. `ID()` returns a descriptive kebab-case identifier
3. `Check()` examines the AST node type and returns findings if dangerous pattern detected
4. Companion `_test.go` with table-driven tests covering: dangerous case, safe case, edge cases

### Rule R-1: Non-concurrent CREATE INDEX

**File:** `internal/analyzer/rules/create_index.go`

- Match: `*pg_query.IndexStmt` where `ConcurrentlyCreated` is false (and it's not a `CREATE UNIQUE INDEX` on a new table within the same migration — though this heuristic can be simplified)
- Severity: HIGH
- Suggestion: "Use CREATE INDEX CONCURRENTLY to avoid locking the table"
- Edge case: `CREATE INDEX CONCURRENTLY` → Safe, no finding

**Tests:** Standard index → HIGH finding. `CONCURRENTLY` → no finding. `CREATE UNIQUE INDEX` → HIGH finding (still needs CONCURRENTLY in production).

### Rule R-2: ADD COLUMN with Volatile DEFAULT

**File:** `internal/analyzer/rules/alter_add_column.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AddColumn` where the column definition has a DEFAULT expression
- PG version awareness:
  - PG 11+: Non-volatile defaults (literals like `'active'`, `0`, `true`) are safe. Only volatile functions (`now()`, `random()`, `gen_random_uuid()`) are dangerous.
  - PG 10 and below: ANY default is dangerous (causes full table rewrite)
- Severity: HIGH (when dangerous)
- Suggestion: "Add column without DEFAULT, then backfill in batches"

**Tests:** `ADD COLUMN status TEXT DEFAULT 'active'` with PG14 → Safe. Same with PG10 → HIGH. `ADD COLUMN created_at DEFAULT now()` → HIGH on any version. `ADD COLUMN bio TEXT` (no default) → Safe.

### Rule R-3: ADD CONSTRAINT without NOT VALID

**File:** `internal/analyzer/rules/alter_add_constraint.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AddConstraint` where the constraint is a CHECK or FOREIGN KEY and `SkipValidation` is false
- Does NOT flag: `NOT NULL` constraints (separate rule), `UNIQUE`/`PRIMARY KEY` (these always need validation)
- Severity: HIGH
- Suggestion: "Add with NOT VALID, then VALIDATE CONSTRAINT in a separate statement"

**Tests:** CHECK constraint without NOT VALID → HIGH. CHECK with NOT VALID → Safe. FOREIGN KEY without NOT VALID → HIGH. PRIMARY KEY → no finding (not applicable).

### Rule R-4: ALTER COLUMN TYPE

**File:** `internal/analyzer/rules/alter_column_type.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AlterColumnType`
- This is almost always a full table rewrite, regardless of PG version
- Exception: Some casts are safe (e.g., `varchar(50)` → `varchar(100)`, `int` → `bigint`) but detecting these reliably is complex. For v1, flag ALL type changes.
- Severity: HIGH
- Suggestion: "Use a staged approach: add new column, backfill data, swap columns, drop old column"

**Tests:** `ALTER COLUMN email TYPE VARCHAR(255)` → HIGH. Any type change → HIGH.

### Rule R-5: SET NOT NULL

**File:** `internal/analyzer/rules/alter_set_not_null.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_SetNotNull`
- PG version awareness:
  - PG 12+: Can use CHECK constraint NOT VALID approach (add CHECK, validate, then SET NOT NULL skips the scan)
  - PG 11 and below: Full table scan required, no workaround
- Severity: MEDIUM (PG12+, has workaround) or HIGH (PG11-, no workaround)
- Suggestion PG12+: "First add CHECK (col IS NOT NULL) NOT VALID, then VALIDATE CONSTRAINT, then SET NOT NULL"
- Suggestion PG11-: "Requires full table scan. Consider application-level enforcement instead."

**Tests:** `SET NOT NULL` with PG14 → MEDIUM. With PG10 → HIGH. Correct suggestions per version.

### Rule R-6: DROP TABLE / TRUNCATE

**File:** `internal/analyzer/rules/drop_table.go`

- Match: `*pg_query.DropStmt` where `RemoveType` is `OBJECT_TABLE`, or `*pg_query.TruncateStmt`
- Severity: CRITICAL
- Suggestion: "This is irreversible. Ensure you have a backup and that no application code references this table."
- Also detect `DROP TABLE IF EXISTS` (same severity, but note the IF EXISTS in the message)

**Tests:** `DROP TABLE users` → CRITICAL. `DROP TABLE IF EXISTS users` → CRITICAL. `TRUNCATE users` → CRITICAL. `DROP INDEX` → no finding (different rule concern).

### Rule R-8: VACUUM FULL

**File:** `internal/analyzer/rules/vacuum_full.go`

- Match: `*pg_query.VacuumStmt` where the options include `VACOPT_FULL`
- Severity: HIGH
- Suggestion: "Use regular VACUUM instead. VACUUM FULL rewrites the entire table and holds an ACCESS EXCLUSIVE lock."

**Tests:** `VACUUM FULL users` → HIGH. `VACUUM users` → no finding. `VACUUM (FULL) users` → HIGH.

### Rule R-9: Explicit LOCK TABLE

**File:** `internal/analyzer/rules/lock_table.go`

- Match: `*pg_query.LockStmt`
- Severity: HIGH
- Suggestion: "Avoid explicit table locks. Let PostgreSQL manage locking through normal operations."

**Tests:** `LOCK TABLE users IN ACCESS EXCLUSIVE MODE` → HIGH. `LOCK TABLE users IN SHARE MODE` → HIGH (still flag any explicit lock).

### Rule R-10: RENAME TABLE / COLUMN

**File:** `internal/analyzer/rules/rename.go`

- Match: `*pg_query.RenameStmt` where `RenameType` is `OBJECT_TABLE` or `OBJECT_COLUMN`
- Severity: MEDIUM
- Suggestion: "Renaming breaks application code that references the old name. Use a staged approach: add new name, update app code, remove old name."

**Tests:** `RENAME COLUMN email TO email_address` → MEDIUM. `RENAME TABLE users TO customers` → MEDIUM. `RENAME INDEX` → no finding (safe operation in PG).

## Step 3.6: Wire Analyze Command

**File:** `internal/cli/analyze.go` (update from stub)

- Load migrations from `--migrations-dir` using `migration.LoadFromDir()`
- Sort with `migration.Sort()`
- Create analyzer with `analyzer.New()` and default registry
- Run `analyzer.AnalyzeAll()`
- Format output based on `--format` flag (text for now, JSON/GH Actions in Phase 8)
- If `--fail-on-high` and any finding is HIGH or CRITICAL, exit with code 1

## Step 3.7: Verify & Commit

- Run `make test-unit` — all rule tests pass with `-race`
- Run `bin/migrate analyze ./testdata/migrations/` and verify:
  - V001 → no findings (safe)
  - V002 → R-1 fires (non-concurrent index)
  - V003 → R-2 may or may not fire depending on target PG version
  - V004 → R-3 fires (constraint without NOT VALID)
  - V005 → R-4 fires (type change)
  - V006 → R-5 fires (SET NOT NULL)
  - V007 → R-6 fires (DROP TABLE)
  - V008 → R-8 fires (VACUUM FULL)
  - V009 → R-9 fires (LOCK TABLE)
  - V010 → R-10 fires (RENAME)
  - V011 → no findings (CONCURRENTLY)
  - V012 → no findings (nullable, no default)
