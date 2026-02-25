# Phase 8: Polish — Output, Status, CI Formats

> **Status:** Pending
> **Commit:** `feat: add colored CLI output, status command, and JSON format`

## Goal

Professional terminal output, status command, JSON/GitHub Actions output for CI/CD.

---

## Step 8.1: Output Formatting

**File:** `internal/cli/output.go`

- Use lipgloss for colored terminal output:
  - CRITICAL: bright red, bold
  - HIGH: red
  - MEDIUM: yellow
  - LOW: cyan
  - SAFE: green
- Table formatting for status and plan commands (use lipgloss table or simple aligned columns)
- JSON output: structured data matching the internal types, one JSON object per migration
- GitHub Actions output: `::warning file={path},line={line}::{message}` and `::error` annotations

## Step 8.2: Status Command

**File:** `internal/cli/status.go` (update from stub)

- Connect to database
- Query tracker for all applied migrations
- Load migrations from disk
- Cross-reference: show applied, pending, checksum mismatches
- Display as a table:
  ```
  Migration Status
  ══════════════════════════════════════════════════

  Version  Name              Status       Applied At            Duration
  ───────  ────              ──────       ──────────            ────────
  V001     create_users      applied      2024-01-15 10:30:00   45ms
  V002     add_email_index   applied      2024-01-15 10:30:01   2.3s
  V003     add_column_def    pending      -                     -

  ⚠ V002 checksum mismatch! File was modified after applying.
  ```

## Step 8.3: Confirmation Prompts

Update `apply.go`:
- When analysis finds HIGH/CRITICAL risks and `--force` is not set:
  ```
  ⚠ 2 HIGH risk operations detected:
    - V002: CREATE INDEX without CONCURRENTLY (ACCESS EXCLUSIVE lock)
    - V005: ALTER COLUMN TYPE (full table rewrite)

  Type "yes" to proceed, or use --force to skip this check:
  ```
- Read from stdin, only proceed if input is exactly "yes"

## Step 8.4: Format Flags

Update all output-producing commands:
- `--format text` (default): colored terminal output
- `--format json`: structured JSON (one object for analyze, array of migrations for status)
- `--format github-actions`: annotation format for GitHub Actions CI

## Step 8.5: Verify & Commit

- Manual testing: `migrate analyze ./testdata/migrations/` shows colored output
- Manual testing: `migrate analyze ./testdata/migrations/ --format json` outputs valid JSON
- Manual testing: `migrate status` shows formatted table (requires DB)
- `make test-unit` passes
