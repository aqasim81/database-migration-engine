package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsConcurrentOp_createConcurrent_returnsTrue(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("CREATE INDEX CONCURRENTLY idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentOp_regularIndex_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("CREATE INDEX idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentOp_noIndex_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("ALTER TABLE users ADD COLUMN age INTEGER;")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentOp_multipleStatements_detectsConcurrent(t *testing.T) {
	t.Parallel()

	sql := `ALTER TABLE users ADD COLUMN email TEXT;
CREATE INDEX CONCURRENTLY idx_users_email ON users (email);`

	got, err := containsConcurrentOp(sql)

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentOp_emptySQL_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentOp_invalidSQL_returnsError(t *testing.T) {
	t.Parallel()

	_, err := containsConcurrentOp("CREATE INDEX CONCURRENTLY ;;; @@@ !!!")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing SQL")
}

func TestContainsConcurrentOp_uniqueConcurrent_returnsTrue(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("CREATE UNIQUE INDEX CONCURRENTLY idx_users_email ON users (email);")

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentOp_dropConcurrent_returnsTrue(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("DROP INDEX CONCURRENTLY IF EXISTS idx_users_email;")

	require.NoError(t, err)
	assert.True(t, got)
}

func TestContainsConcurrentOp_dropRegular_returnsFalse(t *testing.T) {
	t.Parallel()

	got, err := containsConcurrentOp("DROP INDEX idx_users_email;")

	require.NoError(t, err)
	assert.False(t, got)
}

func TestContainsConcurrentOp_dropConcurrentMultiple_returnsTrue(t *testing.T) {
	t.Parallel()

	sql := `ALTER TABLE users DROP COLUMN email;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_status;`

	got, err := containsConcurrentOp(sql)

	require.NoError(t, err)
	assert.True(t, got)
}
