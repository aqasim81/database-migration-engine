# 60-second demo recording

A script for recording `migrate` catching a dangerous migration with
asciinema. It uses the files in `examples/quickstart/`. Every command below is
typed live; nothing on screen is scripted output.

## Before recording (off camera)

```bash
# 1. Build the binary and put it on PATH for this shell.
make build
export PATH="$PWD/bin:$PATH"

# 2. Start a throwaway database on a fixed port.
docker run -d --rm --name migrate-demo -p 55432:5432 \
  -e POSTGRES_PASSWORD=demo postgres:16-alpine
until docker exec migrate-demo pg_isready -q -h 127.0.0.1 -U postgres; do sleep 0.5; done

# 3. Point migrate at it. The URL stays out of the recording; migrate prints it redacted.
export MIGRATE_DATABASE_URL="postgres://postgres:demo@127.0.0.1:55432/postgres?sslmode=disable"

# 4. Work from the example directory with a clean screen and short prompt.
cd examples/quickstart
export PS1='$ '
clear
```

Terminal: 100×30, dark theme, font at least 16pt.

## Record

```bash
asciinema rec --cols 100 --rows 30 --idle-time-limit 2 \
  --title "migrate: catching a blocking CREATE INDEX" migrate-demo.cast
```

| Time | Type | On screen / what to point out |
|------|------|-------------------------------|
| 0:00 | `cat migrations/V002_index_orders_customer.up.sql` | A one-line `CREATE INDEX`. It looks harmless. |
| 0:05 | `migrate analyze migrations` | `[HIGH] CREATE INDEX without CONCURRENTLY locks the table for writes`, with Table, Rule, SQL and Fix. Pause about 4s. |
| 0:15 | `migrate plan --migrations-dir migrations` | The plan table: `002 … pending HIGH 10s-5min`. The connection line shows the password as `***`. |
| 0:25 | `migrate apply --migrations-dir migrations` | The findings again, then the prompt `Type "yes" to proceed…`. Press **Enter** without typing. It prints `apply aborted: dangerous migrations detected` and nothing is applied. |
| 0:35 | `diff migrations/V002_index_orders_customer.up.sql fixed/V002_index_orders_customer.up.sql` | The fix is one word: `CONCURRENTLY`. |
| 0:40 | `migrate apply --migrations-dir fixed` | `No dangerous operations detected.`, then both migrations `done`. |
| 0:50 | `migrate status --migrations-dir fixed` | Both migrations `applied`, with timestamps. |
| 0:55 | `exit` | Ends the recording. |

If a take runs long, drop the `diff` step. The `apply` pair is the part that
matters.

## After recording

```bash
docker rm -f migrate-demo
asciinema play migrate-demo.cast          # check the take
asciinema upload migrate-demo.cast        # optional
```

Re-takes need a fresh database, because step 0:40 records the migrations as
applied. Run `docker rm -f migrate-demo` and repeat step 2 above.
