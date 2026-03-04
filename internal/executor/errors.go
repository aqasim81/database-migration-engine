package executor

import "errors"

// ErrExecutionFailed indicates a migration failed to execute.
var ErrExecutionFailed = errors.New("migration execution failed")

// ErrNoDownSQL indicates a migration has no .down.sql file for rollback.
var ErrNoDownSQL = errors.New("no down migration file")

// ErrNothingToRollback indicates no applied migrations are available to roll back.
var ErrNothingToRollback = errors.New("no applied migrations to roll back")

// ErrTargetNotFound indicates the target version was not found among applied migrations.
var ErrTargetNotFound = errors.New("target version not found in applied migrations")
