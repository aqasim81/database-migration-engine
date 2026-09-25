package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SetLockTimeout sets the lock_timeout for the given transaction only.
// This causes the migration to fail fast if it cannot acquire a lock
// within the specified duration, instead of blocking other queries.
// It uses SET LOCAL, which ends with the transaction; a plain SET would stay
// on the pooled connection after commit and leak into whatever uses it next.
func SetLockTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error {
	sql := fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", timeout.Milliseconds())

	_, err := tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("setting lock_timeout: %w", err)
	}

	return nil
}

// SetStatementTimeout sets the statement_timeout for the given transaction
// only (SET LOCAL; see SetLockTimeout).
// This prevents runaway queries from holding locks indefinitely.
func SetStatementTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error {
	sql := fmt.Sprintf("SET LOCAL statement_timeout = '%dms'", timeout.Milliseconds())

	_, err := tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("setting statement_timeout: %w", err)
	}

	return nil
}
