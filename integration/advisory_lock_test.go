//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/database"
)

func TestAdvisoryLock_acquireAndRelease(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	handle, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, handle)

	err = handle.Release(ctx)
	require.NoError(t, err)
}

func TestAdvisoryLock_doubleAcquire_returnsLockNotAcquired(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	handle1, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, handle1)

	t.Cleanup(func() {
		_ = handle1.Release(context.Background())
	})

	handle2, err := database.TryAcquireLock(ctx, pool)
	assert.Nil(t, handle2)
	require.ErrorIs(t, err, database.ErrLockNotAcquired)
}

func TestAdvisoryLock_releaseAllowsReacquire(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	handle1, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)
	require.NoError(t, handle1.Release(ctx))

	handle2, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, handle2)
	require.NoError(t, handle2.Release(ctx))
}

func TestLockHandle_Release_idempotent(t *testing.T) {
	t.Parallel()

	pool := SetupPostgres(t)
	ctx := context.Background()

	handle, err := database.TryAcquireLock(ctx, pool)
	require.NoError(t, err)

	err = handle.Release(ctx)
	require.NoError(t, err)

	err = handle.Release(ctx)
	require.NoError(t, err)
}

func TestLockHandle_Release_nilHandle_noError(t *testing.T) {
	t.Parallel()

	var handle *database.LockHandle

	err := handle.Release(context.Background())
	require.NoError(t, err)
}
