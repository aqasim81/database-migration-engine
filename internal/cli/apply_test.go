package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
	"github.com/aqasim81/database-migration-engine/internal/migration"
)

func TestLoadAndSortMigrations_validDir_returnsSorted(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	sorted, err := loadAndSortMigrations("./testdata/migrations", buf)

	require.NoError(t, err)
	require.NotNil(t, sorted)
	require.Len(t, sorted, 2)
	assert.Equal(t, "001", sorted[0].Version)
	assert.Equal(t, "002", sorted[1].Version)
}

func TestLoadAndSortMigrations_emptyDir_returnsNil(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	sorted, err := loadAndSortMigrations(t.TempDir(), buf)

	require.NoError(t, err)
	assert.Nil(t, sorted)
	assert.Contains(t, buf.String(), "No migration files found")
}

func TestLoadAndSortMigrations_invalidDir_returnsError(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)

	_, err := loadAndSortMigrations("/nonexistent/path", buf)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading migrations")
}

func TestCheckDangerousMigrations_safeSQL_returnsFalse(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	safe := sorted[:1]
	blocked, err := checkDangerousMigrations(cmd, safe, cfg)

	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestCheckDangerousMigrations_dangerousSQL_noInput_returnsTrue(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetIn(strings.NewReader("")) // EOF — no confirmation
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	blocked, err := checkDangerousMigrations(cmd, sorted, cfg)

	require.NoError(t, err)
	assert.True(t, blocked)
	assert.Contains(t, buf.String(), "Dangerous operations detected")
}

func TestCheckDangerousMigrations_dangerousSQL_confirmedYes_unblocks(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetIn(strings.NewReader("yes\n"))
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	blocked, err := checkDangerousMigrations(cmd, sorted, cfg)

	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestCheckDangerousMigrations_dangerousSQL_confirmedNo_blocked(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetIn(strings.NewReader("no\n"))
	cfg := config.New()

	sorted, err := loadAndSortMigrations("./testdata/migrations", new(bytes.Buffer))
	require.NoError(t, err)

	blocked, err := checkDangerousMigrations(cmd, sorted, cfg)

	require.NoError(t, err)
	assert.True(t, blocked)
}

// Tests below write to the global AppConfig — they must NOT be parallel.

func TestRunApply_noMigrations_printsMessage(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	setTestAppConfig(t, &config.Config{
		DatabaseURL:   "postgres://test:test@localhost/test",
		MigrationsDir: t.TempDir(),
	})

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := runApply(cmd, nil)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No migration files found")
}

// The DB-backed runApply gate tests live in apply_integration_test.go: the
// gate now reads applied state before prompting, so it needs a real database.

func TestPendingMigrations(t *testing.T) {
	t.Parallel()

	all := []migration.Migration{{Version: "001"}, {Version: "002"}, {Version: "003"}}

	tests := []struct {
		name    string
		applied map[string]bool
		want    []string
	}{
		{name: "none applied", applied: map[string]bool{}, want: []string{"001", "002", "003"}},
		{name: "nil applied map", applied: nil, want: []string{"001", "002", "003"}},
		{name: "some applied", applied: map[string]bool{"001": true, "003": true}, want: []string{"002"}},
		{name: "all applied", applied: map[string]bool{"001": true, "002": true, "003": true}, want: []string{}},
		{name: "applied version without file", applied: map[string]bool{"999": true}, want: []string{"001", "002", "003"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pendingMigrations(all, tt.applied)

			versions := make([]string, 0, len(got))
			for _, m := range got {
				versions = append(versions, m.Version)
			}

			assert.Equal(t, tt.want, versions)
		})
	}
}

func TestReadConfirmation_yes(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader("yes\n"))
	require.NoError(t, err)
	assert.True(t, confirmed)
}

func TestReadConfirmation_yesWithSpaces(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader("  yes  \n"))
	require.NoError(t, err)
	assert.True(t, confirmed)
}

func TestReadConfirmation_no(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader("no\n"))
	require.NoError(t, err)
	assert.False(t, confirmed)
}

func TestReadConfirmation_empty(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader("\n"))
	require.NoError(t, err)
	assert.False(t, confirmed)
}

func TestReadConfirmation_eof(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader(""))
	require.NoError(t, err)
	assert.False(t, confirmed)
}

func TestReadConfirmation_caseSensitive(t *testing.T) {
	t.Parallel()

	confirmed, err := readConfirmation(strings.NewReader("Yes\n"))
	require.NoError(t, err)
	assert.False(t, confirmed)
}
