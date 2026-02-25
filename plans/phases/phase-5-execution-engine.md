# Phase 5: Execution Engine

> **Status:** Pending
> **Commit:** `feat: implement migration executor with lock_timeout and transaction safety`

## Goal

Apply pending migrations with transactions, timeouts, advisory locks, and special handling for `CREATE INDEX CONCURRENTLY`.

---

## Step 5.1: Safety Helpers

**File:** `internal/executor/safety.go`

```go
func SetLockTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error
func SetStatementTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error
func ResetTimeouts(ctx context.Context, tx pgx.Tx) error
```

- Execute `SET lock_timeout = '5000ms'` etc. within the transaction
- These protect against runaway locks: if the migration can't acquire a lock within the timeout, it fails fast instead of blocking other queries

## Step 5.2: Transaction Management

**File:** `internal/executor/transaction.go`

```go
func ExecInTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error
func ExecWithoutTransaction(ctx context.Context, pool *pgxpool.Pool, sql string) error
```

- `ExecInTransaction`: begin → run fn → commit (rollback on error)
- `ExecWithoutTransaction`: for `CREATE INDEX CONCURRENTLY` which cannot run inside a transaction

## Step 5.3: Executor

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

## Step 5.4: Wire Apply Command

**File:** `internal/cli/apply.go` (update from stub)

- Load config, connect to database, create tracker and executor
- Load and sort migrations
- If not `--force`, run analyzer first:
  - If HIGH or CRITICAL findings exist, show them and prompt for confirmation
  - User must type "yes" or pass `--force` to proceed
- Call `executor.Apply()`
- Show progress (via callback) and final summary

## Step 5.5: Integration Tests

**File:** `integration/lifecycle_test.go`

- Test: apply 3 safe migrations → all tracked as "applied"
- Test: apply already-applied migrations → skipped
- Test: apply with checksum mismatch → error
- Test: apply with lock_timeout → migration fails if lock can't be acquired (requires a second connection holding a lock)
- Test: `CREATE INDEX CONCURRENTLY` executes outside transaction

## Step 5.6: Verify & Commit

- `make test-unit` passes
- `make test-integration` passes (full apply lifecycle)
