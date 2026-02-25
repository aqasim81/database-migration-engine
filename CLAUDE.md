# Database Migration Engine — CLAUDE.md

## Project Overview

Zero-downtime PostgreSQL schema migration CLI. Parses SQL with the real PG parser, detects dangerous DDL operations, suggests safe alternatives, executes with rollback capability.

**Status:** Phase 1 — CLI skeleton with 5 subcommand stubs and config loading.

## Stack

| Component | Choice |
|---|---|
| Language | Go 1.22+ |
| SQL Parser | `pg_query_go` v5 (wraps libpg_query C library) |
| CLI | Cobra |
| PostgreSQL | pgx v5 |
| Testing | go test + testify + testcontainers-go |
| Config | YAML (gopkg.in/yaml.v3) |
| Terminal UI | lipgloss |

## Build & Quality Commands

```bash
# IMPORTANT: CGO_ENABLED=1 is required for ALL Go commands (pg_query_go wraps C)
make build          # Build binary to bin/migrate
make test           # Unit tests with -race
make test-integration  # Integration tests (requires Docker)
make lint           # golangci-lint
make fmt            # Format with gofumpt + goimports
make coverage       # Coverage report
make coverage-check # Fail if below 80%
make audit          # Full gate: fmt + vet + lint + test + coverage
```

## Architecture

```
cmd/migrate/main.go → internal/cli/ → internal/{parser,migration,analyzer,planner,executor,tracker,database}
```

- `cmd/migrate/` — Thin entry point. No business logic.
- `internal/cli/` — Cobra commands, output formatting.
- `internal/parser/` — Wraps pg_query_go Parse(). Returns typed AST.
- `internal/migration/` — Migration type, file loader, version sorter.
- `internal/analyzer/` — Danger detection engine. Rule interface + 9 rule implementations.
- `internal/analyzer/rules/` — One file per detection rule. Each independently testable.
- `internal/planner/` — Execution plan builder, impact estimation.
- `internal/executor/` — Transaction management, lock/statement timeouts, advisory locks.
- `internal/tracker/` — schema_migrations table CRUD.
- `internal/database/` — pgx pool setup, advisory lock helpers.

## Coding Standards

### Go Conventions
- All code in `internal/` — nothing exported to outside this module
- No `utils/`, `helpers/`, `common/` packages — name packages by domain
- Return `error`, never `panic` in library code
- Wrap errors with context: `fmt.Errorf("loading migration %s: %w", path, err)`
- Sentinel errors as package-level `var`: `var ErrNotFound = errors.New("not found")`
- Use `errors.Is()` and `errors.As()` for error checking
- Interfaces in the consumer package, not the provider

### Linting
- golangci-lint v2 with strict config (see `.golangci.yml`)
- Every `//nolint` must specify the linter AND a reason: `//nolint:funlen // table-driven test with many cases`
- Max cyclomatic complexity: 15 per function
- Max cognitive complexity: 20 per function
- Max function length: 80 lines / 40 statements

### Testing Conventions
- **Table-driven tests** for all pure logic (parser, analyzer rules, sorter, config)
- **`t.Parallel()`** on every `func Test*` AND every `t.Run` subtest
- **`require.*`** for preconditions and setup (stops test on failure)
- **`assert.*`** for assertions (continues to check other assertions)
- **`t.Helper()`** on ALL test helper functions (accurate line numbers in failures)
- **`t.Cleanup()`** instead of `defer` in test helpers (runs even if helper is in separate function)
- Test files next to source: `parser.go` → `parser_test.go`
- Black-box tests use `package foo_test` to test the public API
- White-box tests use `package foo` only when testing unexported logic
- Test naming: `TestFunctionName_scenario_expected` (e.g., `TestParse_invalidSQL_returnsError`)
- No test logic in test helpers — helpers set up state, tests make assertions

### Coverage Targets
- **90%** on `internal/analyzer/rules/` — these are the core value
- **85%** on `internal/executor/`
- **80%** total project
- **70%** per-file floor

### Commit Convention
- `feat:` — new feature
- `fix:` — bug fix
- `test:` — adding or updating tests
- `refactor:` — code change that neither fixes a bug nor adds a feature
- `chore:` — build process, tooling, dependencies
- `docs:` — documentation only

## Key Design Decisions

1. **Real PostgreSQL parser** via pg_query_go (100% accurate for valid PG SQL)
2. **Rule-based analysis** — each danger rule is a standalone `Rule` interface implementation
3. **Advisory locks** — `pg_try_advisory_lock` prevents concurrent migration runs
4. **CREATE INDEX CONCURRENTLY** — detected and executed outside transaction (PG requirement)
5. **PG version-aware** — rules adjust for target PG version (e.g., non-volatile DEFAULT safe on PG 11+)
6. **CGO required** — pg_query_go wraps C library, `CGO_ENABLED=1` in all commands

## References

- `plans/prd.md` — Full product requirements
- `plans/implementation_plan.md` — Detailed 9-phase build plan (master reference)
- `plans/checklist.md` — Phase-by-phase implementation checklist
- `plans/phases/` — Individual phase plans (one file per phase)
- `plans/project.md` — Project overview
- `.golangci.yml` — Linter configuration
- `.testcoverage.yml` — Coverage thresholds
