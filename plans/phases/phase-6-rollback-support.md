# Phase 6: Rollback Support

> **Status:** Pending
> **Commit:** `feat: add rollback support with .down.sql file convention`

## Goal

Revert applied migrations using `.down.sql` files.

---

## Step 6.1: Update Loader for Down Files

The loader from Phase 2 already pairs `.up.sql` and `.down.sql` files. Verify that `Migration.DownSQL` is populated correctly.

If a migration has no `.down.sql` file and rollback is attempted, return a clear error: "No rollback file found for migration V007."

## Step 6.2: Implement Rollback in Executor

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

## Step 6.3: Wire Rollback Command

**File:** `internal/cli/rollback.go` (update from stub)

- `--steps N` (default 1): rollback last N migrations
- `--target VERSION`: rollback all migrations after this version
- Require database connection
- Show what will be rolled back, confirm unless `--force`

## Step 6.4: Integration Tests

Add to `integration/lifecycle_test.go`:
- Apply 3 migrations → rollback 1 → verify only 2 remain applied
- Rollback to target version → correct migrations reverted
- Rollback migration with no `.down.sql` → clear error

## Step 6.5: Verify & Commit

- `make test-unit` and `make test-integration` pass
