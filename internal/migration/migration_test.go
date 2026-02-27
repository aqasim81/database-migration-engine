package migration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/migration"
)

func TestComputeChecksum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		sql   string
		check func(t *testing.T, checksum string)
	}{
		{
			name: "produces 64-char hex string",
			sql:  "CREATE TABLE users (id INT);",
			check: func(t *testing.T, checksum string) {
				t.Helper()
				assert.Len(t, checksum, 64)
				assert.Regexp(t, `^[0-9a-f]{64}$`, checksum)
			},
		},
		{
			name: "deterministic for same input",
			sql:  "CREATE TABLE users (id INT);",
			check: func(t *testing.T, checksum string) {
				t.Helper()
				again := migration.ComputeChecksum("CREATE TABLE users (id INT);")
				assert.Equal(t, checksum, again)
			},
		},
		{
			name: "different SQL produces different checksum",
			sql:  "CREATE TABLE users (id INT);",
			check: func(t *testing.T, checksum string) {
				t.Helper()
				other := migration.ComputeChecksum("CREATE TABLE posts (id INT);")
				assert.NotEqual(t, checksum, other)
			},
		},
		{
			name: "empty string produces valid checksum",
			sql:  "",
			check: func(t *testing.T, checksum string) {
				t.Helper()
				assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", checksum)
			},
		},
		{
			name: "whitespace matters",
			sql:  "SELECT 1",
			check: func(t *testing.T, checksum string) {
				t.Helper()
				withSpace := migration.ComputeChecksum("SELECT 1 ")
				assert.NotEqual(t, checksum, withSpace)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checksum := migration.ComputeChecksum(tt.sql)
			tt.check(t, checksum)
		})
	}
}
