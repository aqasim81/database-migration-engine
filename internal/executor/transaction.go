package executor

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExecInTransaction runs fn inside a database transaction.
// On success the transaction is committed; on error it is rolled back.
func ExecInTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	defer tx.Rollback(ctx) //nolint:errcheck // rollback on committed tx returns ErrTxClosed

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// ExecWithoutTransaction executes SQL directly on the pool, outside any
// transaction. Required for statements like CREATE INDEX CONCURRENTLY
// which cannot run inside a transaction block.
func ExecWithoutTransaction(ctx context.Context, pool *pgxpool.Pool, sql string) error {
	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("executing outside transaction: %w", err)
	}

	return nil
}
