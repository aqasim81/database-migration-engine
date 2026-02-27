package migration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/migration"
)

func TestLoadFromDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(t *testing.T) string // returns directory path
		wantErr     bool
		errContains string
		check       func(t *testing.T, ms []migration.Migration)
	}{
		{
			name: "loads from testdata directory",
			setup: func(t *testing.T) string {
				t.Helper()

				return filepath.Join("..", "..", "testdata", "migrations")
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				assert.Len(t, ms, 12, "expected 12 .up.sql migrations")

				byVersion := indexByVersion(t, ms)

				v001 := byVersion["001"]
				require.NotNil(t, v001, "V001 should exist")
				assert.Equal(t, "create_users", v001.Name)
				assert.Contains(t, v001.UpSQL, "CREATE TABLE")
				assert.Contains(t, v001.DownSQL, "DROP TABLE")
				assert.Len(t, v001.Checksum, 64)
				assert.True(t, strings.HasSuffix(v001.FilePath, "V001_create_users.up.sql"))
			},
		},
		{
			name: "missing directory returns error",
			setup: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr:     true,
			errContains: "reading migrations directory",
		},
		{
			name: "empty directory returns empty slice",
			setup: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				assert.Empty(t, ms)
			},
		},
		{
			name: "non-matching files are skipped",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "README.md", "# readme")
				writeFile(t, dir, "notes.txt", "some notes")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				assert.Empty(t, ms)
			},
		},
		{
			name: "down.sql pairing works",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_test.up.sql", "CREATE TABLE test (id INT);")
				writeFile(t, dir, "V001_test.down.sql", "DROP TABLE test;")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				assert.Equal(t, "DROP TABLE test;", ms[0].DownSQL)
			},
		},
		{
			name: "up.sql without down.sql has empty DownSQL",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_test.up.sql", "CREATE TABLE test (id INT);")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				assert.Empty(t, ms[0].DownSQL)
			},
		},
		{
			name: "orphan down.sql is skipped",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_test.down.sql", "DROP TABLE test;")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				assert.Empty(t, ms)
			},
		},
		{
			name: "timestamp filename pattern works",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "20240101120000_create_posts.up.sql", "CREATE TABLE posts (id INT);")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				assert.Equal(t, "20240101120000", ms[0].Version)
				assert.Equal(t, "create_posts", ms[0].Name)
			},
		},
		{
			name: "V-prefixed filename pattern works",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_create_users.up.sql", "CREATE TABLE users (id INT);")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				assert.Equal(t, "001", ms[0].Version)
				assert.Equal(t, "create_users", ms[0].Name)
			},
		},
		{
			name: "checksum is computed correctly",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_test.up.sql", "SELECT 1;")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				expected := migration.ComputeChecksum("SELECT 1;")
				assert.Equal(t, expected, ms[0].Checksum)
			},
		},
		{
			name: "content is trimmed before checksum",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeFile(t, dir, "V001_test.up.sql", "  SELECT 1;  \n")

				return dir
			},
			check: func(t *testing.T, ms []migration.Migration) {
				t.Helper()
				require.Len(t, ms, 1)
				assert.Equal(t, "SELECT 1;", ms[0].UpSQL)
				expected := migration.ComputeChecksum("SELECT 1;")
				assert.Equal(t, expected, ms[0].Checksum)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := tt.setup(t)
			ms, err := migration.LoadFromDir(dir)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)

			if tt.check != nil {
				tt.check(t, ms)
			}
		})
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func indexByVersion(t *testing.T, ms []migration.Migration) map[string]*migration.Migration {
	t.Helper()

	index := make(map[string]*migration.Migration, len(ms))
	for i := range ms {
		index[ms[i].Version] = &ms[i]
	}

	return index
}
