//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/database"
)

func TestNewPool_validConnection_succeeds(t *testing.T) {
	t.Parallel()

	dsn := SetupPostgresDSN(t)
	ctx := context.Background()

	pool, err := database.NewPool(ctx, dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	var result int

	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestNewPool_invalidURL_returnsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := database.NewPool(ctx, "not-valid")

	require.ErrorIs(t, err, database.ErrInvalidDatabaseURL)
}
