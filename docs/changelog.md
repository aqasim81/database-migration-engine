# Changelog

## v0.1.0 (Unreleased)

### Phase 9: Documentation + Release Configuration
- Added comprehensive README with architecture diagram, command reference, and detection rules table
- Added MIT license
- Added GoReleaser configuration for binary releases (linux/amd64)
- Added GitHub Actions release workflow (tag-triggered)

### Phase 8: Polish, Output Formats, Status Command
- Added lipgloss-colored severity output for terminal display
- Added `--format json` and `--format github-actions` to analyze, plan, and status commands
- Added `status` command with applied/pending cross-reference and checksum mismatch detection
- Added interactive confirmation prompt on HIGH/CRITICAL risks in `apply`

### Phase 7: Planner + Impact Estimation
- Added execution plan builder with impact estimation
- Added table size queries for lock duration heuristics
- Wired `plan` command with text and JSON output

### Phase 6: Rollback Support
- Added down file loading and pairing with up migrations
- Added rollback executor with `--steps` and `--target` flags
- Wired `rollback` command with progress callbacks

### Phase 5: Execution Engine
- Added migration executor with transaction management
- Added lock/statement timeout configuration
- Added `CREATE INDEX CONCURRENTLY` detection (runs outside transaction)
- Added advisory locks to prevent concurrent migration runs
- Wired `apply` command with dry-run support

### Phase 4: Migration Tracking + Database Connection
- Added `schema_migrations` table with CRUD operations
- Added advisory lock helpers via `pg_try_advisory_lock`
- Added pgx connection pool wrapper
- Added integration test setup with testcontainers-go

### Phase 3: Danger Analysis Engine
- Added analyzer framework with Rule interface and Registry
- Implemented 9 detection rules: create-index, add-column, add-constraint, alter-column-type, set-not-null, drop-table, vacuum-full, lock-table, rename
- Added PG version-aware analysis (rules adapt to target PG version)
- Wired `analyze` command

### Phase 2: SQL Parser + Migration Loading
- Added pg_query_go v6 parser wrapper
- Added migration type, file loader, version sorter, and checksum computation
- Added test data migration files

### Phase 1: Project Scaffolding + CLI Skeleton
- Added Cobra CLI with root command and 5 subcommand stubs
- Added YAML config loading with env var and flag merging
- Added config precedence: flags > env > file > defaults

### Phase 0: Quality Infrastructure
- Added golangci-lint configuration with complexity limits
- Added test coverage enforcement (80% minimum)
- Added lefthook pre-commit/pre-push hooks
- Added Makefile with 30+ targets
- Added GitHub Actions CI (lint, test, coverage, integration, vet)
