# Product Requirements Document: Database Migration Engine

## 1. Overview

**Product Name:** `migrate` — Zero-Downtime PostgreSQL Schema Migration CLI

**Category:** Portfolio Project (GitHub showcase)

**One-liner:** A CLI tool that parses SQL migrations using the real PostgreSQL parser, detects dangerous DDL operations that cause table locks and outages, suggests safe alternatives, and executes migrations with rollback capability.

## 2. Problem Statement

Database schema migrations are one of the most dangerous operations in production systems. A single `ALTER TABLE` statement can acquire an `ACCESS EXCLUSIVE` lock, blocking all reads and writes for minutes on large tables. Common pitfalls include:

- `CREATE INDEX` without `CONCURRENTLY` — locks the entire table for the duration of the index build
- `ADD COLUMN` with a volatile `DEFAULT` — rewrites every row in the table while holding an exclusive lock
- `ADD CONSTRAINT` without `NOT VALID` — scans the entire table under an exclusive lock to validate existing rows
- `ALTER COLUMN TYPE` — triggers a full table rewrite
- `SET NOT NULL` — requires a full table scan to verify no NULLs exist

Existing migration tools (golang-migrate, goose, flyway, liquibase) focus on _executing_ migrations but do almost nothing to _prevent_ these dangerous patterns. Teams discover problems during production deployments when it's already too late.

## 3. Target Users

### Primary: Portfolio Reviewers
- Engineering managers and senior engineers evaluating GitHub profiles
- Looking for evidence of PostgreSQL internals knowledge and production engineering maturity
- Value: demonstrates deep understanding of DDL locking, concurrent operations, and migration safety

### Secondary: Backend Engineers & DBAs
- Engineers running schema migrations on production PostgreSQL databases
- Want pre-deployment analysis of migration safety
- Want guardrails integrated into CI/CD pipelines

## 4. Goals

### Portfolio Goals
- Demonstrate deep PostgreSQL internals knowledge (locking modes, DDL behavior, version-specific features)
- Show proficiency in Go systems programming (CLI design, SQL parsing, database drivers)
- Showcase production-grade engineering patterns (advisory locks, transaction management, rollback support)
- Create a genuinely useful open-source tool that could attract stars and real usage

### Technical Goals
- Parse any valid PostgreSQL SQL using the real PG parser (not regex or hand-rolled parsing)
- Detect 9 categories of dangerous DDL operations with zero false negatives on supported patterns
- Provide actionable safe alternatives for every detected danger
- Execute migrations with configurable `lock_timeout` and `statement_timeout` to prevent runaway locks
- Support rollback via `.down.sql` convention
- Track migration state in a `schema_migrations` table with checksums and execution metadata

## 5. Non-Goals

- Supporting databases other than PostgreSQL (MySQL, SQLite, etc.)
- Providing a web UI or dashboard
- Auto-generating migration SQL from schema diffs (this is a migration _runner_ and _analyzer_, not a diff tool)
- Supporting ORM-generated migrations (input is raw SQL files only)
- Providing a migration authoring experience or SQL editor

## 6. Functional Requirements

### FR-1: Migration File Management

| ID | Requirement |
|---|---|
| FR-1.1 | Read `.sql` migration files from a configurable directory |
| FR-1.2 | Support versioned naming: `VXXX_description.up.sql` / `VXXX_description.down.sql` |
| FR-1.3 | Also support timestamp-based naming: `YYYYMMDDHHMMSS_description.up.sql` |
| FR-1.4 | Compute SHA-256 checksum per file to detect tampering of applied migrations |
| FR-1.5 | Sort migrations by version/timestamp for deterministic ordering |

### FR-2: SQL Parsing & Analysis

| ID | Requirement |
|---|---|
| FR-2.1 | Parse SQL using `pg_query_go` v5 (wraps libpg_query — the real PostgreSQL C parser) |
| FR-2.2 | Walk the AST to identify DDL statement types (CREATE, ALTER, DROP, etc.) |
| FR-2.3 | Detect 9 categories of dangerous operations (see Section 7) |
| FR-2.4 | Assign severity levels: SAFE, LOW, MEDIUM, HIGH, CRITICAL |
| FR-2.5 | Provide a specific safe alternative suggestion for each detected danger |
| FR-2.6 | Report affected table name, line number, and lock type for each finding |
| FR-2.7 | Be PostgreSQL version-aware (e.g., non-volatile DEFAULT is safe on PG 11+) |

### FR-3: Danger Detection Rules

| Rule ID | Pattern | Severity | Safe Alternative |
|---|---|---|---|
| R-1 | `CREATE INDEX` (non-concurrent) | HIGH | `CREATE INDEX CONCURRENTLY` |
| R-2 | `ADD COLUMN` with volatile DEFAULT | HIGH | Add column NULL, then backfill in batches |
| R-3 | `ADD CONSTRAINT` without `NOT VALID` | HIGH | `ADD CONSTRAINT ... NOT VALID` then `VALIDATE CONSTRAINT` |
| R-4 | `ALTER COLUMN TYPE` (type change) | HIGH | Staged: add new column, backfill, swap, drop old |
| R-5 | `SET NOT NULL` | MEDIUM | PG12+: add CHECK constraint NOT VALID, validate, then SET NOT NULL |
| R-6 | `DROP TABLE` | CRITICAL | Warning only (irreversible data loss) |
| R-7 | `TRUNCATE` | CRITICAL | Warning only (irreversible data loss) |
| R-8 | `VACUUM FULL` | HIGH | Regular `VACUUM` (non-blocking) |
| R-9 | `LOCK TABLE` | HIGH | Warning (explicit locks are rarely needed) |
| R-10 | `RENAME TABLE` / `RENAME COLUMN` | MEDIUM | Warning (breaks application code referencing old names) |

