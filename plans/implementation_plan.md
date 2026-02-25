# Implementation Plan: Database Migration Engine

## Prerequisites

### Environment Setup
- **Go 1.22+** must be installed (`brew install go`)
- **Docker** must be running (for testcontainers-go integration tests)
- **CGO_ENABLED=1** required for `pg_query_go` (wraps a C library via libpg_query)

### Key Dependencies
| Package | Version | Purpose |
|---|---|---|
| `github.com/pganalyze/pg_query_go/v5` | v5.x | Real PostgreSQL parser (C library wrapper) |
| `github.com/spf13/cobra` | v1.8+ | CLI framework |
| `github.com/jackc/pgx/v5` | v5.x | PostgreSQL driver (pure Go + pgx pool) |
| `github.com/stretchr/testify` | v1.9+ | Test assertions and mocking |
| `github.com/testcontainers/testcontainers-go` | v0.31+ | Docker-based integration tests |
| `gopkg.in/yaml.v3` | v3.x | YAML config parsing |
| `github.com/charmbracelet/lipgloss` | v0.10+ | Terminal styling and colored output |

---

## Project Structure

```
database_migration_engine/
├── cmd/
│   └── migrate/
│       └── main.go                     # Entry point — initializes Cobra root and runs
├── internal/
│   ├── cli/                            # CLI layer — Cobra commands + terminal output
│   │   ├── root.go                     # Root command, global flags (--config, --database-url, --verbose)
│   │   ├── analyze.go                  # `analyze` subcommand
│   │   ├── plan.go                     # `plan` subcommand
│   │   ├── apply.go                    # `apply` subcommand
│   │   ├── rollback.go                # `rollback` subcommand
│   │   ├── status.go                  # `status` subcommand
│   │   └── output.go                  # Shared output formatting (tables, colors, JSON, GH Actions)
│   ├── config/
│   │   └── config.go                   # YAML config loading + env var overrides + flag merging
│   ├── parser/
│   │   ├── parser.go                   # Wraps pg_query_go Parse(), returns typed AST nodes
│   │   └── parser_test.go              # Unit tests for parser
│   ├── migration/
│   │   ├── migration.go                # Migration type: Version, Name, UpSQL, DownSQL, Checksum
│   │   ├── loader.go                  # Read .sql files from disk, pair up/down files
│   │   ├── loader_test.go             # Unit tests for loader
│   │   ├── sorter.go                  # Sort migrations by version/timestamp
│   │   └── sorter_test.go             # Unit tests for sorter
│   ├── analyzer/
│   │   ├── analyzer.go                 # Orchestrator: load migrations → parse → run rules → collect results
│   │   ├── analyzer_test.go            # Integration-style tests for the full pipeline
│   │   ├── rules.go                    # Rule interface + RuleRegistry
│   │   ├── severity.go                # Severity enum: Safe, Low, Medium, High, Critical
│   │   ├── result.go                  # Finding type: Rule, Severity, Table, Line, Message, Suggestion
│   │   └── rules/                      # One file per detection rule
│   │       ├── create_index.go         # R-1: Non-concurrent CREATE INDEX
│   │       ├── create_index_test.go
│   │       ├── alter_add_column.go     # R-2: ADD COLUMN with volatile DEFAULT
│   │       ├── alter_add_column_test.go
│   │       ├── alter_add_constraint.go # R-3: ADD CONSTRAINT without NOT VALID
│   │       ├── alter_add_constraint_test.go
│   │       ├── alter_column_type.go    # R-4: ALTER COLUMN TYPE
│   │       ├── alter_column_type_test.go
│   │       ├── alter_set_not_null.go   # R-5: SET NOT NULL
│   │       ├── alter_set_not_null_test.go
│   │       ├── drop_table.go           # R-6: DROP TABLE + TRUNCATE
│   │       ├── drop_table_test.go
│   │       ├── vacuum_full.go          # R-8: VACUUM FULL
│   │       ├── vacuum_full_test.go
│   │       ├── lock_table.go           # R-9: Explicit LOCK TABLE
│   │       ├── lock_table_test.go
│   │       ├── rename.go              # R-10: RENAME TABLE / COLUMN
│   │       └── rename_test.go
│   ├── planner/
│   │   ├── planner.go                 # Build MigrationPlan from analysis results
│   │   ├── planner_test.go
│   │   └── impact.go                  # Estimate lock duration based on operation + optional table size
│   ├── executor/
│   │   ├── executor.go                 # Execute migrations: acquire lock → set timeouts → run SQL → record
│   │   ├── executor_test.go
│   │   ├── transaction.go             # Transaction wrapping, savepoint management
│   │   └── safety.go                  # SET lock_timeout, SET statement_timeout helpers
│   ├── tracker/
│   │   ├── tracker.go                 # CRUD for schema_migrations table
│   │   ├── tracker_test.go
│   │   └── schema.go                  # DDL for creating/migrating the tracking table
│   └── database/
│       ├── connection.go               # pgx pool setup with connection string parsing
│       ├── connection_test.go
│       └── advisory_lock.go           # pg_try_advisory_lock / pg_advisory_unlock wrappers
├── testdata/
│   └── migrations/                     # Test SQL files covering all rule patterns
│       ├── V001_create_users.up.sql
│       ├── V001_create_users.down.sql
│       ├── V002_add_email_index.up.sql
│       ├── V002_add_email_index.down.sql
│       ├── V003_add_column_default.up.sql
│       ├── V003_add_column_default.down.sql
│       ├── V004_add_constraint.up.sql
│       ├── V004_add_constraint.down.sql
│       ├── V005_alter_column_type.up.sql
│       ├── V005_alter_column_type.down.sql
│       ├── V006_set_not_null.up.sql
│       ├── V006_set_not_null.down.sql
│       ├── V007_drop_table.up.sql
│       ├── V007_drop_table.down.sql
│       ├── V008_vacuum_full.up.sql
│       ├── V009_lock_table.up.sql
│       ├── V010_rename_column.up.sql
│       ├── V010_rename_column.down.sql
│       ├── V011_safe_concurrent_index.up.sql
│       ├── V011_safe_concurrent_index.down.sql
│       └── V012_safe_add_column.up.sql
├── integration/
│   ├── lifecycle_test.go              # Full apply → status → rollback → status test
│   └── testhelpers.go                 # Shared testcontainers setup
├── config.example.yml                  # Example configuration file
├── Makefile                            # Build, test, lint commands
├── CLAUDE.md                           # Project-specific Claude Code guidance
├── prd.md                              # Product requirements document
├── implementation_plan.md              # This file
└── project.md                          # Project overview
```

