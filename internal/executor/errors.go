package executor

import "errors"

// ErrExecutionFailed indicates a migration failed to execute.
var ErrExecutionFailed = errors.New("migration execution failed")

// ErrRollbackNotImplemented indicates rollback is not yet available.
var ErrRollbackNotImplemented = errors.New("rollback not yet implemented")
