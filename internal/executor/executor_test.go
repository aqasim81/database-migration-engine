package executor_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/executor"
	"github.com/aqasim81/database-migration-engine/internal/migration"
)

func TestNew_defaultOptions(t *testing.T) {
	t.Parallel()

	exec := executor.New(nil, nil)

	require.NotNil(t, exec)
}

func TestNew_withOptions(t *testing.T) {
	t.Parallel()

	var received []executor.ProgressEvent
	cb := func(e executor.ProgressEvent) { received = append(received, e) }

	exec := executor.New(nil, nil,
		executor.WithLockTimeout(10*time.Second),
		executor.WithStatementTimeout(30*time.Second),
		executor.WithDryRun(true),
		executor.WithProgressCallback(cb),
	)

	require.NotNil(t, exec)
}

func TestProgressEvent_fields(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{Version: "001", Name: "create_users"}
	testErr := errors.New("test error")

	event := executor.ProgressEvent{
		Migration: m,
		Status:    executor.StatusFailed,
		Duration:  5 * time.Second,
		Error:     testErr,
	}

	assert.Equal(t, m, event.Migration)
	assert.Equal(t, executor.StatusFailed, event.Status)
	assert.Equal(t, 5*time.Second, event.Duration)
	assert.ErrorIs(t, event.Error, testErr)
}

func TestStatusConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "starting", executor.StatusStarting)
	assert.Equal(t, "completed", executor.StatusCompleted)
	assert.Equal(t, "failed", executor.StatusFailed)
	assert.Equal(t, "skipped", executor.StatusSkipped)
	assert.Equal(t, "rolling_back", executor.StatusRollingBack)
}

func TestErrors_sentinel(t *testing.T) {
	t.Parallel()

	t.Run("ErrExecutionFailed", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, executor.ErrExecutionFailed, "migration execution failed")
	})

	t.Run("ErrNoDownSQL", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, executor.ErrNoDownSQL, "no down migration file")
	})

	t.Run("ErrNothingToRollback", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, executor.ErrNothingToRollback, "no applied migrations to roll back")
	})

	t.Run("ErrTargetNotFound", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, executor.ErrTargetNotFound, "target version not found in applied migrations")
	})
}
