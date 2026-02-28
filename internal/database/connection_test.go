package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/database"
)

func TestNewPool_invalidURL_returnsInvalidURLError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := database.NewPool(ctx, "not-a-valid-url")

	require.ErrorIs(t, err, database.ErrInvalidDatabaseURL)
}

func TestNewPool_emptyURL_returnsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := database.NewPool(ctx, "")

	require.Error(t, err)
}
