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

# Snapshot release: build, archive and checksum exactly as the tag-triggered
# release workflow would, without publishing. Output lands in dist/.
# Needs whatever toolchain .goreleaser.yml requires (for CGO projects that can
# mean a specific host OS or cross-compilers; see that file's header).
.PHONY: release-dry
release-dry: ## Dry-run the GoReleaser release (snapshot, nothing published)
	@command -v goreleaser >/dev/null || { echo "goreleaser not found: brew install goreleaser"; exit 1; }
	goreleaser check
	goreleaser release --snapshot --clean
