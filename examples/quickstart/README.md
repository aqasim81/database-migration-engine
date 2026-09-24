# Quickstart

See `migrate` catch a dangerous migration, refuse to apply it, and then apply
the safe version against a throwaway PostgreSQL container.

```bash
git clone https://github.com/aqasim81/database-migration-engine.git
cd database-migration-engine
make demo            # same as: examples/quickstart/run.sh
```

**Needs:** Go 1.25+ with a C compiler (`pg_query_go` uses CGO; on macOS the
Xcode command line tools are enough) and Docker. Without Docker the script
stops after step 1, which needs no database.

**Time:** about 5s warm. On a first run, add roughly 15s for the CGO build and
10s to pull `postgres:16-alpine`, plus Go module downloads.

## What it does

| Step | Command | What you see |
|------|---------|--------------|
| 1 | `migrate analyze migrations --fail-on-high` | `V002` flagged HIGH: `CREATE INDEX` without `CONCURRENTLY` blocks writes to `orders`. Exit code 1, which is what fails a CI job. |
| 2 | `migrate plan` | Per-migration risk and estimated lock window, using the live database. |
| 3 | `migrate apply --migrations-dir migrations` | Refused. Stdin is `/dev/null`, so nobody types `yes`. Nothing is applied. |
| 4 | `migrate apply --migrations-dir fixed` | Both migrations applied. The `CONCURRENTLY` index runs outside a transaction automatically. |
| 5 | `migrate status` | Both migrations recorded in `schema_migrations`. |

The container is started with `--rm` on a random port and removed when the
script exits, even if it fails.

## Files

- `migrations/`: `V001` creates `orders`. `V002` adds an index the unsafe way.
- `fixed/`: the same `V001`, and `V002` rewritten with `CREATE INDEX CONCURRENTLY`.
