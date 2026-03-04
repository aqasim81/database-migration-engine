package planner

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgTableSizer queries PostgreSQL for table sizes.
type PgTableSizer struct {
	pool *pgxpool.Pool
}

// NewPgTableSizer creates a TableSizer backed by a database pool.
func NewPgTableSizer(pool *pgxpool.Pool) *PgTableSizer {
	return &PgTableSizer{pool: pool}
}

// TableSizeBytes returns the total on-disk size of a table (including indexes and TOAST).
// Returns -1 if the table does not exist.
func (s *PgTableSizer) TableSizeBytes(ctx context.Context, tableName string) (int64, error) {
	var size int64

	err := s.pool.QueryRow(ctx,
		`SELECT pg_total_relation_size(c.oid)
		 FROM pg_class c
		 JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE c.relname = $1 AND n.nspname = 'public'`,
		tableName,
	).Scan(&size)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sizeUnknown, nil
		}

		return sizeUnknown, fmt.Errorf("querying table size for %s: %w", tableName, err)
	}

	return size, nil
}
