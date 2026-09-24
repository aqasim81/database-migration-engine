#!/usr/bin/env bash
# Quickstart: watch migrate catch a dangerous migration, refuse to apply it,
# then apply the safe version against a throwaway PostgreSQL container.
#
# Usage: examples/quickstart/run.sh        (or: make demo)
# Needs: Go with a C compiler (CGO), Docker. Without Docker it stops after analyze.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
BIN="$ROOT/bin/migrate"
PG_IMAGE="postgres:16-alpine"
PG_PASSWORD="quickstart"

# Keep the run hermetic: ignore any MIGRATE_* settings from the caller's shell.
unset MIGRATE_DATABASE_URL MIGRATE_MIGRATIONS_DIR MIGRATE_LOCK_TIMEOUT \
	MIGRATE_STATEMENT_TIMEOUT MIGRATE_TARGET_PG_VERSION

step() { printf '\n\033[1m▶ %s\033[0m\n' "$*"; }
cmd() { printf '\033[2m$ migrate %s\033[0m\n' "$*"; }

step "Building migrate"
(cd "$ROOT" && CGO_ENABLED=1 go build -o "$BIN" ./cmd/migrate)

# Run from the example dir so a migrate.yml elsewhere is never picked up.
cd "$HERE"

step "1. Analyze: no database needed"
cmd "analyze migrations --fail-on-high"
if "$BIN" analyze migrations --fail-on-high; then
	echo "Expected analyze to fail on V002." >&2
	exit 1
fi
echo "→ analyze exited non-zero: this is what fails a CI job."

if ! docker info >/dev/null 2>&1; then
	step "Docker is not running. Stopping here."
	echo "Start Docker and re-run to see plan, apply and status against a real database."
	exit 0
fi

step "Starting throwaway PostgreSQL ($PG_IMAGE)"
docker image inspect "$PG_IMAGE" >/dev/null 2>&1 || docker pull -q "$PG_IMAGE" >/dev/null
CID="$(docker run -d --rm -P -e POSTGRES_PASSWORD="$PG_PASSWORD" "$PG_IMAGE")"
trap 'docker rm -f "$CID" >/dev/null 2>&1 || true' EXIT

# The image's init server listens on a socket only, so a TCP check means "really up".
for _ in $(seq 1 60); do
	docker exec "$CID" pg_isready -q -h 127.0.0.1 -U postgres && break
	sleep 0.5
done
docker exec "$CID" pg_isready -q -h 127.0.0.1 -U postgres

PORT="$(docker port "$CID" 5432/tcp | head -n1 | awk -F: '{print $NF}')"
DB_URL="postgres://postgres:${PG_PASSWORD}@127.0.0.1:${PORT}/postgres?sslmode=disable"

step "2. Plan: estimated risk and lock per migration"
cmd "plan --migrations-dir migrations"
"$BIN" plan --migrations-dir migrations --database-url "$DB_URL"

step "3. Apply the dangerous version: migrate refuses without an explicit \"yes\""
cmd "apply --migrations-dir migrations < /dev/null"
if "$BIN" apply --migrations-dir migrations --database-url "$DB_URL" </dev/null; then
	echo "Expected apply to refuse." >&2
	exit 1
fi

step "4. Apply the fixed version (CREATE INDEX CONCURRENTLY)"
cmd "apply --migrations-dir fixed"
"$BIN" apply --migrations-dir fixed --database-url "$DB_URL"

step "5. Status"
cmd "status --migrations-dir fixed"
"$BIN" status --migrations-dir fixed --database-url "$DB_URL"

step "Done. The container is removed on exit."