---

## Phase 0: Quality Infrastructure

**Goal:** Establish linting, formatting, testing standards, coverage enforcement, and pre-commit hooks before any feature code is written. Every line of code from Phase 1 onward is held to these standards.

### Files Created

| File | Purpose |
|---|---|
| `.golangci.yml` | golangci-lint v2 config — standard defaults + errorlint, wrapcheck, gocritic, revive, exhaustive, gosec, cyclop (max 15), funlen (80 lines), testifylint, tparallel, and more. Test file exclusions. nolint requires explanation. |
| `.testcoverage.yml` | Coverage thresholds — 80% total, 75% per-package, 70% per-file. 90% on analyzer rules, 85% on executor. |
| `lefthook.yml` | Pre-commit: gofumpt + goimports + golangci-lint + govet + go mod tidy (parallel, auto-stage fixes). Pre-push: tests + coverage check. |
| `Makefile` | Targets: build, fmt, fmt-check, vet, lint, lint-fix, test, test-short, test-integration, coverage, coverage-html, coverage-check, audit (full gate), clean. |
| `.editorconfig` | LF line endings, UTF-8, tabs for Go/Makefile, 2-space for YAML. |
| `CLAUDE.md` | Project-specific coding standards, testing conventions, architecture overview. |

### Quality Gate

`make audit` runs the full gate: format check → go vet → golangci-lint → tests with race detection → coverage threshold check → go mod tidy/verify.

### Key Standards Enforced

- **Linting:** 25+ linters including error handling, correctness, complexity, security, test quality
- **Formatting:** gofumpt + goimports with local import grouping (auto-fixed on commit)
- **Testing:** Table-driven tests, t.Parallel(), require vs assert, t.Helper(), t.Cleanup()
- **Coverage:** 90% on analyzer rules, 85% on executor, 80% total, 70% per-file
- **Commits:** Pre-commit hooks prevent committing lint errors or unformatted code

**Commit:** `chore: add quality infrastructure — linting, coverage, hooks, makefile`

---

## Phase 1: Project Scaffolding + CLI Skeleton

**Goal:** Runnable binary with 5 subcommand stubs, config loading, and project infrastructure.

### Step 1.1: Initialize Go Module

Create `go.mod` with module path `github.com/ahmad/migrate`.

**File:** `go.mod`
```
module github.com/ahmad/migrate

go 1.22
```

Run `go mod tidy` after adding initial dependencies.

### Step 1.2: Create Entry Point

**File:** `cmd/migrate/main.go`

- Import and execute `internal/cli.Execute()`
- Minimal — just the bridge between `main()` and Cobra

### Step 1.3: Root Command + Global Flags

**File:** `internal/cli/root.go`

- Define root `cobra.Command` with app name `migrate`, version, description
- Global persistent flags:
  - `--config` (string, default `migrate.yml`) — path to config file
  - `--database-url` (string) — PostgreSQL connection string
  - `--migrations-dir` (string, default `./migrations`) — path to migration files
  - `--verbose` (bool) — enable debug logging
- `PersistentPreRunE`: load config file, merge env vars, merge flags (flag > env > file)
- Export `Execute()` function that calls `rootCmd.Execute()`

### Step 1.4: Five Subcommand Stubs

**Files:** `internal/cli/analyze.go`, `plan.go`, `apply.go`, `rollback.go`, `status.go`

Each file:
- Defines a `cobra.Command` with name, short/long description, relevant flags
- `RunE` function that prints "Not yet implemented" (placeholder)
- Registers as subcommand of root in `init()`

**Command-specific flags:**
- `analyze`: `--format` (text/json/github-actions), `--fail-on-high`
- `plan`: `--pending-only`
- `apply`: `--dry-run`, `--force`, `--lock-timeout` (duration), `--statement-timeout` (duration)
- `rollback`: `--steps` (int), `--target` (string)
- `status`: `--format` (text/json)

