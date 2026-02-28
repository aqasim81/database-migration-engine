package tracker

import "errors"

// ErrMigrationNotFound indicates no record exists for the given migration version.
var ErrMigrationNotFound = errors.New("migration not found in schema_migrations")

// ErrChecksumMismatch indicates the recorded checksum differs from the expected one.
var ErrChecksumMismatch = errors.New("migration checksum mismatch")

// ErrTableCreation indicates the schema_migrations table could not be created.
var ErrTableCreation = errors.New("creating schema_migrations table")
