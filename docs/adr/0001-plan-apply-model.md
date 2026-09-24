# 0001. Separate a read-only `plan` from a self-checking `apply`

- **Status:** Accepted
- **Date:** 2026-09-24 (written after the fact; the model shipped in phases 5 and 7)

> This ADR was reconstructed from the code, not from notes made when the
> decision was taken. Statements marked **[Inferred]** are my reading of intent
> and need confirming. Everything else can be checked in the cited files.

## Context

The tool has to answer two different questions:

1. *What would happen if these migrations ran?* This should be answerable in
   code review and CI, often with no database available.
2. *Run them safely.* This needs a live connection, mutual exclusion, and
   timeouts, and it must not run a dangerous statement by accident.

Both questions depend on the same analysis: parse each `.up.sql` with
`pg_query_go`, then run the rule registry (`internal/analyzer`).

## Decision

### `plan` is a read-only preview (`internal/cli/plan.go`, `internal/planner`)

- `plan` loads and sorts migration files, runs the analyzer, and builds a
  `planner.Plan`. Each step has findings, impact estimates, status
  (`pending`/`applied`), and `RunInTx`.
- The database is **optional**. With no `database_url`, every migration is
  `pending` and table sizes are unknown (`sizeUnknown = -1`)
  (`loadAppliedState`).
- With a database, `plan` reads `schema_migrations` for applied state and
  queries `pg_total_relation_size` (public schema only, `table_sizer.go`) to
  choose a lock-duration bucket (`< 1s`, `1-10s`, `10s-5min`, `> 5min`)
  from per-rule heuristics in `impact.go`.
- `RunInTx` is false when the SQL contains `CREATE/DROP INDEX CONCURRENTLY`.
- `plan` writes nothing: no plan file, and no DB writes other than
  `EnsureTable` creating `schema_migrations` if it is missing.

### `apply` does not consume a plan. It re-derives everything (`internal/cli/apply.go`, `internal/executor`)

- `apply` reloads the files, connects, and **re-runs the analyzer itself** on
  the migrations that are still pending (already-applied ones were confirmed
  when they ran; #5). If any finding is HIGH or CRITICAL, it prints the findings and requires the user to
  type `yes` on stdin. `--force` skips the check, and so does `--dry-run`.
  EOF or any other answer aborts with `errDangerousMigrations`.
- The executor takes a session-level advisory lock with
  `pg_try_advisory_lock(123456789)` on a dedicated pooled connection
  (`database/advisory_lock.go`). If another run holds the lock, it fails
  immediately with `ErrLockNotAcquired`.
- Migrations run in version order. For each one:
  - If it is already applied, the stored checksum is compared with the file.
    A mismatch aborts with `tracker.ErrChecksumMismatch`.
  - Otherwise the SQL runs in its own transaction, after
    `SET lock_timeout` / `SET statement_timeout` when those are configured.
    If it contains CONCURRENTLY, it runs outside a transaction and no
    timeouts are set.
  - The `schema_migrations` row is written inside the migration's own
    transaction, so the schema change and its record commit or roll back
    together (#4). CONCURRENTLY migrations can't use a transaction, so they
    are recorded on the pool after the SQL succeeds. Rollback follows the
    same rules.
- Concurrent-statement detection is written twice, once in
  `planner.hasConcurrentOp` and once in `executor.containsConcurrentOp`. The
  code comment says this is deliberate, to avoid cross-package coupling.

## Consequences

**Easier**

- `analyze` and `plan` work in CI with no database. `analyze --fail-on-high`
  and `--format github-actions` are built for that.
- A plan can never go stale against `apply`, because `apply` doesn't read one.
- Two engineers or two deploys can't interleave migrations: the advisory lock
  makes the second one fail fast.

**Harder / known gaps (read from code)**

- Nothing guarantees that what someone reviewed in `plan` is what `apply`
  runs. If files change in between, `apply` runs the new content.
  Applied-migration checksums catch edits only *after* a migration has been
  applied.
- `apply` needs a reachable database before it can show any findings, since
  the gate reads applied state first (#5). `analyze` remains the
  database-free check.
- CONCURRENTLY migrations are recorded after their SQL, outside any
  transaction. A crash between the two leaves the index built but unrecorded,
  and the next `apply` would run it again. `CREATE INDEX CONCURRENTLY IF NOT
  EXISTS` makes that re-run harmless. Until #4 this gap applied to every
  migration.
- CONCURRENTLY migrations get no `lock_timeout`/`statement_timeout`.
- Lock-duration estimates are coarse buckets based on table size. They look
  only at the `public` schema, and tables that don't exist yet count as
  "unknown".
- Concurrent detection is duplicated, so the two copies must be changed
  together.

## Alternatives considered

- **Saved plan artifact, applied verbatim** (as in Terraform's
  `plan -out` / `apply planfile`). **[Inferred]** This was rejected or never
  considered because migration files are already the reviewed, versioned
  artifact in git, and a second artifact adds a staleness problem without
  adding safety. I found no plan-file code, flag, or note in the repo.
- **Gate `apply` on MEDIUM and above.** **[Inferred]** HIGH is the threshold
  because it matches `analyze --fail-on-high` and `Plan.HighRiskCount`. A lower
  threshold would prompt on operations that are routinely safe on small
  tables.
- **Blocking `pg_advisory_lock` instead of `pg_try_advisory_lock`.**
  **[Inferred]** Failing fast was chosen so that a second deploy exits with a
  clear error instead of silently queuing behind the first.
- **Exact lock-time prediction** (e.g. `EXPLAIN` or sampling).
  **[Inferred]** Rejected as unreliable for DDL. Size buckets are enough to
  rank risk, which is what the plan table is for.
- **Record the migration in a separate statement after its SQL commits.**
  This was the original behaviour and was replaced in #4. It meant one code
  path for transactional and CONCURRENTLY migrations, but any crash or tracker
  failure between the two statements caused a re-run.

## Open questions for the author

1. ~~Should the `apply` gate skip migrations that are already applied?~~ Yes;
   done in #5.
2. ~~Should `RecordApplied` share the migration's transaction?~~ Yes; done in #4.
3. Is `plan --out` / `apply --plan` wanted, or is "git is the plan" the
   intended model?
