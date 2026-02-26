# Database Migration Engine — Implementation Checklist

Each phase follows a structured workflow: plan → review → implement → review → test → fix → update checklist → next phase.

---

## Phase 0: Quality Infrastructure

> **Plan file:** [`plans/phases/phase-0-quality-infrastructure.md`](phases/phase-0-quality-infrastructure.md)

- [x] **Plan** — Read phase plan, understand scope and deliverables
- [x] **Review the plan** — Verify approach, clarify any ambiguities
- [x] **Implement** — Set up linting, coverage, hooks, Makefile, editorconfig
- [x] **Review the implementation** — Code review, check against plan requirements
- [x] **Test** — Run `make audit` to verify full quality gate passes
- [x] **Fix** — Address any issues found during review or testing
- [x] **Update checklist** — Mark items complete
- [x] **Next** — Proceed to Phase 1

---

## Phase 1: Project Scaffolding + CLI Skeleton

> **Plan file:** [`plans/phases/phase-1-project-scaffolding-cli-skeleton.md`](phases/phase-1-project-scaffolding-cli-skeleton.md)

- [x] **Plan** — Read phase plan, understand scope and deliverables
- [x] **Review the plan** — Verify approach, clarify any ambiguities
- [x] **Implement** — Create go.mod, entry point, root command, 5 subcommand stubs, config loading
- [x] **Review the implementation** — Code review, check against plan requirements
- [x] **Test** — Run `make build` and verify `bin/migrate --help` shows all commands
- [x] **Fix** — Address any issues found during review or testing
- [x] **Update checklist** — Mark items complete
- [x] **Next** — Proceed to Phase 2

---

## Phase 2: SQL Parser + Migration Loading

> **Plan file:** [`plans/phases/phase-2-sql-parser-migration-loading.md`](phases/phase-2-sql-parser-migration-loading.md)

- [x] **Plan** — Read phase plan, understand scope and deliverables
- [x] **Review the plan** — Verify approach, clarify any ambiguities
- [x] **Implement** — Parser wrapper, migration type, file loader, sorter, test data files
- [x] **Review the implementation** — Code review, check against plan requirements
- [x] **Test** — Run `make test` — all parser and loader tests pass
- [x] **Fix** — Address any issues found during review or testing
- [x] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 3

---

## Phase 3: Danger Analysis Engine (Core Value)

> **Plan file:** [`plans/phases/phase-3-danger-analysis-engine.md`](phases/phase-3-danger-analysis-engine.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Severity enum, result types, rule interface, analyzer orchestrator, 9 rules, wire analyze command
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` and `bin/migrate analyze ./testdata/migrations/` to verify all rules fire correctly
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 4

---

## Phase 4: Migration Tracking + Database Connection

> **Plan file:** [`plans/phases/phase-4-migration-tracking-database-connection.md`](phases/phase-4-migration-tracking-database-connection.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Database connection, advisory locks, schema_migrations table, tracker CRUD, integration test setup
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` and `make test-integration` — tracker tests pass against real PostgreSQL
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 5

---

## Phase 5: Execution Engine

> **Plan file:** [`plans/phases/phase-5-execution-engine.md`](phases/phase-5-execution-engine.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Safety helpers, transaction management, executor, wire apply command, integration tests
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` and `make test-integration` — full apply lifecycle passes
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 6

---

## Phase 6: Rollback Support

> **Plan file:** [`plans/phases/phase-6-rollback-support.md`](phases/phase-6-rollback-support.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Down file loading, rollback in executor, wire rollback command, integration tests
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` and `make test-integration` — rollback lifecycle passes
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 7

---

## Phase 7: Planner + Impact Estimation

> **Plan file:** [`plans/phases/phase-7-planner-impact-estimation.md`](phases/phase-7-planner-impact-estimation.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Impact estimation, planner, wire plan command
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` — planner tests pass
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 8

---

## Phase 8: Polish — Output, Status, CI Formats

> **Plan file:** [`plans/phases/phase-8-polish-output-status-ci-formats.md`](phases/phase-8-polish-output-status-ci-formats.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — Lipgloss output formatting, status command, confirmation prompts, JSON/GitHub Actions formats
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Run `make test` and manually verify colored output, JSON output, status table
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Next** — Proceed to Phase 9

---

## Phase 9: Documentation + Release Configuration

> **Plan file:** [`plans/phases/phase-9-documentation-release-configuration.md`](phases/phase-9-documentation-release-configuration.md)

- [ ] **Plan** — Read phase plan, understand scope and deliverables
- [ ] **Review the plan** — Verify approach, clarify any ambiguities
- [ ] **Implement** — README, GoReleaser config, GitHub Actions CI
- [ ] **Review the implementation** — Code review, check against plan requirements
- [ ] **Test** — Verify README renders, CI passes, all verification checklist items pass
- [ ] **Fix** — Address any issues found during review or testing
- [ ] **Update checklist** — Mark items complete
- [ ] **Done** — Project complete!
