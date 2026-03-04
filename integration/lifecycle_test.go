//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/database"
	"github.com/aqasim81/database-migration-engine/internal/executor"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

func makeMigrations() []migration.Migration {
	m1SQL := "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);"
	m2SQL := "CREATE TABLE posts (id SERIAL PRIMARY KEY, user_id INTEGER REFERENCES users(id), title TEXT);"
	m3SQL := "ALTER TABLE users ADD COLUMN email TEXT;"

	return []migration.Migration{
		{
			Version:  "001",
			Name:     "create_users",
			UpSQL:    m1SQL,
			Checksum: migration.ComputeChecksum(m1SQL),
			FilePath: "migrations/V001_create_users.up.sql",
		},
		{
			Version:  "002",
			Name:     "create_posts",
			UpSQL:    m2SQL,
			Checksum: migration.ComputeChecksum(m2SQL),
			FilePath: "migrations/V002_create_posts.up.sql",
		},
		{
			Version:  "003",
			Name:     "add_email",
			UpSQL:    m3SQL,
			Checksum: migration.ComputeChecksum(m3SQL),
			FilePath: "migrations/V003_add_email.up.sql",
		},
	}
}

func TestApply_safeMigrations_allTracked(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrations()

	var events []executor.ProgressEvent
	exec := executor.New(pool, tr,
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// All 3 should be recorded as applied.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 3)

	assert.Equal(t, "001", applied[0].Version)
	assert.Equal(t, "002", applied[1].Version)
	assert.Equal(t, "003", applied[2].Version)

	for _, a := range applied {
		assert.Equal(t, "applied", a.Status)
		assert.Greater(t, a.DurationMs, -1)
	}

	// Check progress events: 3 starting + 3 completed = 6.
	require.Len(t, events, 6)

	for i := 0; i < 3; i++ {
		assert.Equal(t, executor.StatusStarting, events[i*2].Status)
		assert.Equal(t, executor.StatusCompleted, events[i*2+1].Status)
	}
}

func TestApply_alreadyApplied_skipped(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrations()

	exec := executor.New(pool, tr)

	// First apply.
	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Second apply — all should be skipped.
	var events []executor.ProgressEvent
	exec2 := executor.New(pool, tr,
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err = exec2.Apply(ctx, migrations)
	require.NoError(t, err)

	require.Len(t, events, 3)

	for _, e := range events {
		assert.Equal(t, executor.StatusSkipped, e.Status)
	}
}

func TestApply_checksumMismatch_returnsError(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	// Apply first migration.
	migrations := makeMigrations()[:1]
	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Tamper with the checksum by changing UpSQL.
	tampered := []migration.Migration{
		{
			Version:  "001",
			Name:     "create_users",
			UpSQL:    "CREATE TABLE users (id SERIAL PRIMARY KEY);",
			Checksum: migration.ComputeChecksum("CREATE TABLE users (id SERIAL PRIMARY KEY);"),
			FilePath: "migrations/V001_create_users.up.sql",
		},
	}

	err = exec.Apply(ctx, tampered)
	require.Error(t, err)
	assert.ErrorIs(t, err, tracker.ErrChecksumMismatch)
}

func TestApply_concurrentIndex_executesOutsideTransaction(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	// Create the table first.
	tableSQL := "CREATE TABLE items (id SERIAL PRIMARY KEY, name TEXT);"
	indexSQL := "CREATE INDEX CONCURRENTLY idx_items_name ON items (name);"

	migrations := []migration.Migration{
		{
			Version:  "001",
			Name:     "create_items",
			UpSQL:    tableSQL,
			Checksum: migration.ComputeChecksum(tableSQL),
			FilePath: "migrations/V001_create_items.up.sql",
		},
		{
			Version:  "002",
			Name:     "add_items_index",
			UpSQL:    indexSQL,
			Checksum: migration.ComputeChecksum(indexSQL),
			FilePath: "migrations/V002_add_items_index.up.sql",
		},
	}

	var events []executor.ProgressEvent
	exec := executor.New(pool, tr,
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Both should be applied.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 2)

	// Verify the index actually exists.
	var indexExists bool
	err = pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname = 'idx_items_name')",
	).Scan(&indexExists)
	require.NoError(t, err)
	assert.True(t, indexExists)
}

