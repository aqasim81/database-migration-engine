# Database Migration Engine — CLAUDE.md

## Project Overview

Zero-downtime PostgreSQL schema migration CLI. Parses SQL with the real PG parser, detects dangerous DDL operations, suggests safe alternatives, executes with rollback capability.

**Status:** All phases complete (0-9). v0.1.0 — fully functional CLI with 9 detection rules, zero-downtime execution, rollback support, JSON/GitHub Actions output, GoReleaser release config.

## Stack

Go 1.25+ | pg_query_go v6 (C-backed PG parser) | Cobra CLI | pgx v5 | testify + testcontainers-go | YAML config (gopkg.in/yaml.v3) | lipgloss TUI

## Key Commands

```bash
# CGO_ENABLED=1 is required for ALL Go commands (pg_query_go wraps C)
make audit          # Full gate: fmt + vet + lint + test + coverage (run before committing)
make test           # Unit tests with -race
make test-integration  # Integration tests (requires Docker)
make lint           # golangci-lint
make coverage-check # Fail if below 80%
```

## Architecture

```
cmd/migrate/main.go → internal/cli/ → internal/{parser,migration,analyzer,planner,executor,tracker,database}
```

- `cmd/migrate/` — Thin entry point, no business logic
- `internal/cli/` — Cobra commands, output formatting, only place that prints errors/sets exit codes
- `internal/config/` — YAML config, env var overrides, flag merging, URL redaction
- `internal/parser/` — Wraps pg_query_go Parse(), returns typed AST
- `internal/migration/` — Migration type, file loader, version sorter, checksums
- `internal/analyzer/` — Danger detection engine. Rule interface + implementations in `rules/` (one file per rule)
- `internal/planner/` — Execution plan builder, impact estimation (lock duration heuristics, table size queries)
- `internal/executor/` — Migration executor: Apply flow, transaction mgmt, lock/statement timeouts, concurrent index detection, progress callbacks
- `internal/tracker/` — schema_migrations CRUD (EnsureTable, IsApplied, GetApplied, RecordApplied, RecordRolledBack, GetChecksum)
- `internal/database/` — pgx pool wrapper, advisory lock helpers (LockHandle)

## Key Design Decisions

1. **Real PostgreSQL parser** via pg_query_go — 100% accurate for valid PG SQL
2. **Rule-based analysis** — each danger rule is a standalone `Rule` interface implementation
3. **Advisory locks** — `pg_try_advisory_lock` prevents concurrent migration runs
4. **CREATE INDEX CONCURRENTLY** — detected and executed outside transaction (PG requirement)
5. **PG version-aware** — rules adjust for target PG version (e.g., non-volatile DEFAULT safe on PG 11+)
6. **CGO required** — pg_query_go wraps C library, `CGO_ENABLED=1` in all commands

## Coding Conventions

- All code in `internal/` — nothing exported outside this module
- No `utils/`, `helpers/`, `common/` packages — name packages by domain
- Return `error`, never `panic`; wrap with context: `fmt.Errorf("loading migration %s: %w", path, err)`
- Sentinel errors as package-level `var`; check with `errors.Is()` / `errors.As()`
- Interfaces in the consumer package, not the provider
- No `interface{}` / `any` unless unavoidable — use typed AST nodes from pg_query_go
- No global mutable state — pass dependencies explicitly
- Database errors must include operation context (e.g., which migration was being applied)
- Every `//nolint` must specify linter AND reason: `//nolint:funlen // table-driven test`
- Complexity limits: cyclomatic 15, cognitive 20, function length 80 lines / 40 statements

## Testing

- Table-driven tests for all pure logic (parser, analyzer rules, sorter, config)
- `t.Parallel()` on every `func Test*` AND every `t.Run` subtest
- `require.*` for preconditions (stops on failure), `assert.*` for assertions (continues)
- `t.Helper()` on all test helpers; `t.Cleanup()` instead of `defer` in helpers
- Black-box tests (`package foo_test`) for public API; white-box (`package foo`) only for unexported logic
- Naming: `TestFunctionName_scenario_expected`
- Helpers set up state only — tests make assertions

### Coverage

- **Enforced:** 80% total, computed by `make coverage-check` (same gate in pre-push and CI) after
  excluding four integration-only files: `database/advisory_lock.go`, `tracker/tracker.go`,
  `executor/transaction.go`, `executor/safety.go`. `COVERAGE_MIN` and `COVERAGE_EXCLUDE` in the
  Makefile are the only source of truth; there are no per-package or per-file gates.
- **Raw unit coverage (2026-09-25):** 81.4% total after exclusions; rules 86.9%, executor 76.0%,
  cli 67.9% (part of `runApply` is covered by `internal/cli/apply_integration_test.go`).
  Aim for 90%+ on new rules, but don't block a PR on a package number nothing enforces.
