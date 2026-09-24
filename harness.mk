# harness.mk — portable release targets, included from Makefile.
# Copied verbatim into other repos by scripts/bootstrap-harness.sh; keep it
# free of project-specific names.

# ─────────────────────────────────────────
# RELEASE
# ─────────────────────────────────────────

.PHONY: changelog
changelog: ## Regenerate CHANGELOG.md from conventional commits (git-cliff)
	@command -v git-cliff >/dev/null || { echo "git-cliff not found: brew install git-cliff"; exit 1; }
	git cliff --output CHANGELOG.md
