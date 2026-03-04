# migrate

[![CI](https://github.com/aqasim81/database-migration-engine/actions/workflows/ci.yml/badge.svg)](https://github.com/aqasim81/database-migration-engine/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Zero-downtime PostgreSQL schema migration CLI. Parses SQL with the real PostgreSQL parser, detects dangerous DDL operations that cause table locks and outages, suggests safe alternatives, and executes migrations with rollback capability.

## Why

Most migration tools blindly execute SQL files. A single `CREATE INDEX` on a large table can lock writes for minutes and take down production. **migrate** catches these problems before they reach your database.

- **Real PostgreSQL parser** — uses `pg_query_go` (the same C library that powers pganalyze) for 100% accurate SQL parsing
- **9 built-in detection rules** — catches `CREATE INDEX` without `CONCURRENTLY`, unsafe `ALTER TABLE`, `DROP TABLE`, and more
- **PG version-aware** — rules adapt to your PostgreSQL version (e.g., `ADD COLUMN` with non-volatile `DEFAULT` is safe on PG 11+)
- **Zero-downtime execution** — configurable lock/statement timeouts, advisory locks to prevent concurrent runs, automatic `CONCURRENTLY` handling
- **CI/CD integration** — JSON output and GitHub Actions annotations for automated pipelines

## Architecture

```
                    SQL Migration Files (.up.sql / .down.sql)
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │        Migration Loader        │
                    │   (discover, sort, checksum)   │
                    └───────────────────────────────┘
                                    │
                        ┌───────────┴───────────┐
                        │                       │
                        ▼                       ▼
            ┌───────────────────────┐   ┌──────────────────┐
            │      SQL Parser       │   │   Plan Builder    │
            │ (pg_query_go / libpg) │   │ (impact, sizing)  │
            └───────────────────────┘   └──────────────────┘
                        │                       │
                        ▼                       ▼
            ┌───────────────────────┐   ┌──────────────────┐
            │    Danger Analyzer    │   │     Executor      │
            │    (9 rule checks)    │   │  (tx, timeouts,   │
            └───────────────────────┘   │  advisory locks)  │
                        │               └──────────────────┘
                        ▼                       │
            ┌───────────────────────┐           ▼
            │   Findings Report     │   ┌──────────────────┐
            │ (text/json/GH-actions)│   │ Migration Tracker │
            └───────────────────────┘   │ (schema_migrations)│
                                        └──────────────────┘
                                                │
                                                ▼
                                        ┌──────────────────┐
                                        │   PostgreSQL      │
                                        └──────────────────┘
```

## Quick Start

### Install

**From source** (requires Go 1.22+ and a C compiler — `CGO_ENABLED=1`):

```bash
go install github.com/aqasim81/database-migration-engine/cmd/migrate@latest
```

**Build from source:**

```bash
git clone https://github.com/aqasim81/database-migration-engine.git
cd database-migration-engine
make build
# Binary at ./bin/migrate
```

**Download binary** from [GitHub Releases](https://github.com/aqasim81/database-migration-engine/releases).

### Analyze Your Migrations

```bash
# Scan a directory of SQL files for dangerous patterns
migrate analyze ./migrations

# Output:
# === V001_create_users ===
#   [HIGH] CREATE INDEX without CONCURRENTLY locks the table for writes
#     Table: users
#     Rule:  create-index-not-concurrent
#     Fix:   Use CREATE INDEX CONCURRENTLY to avoid blocking writes
#
# Found 1 finding(s) across 1 migration(s).
```

### Apply Migrations

```bash
# Preview what would run
migrate apply --dry-run --database-url "postgres://localhost/mydb"

# Apply with safety checks (prompts on HIGH/CRITICAL findings)
migrate apply --database-url "postgres://localhost/mydb"

# Check current state
migrate status --database-url "postgres://localhost/mydb"
```

## Commands

### `analyze`

Scan migration files for dangerous DDL operations.

```
Usage: migrate analyze [migration-dir]

Flags:
  --format string      Output format: text, json, github-actions (default "text")
  --fail-on-high       Exit with non-zero code if HIGH/CRITICAL findings exist
```

```bash
# Standard analysis
migrate analyze ./migrations

# CI mode — fail the build on dangerous operations
migrate analyze --fail-on-high ./migrations

# GitHub Actions annotations
migrate analyze --format github-actions ./migrations
```

### `plan`

Show the execution plan for pending migrations with impact estimation.

```
Usage: migrate plan

Flags:
  --pending-only       Show only pending migrations
  --format string      Output format: text, json (default "text")
```

```bash
# Show full plan with applied/pending status and risk levels
migrate plan --database-url "postgres://localhost/mydb"

# JSON output for tooling
migrate plan --format json --database-url "postgres://localhost/mydb"
```

### `apply`

Execute pending migrations against the database.

```
Usage: migrate apply

Flags:
  --dry-run                Show what would be applied without executing
  --force                  Skip safety checks and confirmation prompts
  --lock-timeout duration  Override lock timeout (e.g., 10s, 1m)
  --statement-timeout duration  Override statement timeout (e.g., 30s, 5m)
```

```bash
# Dry run — see what would execute
migrate apply --dry-run

# Apply with custom timeouts
migrate apply --lock-timeout 10s --statement-timeout 5m

# Skip confirmation prompt (for CI/scripts)
migrate apply --force
```

When HIGH or CRITICAL findings are detected, `apply` displays them and prompts for confirmation before proceeding. Use `--force` to skip the prompt.

### `status`

Show the current migration status (applied vs. pending).

```
Usage: migrate status

Flags:
  --format string      Output format: text, json (default "text")
```

```bash
migrate status --database-url "postgres://localhost/mydb"

# Output:
# VERSION    NAME                   STATUS       APPLIED AT              DURATION
# V001       create_users           applied      2025-01-15 10:30:00     12ms
# V002       add_email_index        pending      -                       -
```

Detects **checksum mismatches** — if a migration file was modified after being applied, it's flagged as `mismatch`.

### `rollback`

Roll back previously applied migrations using their `.down.sql` files.

```
Usage: migrate rollback

Flags:
  --steps int          Number of migrations to roll back (default 1)
  --target string      Roll back to a specific migration version
```

```bash
# Roll back the last migration
migrate rollback

# Roll back 3 migrations
migrate rollback --steps 3

# Roll back to a specific version
migrate rollback --target V002
```

`--steps` and `--target` are mutually exclusive.

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `migrate.yml` | Path to configuration file |
| `--database-url` | | PostgreSQL connection string |
| `--migrations-dir` | | Path to migration files directory |
| `--verbose` | `false` | Enable verbose output |

## Detection Rules

| Rule ID | Severity | Detects | Lock Type | Safe Alternative |
|---------|----------|---------|-----------|------------------|
| `create-index-not-concurrent` | HIGH | `CREATE INDEX` without `CONCURRENTLY` | SHARE | Use `CREATE INDEX CONCURRENTLY` |
| `add-column-volatile-default` | HIGH | `ADD COLUMN` with volatile `DEFAULT` | ACCESS EXCLUSIVE | Add column without DEFAULT, backfill in batches |
| `add-constraint-without-not-valid` | HIGH | `ADD CONSTRAINT` without `NOT VALID` | ACCESS EXCLUSIVE | Add with `NOT VALID`, then `VALIDATE CONSTRAINT` separately |
| `alter-column-type` | HIGH | `ALTER COLUMN TYPE` | ACCESS EXCLUSIVE | Add new column, backfill, swap, drop old |
| `set-not-null` | HIGH* | `SET NOT NULL` | ACCESS EXCLUSIVE | Add `CHECK (col IS NOT NULL) NOT VALID`, validate, then `SET NOT NULL` |
| `drop-table` | CRITICAL | `DROP TABLE` / `TRUNCATE` | ACCESS EXCLUSIVE | Ensure backup exists; operation is irreversible |
| `vacuum-full` | HIGH | `VACUUM FULL` | ACCESS EXCLUSIVE | Use regular `VACUUM` instead |
| `lock-table` | HIGH | Explicit `LOCK TABLE` | Varies | Avoid explicit locks; let PostgreSQL manage locking |
| `rename` | MEDIUM | `RENAME TABLE` / `RENAME COLUMN` | ACCESS EXCLUSIVE | Staged approach: add new name (view), update app, drop old |

*`set-not-null` is MEDIUM on PG 12+ (has a safe constraint-based approach).

## Configuration

### Config File

Create a `migrate.yml` file (or copy from `config.example.yml`):

```yaml
# PostgreSQL connection string
database_url: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

# Path to directory containing .sql migration files
migrations_dir: "./migrations"

# Maximum time to wait for a table lock before aborting
lock_timeout: "5s"

# Maximum time a single SQL statement can run
statement_timeout: "30s"

# Target PostgreSQL version for version-aware analysis
# Affects rules like ADD COLUMN with DEFAULT (safe on PG 11+)
target_pg_version: 14
```

### Environment Variables

| Variable | Overrides |
|----------|-----------|
| `MIGRATE_DATABASE_URL` | `database_url` |
| `MIGRATE_MIGRATIONS_DIR` | `migrations_dir` |
| `MIGRATE_LOCK_TIMEOUT` | `lock_timeout` |
| `MIGRATE_STATEMENT_TIMEOUT` | `statement_timeout` |

### Precedence

CLI flags > environment variables > config file > defaults

### Defaults

| Setting | Default |
|---------|---------|
| `migrations_dir` | `./migrations` |
| `lock_timeout` | `5s` |
| `statement_timeout` | `30s` |
| `target_pg_version` | `14` |
| `format` | `text` |

## CI/CD Integration

### GitHub Actions

Use `--format github-actions` to get native annotations in pull requests:

```yaml
- name: Check migrations
  run: migrate analyze --format github-actions --fail-on-high ./migrations
```

This produces `::error` annotations for HIGH/CRITICAL findings and `::warning` for lower severities, displayed inline on the PR diff.

### JSON Output

Pipe JSON output to other tools for custom processing:

```bash
migrate analyze --format json ./migrations | jq '.migrations[].findings[]'
```

## Migration File Convention

```
migrations/
├── V001_create_users.up.sql
├── V001_create_users.down.sql
├── V002_add_email_index.up.sql
├── V002_add_email_index.down.sql
└── ...
```

- **Naming:** `V{NNN}_{description}.up.sql` / `V{NNN}_{description}.down.sql`
- **Versioning:** Files are sorted by version prefix
- **Checksums:** SHA-256 checksums are computed and stored in `schema_migrations` to detect drift
- **Up/Down pairing:** Each migration can have an `.up.sql` (apply) and optional `.down.sql` (rollback)

## Contributing

### Prerequisites

- **Go 1.22+** with a C compiler (`CGO_ENABLED=1` required for `pg_query_go`)
- **Docker** (for integration tests via testcontainers)
- **golangci-lint** (installed via `make install-tools`)

### Development

```bash
# Install dev tools (linter, formatter, gitleaks)
make install-tools

# Run unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Full quality gate (format + vet + lint + test + coverage)
make audit
```

### Code Standards

- All code in `internal/` — nothing is exported
- Error handling: return `error`, wrap with context via `fmt.Errorf`
- Testing: table-driven tests, `t.Parallel()`, `require` for preconditions, `assert` for assertions
- Coverage: 80% minimum overall, 90% on analyzer rules

## License

[MIT](LICENSE)
