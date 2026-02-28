package tracker

// createSchemaSQL is the DDL for the schema_migrations tracking table.
const createSchemaSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version      TEXT PRIMARY KEY,
    filename     TEXT NOT NULL,
    checksum     TEXT NOT NULL,
    applied_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms  INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'applied'
)`
