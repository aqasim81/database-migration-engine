#!/usr/bin/env bash
# bootstrap-harness.sh — install the review/release harness into another repo.
#
# Copies, with the project name and GitHub owner substituted:
#   .github/workflows/ci.yml          CI gate (Go: make audit + integration; Node: pnpm lint/type-check/test)
#   .github/PULL_REQUEST_TEMPLATE.md  PR template
#   cliff.toml                        git-cliff config (CHANGELOG.md from conventional commits)
#   docs/adr/0000-template.md         ADR template
#   harness.mk + Makefile targets     changelog, release-dry, and test/lint (plus audit etc. for Go)
#                                     only where the target Makefile doesn't define them already
#
# Go files are copied from this repo's live versions, so improvements here
# propagate on the next run. Node variants live in scripts/harness/node/.
#
# Usage:
#   scripts/bootstrap-harness.sh <target-repo> <project-name> [--owner <gh-owner>] [--lang go|node] [--force]
#
# Existing files are never overwritten without --force. Re-running is safe.

# Make recipes below contain literal $(...) and $$ that must reach the Makefile
# unexpanded. The directive sits before the first command so it covers the file.
# shellcheck disable=SC2016
set -euo pipefail

SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
	sed -n '2,/^$/{/^$/d;s/^# \{0,1\}//;p;}' "${BASH_SOURCE[0]}"
	exit "${1:-0}"
}

die() {
	echo "error: $*" >&2
	exit 1
}

# ─── Arguments ───────────────────────────────────────────────────────────────

TARGET=""
PROJECT=""
OWNER=""
LANG_OVERRIDE=""
FORCE=0

while [ $# -gt 0 ]; do
	case "$1" in
	-h | --help) usage 0 ;;
	--owner) OWNER="${2:?--owner needs a value}"; shift 2 ;;
	--lang) LANG_OVERRIDE="${2:?--lang needs a value}"; shift 2 ;;
	--force) FORCE=1; shift ;;
	-*) die "unknown flag: $1" ;;
	*)
		if [ -z "$TARGET" ]; then TARGET="$1"
		elif [ -z "$PROJECT" ]; then PROJECT="$1"
		else die "unexpected argument: $1"
		fi
		shift
		;;
	esac
done

[ -n "$TARGET" ] && [ -n "$PROJECT" ] || usage 1
[ -d "$TARGET" ] || die "target directory not found: $TARGET"
TARGET="$(cd "$TARGET" && pwd)"
[ "$TARGET" != "$SRC" ] || die "target is the harness source repo itself"
[[ "$PROJECT" =~ ^[A-Za-z0-9._-]+$ ]] || die "project name must match [A-Za-z0-9._-]+ (the GitHub repo name)"

# Owner: flag, else parsed from the target's origin remote.
if [ -z "$OWNER" ]; then
	remote="$(git -C "$TARGET" remote get-url origin 2>/dev/null || true)"
	OWNER="$(printf '%s' "$remote" | sed -nE 's#^.*[:/]([^/:]+)/[^/]+(\.git)?$#\1#p')"
	[ -n "$OWNER" ] || die "cannot infer GitHub owner from origin remote; pass --owner"
fi
[[ "$OWNER" =~ ^[A-Za-z0-9-]+$ ]] || die "invalid GitHub owner: $OWNER"

# Language: flag, else detected from manifest files.
LANG_KIND="$LANG_OVERRIDE"
if [ -z "$LANG_KIND" ]; then
	has_go=0 has_node=0
	[ -f "$TARGET/go.mod" ] && has_go=1
	[ -f "$TARGET/package.json" ] && has_node=1
	if [ "$has_go" = 1 ] && [ "$has_node" = 1 ]; then
		die "found both go.mod and package.json; pass --lang go|node"
	elif [ "$has_go" = 1 ]; then LANG_KIND=go
	elif [ "$has_node" = 1 ]; then LANG_KIND=node
	else die "no go.mod or package.json in $TARGET; pass --lang go|node"
	fi
fi
case "$LANG_KIND" in go | node) ;; *) die "--lang must be go or node" ;; esac

echo "Installing harness into $TARGET (project=$PROJECT owner=$OWNER lang=$LANG_KIND)"

# ─── File copies ─────────────────────────────────────────────────────────────

WROTE=()
SKIPPED=()

# install_file <source> <dest-relative-to-target>
install_file() {
	local src="$1" rel="$2" dest="$TARGET/$2"
	if [ -e "$dest" ] && [ "$FORCE" != 1 ]; then
		SKIPPED+=("$rel (exists; --force to overwrite)")
		return
	fi
	mkdir -p "$(dirname "$dest")"
	cp "$src" "$dest"
	WROTE+=("$rel")
}

install_file "$SRC/.github/PULL_REQUEST_TEMPLATE.md" .github/PULL_REQUEST_TEMPLATE.md
install_file "$SRC/docs/adr/0000-template.md" docs/adr/0000-template.md

