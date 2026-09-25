# Review instructions (apply to every code review, human or automated)

## Passes
Tag every finding with its pass:
- **Bugs:** logic errors, broken edge cases, regressions, error paths, resource cleanup.
- **Security & privacy:** injection, auth and authorization gaps, secrets in code, PII in logs or fixtures.
- **Invariants:** anything that can break a rule listed under "## Invariants" in CLAUDE.md.
- **Compliance with intent:** the change matches its linked intent/spec/plan and phase; nothing unplanned slipped in.

## What Important means here
Important = breaks behaviour, leaks data, breaks an invariant, or contradicts the plan. Style and naming are nits.

## Cap the nits
At most five nits per review; summarise the rest as a count.

## Do not report
Anything the formatter, linter or type checker already enforces; generated files; lockfiles.

## Project-specific focus (moved from CLAUDE.md, 25 Sept 2026)
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
