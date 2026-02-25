# Phase 0: Quality Infrastructure

> **Status:** Complete
> **Commit:** `chore: add quality infrastructure — linting, coverage, hooks, makefile`

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
├── integration/
│   ├── lifecycle_test.go              # Full apply → status → rollback → status test
│   └── testhelpers.go                 # Shared testcontainers setup
├── plans/                              # Project documentation
│   ├── project.md
│   ├── prd.md
│   ├── implementation_plan.md
│   ├── checklist.md
│   └── phases/
├── config.example.yml                  # Example configuration file
├── Makefile                            # Build, test, lint commands
└── CLAUDE.md                           # Project-specific Claude Code guidance
```

---

## Goal

Establish linting, formatting, testing standards, coverage enforcement, and pre-commit hooks before any feature code is written. Every line of code from Phase 1 onward is held to these standards.

## Files Created

| File | Purpose |
|---|---|
| `.golangci.yml` | golangci-lint v2 config — standard defaults + errorlint, wrapcheck, gocritic, revive, exhaustive, gosec, cyclop (max 15), funlen (80 lines), testifylint, tparallel, and more. Test file exclusions. nolint requires explanation. |
| `.testcoverage.yml` | Coverage thresholds — 80% total, 75% per-package, 70% per-file. 90% on analyzer rules, 85% on executor. |
| `lefthook.yml` | Pre-commit: gofumpt + goimports + golangci-lint + govet + go mod tidy (parallel, auto-stage fixes). Pre-push: tests + coverage check. |
| `Makefile` | Targets: build, fmt, fmt-check, vet, lint, lint-fix, test, test-short, test-integration, coverage, coverage-html, coverage-check, audit (full gate), clean. |
| `.editorconfig` | LF line endings, UTF-8, tabs for Go/Makefile, 2-space for YAML. |
| `CLAUDE.md` | Project-specific coding standards, testing conventions, architecture overview. |

## Quality Gate

`make audit` runs the full gate: format check → go vet → golangci-lint → tests with race detection → coverage threshold check → go mod tidy/verify.

## Key Standards Enforced

- **Linting:** 25+ linters including error handling, correctness, complexity, security, test quality
- **Formatting:** gofumpt + goimports with local import grouping (auto-fixed on commit)
- **Testing:** Table-driven tests, t.Parallel(), require vs assert, t.Helper(), t.Cleanup()
- **Coverage:** 90% on analyzer rules, 85% on executor, 80% total, 70% per-file
- **Commits:** Pre-commit hooks prevent committing lint errors or unformatted code
