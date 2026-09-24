# harness.mk — portable release targets, included from Makefile.
# Installed by bootstrap-harness.sh; keep it free of project-specific names.

# ─────────────────────────────────────────
# RELEASE
# ─────────────────────────────────────────

.PHONY: changelog
changelog: ## Regenerate CHANGELOG.md from conventional commits (git-cliff)
	@command -v git-cliff >/dev/null || { echo "git-cliff not found: brew install git-cliff"; exit 1; }
	git cliff --output CHANGELOG.md

# Production build plus a preview of the next version and its release notes.
# Nothing is tagged, pushed or published.
.PHONY: release-dry
release-dry: ## Build and preview the next release's version and notes
	@command -v git-cliff >/dev/null || { echo "git-cliff not found: brew install git-cliff"; exit 1; }
	pnpm build
	@echo "Next version: $$(git cliff --bumped-version 2>/dev/null)"
	git cliff --unreleased --bump
