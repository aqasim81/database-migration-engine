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
	StatusStarting    = "starting"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusSkipped     = "skipped"
	StatusRollingBack = "rolling_back"
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
	GetApplied(ctx context.Context) ([]tracker.AppliedMigration, error)
	RecordRolledBack(ctx context.Context, version string) error
}

// lockReleaser is returned by lockFn and must be released when done.
type lockReleaser interface {
	Release(ctx context.Context) error
}

// lockFunc acquires an advisory lock and returns a releaser.
type lockFunc func(ctx context.Context) (lockReleaser, error)

// runSQLFunc executes SQL with a descriptive label for error wrapping.
type runSQLFunc func(ctx context.Context, sql, label string) error

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
	execSQL          runSQLFunc
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
		e.execSQL = e.runSQL
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

// Rollback reverses the most recent `steps` applied migrations using their
// down migration files. The caller must provide all known migrations so
// their DownSQL can be looked up by version.
func (e *Executor) Rollback(ctx context.Context, migrations []migration.Migration, steps int) error {
	if steps <= 0 {
		return nil
	}

	return e.withRollbackLock(ctx, migrations, func(applied []tracker.AppliedMigration) ([]tracker.AppliedMigration, error) {
		if len(applied) == 0 {
			return nil, ErrNothingToRollback
		}

		targets := reverseApplied(applied)
		if steps < len(targets) {
			targets = targets[:steps]
		}

		return targets, nil
	})
}

// RollbackToVersion reverses all applied migrations with versions greater
// than the target version. The target version itself is NOT rolled back.
func (e *Executor) RollbackToVersion(ctx context.Context, migrations []migration.Migration, target string) error {
	return e.withRollbackLock(ctx, migrations, func(applied []tracker.AppliedMigration) ([]tracker.AppliedMigration, error) {
		targets, err := appliedAfterVersion(applied, target)
		if err != nil {
			return nil, err
		}

		if len(targets) == 0 {
			return nil, ErrNothingToRollback
		}

		return targets, nil
	})
}

// withRollbackLock handles the shared rollback preamble: advisory lock,
// ensure table, get applied list, compute targets via selectFn, and execute.
func (e *Executor) withRollbackLock(
	ctx context.Context,
	migrations []migration.Migration,
	selectFn func([]tracker.AppliedMigration) ([]tracker.AppliedMigration, error),
) error {
	lock, err := e.acquireLock(ctx)
	if err != nil {
		return fmt.Errorf("acquiring migration lock: %w", err)
	}
	defer lock.Release(ctx) //nolint:errcheck // best-effort release on return

	if err := e.tracker.EnsureTable(ctx); err != nil {
		return err
	}

	applied, err := e.tracker.GetApplied(ctx)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	targets, err := selectFn(applied)
	if err != nil {
		return err
	}

	return e.rollbackTargets(ctx, targets, migrations)
}

// rollbackTargets executes the down SQL for each target in order.
func (e *Executor) rollbackTargets(
	ctx context.Context,
	targets []tracker.AppliedMigration,
	migrations []migration.Migration,
) error {
	lookup := buildMigrationLookup(migrations)

	for i := range targets {
		if err := e.rollbackOne(ctx, &targets[i], lookup); err != nil {
			return err
		}
	}

	return nil
}

// rollbackOne handles a single rollback: validate DownSQL exists, execute
// it, update tracker, and fire progress.
func (e *Executor) rollbackOne(
	ctx context.Context,
	applied *tracker.AppliedMigration,
	lookup map[string]*migration.Migration,
) error {
	m, ok := lookup[applied.Version]
	if !ok {
		return fmt.Errorf("migration %s: no migration file found for rollback", applied.Version)
	}

	if m.DownSQL == "" {
		return fmt.Errorf("migration %s (%s): %w", m.Version, m.Name, ErrNoDownSQL)
	}

	if e.dryRun {
		e.fireProgress(ProgressEvent{Migration: m, Status: StatusSkipped})
		return nil
	}

	e.fireProgress(ProgressEvent{Migration: m, Status: StatusRollingBack})

	start := time.Now()
	execErr := e.execSQL(ctx, m.DownSQL, "executing down SQL")
	duration := time.Since(start)

	if execErr != nil {
		e.fireProgress(ProgressEvent{
			Migration: m,
			Status:    StatusFailed,
			Duration:  duration,
			Error:     execErr,
		})

		return fmt.Errorf("rolling back migration %s: %w", m.Version, execErr)
	}

	if err := e.tracker.RecordRolledBack(ctx, m.Version); err != nil {
		return fmt.Errorf("recording rollback for %s: %w", m.Version, err)
	}

	e.fireProgress(ProgressEvent{
		Migration: m,
		Status:    StatusCompleted,
		Duration:  duration,
	})

	return nil
}

// runSQL executes a SQL string, choosing between transactional and
// non-transactional execution based on whether it contains concurrent
// operations (CREATE/DROP INDEX CONCURRENTLY).
func (e *Executor) runSQL(ctx context.Context, sql, label string) error {
	concurrent, err := containsConcurrentOp(sql)
	if err != nil {
		return err
	}

	if concurrent {
		return ExecWithoutTransaction(ctx, e.pool, sql)
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

		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}

		return nil
	})
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
	execErr := e.execSQL(ctx, m.UpSQL, "executing SQL")
	duration := time.Since(start)

	if execErr != nil {
		e.fireProgress(ProgressEvent{
			Migration: m,
			Status:    StatusFailed,
			Duration:  duration,
			Error:     execErr,
		})

		return fmt.Errorf("applying migration %s: %w", m.Version, execErr)
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

func (e *Executor) fireProgress(event ProgressEvent) {
	if e.onProgress != nil {
		e.onProgress(event)
	}
}
