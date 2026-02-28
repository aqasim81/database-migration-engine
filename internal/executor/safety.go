package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SetLockTimeout sets the lock_timeout for the given transaction.
// This causes the migration to fail fast if it cannot acquire a lock
// within the specified duration, instead of blocking other queries.
func SetLockTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error {
	sql := fmt.Sprintf("SET lock_timeout = '%dms'", timeout.Milliseconds())

	_, err := tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("setting lock_timeout: %w", err)
	}

	return nil
}

// SetStatementTimeout sets the statement_timeout for the given transaction.
// This prevents runaway queries from holding locks indefinitely.
func SetStatementTimeout(ctx context.Context, tx pgx.Tx, timeout time.Duration) error {
	sql := fmt.Sprintf("SET statement_timeout = '%dms'", timeout.Milliseconds())

	_, err := tx.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("setting statement_timeout: %w", err)
	}

	return nil
}

// ResetTimeouts resets both lock_timeout and statement_timeout to zero (unlimited).
func ResetTimeouts(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SET lock_timeout = '0'; SET statement_timeout = '0'")
	if err != nil {
		return fmt.Errorf("resetting timeouts: %w", err)
	}

	return nil
}
