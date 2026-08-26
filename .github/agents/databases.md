# Postgres and MongoDB development

Read this only when you need to test against Postgres or MongoDB, or when you
are writing a SQL migration. **You do not need this file for normal work** —
SQLite is the default and needs no setup at all.

## Default: SQLite, zero setup

`DB_PROVIDER=sqlite` (the default) creates the database file automatically at:

| OS | Path |
|---|---|
| Linux, macOS | `~/.config/device-management-toolkit/console.db` |
| Windows | `%APPDATA%\device-management-toolkit\console.db` |

If you want a clean database, delete that file and restart the app.

## Running Postgres or MongoDB

```sh
docker compose up -d postgres                # start Postgres
docker compose --profile mongo up -d mongo   # start MongoDB
docker compose down --remove-orphans         # stop everything
```

MongoDB sits behind a Compose profile, so plain `docker compose up -d` will
not start it. You must pass `--profile mongo`.

Then set `DB_PROVIDER` (and `DB_URL` for Postgres) in your `.env`.

## Migrations

Migrations apply to **Postgres only**:

- **SQLite** runs migrations automatically at startup.
- **MongoDB** does not use migrations at all — it has no fixed schema.

```sh
make bin-deps          # one time: installs golang-migrate and mockgen into ./bin/
make migrate-create    # create a new empty migration pair (up + down)
make migrate-up        # apply all pending migrations
```

`make migrate-create` writes two files under `internal/app/migrations/`: one
`.up.sql` (applies the change) and one `.down.sql` (undoes it). **Fill in
both.** A migration without a working `.down.sql` cannot be rolled back.

## Adding a repository method

A new method on a `Repository` interface must work on all three databases.
Implement it in **both** packages:

1. `internal/usecase/sqldb/<feature>.go` — serves both Postgres and SQLite.
2. `internal/usecase/nosqldb/mongo/<feature>.go` — serves MongoDB.
3. Add a Postgres migration under `internal/app/migrations/` if the change
   needs a new column or table.
4. Run `make mock`.

Both implementations must return the same results for the same input. The use
case cannot tell which database is running, so a method that only works on one
backend is a bug.