if [ "$LANG_KIND" = go ]; then
	install_file "$SRC/.github/workflows/ci.yml" .github/workflows/ci.yml
	install_file "$SRC/harness.mk" harness.mk
else
	install_file "$SRC/scripts/harness/node/ci.yml" .github/workflows/ci.yml
	install_file "$SRC/scripts/harness/node/harness.mk" harness.mk
fi

# cliff.toml: point the <REPO> commit-link postprocessor at the target repo.
if [ -e "$TARGET/cliff.toml" ] && [ "$FORCE" != 1 ]; then
	SKIPPED+=("cliff.toml (exists; --force to overwrite)")
else
	sed -E "s#(replace = \"https://github\\.com/)[^\"]+\"#\\1$OWNER/$PROJECT\"#" \
		"$SRC/cliff.toml" >"$TARGET/cliff.toml"
	grep -q "github.com/$OWNER/$PROJECT\"" "$TARGET/cliff.toml" ||
		die "cliff.toml substitution failed; the <REPO> postprocessor line changed shape"
	WROTE+=("cliff.toml")
fi

# ─── Makefile ────────────────────────────────────────────────────────────────

MAKEFILE="$TARGET/Makefile"
if [ ! -f "$MAKEFILE" ]; then
	printf '# %s — Makefile\n\n.DEFAULT_GOAL := help\n\n.PHONY: help\nhelp: ## Show this help message\n\t@grep -hE %s $(MAKEFILE_LIST) | awk %s\n' \
		"$PROJECT" \
		"'^[a-zA-Z_-]+:.*?## .*\$\$'" \
		"'BEGIN {FS = \":.*?## \"}; {printf \"  %-18s %s\\n\", \$\$1, \$\$2}'" >"$MAKEFILE"
	WROTE+=("Makefile (new)")
fi

has_target() { grep -qE "^$1:" "$MAKEFILE"; }

# add_target <name> <help> <recipe-line>...
add_target() {
	local name="$1" help="$2"
	shift 2
	if has_target "$name"; then
		SKIPPED+=("make $name (already defined)")
		return
	fi
	{
		printf '\n.PHONY: %s\n%s: ## %s\n' "$name" "$name" "$help"
		for line in "$@"; do printf '\t%s\n' "$line"; done
	} >>"$MAKEFILE"
	WROTE+=("make $name")
}

# Targets from harness.mk would clash with same-named targets in Makefile.
for t in changelog release-dry; do
	has_target "$t" && SKIPPED+=("WARNING: Makefile already defines '$t'; it will clash with harness.mk")
done

if [ "$LANG_KIND" = go ]; then
	add_target vet "Run go vet" 'go vet ./...'
	add_target lint "Run golangci-lint" 'golangci-lint run ./...'
	add_target test "Run unit tests with race detection" 'go test -race -count=1 ./...'
	add_target test-integration "Run integration-tagged tests" 'go test -race -count=1 -tags=integration ./...'
	add_target coverage-check "Fail if total coverage is below COVERAGE_MIN (default 80)" \
		'go test -coverprofile=coverage.out ./... >/dev/null' \
		'@go tool cover -func=coverage.out | awk -v min=$${COVERAGE_MIN:-80} '"'"'/^total:/ { sub("%","",$$3); print "Total coverage: "$$3"% (threshold: "min"%)"; if ($$3+0 < min) exit 1 }'"'"
	add_target audit "Full quality gate (CI runs this)" \
		'$(MAKE) vet lint test coverage-check' \
		'go mod tidy -diff'
else
	add_target lint "Run the linter" 'pnpm lint'
	add_target test "Run tests" 'pnpm test'
fi

if grep -qE '^include harness\.mk' "$MAKEFILE"; then
	SKIPPED+=("Makefile include (already present)")
else
	printf '\n# Portable release targets (changelog, release-dry), installed by bootstrap-harness.sh.\ninclude harness.mk\n' >>"$MAKEFILE"
	WROTE+=("Makefile: include harness.mk")
fi

# ─── Report ──────────────────────────────────────────────────────────────────

echo
echo "Wrote:"
if [ ${#WROTE[@]} -eq 0 ]; then echo "  (nothing)"; else printf '  %s\n' "${WROTE[@]}"; fi
if [ ${#SKIPPED[@]} -gt 0 ]; then
	echo "Skipped:"
	printf '  %s\n' "${SKIPPED[@]}"
fi

echo
echo "Next steps:"
echo "  - make changelog   (needs git-cliff) to create CHANGELOG.md"
if [ "$LANG_KIND" = go ]; then
	echo "  - make release-dry needs a .goreleaser.yml; CI pins golangci-lint v2 (v2 config format)"
else
	grep -q '"packageManager"' "$TARGET/package.json" 2>/dev/null ||
		echo "  - add \"packageManager\": \"pnpm@<version>\" to package.json (CI's pnpm setup reads it)"
fi
echo "  - add a 'demo' target and project-specific review notes to CLAUDE.md by hand"