### Step 1.5: Configuration

**File:** `internal/config/config.go`

```go
type Config struct {
    DatabaseURL      string        `yaml:"database_url"`
    MigrationsDir    string        `yaml:"migrations_dir"`
    LockTimeout      time.Duration `yaml:"lock_timeout"`
    StatementTimeout time.Duration `yaml:"statement_timeout"`
    TargetPGVersion  int           `yaml:"target_pg_version"` // e.g., 14
    Format           string        `yaml:"format"`
}
```

- `Load(path string) (*Config, error)` — reads YAML file
- `MergeEnv(cfg *Config)` — override with `MIGRATE_DATABASE_URL`, `MIGRATE_MIGRATIONS_DIR`, etc.
- Provide sensible defaults: lock_timeout=5s, statement_timeout=30s, target_pg_version=14

### Step 1.6: Example Config

**File:** `config.example.yml`

```yaml
database_url: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
migrations_dir: "./migrations"
lock_timeout: "5s"
statement_timeout: "30s"
target_pg_version: 14
```

### Step 1.7: Makefile

**File:** `Makefile`

```makefile
.PHONY: build test test-unit test-integration lint clean

build:
	CGO_ENABLED=1 go build -o bin/migrate ./cmd/migrate

test: test-unit

test-unit:
	CGO_ENABLED=1 go test -race -count=1 ./internal/...

test-integration:
	CGO_ENABLED=1 go test -race -count=1 -tags=integration ./integration/...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
```

### Step 1.8: CLAUDE.md

**File:** `CLAUDE.md`

Project-specific guidance for Claude Code: stack, architecture, commands, conventions, current status.

### Step 1.9: Verify & Commit

- Run `go mod tidy` to resolve all dependencies
- Run `make build` to verify the binary compiles
- Run `bin/migrate --help` to verify all 5 subcommands appear
- Run `bin/migrate analyze` to verify it prints "not yet implemented"

**Commit:** `feat: scaffold project structure with CLI skeleton and config loading`

---

## Phase 2: SQL Parser + Migration Loading

**Goal:** Parse any valid PostgreSQL SQL into an AST, load migration files from disk, sort by version.

### Step 2.1: SQL Parser Wrapper

**File:** `internal/parser/parser.go`

```go
type ParseResult struct {
    Stmts []*pg_query.RawStmt  // Typed AST nodes from pg_query_go
    SQL   string                // Original SQL for reference
}

func Parse(sql string) (*ParseResult, error)
```

- Call `pg_query.Parse(sql)` which returns a `*pg_query.ParseResult`
- Return the list of `RawStmt` nodes
- Handle parse errors with descriptive messages (line number, position)

**File:** `internal/parser/parser_test.go`

Table-driven tests:
- Valid `CREATE TABLE` → 1 statement, correct node type
- Valid multi-statement SQL → correct count
- `CREATE INDEX CONCURRENTLY` → parses correctly
- `ALTER TABLE ADD COLUMN` → correct node type
- Invalid SQL → returns error with position info
- Empty string → returns 0 statements

### Step 2.2: Migration Type

**File:** `internal/migration/migration.go`

```go
type Migration struct {
    Version   string    // "001" or "20240101120000"
    Name      string    // "create_users"
    UpSQL     string    // Contents of .up.sql
    DownSQL   string    // Contents of .down.sql (may be empty)
    Checksum  string    // SHA-256 of UpSQL
    FilePath  string    // Path to .up.sql file
}
```

- `ComputeChecksum(sql string) string` — SHA-256 hex digest

### Step 2.3: Migration File Loader

**File:** `internal/migration/loader.go`

```go
func LoadFromDir(dir string) ([]Migration, error)
```

- Scan directory for `*.up.sql` files
- Parse filename to extract version and name:
  - Pattern 1: `V{version}_{name}.up.sql` (e.g., `V001_create_users.up.sql`)
  - Pattern 2: `{timestamp}_{name}.up.sql` (e.g., `20240101120000_create_users.up.sql`)
- Look for matching `.down.sql` file
- Read file contents, compute checksum
- Return unsorted slice

**File:** `internal/migration/loader_test.go`

- Load from `testdata/migrations/` — correct count and content
- Missing directory → error
- Empty directory → empty slice, no error
- File without matching pattern → skipped with warning
- `.down.sql` pairing works correctly

### Step 2.4: Migration Sorter

**File:** `internal/migration/sorter.go`

```go
func Sort(migrations []Migration) []Migration
```

