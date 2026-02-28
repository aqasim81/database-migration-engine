package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aqasim81/database-migration-engine/internal/database"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

// Progress status constants reported via ProgressEvent.
const (
	StatusStarting  = "starting"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"
)

// ProgressEvent is emitted by the executor for each migration processed.
type ProgressEvent struct {
	Migration *migration.Migration
	Status    string
	Duration  time.Duration
	Error     error
}

// MigrationTracker abstracts schema_migrations operations for testability.
type MigrationTracker interface {
	EnsureTable(ctx context.Context) error
	IsApplied(ctx context.Context, version string) (bool, error)
	GetChecksum(ctx context.Context, version string) (string, error)
	RecordApplied(ctx context.Context, p tracker.RecordParams) error
}

// lockReleaser is returned by lockFn and must be released when done.
type lockReleaser interface {
	Release(ctx context.Context) error
}

// lockFunc acquires an advisory lock and returns a releaser.
type lockFunc func(ctx context.Context) (lockReleaser, error)

// sqlExecFunc executes a single migration's SQL.
type sqlExecFunc func(ctx context.Context, m *migration.Migration) error

// Executor applies pending migrations with transaction safety, timeouts,
// and advisory locks to prevent concurrent runs.
type Executor struct {
	pool             *pgxpool.Pool
	tracker          MigrationTracker
	lockTimeout      time.Duration
	statementTimeout time.Duration
	dryRun           bool
	onProgress       func(ProgressEvent)
	acquireLock      lockFunc
	execSQL          sqlExecFunc
}

// Option configures an Executor.
type Option func(*Executor)

// WithLockTimeout sets the per-transaction lock_timeout.
func WithLockTimeout(d time.Duration) Option {
	return func(e *Executor) { e.lockTimeout = d }
}

// WithStatementTimeout sets the per-transaction statement_timeout.
func WithStatementTimeout(d time.Duration) Option {
	return func(e *Executor) { e.statementTimeout = d }
}

// WithDryRun enables dry-run mode where no SQL is executed.
func WithDryRun(b bool) Option {
	return func(e *Executor) { e.dryRun = b }
}

// WithProgressCallback sets a function called for each migration processed.
func WithProgressCallback(fn func(ProgressEvent)) Option {
	return func(e *Executor) { e.onProgress = fn }
}

// New creates an Executor with the given pool, tracker, and options.
func New(pool *pgxpool.Pool, t MigrationTracker, opts ...Option) *Executor {
	e := &Executor{
		pool:    pool,
		tracker: t,
	}

	for _, opt := range opts {
		opt(e)
	}

	// Set defaults for injectable functions after options are applied,
	// so tests can override them via options.
	if e.acquireLock == nil {
		e.acquireLock = func(ctx context.Context) (lockReleaser, error) {
			return database.TryAcquireLock(ctx, e.pool)
		}
	}

	if e.execSQL == nil {
		e.execSQL = e.executeMigration
	}

	return e
}

// Apply executes pending migrations in order. Already-applied migrations
// are skipped after verifying their checksum. The advisory lock prevents
// concurrent migration runs.
func (e *Executor) Apply(ctx context.Context, migrations []migration.Migration) error {
	lock, err := e.acquireLock(ctx)
	if err != nil {
		return fmt.Errorf("acquiring migration lock: %w", err)
	}
	defer lock.Release(ctx) //nolint:errcheck // best-effort release on return

	if err := e.tracker.EnsureTable(ctx); err != nil {
		return err
	}

	for i := range migrations {
		if err := e.applyOne(ctx, &migrations[i]); err != nil {
			return err
		}
	}

	return nil
}

// Rollback reverses the most recent `steps` applied migrations.
// Not yet implemented — will be completed in Phase 6.
func (e *Executor) Rollback(_ context.Context, _ int) error {
	return ErrRollbackNotImplemented
}

// RollbackToVersion reverses all migrations applied after the target version.
// Not yet implemented — will be completed in Phase 6.
func (e *Executor) RollbackToVersion(_ context.Context, _ string) error {
	return ErrRollbackNotImplemented
}

// applyOne handles a single migration: skip if applied, dry-run check,
// execute, record, and fire progress.
func (e *Executor) applyOne(ctx context.Context, m *migration.Migration) error {
	skip, err := e.shouldSkip(ctx, m)
	if err != nil {
		return err
	}

	if skip {
		e.fireProgress(ProgressEvent{Migration: m, Status: StatusSkipped})
		return nil
	}

	if e.dryRun {
		e.fireProgress(ProgressEvent{Migration: m, Status: StatusSkipped})
		return nil
	}

	e.fireProgress(ProgressEvent{Migration: m, Status: StatusStarting})

	start := time.Now()
	execErr := e.execSQL(ctx, m)
	duration := time.Since(start)

	if execErr != nil {
		e.fireProgress(ProgressEvent{
			Migration: m,
			Status:    StatusFailed,
			Duration:  duration,
			Error:     execErr,
		})

		return fmt.Errorf("executing migration %s: %w", m.Version, execErr)
	}

	if err := e.tracker.RecordApplied(ctx, tracker.RecordParams{
		Version:    m.Version,
		Filename:   filepath.Base(m.FilePath),
		Checksum:   m.Checksum,
		DurationMs: int(duration.Milliseconds()),
	}); err != nil {
		return fmt.Errorf("recording migration %s: %w", m.Version, err)
	}

	e.fireProgress(ProgressEvent{
		Migration: m,
		Status:    StatusCompleted,
		Duration:  duration,
	})

	return nil
}

// shouldSkip returns true if the migration is already applied.
// Verifies the checksum of applied migrations to catch file tampering.
func (e *Executor) shouldSkip(ctx context.Context, m *migration.Migration) (bool, error) {
	applied, err := e.tracker.IsApplied(ctx, m.Version)
	if err != nil {
		return false, fmt.Errorf("checking migration %s: %w", m.Version, err)
	}

	if !applied {
		return false, nil
	}

	storedChecksum, err := e.tracker.GetChecksum(ctx, m.Version)
	if err != nil {
		return false, fmt.Errorf("getting checksum for %s: %w", m.Version, err)
	}

	if storedChecksum != m.Checksum {
		return false, fmt.Errorf(
			"migration %s: %w: stored=%s computed=%s",
			m.Version, tracker.ErrChecksumMismatch, storedChecksum, m.Checksum,
		)
	}

	return true, nil
}

// executeMigration runs the SQL for a single migration, choosing between
// transactional and non-transactional execution based on whether the
// migration contains CREATE INDEX CONCURRENTLY.
func (e *Executor) executeMigration(ctx context.Context, m *migration.Migration) error {
	concurrent, err := containsConcurrentIndex(m.UpSQL)
	if err != nil {
		return err
	}

	if concurrent {
		return ExecWithoutTransaction(ctx, e.pool, m.UpSQL)
	}

	return ExecInTransaction(ctx, e.pool, func(tx pgx.Tx) error {
		if e.lockTimeout > 0 {
			if err := SetLockTimeout(ctx, tx, e.lockTimeout); err != nil {
				return err
			}
		}

		if e.statementTimeout > 0 {
			if err := SetStatementTimeout(ctx, tx, e.statementTimeout); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			return fmt.Errorf("executing SQL: %w", err)
		}

		return nil
	})
}

func (e *Executor) fireProgress(event ProgressEvent) {
	if e.onProgress != nil {
		e.onProgress(event)
	}
}
