package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationLockID is the advisory lock identifier used to prevent
// concurrent migration runs.
const MigrationLockID int64 = 123456789

// LockHandle wraps a dedicated pooled connection that holds a
// session-level advisory lock. Call Release to unlock and return
// the connection to the pool.
type LockHandle struct {
	conn *pgxpool.Conn
}

// TryAcquireLock attempts to acquire a session-level advisory lock.
// Returns a LockHandle if successful, or ErrLockNotAcquired if the
// lock is already held by another process. The caller must call
// handle.Release() when done.
func TryAcquireLock(ctx context.Context, pool *pgxpool.Pool) (*LockHandle, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection for advisory lock: %w", err)
	}

	var acquired bool

	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", MigrationLockID).Scan(&acquired)
	if err != nil {
		conn.Release()

		return nil, fmt.Errorf("executing pg_try_advisory_lock: %w", err)
	}

	if !acquired {
		conn.Release()

		return nil, ErrLockNotAcquired
	}

	return &LockHandle{conn: conn}, nil
}

// Release unlocks the advisory lock and returns the connection to the pool.
// Safe to call multiple times; subsequent calls are no-ops.
func (h *LockHandle) Release(ctx context.Context) error {
	if h == nil || h.conn == nil {
		return nil
	}

	_, err := h.conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", MigrationLockID)
	h.conn.Release()
	h.conn = nil

	if err != nil {
		return fmt.Errorf("releasing advisory lock: %w", err)
	}

	return nil
}
