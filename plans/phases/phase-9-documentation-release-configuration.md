# Phase 9: Documentation + Release Configuration

> **Status:** Pending
> **Commit:** `docs: add README with architecture diagram and usage examples`

## Goal

Production-ready README, release configuration, CI/CD.

---

## Step 9.1: README.md

Comprehensive README with:
- Project title, badges (Go version, test status, license)
- One-paragraph description
- Architecture diagram (ASCII art showing: SQL Files → Parser → Analyzer → Rules → Plan → Executor → PostgreSQL)
- Installation instructions (go install, brew, binary download)
- Quick start with examples of each command
- Full command reference
- Configuration reference
- List of all detection rules with examples
- Contributing section

## Step 9.2: GoReleaser Configuration

**File:** `.goreleaser.yml`

- Build for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- CGO_ENABLED=1 (note: cross-compilation with CGO requires special setup)
- Generate checksums, create GitHub release

## Step 9.3: GitHub Actions CI

**File:** `.github/workflows/ci.yml`

- Trigger on push and PR to main
- Jobs: lint (golangci-lint), test-unit, test-integration (with postgres service container)
- Cache Go modules

## Step 9.4: Verify & Commit

- README renders correctly on GitHub
- CI passes

---

## Verification Checklist

### Unit Tests (no Docker required)
- [ ] Parser: valid SQL → correct AST, invalid SQL → error
- [ ] Loader: reads files, pairs up/down, computes checksums
- [ ] Sorter: orders by version
- [ ] All 9+ analyzer rules: table-driven tests with safe/dangerous/edge cases
- [ ] Planner: builds plan from analysis results
- [ ] Config: loads YAML, merges env vars

### Integration Tests (require Docker)
- [ ] Connect to real PostgreSQL via testcontainers
- [ ] Tracker: create table, record, query, checksum verification
- [ ] Executor: apply migrations, verify they're recorded
- [ ] Executor: CREATE INDEX CONCURRENTLY runs outside transaction
- [ ] Executor: lock_timeout causes fast failure
- [ ] Full lifecycle: apply → status → rollback → status

### Manual Smoke Tests
- [ ] `migrate analyze ./testdata/migrations/` — all rules fire on expected files
- [ ] `migrate analyze --format json` — valid JSON output
- [ ] `migrate analyze --fail-on-high` — exits with code 1
- [ ] `migrate plan ./testdata/migrations/` — shows plan table
- [ ] `migrate apply --dry-run` — shows what would execute without changes
- [ ] `migrate apply` — executes and tracks migrations
- [ ] `migrate status` — shows applied/pending table
- [ ] `migrate rollback --steps 1` — reverts last migration

---

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| `pg_query_go` CGO complexity | Pin to v5, test build on CI early. Use `CGO_ENABLED=1` everywhere. |
| AST node types may change across pg_query versions | Pin dependency, test extensively, add version to go.mod |
| testcontainers-go requires Docker | CI runs Docker. Local dev can skip with `make test-unit`. |
| Cross-compilation with CGO is hard | Phase 9 goreleaser may need Docker-based cross-compilation or platform-specific CI jobs. This is a packaging concern, not a functionality concern. |
| Rule false positives | Start conservative (flag more, not fewer). Users can use `--force` to override. |

---

## Estimated Session Breakdown

| Session | Phases | Focus |
|---|---|---|
| Session 0 | Phase 0 | Quality infrastructure — linting, coverage, hooks, makefile |
| Session 1 | Phase 1 | Scaffolding, CLI, config |
| Session 2 | Phase 2 | Parser, loader, test data |
| Session 3 | Phase 3 (part 1) | Analyzer framework + rules R-1 through R-5 |
| Session 4 | Phase 3 (part 2) | Rules R-6 through R-10 + wire analyze command |
| Session 5 | Phase 4 | Database, tracker, advisory locks, integration test setup |
| Session 6 | Phase 5 | Executor, apply command, integration tests |
| Session 7 | Phase 6 | Rollback support |
| Session 8 | Phase 7 | Planner, impact estimation, plan command |
| Session 9 | Phase 8 | Polish: output formatting, status command, CI formats |
| Session 10 | Phase 9 | README, goreleaser, GitHub Actions |

Each session follows: `/clear` → read CLAUDE.md → implement phase → test → commit.
