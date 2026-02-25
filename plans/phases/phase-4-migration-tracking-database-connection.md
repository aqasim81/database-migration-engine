# Phase 4: Migration Tracking + Database Connection

> **Status:** Pending
> **Commit:** `feat: add migration tracking with schema_migrations table`

## Goal

Connect to PostgreSQL, manage the `schema_migrations` table, enforce exclusive access with advisory locks.

---

## Step 4.1: Database Connection

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

## Step 4.2: Advisory Locks

**File:** `internal/database/advisory_lock.go`

```go
const MigrationLockID = 123456789  // Arbitrary unique lock ID

func TryAcquireLock(ctx context.Context, pool *pgxpool.Pool) (bool, error)
func ReleaseLock(ctx context.Context, pool *pgxpool.Pool) error
```

- Use `pg_try_advisory_lock(MigrationLockID)` — non-blocking, returns false if already held
- Use `pg_advisory_unlock(MigrationLockID)` for release
- Critical: use session-level locks (not transaction-level) so the lock persists across multiple transactions during multi-migration execution

## Step 4.3: Schema Migrations Table

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

## Step 4.4: Migration Tracker

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

## Step 4.5: Integration Test Setup

**File:** `integration/testhelpers.go`

```go
func SetupPostgres(t *testing.T) (*pgxpool.Pool, func())
```

- Use `testcontainers-go` to start a PostgreSQL 16 container
- Return a connection pool and a cleanup function
- Build tag: `//go:build integration`

**File:** `integration/lifecycle_test.go` (stub for now, flesh out in Phase 5)

- Test: ensure tracker creates table, records migration, queries applied

## Step 4.6: Verify & Commit

- Run `make test-unit` for non-DB tests
- Run `make test-integration` — tracker tests pass against real PostgreSQL
