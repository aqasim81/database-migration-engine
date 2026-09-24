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
# CGO targets can't be cross-compiled from macOS, so off Linux this runs inside
# the goreleaser container on the release platform (slow under emulation).
GORELEASER_VERSION ?= v2.18.2
RELEASE_PLATFORM   ?= linux/amd64

.PHONY: release-dry
release-dry: ## Dry-run the GoReleaser release (snapshot, nothing published)
	@if [ "$$(uname -s)" = "Linux" ] && command -v goreleaser >/dev/null; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "Running goreleaser $(GORELEASER_VERSION) in Docker ($(RELEASE_PLATFORM))"; \
		docker run --rm --platform $(RELEASE_PLATFORM) \
			-v "$(CURDIR)":/src -w /src \
			-v "$$(go env GOMODCACHE)":/go/pkg/mod \
			-e CGO_ENABLED=1 \
			goreleaser/goreleaser:$(GORELEASER_VERSION) release --snapshot --clean; \
	fi
