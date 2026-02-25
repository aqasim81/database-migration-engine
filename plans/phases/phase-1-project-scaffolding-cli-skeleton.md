# Phase 1: Project Scaffolding + CLI Skeleton

> **Status:** Pending
> **Commit:** `feat: scaffold project structure with CLI skeleton and config loading`

## Goal

Runnable binary with 5 subcommand stubs, config loading, and project infrastructure.

---

## Step 1.1: Initialize Go Module

Create `go.mod` with module path `github.com/ahmad/migrate`.

**File:** `go.mod`
```
module github.com/ahmad/migrate

go 1.22
```

Run `go mod tidy` after adding initial dependencies.

## Step 1.2: Create Entry Point

**File:** `cmd/migrate/main.go`

- Import and execute `internal/cli.Execute()`
- Minimal — just the bridge between `main()` and Cobra

## Step 1.3: Root Command + Global Flags

**File:** `internal/cli/root.go`

- Define root `cobra.Command` with app name `migrate`, version, description
- Global persistent flags:
  - `--config` (string, default `migrate.yml`) — path to config file
  - `--database-url` (string) — PostgreSQL connection string
  - `--migrations-dir` (string, default `./migrations`) — path to migration files
  - `--verbose` (bool) — enable debug logging
- `PersistentPreRunE`: load config file, merge env vars, merge flags (flag > env > file)
- Export `Execute()` function that calls `rootCmd.Execute()`

## Step 1.4: Five Subcommand Stubs

**Files:** `internal/cli/analyze.go`, `plan.go`, `apply.go`, `rollback.go`, `status.go`

Each file:
- Defines a `cobra.Command` with name, short/long description, relevant flags
- `RunE` function that prints "Not yet implemented" (placeholder)
- Registers as subcommand of root in `init()`

**Command-specific flags:**
- `analyze`: `--format` (text/json/github-actions), `--fail-on-high`
- `plan`: `--pending-only`
- `apply`: `--dry-run`, `--force`, `--lock-timeout` (duration), `--statement-timeout` (duration)
- `rollback`: `--steps` (int), `--target` (string)
- `status`: `--format` (text/json)

## Step 1.5: Configuration

**File:** `internal/config/config.go`

```go
type Config struct {
    DatabaseURL      string        `yaml:"database_url"`
    MigrationsDir    string        `yaml:"migrations_dir"`
    LockTimeout      time.Duration `yaml:"lock_timeout"`
    StatementTimeout time.Duration `yaml:"statement_timeout"`
    TargetPGVersion  int           `yaml:"target_pg_version"` // e.g., 14
    Format           string        `yaml:"format"`
}
```

- `Load(path string) (*Config, error)` — reads YAML file
- `MergeEnv(cfg *Config)` — override with `MIGRATE_DATABASE_URL`, `MIGRATE_MIGRATIONS_DIR`, etc.
- Provide sensible defaults: lock_timeout=5s, statement_timeout=30s, target_pg_version=14

## Step 1.6: Example Config

**File:** `config.example.yml`

```yaml
database_url: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
migrations_dir: "./migrations"
lock_timeout: "5s"
statement_timeout: "30s"
target_pg_version: 14
```

## Step 1.7: Makefile

**File:** `Makefile`

```makefile
.PHONY: build test test-unit test-integration lint clean

build:
	CGO_ENABLED=1 go build -o bin/migrate ./cmd/migrate

test: test-unit

test-unit:
	CGO_ENABLED=1 go test -race -count=1 ./internal/...

test-integration:
	CGO_ENABLED=1 go test -race -count=1 -tags=integration ./integration/...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
```

## Step 1.8: CLAUDE.md

**File:** `CLAUDE.md`

Project-specific guidance for Claude Code: stack, architecture, commands, conventions, current status.

## Step 1.9: Verify & Commit

- Run `go mod tidy` to resolve all dependencies
- Run `make build` to verify the binary compiles
- Run `bin/migrate --help` to verify all 5 subcommands appear
- Run `bin/migrate analyze` to verify it prints "not yet implemented"
