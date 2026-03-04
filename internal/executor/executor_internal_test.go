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
	ensureErr     error
	applied       map[string]bool
	checksums     map[string]string
	recorded      []tracker.RecordParams
	isAppliedErr  error
	checksumErr   error
	recordErr     error
	appliedList   []tracker.AppliedMigration
	getAppliedErr error
	rolledBack    []string
	rollbackErr   error
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

func (m *mockTracker) GetApplied(_ context.Context) ([]tracker.AppliedMigration, error) {
	if m.getAppliedErr != nil {
		return nil, m.getAppliedErr
	}

	return m.appliedList, nil
}

func (m *mockTracker) RecordRolledBack(_ context.Context, version string) error {
	if m.rollbackErr != nil {
		return m.rollbackErr
	}

	m.rolledBack = append(m.rolledBack, version)
	m.applied[version] = false

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

func noopExecFn(_ context.Context, _, _ string) error {
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
		execSQL:    func(_ context.Context, _ string, _ string) error { return execErr },
	}

	m := testMigration("001", "CREATE TABLE t (id INT);")

	err := e.applyOne(context.Background(), &m)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying migration 001")

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

// --- helper for rollback tests ---

func testMigrationWithDown(version, upSQL, downSQL string) migration.Migration {
	return migration.Migration{
		Version:  version,
		Name:     "test_" + version,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Checksum: migration.ComputeChecksum(upSQL),
		FilePath: "migrations/V" + version + "_test.up.sql",
	}
}

func makeAppliedList(versions ...string) []tracker.AppliedMigration {
	var list []tracker.AppliedMigration
	for _, v := range versions {
		list = append(list, tracker.AppliedMigration{Version: v, Filename: "V" + v + "_test.up.sql"})
	}

	return list
}

// --- rollbackOne tests ---

func TestRollbackOne_validDownSQL_executesAndRecords(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:    mt,
		execSQL:    noopExecFn,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
	}

	m := testMigrationWithDown("001", "CREATE TABLE t (id INT);", "DROP TABLE t;")
	lookup := buildMigrationLookup([]migration.Migration{m})
	applied := &tracker.AppliedMigration{Version: "001"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, StatusRollingBack, events[0].Status)
	assert.Equal(t, StatusCompleted, events[1].Status)
	require.Len(t, mt.rolledBack, 1)
	assert.Equal(t, "001", mt.rolledBack[0])
}

func TestRollbackOne_emptyDownSQL_returnsErrNoDownSQL(t *testing.T) {
	t.Parallel()

	e := &Executor{tracker: newMockTracker(), execSQL: noopExecFn}

	m := testMigrationWithDown("001", "CREATE TABLE t (id INT);", "")
	lookup := buildMigrationLookup([]migration.Migration{m})
	applied := &tracker.AppliedMigration{Version: "001"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoDownSQL)
}

func TestRollbackOne_migrationNotFound_returnsError(t *testing.T) {
	t.Parallel()

	e := &Executor{tracker: newMockTracker(), execSQL: noopExecFn}

	lookup := make(map[string]*migration.Migration)
	applied := &tracker.AppliedMigration{Version: "999"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration 999: no migration file found")
}

func TestRollbackOne_execError_reportsFailed(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent
	execErr := errors.New("SQL error")

	e := &Executor{
		tracker:    mt,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
		execSQL: func(_ context.Context, _ string, _ string) error {
			return execErr
		},
	}

	m := testMigrationWithDown("001", "CREATE TABLE t (id INT);", "DROP TABLE t;")
	lookup := buildMigrationLookup([]migration.Migration{m})
	applied := &tracker.AppliedMigration{Version: "001"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rolling back migration 001")
	require.Len(t, events, 2)
	assert.Equal(t, StatusRollingBack, events[0].Status)
	assert.Equal(t, StatusFailed, events[1].Status)
	assert.Empty(t, mt.rolledBack)
}

func TestRollbackOne_recordError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.rollbackErr = errors.New("record failed")

	e := &Executor{tracker: mt, execSQL: noopExecFn}

	m := testMigrationWithDown("001", "CREATE TABLE t (id INT);", "DROP TABLE t;")
	lookup := buildMigrationLookup([]migration.Migration{m})
	applied := &tracker.AppliedMigration{Version: "001"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recording rollback for 001")
}

func TestRollbackOne_dryRun_skipsExecution(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	var events []ProgressEvent

	e := &Executor{
		tracker:    mt,
		dryRun:     true,
		execSQL:    noopExecFn,
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
	}

	m := testMigrationWithDown("001", "CREATE TABLE t (id INT);", "DROP TABLE t;")
	lookup := buildMigrationLookup([]migration.Migration{m})
	applied := &tracker.AppliedMigration{Version: "001"}

	err := e.rollbackOne(context.Background(), applied, lookup)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, StatusSkipped, events[0].Status)
	assert.Empty(t, mt.rolledBack)
}

// --- Rollback tests (steps-based) ---

func TestRollback_fullFlow_rollsBackLastN(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002", "003")

	var events []ProgressEvent

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
		onProgress:  func(ev ProgressEvent) { events = append(events, ev) },
	}

	migrations := []migration.Migration{
		testMigrationWithDown("001", "CREATE TABLE a (id INT);", "DROP TABLE a;"),
		testMigrationWithDown("002", "CREATE TABLE b (id INT);", "DROP TABLE b;"),
		testMigrationWithDown("003", "CREATE TABLE c (id INT);", "DROP TABLE c;"),
	}

	err := e.Rollback(context.Background(), migrations, 2)

	require.NoError(t, err)
	require.Len(t, mt.rolledBack, 2)
	assert.Equal(t, "003", mt.rolledBack[0])
	assert.Equal(t, "002", mt.rolledBack[1])
	// 2 rolling_back + 2 completed = 4 events.
	require.Len(t, events, 4)
}

