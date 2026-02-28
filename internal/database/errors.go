package database

import "errors"

// ErrInvalidDatabaseURL indicates the provided database URL could not be parsed.
var ErrInvalidDatabaseURL = errors.New("invalid database URL")

// ErrConnectionFailed indicates a connection to the database could not be established.
var ErrConnectionFailed = errors.New("database connection failed")

// ErrLockNotAcquired indicates the advisory lock is already held by another process.
var ErrLockNotAcquired = errors.New("migration lock not acquired")
