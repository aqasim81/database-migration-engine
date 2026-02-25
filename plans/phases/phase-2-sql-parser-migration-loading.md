# Phase 2: SQL Parser + Migration Loading

> **Status:** Pending
> **Commit:** `feat: integrate pg_query_go parser with migration file loading`

## Goal

Parse any valid PostgreSQL SQL into an AST, load migration files from disk, sort by version.

---

## Step 2.1: SQL Parser Wrapper

**File:** `internal/parser/parser.go`

```go
type ParseResult struct {
    Stmts []*pg_query.RawStmt  // Typed AST nodes from pg_query_go
    SQL   string                // Original SQL for reference
}

func Parse(sql string) (*ParseResult, error)
```

- Call `pg_query.Parse(sql)` which returns a `*pg_query.ParseResult`
- Return the list of `RawStmt` nodes
- Handle parse errors with descriptive messages (line number, position)

**File:** `internal/parser/parser_test.go`

Table-driven tests:
- Valid `CREATE TABLE` → 1 statement, correct node type
- Valid multi-statement SQL → correct count
- `CREATE INDEX CONCURRENTLY` → parses correctly
- `ALTER TABLE ADD COLUMN` → correct node type
- Invalid SQL → returns error with position info
- Empty string → returns 0 statements

## Step 2.2: Migration Type

**File:** `internal/migration/migration.go`

```go
type Migration struct {
    Version   string    // "001" or "20240101120000"
    Name      string    // "create_users"
    UpSQL     string    // Contents of .up.sql
    DownSQL   string    // Contents of .down.sql (may be empty)
    Checksum  string    // SHA-256 of UpSQL
    FilePath  string    // Path to .up.sql file
}
```

- `ComputeChecksum(sql string) string` — SHA-256 hex digest

## Step 2.3: Migration File Loader

**File:** `internal/migration/loader.go`

```go
func LoadFromDir(dir string) ([]Migration, error)
```

- Scan directory for `*.up.sql` files
- Parse filename to extract version and name:
  - Pattern 1: `V{version}_{name}.up.sql` (e.g., `V001_create_users.up.sql`)
  - Pattern 2: `{timestamp}_{name}.up.sql` (e.g., `20240101120000_create_users.up.sql`)
- Look for matching `.down.sql` file
- Read file contents, compute checksum
- Return unsorted slice

**File:** `internal/migration/loader_test.go`

- Load from `testdata/migrations/` — correct count and content
- Missing directory → error
- Empty directory → empty slice, no error
- File without matching pattern → skipped with warning
- `.down.sql` pairing works correctly

## Step 2.4: Migration Sorter

**File:** `internal/migration/sorter.go`

```go
func Sort(migrations []Migration) []Migration
```

- Sort by version string (lexicographic — works for both zero-padded numbers and timestamps)
- Stable sort to preserve order of equal versions (shouldn't happen, but be safe)

**File:** `internal/migration/sorter_test.go`

- Already sorted → unchanged
- Reverse order → correctly sorted
- Mixed version formats → sorted correctly
- Empty slice → no panic

## Step 2.5: Test Data Files

**Directory:** `testdata/migrations/`

Create 12 migration files that cover every danger rule. Each file should be a realistic, minimal example:

| File | Content | Purpose |
|---|---|---|
| `V001_create_users.up.sql` | `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW());` | Safe migration baseline |
| `V001_create_users.down.sql` | `DROP TABLE IF EXISTS users;` | Rollback for V001 |
| `V002_add_email_index.up.sql` | `CREATE INDEX idx_users_email ON users (email);` | **R-1: Non-concurrent index** |
| `V002_add_email_index.down.sql` | `DROP INDEX IF EXISTS idx_users_email;` | Rollback for V002 |
| `V003_add_column_default.up.sql` | `ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';` | **R-2: ADD COLUMN with DEFAULT** (safe on PG11+, dangerous on PG10) |
| `V003_add_column_default.down.sql` | `ALTER TABLE users DROP COLUMN IF EXISTS status;` | Rollback |
| `V004_add_constraint.up.sql` | `ALTER TABLE users ADD CONSTRAINT chk_email CHECK (email ~* '^.+@.+$');` | **R-3: ADD CONSTRAINT without NOT VALID** |
| `V004_add_constraint.down.sql` | `ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_email;` | Rollback |
| `V005_alter_column_type.up.sql` | `ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255);` | **R-4: ALTER COLUMN TYPE** |
| `V005_alter_column_type.down.sql` | `ALTER TABLE users ALTER COLUMN email TYPE TEXT;` | Rollback |
| `V006_set_not_null.up.sql` | `ALTER TABLE users ALTER COLUMN status SET NOT NULL;` | **R-5: SET NOT NULL** |
| `V006_set_not_null.down.sql` | `ALTER TABLE users ALTER COLUMN status DROP NOT NULL;` | Rollback |
| `V007_drop_table.up.sql` | `DROP TABLE users;` | **R-6: DROP TABLE** |
| `V007_drop_table.down.sql` | `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL);` | Rollback |
| `V008_vacuum_full.up.sql` | `VACUUM FULL users;` | **R-8: VACUUM FULL** |
| `V009_lock_table.up.sql` | `LOCK TABLE users IN ACCESS EXCLUSIVE MODE;` | **R-9: LOCK TABLE** |
| `V010_rename_column.up.sql` | `ALTER TABLE users RENAME COLUMN email TO email_address;` | **R-10: RENAME** |
| `V010_rename_column.down.sql` | `ALTER TABLE users RENAME COLUMN email_address TO email;` | Rollback |
| `V011_safe_concurrent_index.up.sql` | `CREATE INDEX CONCURRENTLY idx_users_status ON users (status);` | Safe — should NOT trigger R-1 |
| `V011_safe_concurrent_index.down.sql` | `DROP INDEX CONCURRENTLY IF EXISTS idx_users_status;` | Rollback |
| `V012_safe_add_column.up.sql` | `ALTER TABLE users ADD COLUMN bio TEXT;` | Safe — nullable, no default |

## Step 2.6: Verify & Commit

- Run `make test-unit` — all parser and loader tests pass
- Verify test data files parse without errors
