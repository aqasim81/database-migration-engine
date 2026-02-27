package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/config"
)

func TestNew_returnsDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.New()

	assert.Empty(t, cfg.DatabaseURL)
	assert.Equal(t, config.DefaultMigrationsDir, cfg.MigrationsDir)
	assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
	assert.Equal(t, config.DefaultStatementTimeout, cfg.StatementTimeout)
	assert.Equal(t, config.DefaultTargetPGVersion, cfg.TargetPGVersion)
	assert.Equal(t, config.DefaultFormat, cfg.Format)
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		allowMissing bool
		writeFile    bool
		wantErr      bool
		errContains  string
		check        func(t *testing.T, cfg *config.Config)
	}{
		{
			name:      "valid file parses all fields",
			writeFile: true,
			content: `database_url: "postgres://localhost:5432/testdb"
migrations_dir: "./db/migrations"
lock_timeout: "10s"
statement_timeout: "1m"
target_pg_version: 15
format: "json"
`,
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, "postgres://localhost:5432/testdb", cfg.DatabaseURL)
				assert.Equal(t, "./db/migrations", cfg.MigrationsDir)
				assert.Equal(t, 10*time.Second, cfg.LockTimeout)
				assert.Equal(t, time.Minute, cfg.StatementTimeout)
				assert.Equal(t, 15, cfg.TargetPGVersion)
				assert.Equal(t, "json", cfg.Format)
			},
		},
		{
			name:      "partial file applies defaults",
			writeFile: true,
			content:   `database_url: "postgres://localhost/mydb"`,
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, "postgres://localhost/mydb", cfg.DatabaseURL)
				assert.Equal(t, config.DefaultMigrationsDir, cfg.MigrationsDir)
				assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
				assert.Equal(t, config.DefaultStatementTimeout, cfg.StatementTimeout)
				assert.Equal(t, config.DefaultTargetPGVersion, cfg.TargetPGVersion)
				assert.Equal(t, config.DefaultFormat, cfg.Format)
			},
		},
		{
			name:      "empty file returns defaults",
			writeFile: true,
			content:   "",
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, config.DefaultMigrationsDir, cfg.MigrationsDir)
				assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
			},
		},
		{
			name:         "missing file with allowMissing returns defaults",
			writeFile:    false,
			allowMissing: true,
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, config.DefaultMigrationsDir, cfg.MigrationsDir)
				assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
			},
		},
		{
			name:         "missing file without allowMissing returns error",
			writeFile:    false,
			allowMissing: false,
			wantErr:      true,
			errContains:  "reading config file",
		},
		{
			name:        "invalid YAML returns error",
			writeFile:   true,
			content:     "{{{invalid yaml",
			wantErr:     true,
			errContains: "parsing config file",
		},
		{
			name:        "invalid lock_timeout duration returns error",
			writeFile:   true,
			content:     `lock_timeout: "not-a-duration"`,
			wantErr:     true,
			errContains: "parsing lock_timeout",
		},
		{
			name:        "invalid statement_timeout duration returns error",
			writeFile:   true,
			content:     `statement_timeout: "garbage"`,
			wantErr:     true,
			errContains: "parsing statement_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "migrate.yml")

			if tt.writeFile {
				require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o644))
			}

			cfg, err := config.Load(path, tt.allowMissing)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestMergeEnv_overridesFields(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		check func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "overrides database URL",
			env:  map[string]string{"MIGRATE_DATABASE_URL": "postgres://env-host/db"},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, "postgres://env-host/db", cfg.DatabaseURL)
			},
		},
		{
			name: "overrides migrations dir",
			env:  map[string]string{"MIGRATE_MIGRATIONS_DIR": "/custom/path"},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, "/custom/path", cfg.MigrationsDir)
			},
		},
		{
			name: "overrides lock timeout",
			env:  map[string]string{"MIGRATE_LOCK_TIMEOUT": "15s"},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, 15*time.Second, cfg.LockTimeout)
			},
		},
		{
			name: "overrides statement timeout",
			env:  map[string]string{"MIGRATE_STATEMENT_TIMEOUT": "2m"},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, 2*time.Minute, cfg.StatementTimeout)
			},
		},
		{
			name: "invalid duration preserves original",
			env:  map[string]string{"MIGRATE_LOCK_TIMEOUT": "not-valid"},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
			},
		},
		{
			name: "unset env vars preserve original",
			env:  map[string]string{},
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.Equal(t, config.DefaultMigrationsDir, cfg.MigrationsDir)
				assert.Equal(t, config.DefaultLockTimeout, cfg.LockTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg := config.New()
			config.MergeEnv(cfg)

			tt.check(t, cfg)
		})
	}
}
