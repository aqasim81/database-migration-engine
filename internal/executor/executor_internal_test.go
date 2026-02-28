package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

// mockLock implements lockReleaser for testing.
type mockLock struct {
	released bool
}

func (m *mockLock) Release(_ context.Context) error {
	m.released = true
	return nil
}

// mockTracker implements MigrationTracker for testing.
type mockTracker struct {
	ensureErr    error
	applied      map[string]bool
	checksums    map[string]string
	recorded     []tracker.RecordParams
	isAppliedErr error
	checksumErr  error
	recordErr    error
}

func newMockTracker() *mockTracker {
	return &mockTracker{
		applied:   make(map[string]bool),
		checksums: make(map[string]string),
	}
}

func (m *mockTracker) EnsureTable(_ context.Context) error {
	return m.ensureErr
}

func (m *mockTracker) IsApplied(_ context.Context, version string) (bool, error) {
	if m.isAppliedErr != nil {
		return false, m.isAppliedErr
	}

	return m.applied[version], nil
}

func (m *mockTracker) GetChecksum(_ context.Context, version string) (string, error) {
	if m.checksumErr != nil {
		return "", m.checksumErr
	}

	cs, ok := m.checksums[version]
	if !ok {
		return "", tracker.ErrMigrationNotFound
	}

	return cs, nil
}

func (m *mockTracker) RecordApplied(_ context.Context, p tracker.RecordParams) error {
	if m.recordErr != nil {
		return m.recordErr
	}

	m.recorded = append(m.recorded, p)
	m.applied[p.Version] = true
	m.checksums[p.Version] = p.Checksum

	return nil
}

func testMigration(version, sql string) migration.Migration {
	return migration.Migration{
		Version:  version,
		Name:     "test_" + version,
		UpSQL:    sql,
		Checksum: migration.ComputeChecksum(sql),
		FilePath: "migrations/V" + version + "_test.up.sql",
	}
}

func noopLockFn(_ context.Context) (lockReleaser, error) {
	return &mockLock{}, nil
}

func noopExecFn(_ context.Context, _ *migration.Migration) error {
	return nil
}

// --- shouldSkip tests ---

func TestShouldSkip_notApplied_returnsFalse(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	e := &Executor{tracker: mt}
	m := testMigration("001", "CREATE TABLE t (id INT);")

	skip, err := e.shouldSkip(context.Background(), &m)

	require.NoError(t, err)
	assert.False(t, skip)
}

func TestShouldSkip_applied_checksumMatch_returnsTrue(t *testing.T) {
	t.Parallel()

	m := testMigration("001", "CREATE TABLE t (id INT);")
	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksums["001"] = m.Checksum
	e := &Executor{tracker: mt}

	skip, err := e.shouldSkip(context.Background(), &m)

	require.NoError(t, err)
	assert.True(t, skip)
}

func TestShouldSkip_applied_checksumMismatch_returnsError(t *testing.T) {
	t.Parallel()

	m := testMigration("001", "CREATE TABLE t (id INT);")
	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksums["001"] = "wrong-checksum"
	e := &Executor{tracker: mt}

	_, err := e.shouldSkip(context.Background(), &m)

	require.Error(t, err)
	assert.ErrorIs(t, err, tracker.ErrChecksumMismatch)
}

func TestShouldSkip_isAppliedError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.isAppliedErr = errors.New("db error")
	e := &Executor{tracker: mt}
	m := testMigration("001", "CREATE TABLE t (id INT);")

	_, err := e.shouldSkip(context.Background(), &m)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checking migration 001")
}

func TestShouldSkip_getChecksumError_returnsError(t *testing.T) {
	t.Parallel()

	m := testMigration("001", "CREATE TABLE t (id INT);")
	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksumErr = errors.New("db error")
	e := &Executor{tracker: mt}

	_, err := e.shouldSkip(context.Background(), &m)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting checksum for 001")
}

// --- fireProgress tests ---

func TestFireProgress_withCallback_callsIt(t *testing.T) {
	t.Parallel()

	var received ProgressEvent
	e := &Executor{onProgress: func(ev ProgressEvent) { received = ev }}

	m := testMigration("001", "SELECT 1;")
	e.fireProgress(ProgressEvent{Migration: &m, Status: StatusStarting})

	assert.Equal(t, StatusStarting, received.Status)
	assert.Equal(t, "001", received.Migration.Version)
}

func TestFireProgress_nilCallback_noPanic(t *testing.T) {
	t.Parallel()

	e := &Executor{}
	m := testMigration("001", "SELECT 1;")

	assert.NotPanics(t, func() {
		e.fireProgress(ProgressEvent{Migration: &m, Status: StatusCompleted})
	})
}

// --- applyOne tests ---

func TestApplyOne_dryRun_skipsExecution(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:    mt,
		dryRun:     true,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
		execSQL:    noopExecFn,
	}

	m := testMigration("001", "CREATE TABLE t (id INT);")

	err := e.applyOne(context.Background(), &m)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, StatusSkipped, events[0].Status)
	assert.Empty(t, mt.recorded)
}

func TestApplyOne_alreadyApplied_skips(t *testing.T) {
	t.Parallel()

	m := testMigration("001", "CREATE TABLE t (id INT);")
	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksums["001"] = m.Checksum

	var events []ProgressEvent
	e := &Executor{
		tracker:    mt,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
		execSQL:    noopExecFn,
	}

	err := e.applyOne(context.Background(), &m)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, StatusSkipped, events[0].Status)
}

