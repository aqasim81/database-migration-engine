package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

func TestReverseApplied_reverses(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{
		{Version: "001"},
		{Version: "002"},
		{Version: "003"},
	}

	got := reverseApplied(applied)

	require.Len(t, got, 3)
	assert.Equal(t, "003", got[0].Version)
	assert.Equal(t, "002", got[1].Version)
	assert.Equal(t, "001", got[2].Version)

	// Original should be unchanged.
	assert.Equal(t, "001", applied[0].Version)
}

func TestReverseApplied_empty(t *testing.T) {
	t.Parallel()

	got := reverseApplied(nil)

	assert.Empty(t, got)
}

func TestReverseApplied_single(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{{Version: "001"}}

	got := reverseApplied(applied)

	require.Len(t, got, 1)
	assert.Equal(t, "001", got[0].Version)
}

func TestAppliedAfterVersion_findsAfterTarget(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{
		{Version: "001"},
		{Version: "002"},
		{Version: "003"},
	}

	got, err := appliedAfterVersion(applied, "001")

	require.NoError(t, err)
	require.Len(t, got, 2)
	// Should be in reverse order (last applied first).
	assert.Equal(t, "003", got[0].Version)
	assert.Equal(t, "002", got[1].Version)
}

func TestAppliedAfterVersion_targetNotFound_returnsError(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{
		{Version: "001"},
		{Version: "002"},
	}

	_, err := appliedAfterVersion(applied, "999")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTargetNotFound)
}

func TestAppliedAfterVersion_nothingAfter_returnsEmpty(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{
		{Version: "001"},
		{Version: "002"},
	}

	got, err := appliedAfterVersion(applied, "002")

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAppliedAfterVersion_targetIsFirst_returnsAllOthers(t *testing.T) {
	t.Parallel()

	applied := []tracker.AppliedMigration{
		{Version: "001"},
		{Version: "002"},
		{Version: "003"},
	}

	got, err := appliedAfterVersion(applied, "001")

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "003", got[0].Version)
	assert.Equal(t, "002", got[1].Version)
}

func TestBuildMigrationLookup_buildsMap(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "create_users"},
		{Version: "002", Name: "create_posts"},
	}

	lookup := buildMigrationLookup(migrations)

	require.Len(t, lookup, 2)
	assert.Equal(t, "create_users", lookup["001"].Name)
	assert.Equal(t, "create_posts", lookup["002"].Name)
}

func TestBuildMigrationLookup_empty(t *testing.T) {
	t.Parallel()

	lookup := buildMigrationLookup(nil)

	assert.Empty(t, lookup)
}

func TestBuildMigrationLookup_pointsToOriginal(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "create_users", DownSQL: "DROP TABLE users;"},
	}

	lookup := buildMigrationLookup(migrations)

	// Should point to the original slice element, preserving DownSQL.
	assert.Equal(t, "DROP TABLE users;", lookup["001"].DownSQL)
}
