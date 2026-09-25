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
make verify         # Same gate under the starter-kit name
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

## Invariants

Rules that must never break (checked by the `invariant-auditor` subagent and the Invariants pass in REVIEW.md):
1. Rule severity and lock type match real PostgreSQL behaviour for the target version (`TargetPGVersion`).
2. CONCURRENTLY statements run outside a transaction; everything else runs in one transaction per migration
   with lock/statement timeouts. Concurrent detection is changed in both `planner.hasConcurrentOp` and
   `executor.containsConcurrentOp`.
3. Advisory locks, pooled connections and transactions are released/rolled back on every path, including errors.
4. A migration is recorded in `schema_migrations` in the same transaction as its SQL; checksum computation
   never changes silently (existing databases must not re-run or reject migrations).
5. Database URLs are printed only through `config.RedactURL()`; no raw credentials in output or logs.
6. The CLI contract (exit codes, `--format json` / `github-actions` shape, `--fail-on-high`) is stable.

Review standards (what matters, what lint already covers, error-handling and test conventions) live in `REVIEW.md`.

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

## Workflow
Workflow rules: `.claude/rules/ai-native-workflow.md` (local). Review policy: `REVIEW.md`.

## Known mistakes to avoid
(When the same mistake happens twice, add the correction here.)