func TestRollback_stepsZero_noop(t *testing.T) {
	t.Parallel()

	e := &Executor{tracker: newMockTracker(), acquireLock: noopLockFn}

	err := e.Rollback(context.Background(), nil, 0)

	require.NoError(t, err)
}

func TestRollback_noAppliedMigrations_returnsErrNothingToRollback(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = nil

	e := &Executor{tracker: mt, acquireLock: noopLockFn}

	err := e.Rollback(context.Background(), nil, 1)

	require.ErrorIs(t, err, ErrNothingToRollback)
}

func TestRollback_stepsExceedsApplied_rollsBackAll(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002")

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
	}

	migrations := []migration.Migration{
		testMigrationWithDown("001", "CREATE TABLE a (id INT);", "DROP TABLE a;"),
		testMigrationWithDown("002", "CREATE TABLE b (id INT);", "DROP TABLE b;"),
	}

	err := e.Rollback(context.Background(), migrations, 10)

	require.NoError(t, err)
	require.Len(t, mt.rolledBack, 2)
	assert.Equal(t, "002", mt.rolledBack[0])
	assert.Equal(t, "001", mt.rolledBack[1])
}

func TestRollback_lockError_returnsError(t *testing.T) {
	t.Parallel()

	e := &Executor{
		tracker: newMockTracker(),
		acquireLock: func(_ context.Context) (lockReleaser, error) {
			return nil, errors.New("lock failed")
		},
	}

	err := e.Rollback(context.Background(), nil, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "acquiring migration lock")
}

func TestRollback_ensureTableError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.ensureErr = errors.New("create table failed")

	e := &Executor{tracker: mt, acquireLock: noopLockFn}

	err := e.Rollback(context.Background(), nil, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create table failed")
}

func TestRollback_getAppliedError_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.getAppliedErr = errors.New("db error")

	e := &Executor{tracker: mt, acquireLock: noopLockFn}

	err := e.Rollback(context.Background(), nil, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting applied migrations")
}

func TestRollback_lockReleased(t *testing.T) {
	t.Parallel()

	lock := &mockLock{}
	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001")

	e := &Executor{
		tracker: mt,
		acquireLock: func(_ context.Context) (lockReleaser, error) {
			return lock, nil
		},
		execSQL: noopExecFn,
	}

	migrations := []migration.Migration{
		testMigrationWithDown("001", "CREATE TABLE a (id INT);", "DROP TABLE a;"),
	}

	err := e.Rollback(context.Background(), migrations, 1)

	require.NoError(t, err)
	assert.True(t, lock.released)
}

func TestRollback_partialFailure_earlierRollbacksTracked(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002", "003")

	callCount := 0
	execErr := errors.New("SQL error on second rollback")

	var events []ProgressEvent

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL: func(_ context.Context, _, _ string) error {
			callCount++
			if callCount == 2 {
				return execErr
			}

			return nil
		},
		onProgress: func(ev ProgressEvent) { events = append(events, ev) },
	}

	migrations := []migration.Migration{
		testMigrationWithDown("001", "CREATE TABLE a (id INT);", "DROP TABLE a;"),
		testMigrationWithDown("002", "CREATE TABLE b (id INT);", "DROP TABLE b;"),
		testMigrationWithDown("003", "CREATE TABLE c (id INT);", "DROP TABLE c;"),
	}

	err := e.Rollback(context.Background(), migrations, 3)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rolling back migration 002")

	// First rollback (003) should have succeeded and been recorded.
	require.Len(t, mt.rolledBack, 1)
	assert.Equal(t, "003", mt.rolledBack[0])

	// Events: rolling_back(003), completed(003), rolling_back(002), failed(002).
	require.Len(t, events, 4)
	assert.Equal(t, StatusRollingBack, events[0].Status)
	assert.Equal(t, StatusCompleted, events[1].Status)
	assert.Equal(t, StatusRollingBack, events[2].Status)
	assert.Equal(t, StatusFailed, events[3].Status)
}

// --- RollbackToVersion tests ---

func TestRollbackToVersion_rollsBackAfterTarget(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002", "003")

	e := &Executor{
		tracker:     mt,
		acquireLock: noopLockFn,
		execSQL:     noopExecFn,
	}

	migrations := []migration.Migration{
		testMigrationWithDown("001", "CREATE TABLE a (id INT);", "DROP TABLE a;"),
		testMigrationWithDown("002", "CREATE TABLE b (id INT);", "DROP TABLE b;"),
		testMigrationWithDown("003", "CREATE TABLE c (id INT);", "DROP TABLE c;"),
	}

	err := e.RollbackToVersion(context.Background(), migrations, "001")

	require.NoError(t, err)
	require.Len(t, mt.rolledBack, 2)
	assert.Equal(t, "003", mt.rolledBack[0])
	assert.Equal(t, "002", mt.rolledBack[1])
}

func TestRollbackToVersion_targetNotFound_returnsError(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002")

	e := &Executor{tracker: mt, acquireLock: noopLockFn}

	err := e.RollbackToVersion(context.Background(), nil, "999")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTargetNotFound)
}

func TestRollbackToVersion_nothingAfterTarget_returnsErrNothingToRollback(t *testing.T) {
	t.Parallel()

	mt := newMockTracker()
	mt.appliedList = makeAppliedList("001", "002")

	e := &Executor{tracker: mt, acquireLock: noopLockFn}

	err := e.RollbackToVersion(context.Background(), nil, "002")

	require.ErrorIs(t, err, ErrNothingToRollback)
}
