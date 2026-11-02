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

**Postgres and SQLite run the same migrations.** `internal/app/migrations/` is
embedded into the binary (`//go:embed all:migrations` in
`internal/app/migrate.go`) and applied automatically at startup:

| Provider | What happens at startup |
|---|---|
| Postgres | `setupHostedDB` applies the embedded migrations |
| SQLite | `setupLocalDB` applies the same embedded migrations |
| MongoDB | Skipped entirely — `Init` returns early. Mongo is schemaless; `ensureIndexes` in `internal/usecase/nosqldb/mongo` creates the unique constraints instead. |

So a new migration file affects **both** SQL backends. Write SQL that works on
Postgres and SQLite, and test on both.

### Writing a migration

Add a matching pair of files to `internal/app/migrations/`, following the
naming of the files already there: one `.up.sql` (applies the change) and one
`.down.sql` (undoes it). **Fill in both.** A migration without a working
`.down.sql` cannot be rolled back.

You do not need any tooling to do this — the files are plain SQL, and the app
applies them on next start.

### The make migrate-* targets

These drive the `golang-migrate` CLI against a **running Postgres** instance.
They are for operating on a live database, not for normal development.

```sh
make bin-deps          # one time: installs golang-migrate and mockgen into ./bin/
make migrate-up        # apply pending migrations to $DB_URL
make migrate-create    # scaffold an empty migration pair
```

`make bin-deps` installs `golang-migrate` with `-tags 'postgres'`, so the CLI
only speaks Postgres. `make migrate-up` needs `DB_URL` set.

> **Known bug:** both targets pass `-dir /internal/app/migrations` and
> `-path /internal/app/migrations` — an **absolute** path, which resolves to
> the filesystem root, not to this repository. As written, `make migrate-create`
> will not put files in `internal/app/migrations/`. Until that is fixed, add
> migration files by hand as described above.

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