func TestApplyOne_executes_records_andReportsProgress(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:    mt,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
		execSQL:    noopExecFn,
	}

	m := testMigration("001", "CREATE TABLE t (id INT);")

	err := e.applyOne(context.Background(), &m)

	require.NoError(t, err)

	// Should have starting + completed events.
	require.Len(t, events, 2)
	assert.Equal(t, StatusStarting, events[0].Status)
	assert.Equal(t, StatusCompleted, events[1].Status)

	// Should be recorded.
	require.Len(t, mt.recorded, 1)
	assert.Equal(t, "001", mt.recorded[0].Version)
	assert.Equal(t, m.Checksum, mt.recorded[0].Checksum)
}

func TestApplyOne_execError_reportsFailed(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	execErr := errors.New("SQL error")
	e := &Executor{
		tracker:    mt,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
		execSQL:    func(_ context.Context, _ *migration.Migration) error { return execErr },
	}

	m := testMigration("001", "CREATE TABLE t (id INT);")

	err := e.applyOne(context.Background(), &m)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "executing migration 001")

	require.Len(t, events, 2)
	assert.Equal(t, StatusStarting, events[0].Status)
	assert.Equal(t, StatusFailed, events[1].Status)
	assert.ErrorIs(t, events[1].Error, execErr)
}

func TestApplyOne_recordError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.recordErr = errors.New("record failed")

	e := &Executor{
		tracker: mt,
		execSQL: noopExecFn,
	}

	m := testMigration("001", "CREATE TABLE t (id INT);")

	err := e.applyOne(context.Background(), &m)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recording migration 001")
}

// --- Apply tests (with mock lock + tracker) ---

func TestApply_fullFlow_appliesAll(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
		onProgress:  func(ev ProgressEvent) { events = append(events, ev) },
	}

	migrations := []migration.Migration{
		testMigration("001", "CREATE TABLE a (id INT);"),
		testMigration("002", "CREATE TABLE b (id INT);"),
	}

	err := e.Apply(context.Background(), migrations)

	require.NoError(t, err)
	require.Len(t, mt.recorded, 2)
	assert.Equal(t, "001", mt.recorded[0].Version)
	assert.Equal(t, "002", mt.recorded[1].Version)

	// 2 starting + 2 completed = 4 events.
	require.Len(t, events, 4)
}

func TestApply_lockError_returnsError(t *testing.T) {
	t.Parallel()

	lockErr := errors.New("lock failed")
	e := &Executor{
		tracker: newMockTracker(),
		acquireLock: func(_ context.Context) (lockReleaser, error) {
			return nil, lockErr
		},
	}

	err := e.Apply(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "acquiring migration lock")
}

func TestApply_ensureTableError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.ensureErr = errors.New("create table failed")

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
	}

	err := e.Apply(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create table failed")
}

func TestApply_emptyMigrations_succeeds(t *testing.T) {
	t.Parallel()

	e := &Executor{
		tracker:     newMockTracker(),
		acquireLock: noopLockFn,
	}

	err := e.Apply(context.Background(), []migration.Migration{})

	require.NoError(t, err)
}

func TestApply_dryRun_nothingRecorded(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
		dryRun:      true,
		onProgress:  func(ev ProgressEvent) { events = append(events, ev) },
	}

	migrations := []migration.Migration{
		testMigration("001", "CREATE TABLE a (id INT);"),
	}

	err := e.Apply(context.Background(), migrations)

	require.NoError(t, err)
	assert.Empty(t, mt.recorded)
	require.Len(t, events, 1)
	assert.Equal(t, StatusSkipped, events[0].Status)
}

func TestApply_skipsAlreadyApplied_andAppliesPending(t *testing.T) {
	t.Parallel()

	m1 := testMigration("001", "CREATE TABLE a (id INT);")
	m2 := testMigration("002", "CREATE TABLE b (id INT);")

	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksums["001"] = m1.Checksum

	var events []ProgressEvent
	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
		onProgress:  func(ev ProgressEvent) { events = append(events, ev) },
	}

	err := e.Apply(context.Background(), []migration.Migration{m1, m2})

	require.NoError(t, err)
	require.Len(t, mt.recorded, 1)
	assert.Equal(t, "002", mt.recorded[0].Version)

	// 1 skip + 1 starting + 1 completed = 3 events.
	require.Len(t, events, 3)
	assert.Equal(t, StatusSkipped, events[0].Status)
	assert.Equal(t, StatusStarting, events[1].Status)
	assert.Equal(t, StatusCompleted, events[2].Status)
}

func TestApply_checksumMismatch_stopsEarly(t *testing.T) {
	t.Parallel()

	m1 := testMigration("001", "CREATE TABLE a (id INT);")
	m2 := testMigration("002", "CREATE TABLE b (id INT);")

	mt := newMockTracker()
	mt.applied["001"] = true
	mt.checksums["001"] = "tampered-checksum"

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
	}

	err := e.Apply(context.Background(), []migration.Migration{m1, m2})

	require.ErrorIs(t, err, tracker.ErrChecksumMismatch)
	assert.Empty(t, mt.recorded)
}

func TestApply_lockReleased(t *testing.T) {
	t.Parallel()

	lock := &mockLock{}

	e := &Executor{
		tracker: newMockTracker(),
		acquireLock: func(_ context.Context) (lockReleaser, error) {
			return lock, nil
		},
	}

	err := e.Apply(context.Background(), []migration.Migration{})

	require.NoError(t, err)
	assert.True(t, lock.released)
}