### FR-4: Migration Planning

| ID | Requirement |
|---|---|
| FR-4.1 | Generate an execution plan showing migration order and detected risks |
| FR-4.2 | Estimate lock duration impact (heuristic based on operation type) |
| FR-4.3 | When connected to a database, query `pg_class` for table sizes to improve estimates |
| FR-4.4 | Show only pending migrations with `--pending-only` flag |

### FR-5: Migration Execution

| ID | Requirement |
|---|---|
| FR-5.1 | Execute migrations within a transaction (one transaction per migration file) |
| FR-5.2 | Set configurable `lock_timeout` before each migration (default: 5s) |
| FR-5.3 | Set configurable `statement_timeout` before each migration (default: 30s) |
| FR-5.4 | Detect `CREATE INDEX CONCURRENTLY` and execute outside a transaction (PostgreSQL requirement) |
| FR-5.5 | Acquire a PostgreSQL advisory lock before executing to prevent concurrent migration runs |
| FR-5.6 | Support `--dry-run` mode that reports what would execute without making changes |
| FR-5.7 | Prompt for confirmation when HIGH or CRITICAL risk operations are detected (unless `--force`) |
| FR-5.8 | Record each migration's result (version, checksum, duration, status) in tracking table |

### FR-6: Rollback

| ID | Requirement |
|---|---|
| FR-6.1 | Execute `.down.sql` files to revert migrations |
| FR-6.2 | Support `--steps N` to rollback N migrations |
| FR-6.3 | Support `--target VERSION` to rollback to a specific version |
| FR-6.4 | Remove entries from the tracking table on successful rollback |

### FR-7: Status Tracking

| ID | Requirement |
|---|---|
| FR-7.1 | Maintain a `schema_migrations` table in the target database |
| FR-7.2 | Track: version, filename, checksum (SHA-256), applied_at, duration_ms, status (applied/failed/rolled_back) |
| FR-7.3 | Detect checksum mismatches (applied migration was modified after execution) |
| FR-7.4 | `status` command shows a table of all migrations and their state |

### FR-8: CLI Interface

| ID | Requirement |
|---|---|
| FR-8.1 | Five commands: `analyze`, `plan`, `apply`, `rollback`, `status` |
| FR-8.2 | Global flags: `--config`, `--database-url`, `--migrations-dir`, `--verbose` |
| FR-8.3 | Output formats: `text` (colored terminal), `json` (CI/CD), `github-actions` (PR annotations) |
| FR-8.4 | `--fail-on-high` flag for CI gating (exit code 1 on HIGH/CRITICAL findings) |
| FR-8.5 | Colored severity output using lipgloss |

### FR-9: Configuration

| ID | Requirement |
|---|---|
| FR-9.1 | YAML config file (`migrate.yml`) for defaults |
| FR-9.2 | Environment variable overrides (`MIGRATE_DATABASE_URL`, etc.) |
| FR-9.3 | CLI flag overrides take highest precedence |
| FR-9.4 | Configurable: database URL, migrations directory, lock timeout, statement timeout, target PG version |

## 7. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-1 | Single statically-linked binary (distribute via goreleaser) |
| NFR-2 | Analyze 100 migration files in < 500ms (parsing is CPU-bound, not I/O) |
| NFR-3 | Zero external runtime dependencies (PostgreSQL client libs bundled via CGO) |
| NFR-4 | Unit test coverage > 80% on analyzer rules and migration logic |
| NFR-5 | Integration tests use testcontainers-go (real PostgreSQL, no mocks) |
| NFR-6 | Works with PostgreSQL 12, 13, 14, 15, 16 |
| NFR-7 | CGO_ENABLED=1 required (pg_query_go wraps a C library) |

## 8. Success Criteria

### Portfolio Success
- [ ] Clean, well-documented GitHub repository with a compelling README
- [ ] Architecture diagram showing the parse → analyze → plan → execute pipeline
- [ ] All 9+ danger rules with comprehensive test coverage
- [ ] Real-world example migrations demonstrating each detected pattern
- [ ] CI/CD pipeline (GitHub Actions) running tests on every push

### Technical Success
- [ ] `migrate analyze` correctly identifies all 9 dangerous patterns with zero false negatives on test cases
- [ ] `migrate apply` executes migrations with advisory lock protection and configurable timeouts
- [ ] `migrate rollback` successfully reverts migrations using `.down.sql` files
- [ ] Full lifecycle integration test passes: apply → status → rollback → status
- [ ] JSON output mode parseable by CI/CD systems

## 9. Open Questions & Future Considerations

These are explicitly **out of scope** for the initial build but worth noting:

1. **Auto-rewrite**: Automatically transform dangerous SQL into safe alternatives (vs. just suggesting)
2. **Schema drift detection**: Compare live database schema against migration history
3. **Batched backfills**: Built-in tooling for batched UPDATE operations during column additions
4. **Slack/PagerDuty integration**: Notify on migration start/complete/failure
5. **Multi-database support**: MySQL, CockroachDB (would require different parsers)
6. **Migration generation**: `migrate create <name>` to scaffold new migration files