func TestApply_dryRun_noChanges(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrations()

	var events []executor.ProgressEvent
	exec := executor.New(pool, tr,
		executor.WithDryRun(true),
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// All should be skipped in dry-run.
	require.Len(t, events, 3)

	for _, e := range events {
		assert.Equal(t, executor.StatusSkipped, e.Status)
	}

	// No migrations should be recorded.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	assert.Empty(t, applied)
}

func TestApply_advisoryLock_preventsConcurrentRuns(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	// Acquire the advisory lock before calling Apply.
	lock, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)
	defer lock.Release(ctx) //nolint:errcheck // test cleanup

	tr := tracker.New(pool)
	exec := executor.New(pool, tr)

	err = exec.Apply(ctx, makeMigrations())
	require.Error(t, err)
	assert.ErrorIs(t, err, database.ErrLockNotAcquired)
}

func TestApply_withTimeouts_succeeds(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrations()[:1]

	exec := executor.New(pool, tr,
		executor.WithLockTimeout(10000000000),      // 10s
		executor.WithStatementTimeout(30000000000), // 30s
	)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
}

func TestApply_failedMigration_reportsError(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	badSQL := "CREATE TABLE missing_ref (id SERIAL, fk INTEGER REFERENCES nonexistent(id));"
	migrations := []migration.Migration{
		{
			Version:  "001",
			Name:     "bad_migration",
			UpSQL:    badSQL,
			Checksum: migration.ComputeChecksum(badSQL),
			FilePath: "migrations/V001_bad_migration.up.sql",
		},
	}

	var events []executor.ProgressEvent
	exec := executor.New(pool, tr,
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err := exec.Apply(ctx, migrations)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "executing migration 001")

	// Should have starting + failed events.
	require.Len(t, events, 2)
	assert.Equal(t, executor.StatusStarting, events[0].Status)
	assert.Equal(t, executor.StatusFailed, events[1].Status)
	assert.Error(t, events[1].Error)
}

func TestApply_partialFailure_earlierMigrationsTracked(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	goodSQL := "CREATE TABLE widgets (id SERIAL PRIMARY KEY);"
	badSQL := "CREATE TABLE bad (id SERIAL, fk INTEGER REFERENCES nonexistent(id));"

	migrations := []migration.Migration{
		{
			Version:  "001",
			Name:     "good",
			UpSQL:    goodSQL,
			Checksum: migration.ComputeChecksum(goodSQL),
			FilePath: "migrations/V001_good.up.sql",
		},
		{
			Version:  "002",
			Name:     "bad",
			UpSQL:    badSQL,
			Checksum: migration.ComputeChecksum(badSQL),
			FilePath: "migrations/V002_bad.up.sql",
		},
	}

	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, migrations)
	require.Error(t, err)

	// First migration should be recorded.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, "001", applied[0].Version)
}

func TestApply_emptyList_succeeds(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, []migration.Migration{})
	require.NoError(t, err)
}

func TestApply_lockReleasedAfterCompletion(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	exec := executor.New(pool, tr)

	// First apply should succeed and release the lock.
	err := exec.Apply(ctx, makeMigrations())
	require.NoError(t, err)

	// Second apply should also succeed (lock was released).
	err = exec.Apply(ctx, makeMigrations())
	require.NoError(t, err)
}

func TestApply_concurrentApply_oneSucceeds(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	var wg sync.WaitGroup

	errs := make([]error, 2)

	for i := range 2 {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			tr := tracker.New(pool)
			exec := executor.New(pool, tr)
			errs[idx] = exec.Apply(ctx, makeMigrations())
		}(i)
	}

	wg.Wait()

	// At least one should succeed; the other may get ErrLockNotAcquired.
	successes := 0

	for _, err := range errs {
		if err == nil {
			successes++
		}
	}

	assert.GreaterOrEqual(t, successes, 1)
}

