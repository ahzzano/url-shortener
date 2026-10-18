# URL shortener

## Developer-run migrations

All migrations are run manually by developers before starting the API. Neither
Docker images nor Compose install Goose or execute migrations automatically.

For a new checkout, copy `.env.example` to `.env` and adjust the development
settings. Do not overwrite an existing `.env`. Install PostgreSQL's `psql` client
and Goose locally; ensure the Go binary installation directory is on `PATH`.

```sh
go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
docker compose up -d --wait postgresql
```

From the repository root, export your configured `.env` values and apply the
migrations (POSIX shell):

```sh
set -a
. ./.env
set +a
psql "$GOOSE_DBSTRING" --set ON_ERROR_STOP=1 \
    --command 'CREATE SCHEMA IF NOT EXISTS custom;' &&
goose up
```

Only after migrations succeed, start the API with Docker Compose:

```sh
docker compose up --build
```

Alternatively, run `go run .` in the shell where the environment was exported.
The application does not load `.env` itself. `.env.example` supplies host URLs
using `localhost:5432`; Compose uses its internal PostgreSQL hostname instead.

Before later schema changes, stop the API, repeat the migration commands, and
start it again only after they succeed. Compose checks PostgreSQL availability,
not schema readiness; starting or restarting containers never runs migrations.

## Existing databases

The local Goose configuration uses `custom.goose_migrations` for migration
history. Before adopting it for an existing database, check that this table
records its applied migrations. If `urls` already exists without matching
history, or history lives in a different table, reconcile that history first.
Do not change tracking tables, automatically baseline existing data, run `goose
down`, or delete database volumes to work around a migration failure.
