package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultMaxConns = 5

// NewPool creates a pgx connection pool for the given database URL.
// It parses the connection string, sets a conservative max connection limit,
// and pings the database to verify connectivity.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDatabaseURL, err)
	}

	poolCfg.MaxConns = defaultMaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	return pool, nil
}