// --- Rollback integration tests ---

func makeMigrationsWithDown() []migration.Migration {
	m1Up := "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);"
	m1Down := "DROP TABLE IF EXISTS users;"
	m2Up := "CREATE TABLE posts (id SERIAL PRIMARY KEY, user_id INTEGER REFERENCES users(id), title TEXT);"
	m2Down := "DROP TABLE IF EXISTS posts;"
	m3Up := "ALTER TABLE users ADD COLUMN email TEXT;"
	m3Down := "ALTER TABLE users DROP COLUMN IF EXISTS email;"

	return []migration.Migration{
		{
			Version:  "001",
			Name:     "create_users",
			UpSQL:    m1Up,
			DownSQL:  m1Down,
			Checksum: migration.ComputeChecksum(m1Up),
			FilePath: "migrations/V001_create_users.up.sql",
		},
		{
			Version:  "002",
			Name:     "create_posts",
			UpSQL:    m2Up,
			DownSQL:  m2Down,
			Checksum: migration.ComputeChecksum(m2Up),
			FilePath: "migrations/V002_create_posts.up.sql",
		},
		{
			Version:  "003",
			Name:     "add_email",
			UpSQL:    m3Up,
			DownSQL:  m3Down,
			Checksum: migration.ComputeChecksum(m3Up),
			FilePath: "migrations/V003_add_email.up.sql",
		},
	}
}

func TestRollback_lastOne_leavesOthersApplied(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrationsWithDown()

	exec := executor.New(pool, tr)

	// Apply 3.
	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Rollback 1.
	var events []executor.ProgressEvent
	exec2 := executor.New(pool, tr,
		executor.WithProgressCallback(func(e executor.ProgressEvent) {
			events = append(events, e)
		}),
	)

	err = exec2.Rollback(ctx, migrations, 1)
	require.NoError(t, err)

	// Verify 2 remain.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 2)
	assert.Equal(t, "001", applied[0].Version)
	assert.Equal(t, "002", applied[1].Version)

	// Check progress events: 1 rolling_back + 1 completed = 2.
	require.Len(t, events, 2)
	assert.Equal(t, executor.StatusRollingBack, events[0].Status)
	assert.Equal(t, executor.StatusCompleted, events[1].Status)
}

func TestRollbackToVersion_rollsBackAfterTarget(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrationsWithDown()

	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Rollback to V001 — V002 and V003 should be rolled back.
	err = exec.RollbackToVersion(ctx, migrations, "001")
	require.NoError(t, err)

	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, "001", applied[0].Version)
}

func TestRollback_noDownFile_returnsError(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	upSQL := "CREATE TABLE nodown (id SERIAL PRIMARY KEY);"
	migrations := []migration.Migration{
		{
			Version:  "001",
			Name:     "no_down",
			UpSQL:    upSQL,
			DownSQL:  "",
			Checksum: migration.ComputeChecksum(upSQL),
			FilePath: "migrations/V001_no_down.up.sql",
		},
	}

	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	err = exec.Rollback(ctx, migrations, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, executor.ErrNoDownSQL)
}

func TestRollback_alreadyRolledBack_returnsErrNothingToRollback(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrationsWithDown()

	exec := executor.New(pool, tr)

	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	// Rollback all 3.
	err = exec.Rollback(ctx, migrations, 3)
	require.NoError(t, err)

	// Rollback again — nothing to roll back.
	err = exec.Rollback(ctx, migrations, 1)
	require.ErrorIs(t, err, executor.ErrNothingToRollback)
}

func TestRollback_reapplyAfterRollback(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)
	migrations := makeMigrationsWithDown()

	exec := executor.New(pool, tr)

	// Apply, rollback, re-apply.
	err := exec.Apply(ctx, migrations)
	require.NoError(t, err)

	err = exec.Rollback(ctx, migrations, 1)
	require.NoError(t, err)

	err = exec.Apply(ctx, migrations)
	require.NoError(t, err)

	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 3)
}
