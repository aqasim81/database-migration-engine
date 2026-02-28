package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsConcurrentIndex_concurrent_returnsTrue(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentIndex("CREATE INDEX CONCURRENTLY idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentIndex_regularIndex_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentIndex("CREATE INDEX idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentIndex_noIndex_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentIndex("ALTER TABLE users ADD COLUMN age INTEGER;")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentIndex_multipleStatements_detectsConcurrent(t *testing.T) {
	t.Parallel()

	sql := `ALTER TABLE users ADD COLUMN email TEXT;
CREATE INDEX CONCURRENTLY idx_users_email ON users (email);`

	got, err := containsConcurrentIndex(sql)

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentIndex_emptySQL_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentIndex("")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentIndex_invalidSQL_returnsError(t *testing.T) {
	t.Parallel()

	_, err := containsConcurrentIndex("NOT VALID SQL ;;; @@@ !!!")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing SQL")
}

func TestContainsConcurrentIndex_uniqueConcurrent_returnsTrue(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentIndex("CREATE UNIQUE INDEX CONCURRENTLY idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.True(t, got)
}
