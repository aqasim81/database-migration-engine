package tracker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Execer runs a statement. *pgxpool.Pool and pgx.Tx both satisfy it, which lets
// callers record a migration inside the transaction that applied it.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// AppliedMigration represents a migration record from the schema_migrations table.
type AppliedMigration struct {
	Version    string
	Filename   string
	Checksum   string
	AppliedAt  time.Time
	DurationMs int
	Status     string
}

// RecordParams contains the fields needed to record a migration as applied.
type RecordParams struct {
	Version    string
	Filename   string
	Checksum   string
	DurationMs int
}

// Tracker manages the schema_migrations table.
type Tracker struct {
	pool *pgxpool.Pool
}

// New creates a Tracker backed by the given connection pool.
func New(pool *pgxpool.Pool) *Tracker {
	return &Tracker{pool: pool}
}

// EnsureTable creates the schema_migrations table if it does not exist.
func (t *Tracker) EnsureTable(ctx context.Context) error {
	_, err := t.pool.Exec(ctx, createSchemaSQL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTableCreation, err)
	}

	return nil
}

// IsApplied checks whether a migration version has been successfully applied.
func (t *Tracker) IsApplied(ctx context.Context, version string) (bool, error) {
	var exists bool

	err := t.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1 AND status = 'applied')`,
		version,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking if migration %s is applied: %w", version, err)
	}

	return exists, nil
}

// GetApplied returns all applied migrations ordered by version.
func (t *Tracker) GetApplied(ctx context.Context) ([]AppliedMigration, error) {
	rows, err := t.pool.Query(ctx,
		`SELECT version, filename, checksum, applied_at, duration_ms, status
		 FROM schema_migrations
		 WHERE status = 'applied'
		 ORDER BY version`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying applied migrations: %w", err)
	}
	defer rows.Close()

	applied, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (AppliedMigration, error) {
		var m AppliedMigration
		if scanErr := row.Scan(&m.Version, &m.Filename, &m.Checksum, &m.AppliedAt, &m.DurationMs, &m.Status); scanErr != nil {
			return AppliedMigration{}, fmt.Errorf("scanning migration row: %w", scanErr)
		}

		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning applied migrations: %w", err)
	}

	return applied, nil
}

// RecordApplied inserts or updates a migration record with status 'applied'.
// Uses upsert to handle re-applying a previously rolled-back migration.
func (t *Tracker) RecordApplied(ctx context.Context, p RecordParams) error {
	return t.RecordAppliedIn(ctx, t.pool, p)
}

// RecordAppliedIn is RecordApplied executed on db, typically the transaction
// that ran the migration so the schema change and its record commit together.
func (t *Tracker) RecordAppliedIn(ctx context.Context, db Execer, p RecordParams) error {
	_, err := db.Exec(ctx,
		`INSERT INTO schema_migrations (version, filename, checksum, duration_ms, status)
		 VALUES ($1, $2, $3, $4, 'applied')
		 ON CONFLICT (version) DO UPDATE SET
		     filename = EXCLUDED.filename,
		     checksum = EXCLUDED.checksum,
		     applied_at = NOW(),
		     duration_ms = EXCLUDED.duration_ms,
		     status = 'applied'`,
		p.Version, p.Filename, p.Checksum, p.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("recording migration %s as applied: %w", p.Version, err)
	}

	return nil
}

// RecordRolledBack updates a migration's status to 'rolled_back'.
func (t *Tracker) RecordRolledBack(ctx context.Context, version string) error {
	return t.RecordRolledBackIn(ctx, t.pool, version)
}

// RecordRolledBackIn is RecordRolledBack executed on db, typically the
// transaction that ran the down migration.
func (t *Tracker) RecordRolledBackIn(ctx context.Context, db Execer, version string) error {
	tag, err := db.Exec(ctx,
		`UPDATE schema_migrations SET status = 'rolled_back' WHERE version = $1`,
		version,
	)
	if err != nil {
		return fmt.Errorf("recording migration %s as rolled back: %w", version, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("migration %s: %w", version, ErrMigrationNotFound)
	}

	return nil
}

// GetChecksum returns the recorded checksum for a migration version.
func (t *Tracker) GetChecksum(ctx context.Context, version string) (string, error) {
	var checksum string

	err := t.pool.QueryRow(ctx,
		`SELECT checksum FROM schema_migrations WHERE version = $1`,
		version,
	).Scan(&checksum)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("migration %s: %w", version, ErrMigrationNotFound)
		}

		return "", fmt.Errorf("getting checksum for migration %s: %w", version, err)
	}

	return checksum, nil
}
