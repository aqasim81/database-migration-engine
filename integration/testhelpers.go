//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage = "postgres:16-alpine"
	testDB        = "migrate_test"
	testUser      = "migrate"
	testPassword  = "migrate"
)

// SetupPostgres starts a PostgreSQL 16 container and returns a connection pool.
// The container and pool are automatically cleaned up when the test completes.
func SetupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        postgresImage,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       testDB,
			"POSTGRES_USER":     testUser,
			"POSTGRES_PASSWORD": testPassword,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := "postgres://" + testUser + ":" + testPassword + "@" + host + ":" + port.Port() + "/" + testDB + "?sslmode=disable"

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	require.NoError(t, pool.Ping(ctx))

	return pool
}