- **Low by design:** `internal/tracker` (2.6%) and `internal/database` (29%) are thin pgx wrappers
  exercised by `integration/` (testcontainers, `-tags=integration`, `make test-integration`).
  Don't ask for unit tests with a mocked pool there; ask for an integration test.

## Security

- All credentials via `MIGRATE_*` env vars or CLI flags — no secrets in code
- `migrate.yml` and `.env` are gitignored; only `config.example.yml` committed
- Use `config.RedactURL()` for database URLs in CLI output — never log raw URLs
- `gitleaks` on every commit via lefthook; `gosec` enabled in linter

## Code Review

### What matters here
- **Rule correctness against real PostgreSQL semantics.** Each finding's severity and lock type must
  match what PG actually does, including version gating (`TargetPGVersion`, e.g. non-volatile
  DEFAULT safe on 11+). A new or changed rule needs table-driven cases for the safe variant,
  the dangerous variant, and nil/odd AST shapes.
- **Transaction boundaries.** CONCURRENTLY statements must run outside a transaction; everything
  else runs in one transaction per migration with lock/statement timeouts. Concurrent detection
  lives in two places (`planner.hasConcurrentOp`, `executor.containsConcurrentOp`); change both.
- **Lock and connection release on every path.** Advisory lock handles, pooled conns, and tx
  rollback in error branches.
- **Tracker consistency.** Anything that changes when `RecordApplied`/`RecordRolledBack` runs, or
  how checksums are computed, can make existing databases re-run or reject migrations.
- **Output safety.** Database URLs go through `config.RedactURL()`; nothing prints raw credentials.
- **Error context.** Errors name the migration version and the operation.
- **CLI contract.** Exit codes, `--format json` / `github-actions` shape, and `--fail-on-high`
  are consumed by CI pipelines; changes are breaking.

### Don't comment on (golangci-lint already enforces it)
Formatting and import grouping (gofumpt, goimports), complexity and length (cyclop 15,
gocognit 20, funlen 80/40, nestif), error wrapping and `errors.Is` use (wrapcheck, errorlint,
nilerr), comment punctuation (godot), `//nolint` without linter + reason (nolintlint),
security patterns (gosec), testify misuse and missing `t.Parallel` (testifylint, tparallel),
unused params/results (unparam, revive). If lint passes, these are settled.

### Error handling (as practiced)
- Return errors, never panic. Wrap with `fmt.Errorf("doing X for %s: %w", version, err)`.
- Sentinels are package-level `var`s: in `errors.go` (`executor`, `database`) or beside their
  single use (`cli/apply.go`). Callers match with `errors.Is`.
- Only `cli.Execute` prints errors and sets the exit code; `RunE` functions return errors.
- `wrapcheck` ignores this module's own packages, so internal errors pass through unwrapped
  when the callee already added context.

### Tests (as practiced)
- Table-driven, `t.Parallel()` at both levels, `require` for preconditions, `assert` for checks.
- Black-box `_test` packages by default. Executor internals are tested white-box by injecting
  `acquireLock` / `execSQL` fakes (`executor_internal_test.go`), not a mocked pgx pool.
- Bug fixes need a regression test that fails without the fix.
- Anything touching real SQL execution, locks, or `schema_migrations` belongs in `integration/`.

## Git & Workflow

- Branch naming: `feat/`, `fix/`, `chore/` prefix
- Conventional commits: `feat:`, `fix:`, `test:`, `refactor:`, `chore:`, `docs:`
- **No AI attribution** — no `Co-Authored-By`, "Generated by", or AI references in any artifact
- Pre-commit hooks: gofumpt, goimports, golangci-lint, go vet, gitleaks
- Pre-push hooks: tests with race detector, coverage ≥80%

## Session Workflow

Internal planning and operational docs live in `plans/`, which is gitignored and local only; they are not in the public repo.

1. `/clear` → read `CLAUDE.md` → check `plans/status.md` for current state and `plans/runbook.md` for how-tos
2. For new work, read or create the relevant plan in `plans/`
3. Implement in small chunks, test after each
4. `make audit` before committing
5. Update `plans/status.md` (and `plans/checklist.md` for phase work) when a step completes

## References

Public: `.golangci.yml` | `.github/workflows/ci.yml` | `.github/workflows/release.yml` | `docs/adr/`

Local only (`plans/`, gitignored): `runbook.md` (release, env quirks, harness reuse, to-dos) | `demo.md` (demo recording script) | `status.md` (project status) | `prd.md` (requirements) | `implementation_plan.md` (9-phase plan) | `checklist.md` | `phases/`
