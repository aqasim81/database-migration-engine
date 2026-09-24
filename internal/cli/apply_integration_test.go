//go:build integration

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/integration"
	"github.com/aqasim81/database-migration-engine/internal/config"
)

const (
	safeCreateSQL      = "CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT);"
	dangerousIndexSQL  = "CREATE INDEX idx_users_email ON users (email);"
	safeAddColumnSQL   = "ALTER TABLE users ADD COLUMN name TEXT;"
	dangerPromptPrefix = "Dangerous operations detected"
)

// writeMigration writes an up migration file into dir.
func writeMigration(t *testing.T, dir, filename, sql string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(sql), 0o600))
}

// runApplyWithInput runs the apply command against dsn/dir with stdin set to input.
func runApplyWithInput(t *testing.T, dsn, dir, input string) (string, error) {
	t.Helper()

	setTestAppConfig(t, &config.Config{
		DatabaseURL:     dsn,
		MigrationsDir:   dir,
		TargetPGVersion: 14,
	})

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetIn(strings.NewReader(input))

	err := runApply(cmd, nil)

	return buf.String(), err
}

// appliedCount returns how many migrations are recorded as applied, treating a
// missing schema_migrations table as zero.
func appliedCount(t *testing.T, dsn string) int {
	t.Helper()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var exists bool
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT to_regclass('public.schema_migrations') IS NOT NULL").Scan(&exists))

	if !exists {
		return 0
	}

	var n int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM schema_migrations WHERE status = 'applied'").Scan(&n))

	return n
}

// Regression test for #5: a dangerous migration that is already applied must
// not trigger the confirmation prompt on later runs.
func TestRunApply_dangerousMigrationAlreadyApplied_noPrompt(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	dsn := integration.SetupPostgresDSN(t)
	dir := t.TempDir()

	writeMigration(t, dir, "V001_create_users.up.sql", safeCreateSQL)
	writeMigration(t, dir, "V002_index_email.up.sql", dangerousIndexSQL)

	out, err := runApplyWithInput(t, dsn, dir, "yes\n")
	require.NoError(t, err)
	require.Contains(t, out, dangerPromptPrefix)
	require.Equal(t, 2, appliedCount(t, dsn))

	writeMigration(t, dir, "V003_add_name.up.sql", safeAddColumnSQL)

	out, err = runApplyWithInput(t, dsn, dir, "no\n")

	require.NoError(t, err)
	assert.NotContains(t, out, dangerPromptPrefix)
	assert.Equal(t, 3, appliedCount(t, dsn))
}

func TestRunApply_dangerousMigrationPending_blockedWithoutYes(t *testing.T) { //nolint:paralleltest // writes global AppConfig
	dsn := integration.SetupPostgresDSN(t)
	dir := t.TempDir()

	writeMigration(t, dir, "V001_create_users.up.sql", safeCreateSQL)
	writeMigration(t, dir, "V002_index_email.up.sql", dangerousIndexSQL)

	out, err := runApplyWithInput(t, dsn, dir, "no\n")

	require.ErrorIs(t, err, errDangerousMigrations)
	assert.Contains(t, out, dangerPromptPrefix)
	assert.Equal(t, 0, appliedCount(t, dsn))
}