- Sort by version string (lexicographic — works for both zero-padded numbers and timestamps)
- Stable sort to preserve order of equal versions (shouldn't happen, but be safe)

**File:** `internal/migration/sorter_test.go`

- Already sorted → unchanged
- Reverse order → correctly sorted
- Mixed version formats → sorted correctly
- Empty slice → no panic

### Step 2.5: Test Data Files

**Directory:** `testdata/migrations/`

Create 12 migration files that cover every danger rule. Each file should be a realistic, minimal example:

| File | Content | Purpose |
|---|---|---|
| `V001_create_users.up.sql` | `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW());` | Safe migration baseline |
| `V001_create_users.down.sql` | `DROP TABLE IF EXISTS users;` | Rollback for V001 |
| `V002_add_email_index.up.sql` | `CREATE INDEX idx_users_email ON users (email);` | **R-1: Non-concurrent index** |
| `V002_add_email_index.down.sql` | `DROP INDEX IF EXISTS idx_users_email;` | Rollback for V002 |
| `V003_add_column_default.up.sql` | `ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';` | **R-2: ADD COLUMN with DEFAULT** (safe on PG11+, dangerous on PG10) |
| `V003_add_column_default.down.sql` | `ALTER TABLE users DROP COLUMN IF EXISTS status;` | Rollback |
| `V004_add_constraint.up.sql` | `ALTER TABLE users ADD CONSTRAINT chk_email CHECK (email ~* '^.+@.+$');` | **R-3: ADD CONSTRAINT without NOT VALID** |
| `V004_add_constraint.down.sql` | `ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_email;` | Rollback |
| `V005_alter_column_type.up.sql` | `ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255);` | **R-4: ALTER COLUMN TYPE** |
| `V005_alter_column_type.down.sql` | `ALTER TABLE users ALTER COLUMN email TYPE TEXT;` | Rollback |
| `V006_set_not_null.up.sql` | `ALTER TABLE users ALTER COLUMN status SET NOT NULL;` | **R-5: SET NOT NULL** |
| `V006_set_not_null.down.sql` | `ALTER TABLE users ALTER COLUMN status DROP NOT NULL;` | Rollback |
| `V007_drop_table.up.sql` | `DROP TABLE users;` | **R-6: DROP TABLE** |
| `V007_drop_table.down.sql` | `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL);` | Rollback |
| `V008_vacuum_full.up.sql` | `VACUUM FULL users;` | **R-8: VACUUM FULL** |
| `V009_lock_table.up.sql` | `LOCK TABLE users IN ACCESS EXCLUSIVE MODE;` | **R-9: LOCK TABLE** |
| `V010_rename_column.up.sql` | `ALTER TABLE users RENAME COLUMN email TO email_address;` | **R-10: RENAME** |
| `V010_rename_column.down.sql` | `ALTER TABLE users RENAME COLUMN email_address TO email;` | Rollback |
| `V011_safe_concurrent_index.up.sql` | `CREATE INDEX CONCURRENTLY idx_users_status ON users (status);` | Safe — should NOT trigger R-1 |
| `V011_safe_concurrent_index.down.sql` | `DROP INDEX CONCURRENTLY IF EXISTS idx_users_status;` | Rollback |
| `V012_safe_add_column.up.sql` | `ALTER TABLE users ADD COLUMN bio TEXT;` | Safe — nullable, no default |

### Step 2.6: Verify & Commit

- Run `make test-unit` — all parser and loader tests pass
- Verify test data files parse without errors

**Commit:** `feat: integrate pg_query_go parser with migration file loading`

---

## Phase 3: Danger Analysis Engine (Core Value)

**Goal:** The heart of the tool — detect dangerous DDL operations and provide actionable suggestions.

### Step 3.1: Severity Enum

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

### Step 3.2: Analysis Result Types

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

### Step 3.3: Rule Interface + Registry

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

### Step 3.4: Analyzer Orchestrator

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

### Step 3.5: Rule Implementations

Each rule file follows the same pattern:
1. Define a struct implementing `Rule`
2. `ID()` returns a descriptive kebab-case identifier
3. `Check()` examines the AST node type and returns findings if dangerous pattern detected
4. Companion `_test.go` with table-driven tests covering: dangerous case, safe case, edge cases

#### Rule R-1: Non-concurrent CREATE INDEX

**File:** `internal/analyzer/rules/create_index.go`

- Match: `*pg_query.IndexStmt` where `ConcurrentlyCreated` is false (and it's not a `CREATE UNIQUE INDEX` on a new table within the same migration — though this heuristic can be simplified)
- Severity: HIGH
- Suggestion: "Use CREATE INDEX CONCURRENTLY to avoid locking the table"
- Edge case: `CREATE INDEX CONCURRENTLY` → Safe, no finding

**Tests:** Standard index → HIGH finding. `CONCURRENTLY` → no finding. `CREATE UNIQUE INDEX` → HIGH finding (still needs CONCURRENTLY in production).

#### Rule R-2: ADD COLUMN with Volatile DEFAULT

**File:** `internal/analyzer/rules/alter_add_column.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AddColumn` where the column definition has a DEFAULT expression
- PG version awareness:
  - PG 11+: Non-volatile defaults (literals like `'active'`, `0`, `true`) are safe. Only volatile functions (`now()`, `random()`, `gen_random_uuid()`) are dangerous.
  - PG 10 and below: ANY default is dangerous (causes full table rewrite)
- Severity: HIGH (when dangerous)
- Suggestion: "Add column without DEFAULT, then backfill in batches"

**Tests:** `ADD COLUMN status TEXT DEFAULT 'active'` with PG14 → Safe. Same with PG10 → HIGH. `ADD COLUMN created_at DEFAULT now()` → HIGH on any version. `ADD COLUMN bio TEXT` (no default) → Safe.

#### Rule R-3: ADD CONSTRAINT without NOT VALID

**File:** `internal/analyzer/rules/alter_add_constraint.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AddConstraint` where the constraint is a CHECK or FOREIGN KEY and `SkipValidation` is false
- Does NOT flag: `NOT NULL` constraints (separate rule), `UNIQUE`/`PRIMARY KEY` (these always need validation)
- Severity: HIGH
- Suggestion: "Add with NOT VALID, then VALIDATE CONSTRAINT in a separate statement"

**Tests:** CHECK constraint without NOT VALID → HIGH. CHECK with NOT VALID → Safe. FOREIGN KEY without NOT VALID → HIGH. PRIMARY KEY → no finding (not applicable).

#### Rule R-4: ALTER COLUMN TYPE

**File:** `internal/analyzer/rules/alter_column_type.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_AlterColumnType`
- This is almost always a full table rewrite, regardless of PG version
- Exception: Some casts are safe (e.g., `varchar(50)` → `varchar(100)`, `int` → `bigint`) but detecting these reliably is complex. For v1, flag ALL type changes.
- Severity: HIGH
- Suggestion: "Use a staged approach: add new column, backfill data, swap columns, drop old column"

**Tests:** `ALTER COLUMN email TYPE VARCHAR(255)` → HIGH. Any type change → HIGH.

#### Rule R-5: SET NOT NULL

**File:** `internal/analyzer/rules/alter_set_not_null.go`

- Match: `*pg_query.AlterTableStmt` with subcommand `AT_SetNotNull`
- PG version awareness:
  - PG 12+: Can use CHECK constraint NOT VALID approach (add CHECK, validate, then SET NOT NULL skips the scan)
  - PG 11 and below: Full table scan required, no workaround
- Severity: MEDIUM (PG12+, has workaround) or HIGH (PG11-, no workaround)
- Suggestion PG12+: "First add CHECK (col IS NOT NULL) NOT VALID, then VALIDATE CONSTRAINT, then SET NOT NULL"
- Suggestion PG11-: "Requires full table scan. Consider application-level enforcement instead."

**Tests:** `SET NOT NULL` with PG14 → MEDIUM. With PG10 → HIGH. Correct suggestions per version.

#### Rule R-6: DROP TABLE / TRUNCATE

**File:** `internal/analyzer/rules/drop_table.go`

- Match: `*pg_query.DropStmt` where `RemoveType` is `OBJECT_TABLE`, or `*pg_query.TruncateStmt`
- Severity: CRITICAL
- Suggestion: "This is irreversible. Ensure you have a backup and that no application code references this table."
- Also detect `DROP TABLE IF EXISTS` (same severity, but note the IF EXISTS in the message)

**Tests:** `DROP TABLE users` → CRITICAL. `DROP TABLE IF EXISTS users` → CRITICAL. `TRUNCATE users` → CRITICAL. `DROP INDEX` → no finding (different rule concern).

#### Rule R-8: VACUUM FULL

**File:** `internal/analyzer/rules/vacuum_full.go`

- Match: `*pg_query.VacuumStmt` where the options include `VACOPT_FULL`
- Severity: HIGH
- Suggestion: "Use regular VACUUM instead. VACUUM FULL rewrites the entire table and holds an ACCESS EXCLUSIVE lock."

**Tests:** `VACUUM FULL users` → HIGH. `VACUUM users` → no finding. `VACUUM (FULL) users` → HIGH.

#### Rule R-9: Explicit LOCK TABLE

**File:** `internal/analyzer/rules/lock_table.go`

- Match: `*pg_query.LockStmt`
- Severity: HIGH
- Suggestion: "Avoid explicit table locks. Let PostgreSQL manage locking through normal operations."

**Tests:** `LOCK TABLE users IN ACCESS EXCLUSIVE MODE` → HIGH. `LOCK TABLE users IN SHARE MODE` → HIGH (still flag any explicit lock).

#### Rule R-10: RENAME TABLE / COLUMN

**File:** `internal/analyzer/rules/rename.go`

- Match: `*pg_query.RenameStmt` where `RenameType` is `OBJECT_TABLE` or `OBJECT_COLUMN`
- Severity: MEDIUM
- Suggestion: "Renaming breaks application code that references the old name. Use a staged approach: add new name, update app code, remove old name."

**Tests:** `RENAME COLUMN email TO email_address` → MEDIUM. `RENAME TABLE users TO customers` → MEDIUM. `RENAME INDEX` → no finding (safe operation in PG).

### Step 3.6: Wire Analyze Command

**File:** `internal/cli/analyze.go` (update from stub)

- Load migrations from `--migrations-dir` using `migration.LoadFromDir()`
- Sort with `migration.Sort()`
- Create analyzer with `analyzer.New()` and default registry
- Run `analyzer.AnalyzeAll()`
- Format output based on `--format` flag (text for now, JSON/GH Actions in Phase 8)
- If `--fail-on-high` and any finding is HIGH or CRITICAL, exit with code 1

### Step 3.7: Verify & Commit

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

**Commit:** `feat: implement danger analysis engine with 9 detection rules`

---

## Phase 4: Migration Tracking + Database Connection

**Goal:** Connect to PostgreSQL, manage the `schema_migrations` table, enforce exclusive access with advisory locks.

### Step 4.1: Database Connection

**File:** `internal/database/connection.go`

```go
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)
```

- Parse connection string with `pgxpool.ParseConfig()`
- Set reasonable pool defaults (max 5 connections for a migration tool)
- Ping to verify connectivity
- Return the pool

**File:** `internal/database/connection_test.go`

- Test with invalid URL → error
- (Integration test will cover real connection)

### Step 4.2: Advisory Locks

**File:** `internal/database/advisory_lock.go`

```go
const MigrationLockID = 123456789  // Arbitrary unique lock ID

func TryAcquireLock(ctx context.Context, pool *pgxpool.Pool) (bool, error)
func ReleaseLock(ctx context.Context, pool *pgxpool.Pool) error
```

- Use `pg_try_advisory_lock(MigrationLockID)` — non-blocking, returns false if already held
- Use `pg_advisory_unlock(MigrationLockID)` for release
- Critical: use session-level locks (not transaction-level) so the lock persists across multiple transactions during multi-migration execution

### Step 4.3: Schema Migrations Table

**File:** `internal/tracker/schema.go`

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version      TEXT PRIMARY KEY,
    filename     TEXT NOT NULL,
    checksum     TEXT NOT NULL,
    applied_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms  INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'applied'  -- 'applied', 'failed', 'rolled_back'
);
```

- `EnsureTable(ctx, pool)` — creates the table if not exists

### Step 4.4: Migration Tracker

**File:** `internal/tracker/tracker.go`

```go
type Tracker struct {
    pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Tracker
func (t *Tracker) EnsureTable(ctx context.Context) error
func (t *Tracker) IsApplied(ctx context.Context, version string) (bool, error)
func (t *Tracker) GetApplied(ctx context.Context) ([]AppliedMigration, error)
func (t *Tracker) RecordApplied(ctx context.Context, m RecordParams) error
func (t *Tracker) RecordRolledBack(ctx context.Context, version string) error
func (t *Tracker) GetChecksum(ctx context.Context, version string) (string, error)
```

`AppliedMigration` mirrors the table columns. `RecordParams` includes version, filename, checksum, duration_ms.

**File:** `internal/tracker/tracker_test.go`

- These will be integration tests using testcontainers-go (see Step 4.5)

### Step 4.5: Integration Test Setup

**File:** `integration/testhelpers.go`

```go
func SetupPostgres(t *testing.T) (*pgxpool.Pool, func())
```

- Use `testcontainers-go` to start a PostgreSQL 16 container
- Return a connection pool and a cleanup function
- Build tag: `//go:build integration`

**File:** `integration/lifecycle_test.go` (stub for now, flesh out in Phase 5)

- Test: ensure tracker creates table, records migration, queries applied

### Step 4.6: Verify & Commit

- Run `make test-unit` for non-DB tests
- Run `make test-integration` — tracker tests pass against real PostgreSQL

**Commit:** `feat: add migration tracking with schema_migrations table`

---

## Phase 5: Execution Engine

**Goal:** Apply pending migrations with transactions, timeouts, advisory locks, and special handling for `CREATE INDEX CONCURRENTLY`.

### Step 5.1: Safety Helpers

**File:** `internal/executor/safety.go`

```go
func SetLockTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error
func SetStatementTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error
func ResetTimeouts(ctx context.Context, tx pgx.Tx) error
```

- Execute `SET lock_timeout = '5000ms'` etc. within the transaction
- These protect against runaway locks: if the migration can't acquire a lock within the timeout, it fails fast instead of blocking other queries

### Step 5.2: Transaction Management

**File:** `internal/executor/transaction.go`

```go
func ExecInTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error
func ExecWithoutTransaction(ctx context.Context, pool *pgxpool.Pool, sql string) error
```

- `ExecInTransaction`: begin → run fn → commit (rollback on error)
- `ExecWithoutTransaction`: for `CREATE INDEX CONCURRENTLY` which cannot run inside a transaction

### Step 5.3: Executor

**File:** `internal/executor/executor.go`

```go
type Executor struct {
    pool             *pgxpool.Pool
    tracker          *tracker.Tracker
    lockTimeout      time.Duration
    statementTimeout time.Duration
    dryRun           bool
    force            bool
    onProgress       func(event ProgressEvent)  // callback for CLI output
}

type ProgressEvent struct {
    Migration  *migration.Migration
    Status     string  // "starting", "completed", "failed", "skipped"
    Duration   time.Duration
    Error      error
}

func New(pool *pgxpool.Pool, tracker *tracker.Tracker, opts ...Option) *Executor

func (e *Executor) Apply(ctx context.Context, migrations []migration.Migration) error
func (e *Executor) Rollback(ctx context.Context, steps int) error
func (e *Executor) RollbackToVersion(ctx context.Context, targetVersion string) error
```

**Apply flow:**
1. Acquire advisory lock (fail if already held — another migration is running)
2. Ensure `schema_migrations` table exists
3. For each migration (in order):
   a. Check if already applied (skip if so, verify checksum matches)
   b. If `--dry-run`, log what would execute and skip
   c. Detect if SQL contains `CREATE INDEX CONCURRENTLY`
   d. If concurrent index: execute WITHOUT a transaction
   e. Otherwise: begin transaction → set lock_timeout → set statement_timeout → execute SQL → commit
   f. Record result in tracker (version, checksum, duration, status)
   g. Fire progress callback
   h. On error: record "failed" status, release advisory lock, return error
4. Release advisory lock

**Rollback flow:**
1. Acquire advisory lock
2. Get list of applied migrations (most recent first)
3. For each migration to rollback (up to `steps` count):
   a. Find the `.down.sql` content
   b. Execute within a transaction
   c. Update tracker status to "rolled_back"
4. Release advisory lock

### Step 5.4: Wire Apply Command

**File:** `internal/cli/apply.go` (update from stub)

- Load config, connect to database, create tracker and executor
- Load and sort migrations
- If not `--force`, run analyzer first:
  - If HIGH or CRITICAL findings exist, show them and prompt for confirmation
  - User must type "yes" or pass `--force` to proceed
- Call `executor.Apply()`
- Show progress (via callback) and final summary

### Step 5.5: Integration Tests

**File:** `integration/lifecycle_test.go`

- Test: apply 3 safe migrations → all tracked as "applied"
- Test: apply already-applied migrations → skipped
- Test: apply with checksum mismatch → error
- Test: apply with lock_timeout → migration fails if lock can't be acquired (requires a second connection holding a lock)
- Test: `CREATE INDEX CONCURRENTLY` executes outside transaction

### Step 5.6: Verify & Commit

- `make test-unit` passes
- `make test-integration` passes (full apply lifecycle)

**Commit:** `feat: implement migration executor with lock_timeout and transaction safety`

---

## Phase 6: Rollback Support

**Goal:** Revert applied migrations using `.down.sql` files.

### Step 6.1: Update Loader for Down Files

The loader from Phase 2 already pairs `.up.sql` and `.down.sql` files. Verify that `Migration.DownSQL` is populated correctly.

If a migration has no `.down.sql` file and rollback is attempted, return a clear error: "No rollback file found for migration V007."

### Step 6.2: Implement Rollback in Executor

Already stubbed in Phase 5. Flesh out:

```go
func (e *Executor) Rollback(ctx context.Context, migrations []migration.Migration, steps int) error
```

- Get applied migrations from tracker (ordered by version DESC)
- Take first `steps` entries
- For each, find the matching `Migration` object to get `DownSQL`
- Execute `DownSQL` in a transaction
- Update tracker: set status to "rolled_back"
- Fire progress callback

### Step 6.3: Wire Rollback Command

**File:** `internal/cli/rollback.go` (update from stub)

- `--steps N` (default 1): rollback last N migrations
- `--target VERSION`: rollback all migrations after this version
- Require database connection
- Show what will be rolled back, confirm unless `--force`

### Step 6.4: Integration Tests

Add to `integration/lifecycle_test.go`:
- Apply 3 migrations → rollback 1 → verify only 2 remain applied
- Rollback to target version → correct migrations reverted
- Rollback migration with no `.down.sql` → clear error

### Step 6.5: Verify & Commit

- `make test-unit` and `make test-integration` pass

**Commit:** `feat: add rollback support with .down.sql file convention`

---

## Phase 7: Planner + Impact Estimation

**Goal:** Show a detailed execution plan with risk assessment and estimated impact before running `apply`.

### Step 7.1: Impact Estimation

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

### Step 7.2: Planner

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

### Step 7.3: Wire Plan Command

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

### Step 7.4: Verify & Commit

- `make test-unit` — planner tests pass

**Commit:** `feat: add migration planner with impact estimation`

---

## Phase 8: Polish — Output, Status, CI Formats

**Goal:** Professional terminal output, status command, JSON/GitHub Actions output for CI/CD.

### Step 8.1: Output Formatting

**File:** `internal/cli/output.go`

- Use lipgloss for colored terminal output:
  - CRITICAL: bright red, bold
  - HIGH: red
  - MEDIUM: yellow
  - LOW: cyan
  - SAFE: green
- Table formatting for status and plan commands (use lipgloss table or simple aligned columns)
- JSON output: structured data matching the internal types, one JSON object per migration
- GitHub Actions output: `::warning file={path},line={line}::{message}` and `::error` annotations

### Step 8.2: Status Command

**File:** `internal/cli/status.go` (update from stub)

- Connect to database
- Query tracker for all applied migrations
- Load migrations from disk
- Cross-reference: show applied, pending, checksum mismatches
- Display as a table:
  ```
  Migration Status
  ══════════════════════════════════════════════════

  Version  Name              Status       Applied At            Duration
  ───────  ────              ──────       ──────────            ────────
  V001     create_users      applied      2024-01-15 10:30:00   45ms
  V002     add_email_index   applied      2024-01-15 10:30:01   2.3s
  V003     add_column_def    pending      -                     -

  ⚠ V002 checksum mismatch! File was modified after applying.
  ```

### Step 8.3: Confirmation Prompts

Update `apply.go`:
- When analysis finds HIGH/CRITICAL risks and `--force` is not set:
  ```
  ⚠ 2 HIGH risk operations detected:
    - V002: CREATE INDEX without CONCURRENTLY (ACCESS EXCLUSIVE lock)
    - V005: ALTER COLUMN TYPE (full table rewrite)

  Type "yes" to proceed, or use --force to skip this check:
  ```
- Read from stdin, only proceed if input is exactly "yes"

### Step 8.4: Format Flags

Update all output-producing commands:
- `--format text` (default): colored terminal output
- `--format json`: structured JSON (one object for analyze, array of migrations for status)
- `--format github-actions`: annotation format for GitHub Actions CI

### Step 8.5: Verify & Commit

- Manual testing: `migrate analyze ./testdata/migrations/` shows colored output
- Manual testing: `migrate analyze ./testdata/migrations/ --format json` outputs valid JSON
- Manual testing: `migrate status` shows formatted table (requires DB)
- `make test-unit` passes

**Commit:** `feat: add colored CLI output, status command, and JSON format`

---

## Phase 9: Documentation + Release Configuration

**Goal:** Production-ready README, release configuration, CI/CD.

### Step 9.1: README.md

Comprehensive README with:
- Project title, badges (Go version, test status, license)
- One-paragraph description
- Architecture diagram (ASCII art showing: SQL Files → Parser → Analyzer → Rules → Plan → Executor → PostgreSQL)
- Installation instructions (go install, brew, binary download)
- Quick start with examples of each command
- Full command reference
- Configuration reference
- List of all detection rules with examples
- Contributing section

### Step 9.2: GoReleaser Configuration

**File:** `.goreleaser.yml`

- Build for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- CGO_ENABLED=1 (note: cross-compilation with CGO requires special setup)
- Generate checksums, create GitHub release

### Step 9.3: GitHub Actions CI

**File:** `.github/workflows/ci.yml`

- Trigger on push and PR to main
- Jobs: lint (golangci-lint), test-unit, test-integration (with postgres service container)
- Cache Go modules

### Step 9.4: Verify & Commit

- README renders correctly on GitHub
- CI passes

**Commit:** `docs: add README with architecture diagram and usage examples`

---

## Verification Checklist

### Unit Tests (no Docker required)
- [ ] Parser: valid SQL → correct AST, invalid SQL → error
- [ ] Loader: reads files, pairs up/down, computes checksums
- [ ] Sorter: orders by version
- [ ] All 9+ analyzer rules: table-driven tests with safe/dangerous/edge cases
- [ ] Planner: builds plan from analysis results
- [ ] Config: loads YAML, merges env vars

### Integration Tests (require Docker)
- [ ] Connect to real PostgreSQL via testcontainers
- [ ] Tracker: create table, record, query, checksum verification
- [ ] Executor: apply migrations, verify they're recorded
- [ ] Executor: CREATE INDEX CONCURRENTLY runs outside transaction
- [ ] Executor: lock_timeout causes fast failure
- [ ] Full lifecycle: apply → status → rollback → status

### Manual Smoke Tests
- [ ] `migrate analyze ./testdata/migrations/` — all rules fire on expected files
- [ ] `migrate analyze --format json` — valid JSON output
- [ ] `migrate analyze --fail-on-high` — exits with code 1
- [ ] `migrate plan ./testdata/migrations/` — shows plan table
- [ ] `migrate apply --dry-run` — shows what would execute without changes
- [ ] `migrate apply` — executes and tracks migrations
- [ ] `migrate status` — shows applied/pending table
- [ ] `migrate rollback --steps 1` — reverts last migration

---

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| `pg_query_go` CGO complexity | Pin to v5, test build on CI early. Use `CGO_ENABLED=1` everywhere. |
| AST node types may change across pg_query versions | Pin dependency, test extensively, add version to go.mod |
| testcontainers-go requires Docker | CI runs Docker. Local dev can skip with `make test-unit`. |
| Cross-compilation with CGO is hard | Phase 9 goreleaser may need Docker-based cross-compilation or platform-specific CI jobs. This is a packaging concern, not a functionality concern. |
| Rule false positives | Start conservative (flag more, not fewer). Users can use `--force` to override. |

---

## Estimated Session Breakdown

| Session | Phases | Focus |
|---|---|---|
| Session 0 | Phase 0 | Quality infrastructure — linting, coverage, hooks, makefile |
| Session 1 | Phase 1 | Scaffolding, CLI, config |
| Session 2 | Phase 2 | Parser, loader, test data |
| Session 3 | Phase 3 (part 1) | Analyzer framework + rules R-1 through R-5 |
| Session 4 | Phase 3 (part 2) | Rules R-6 through R-10 + wire analyze command |
| Session 5 | Phase 4 | Database, tracker, advisory locks, integration test setup |
| Session 6 | Phase 5 | Executor, apply command, integration tests |
| Session 7 | Phase 6 | Rollback support |
| Session 8 | Phase 7 | Planner, impact estimation, plan command |
| Session 9 | Phase 8 | Polish: output formatting, status command, CI formats |
| Session 10 | Phase 9 | README, goreleaser, GitHub Actions |

Each session follows: `/clear` → read CLAUDE.md → implement phase → test → commit.
