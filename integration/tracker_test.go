//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

func TestTracker_fullLifecycle(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	// EnsureTable creates the table.
	err := tr.EnsureTable(ctx)
	require.NoError(t, err)

	// EnsureTable is idempotent.
	err = tr.EnsureTable(ctx)
	require.NoError(t, err)

	// No migrations applied initially.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	assert.Empty(t, applied)

	// IsApplied returns false for unknown version.
	ok, err := tr.IsApplied(ctx, "001")
	require.NoError(t, err)
	assert.False(t, ok)

	// Record a migration.
	err = tr.RecordApplied(ctx, tracker.RecordParams{
		Version:    "001",
		Filename:   "V001_create_users.up.sql",
		Checksum:   "abc123",
		DurationMs: 42,
	})
	require.NoError(t, err)

	// IsApplied returns true after recording.
	ok, err = tr.IsApplied(ctx, "001")
	require.NoError(t, err)
	assert.True(t, ok)

	// GetApplied returns the recorded migration.
	applied, err = tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, "001", applied[0].Version)
	assert.Equal(t, "V001_create_users.up.sql", applied[0].Filename)
	assert.Equal(t, "abc123", applied[0].Checksum)
	assert.Equal(t, 42, applied[0].DurationMs)
	assert.Equal(t, "applied", applied[0].Status)
	assert.False(t, applied[0].AppliedAt.IsZero())

	// GetChecksum returns correct value.
	cs, err := tr.GetChecksum(ctx, "001")
	require.NoError(t, err)
	assert.Equal(t, "abc123", cs)

	// GetChecksum for unknown version returns ErrMigrationNotFound.
	_, err = tr.GetChecksum(ctx, "999")
	require.ErrorIs(t, err, tracker.ErrMigrationNotFound)

	// RecordRolledBack updates status.
	err = tr.RecordRolledBack(ctx, "001")
	require.NoError(t, err)

	// After rollback, IsApplied returns false.
	ok, err = tr.IsApplied(ctx, "001")
	require.NoError(t, err)
	assert.False(t, ok)

	// GetApplied excludes rolled-back migrations.
	applied, err = tr.GetApplied(ctx)
	require.NoError(t, err)
	assert.Empty(t, applied)

	// RecordRolledBack for unknown version returns ErrMigrationNotFound.
	err = tr.RecordRolledBack(ctx, "999")
	require.ErrorIs(t, err, tracker.ErrMigrationNotFound)
}

func TestTracker_RecordApplied_upsertAfterRollback(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()
	tr := tracker.New(pool)

	require.NoError(t, tr.EnsureTable(ctx))

	// Record, rollback, then re-apply.
	err := tr.RecordApplied(ctx, tracker.RecordParams{
		Version:    "001",
		Filename:   "V001_create_users.up.sql",
		Checksum:   "abc123",
		DurationMs: 42,
	})
	require.NoError(t, err)

	require.NoError(t, tr.RecordRolledBack(ctx, "001"))

	// Re-apply should succeed (upsert).
	err = tr.RecordApplied(ctx, tracker.RecordParams{
		Version:    "001",
		Filename:   "V001_create_users.up.sql",
		Checksum:   "abc123",
		DurationMs: 35,
	})
	require.NoError(t, err)

	ok, err := tr.IsApplied(ctx, "001")
	require.NoError(t, err)
	assert.True(t, ok)

	// Duration should be updated.
	applied, err := tr.GetApplied(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, 35, applied[0].DurationMs)
}
