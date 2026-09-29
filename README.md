# Product Price Tracker

A general-purpose tracker for retailer listing prices, price history, promotions,
and alerts. This repository currently contains a Go backend with liveness and
database readiness checks, local PostgreSQL, and a minimal Next.js home page.
Catalog APIs, immutable observation persistence, an opt-in Fake collection runtime,
and current-price APIs are available. No real retailer is collected yet.

## Repository layout

```text
backend/            Go HTTP server, PostgreSQL connection setup, and tests
backend/migrations/ Versioned SQL migrations managed by Goose
frontend/           Next.js App Router application with TypeScript and ESLint
docs/               Project plan, architecture, domain model, workflow, and tasks
docker-compose.yml  Local PostgreSQL only, with a persistent named volume
.env.example        Public local-development configuration examples
```

See [the project plan](docs/PROJECT_PLAN.md) and [task roadmap](docs/TASKS.md).

## Prerequisites

- Go 1.25 or newer.
- Node.js 20.9 or newer and npm. Node.js 24 is used for local verification.
- Docker with Compose v2 or newer (Docker Desktop with Linux containers on Windows).

The frontend runs independently. PostgreSQL is required for backend readiness,
but not for liveness or ordinary unit tests.

## Local PostgreSQL

Start Docker Desktop/the Docker engine first. From the repository root, create
local configuration once (do not overwrite an existing `.env`):

```powershell
Copy-Item .env.example .env
```

On macOS/Linux, use `cp .env.example .env` instead. `.env` is ignored by Git.
The example password is a public development-only value, not a real secret.

The configuration uses `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`,
and `PGSSLMODE`. Compose reads the root `.env` automatically and maps the
database/user/password to the image's `POSTGRES_*` initialization variables.
The backend reads the same `PG*` variables from its process environment; it does
not automatically read `.env`. Shell environment variables override Compose's
`.env` values, so use the same configuration in both terminals.

Keep `PGHOST=127.0.0.1` for the host-run backend. If port 5432 is occupied,
change `PGPORT` in `.env` before starting. `PGSSLMODE=disable` is for this
local-only environment. The database port is bound to localhost, not the LAN.

Start PostgreSQL and wait for its health check:

```sh
docker compose up -d --wait postgres
```

The PostgreSQL 18.6 image stores data in the Compose-managed `postgres_data`
named volume mounted at `/var/lib/postgresql`. Initialization settings apply only
to an empty volume; changing `.env` does not change credentials in an existing
database. Container startup does not run migrations; apply them explicitly below.

## Backend

From the repository root, load the plain `KEY=value` configuration into the
current shell and run the backend. PowerShell:

```powershell
Get-Content .env | ForEach-Object {
  if ($_ -match '^([A-Z][A-Z0-9_]*)=(.*)$') {
    [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process')
  }
}
Set-Location backend
go run .
```

Bash (macOS/Linux):

```sh
set -a
. ./.env
set +a
cd backend
go run .
```

The server listens on `http://127.0.0.1:8080`. Database connections are opened
lazily, so an unavailable database does not prevent HTTP startup. Invalid
connection configuration prevents startup with a generic error.

In another terminal, check liveness and readiness (use `curl.exe` in Windows
PowerShell if `curl` is an alias):

```sh
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz
```

- `/healthz`: HTTP 200, `ok` followed by a newline, independently of DB state.
- `/readyz`: pings PostgreSQL with the request context and a two-second timeout;
  HTTP 200 with `ready` when connected, otherwise HTTP 503 with `not ready`.
  Responses never include credentials or raw driver errors.

To verify a database outage, keep the backend running and, from the repository
root in another terminal, run:

```sh
docker compose stop postgres
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz
```

Expect liveness 200 and readiness 503. Restart PostgreSQL with
`docker compose up -d --wait postgres`; readiness returns to 200 without
restarting the backend. Stop the backend with Ctrl+C.

To stop and remove the local container/network while keeping the database volume:

```sh
docker compose down
```

Do not add `--volumes` unless you intend to delete the local database data.

Run deterministic unit tests from `backend/`; these do not require PostgreSQL
or loaded environment configuration:

```sh
go test ./...
```

## Database migrations

Use the standalone [Goose CLI](https://github.com/pressly/goose), pinned to
**v3.28.0**. It supplies SQL migration ordering, transactional execution, version
metadata, status, and rollback without an ORM or a custom migration engine.
Migration tooling is separate from the backend runtime: `backend/go.mod` and
`backend/go.sum` do not gain migration dependencies.

Install once, with unrelated database drivers excluded:

```sh
go install -tags='no_clickhouse,no_libsql,no_mssql,no_mysql,no_sqlite3,no_vertica,no_ydb' github.com/pressly/goose/v3/cmd/goose@v3.28.0
```

Building this CLI requires Go 1.26 or newer. With Go's default automatic
toolchain selection, the install command downloads a compatible toolchain if
needed; it does not change the backend's Go 1.25 requirement. If automatic
toolchain downloads are disabled, use Go 1.26+ for this installation.

Ensure the Go binary directory (`GOBIN`, or `go env GOPATH` plus `/bin` when
`GOBIN` is unset) is on `PATH`. For the default installation location, add it
to the current shell:

```powershell
$env:Path = "$(go env GOPATH)\bin;$env:Path"
```

Bash equivalent: `export PATH="$(go env GOPATH)/bin:$PATH"`.
Confirm the installed version with `goose -version` (expected v3.28.0).

### Apply, inspect, rollback, and reapply

Run these commands from the repository root with PostgreSQL running and `.env`
created as described above:

```sh
goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" up
goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" status
goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" version
```

Goose loads `.env` directly. Its PostgreSQL driver uses the same `PGHOST`,
`PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, and `PGSSLMODE` configuration as
the backend. Already-exported variables take precedence over `.env`. The quoted
argument adds only an application label and a five-second connection timeout;
it contains no credentials. No second credentials file or database URL is needed.

`up` applies pending migrations in order. Repeating it at the latest version is
a safe no-op. To roll back the latest migration and reapply it:

```sh
goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" down
goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" up
```

The initial file, `backend/migrations/00001_migration_bootstrap.sql`, contains
`-- +goose Up` and `-- +goose Down` sections that each execute `SELECT 1;`.
Its purpose is to prove the workflow. It creates no application objects.
Goose owns `public.goose_db_version` and its supporting primary-key index and
sequence. Migration 2 adds `products`, `retailers`, and `listings`. Migration 3
adds `price_observations`; the latest version is 3. Rolling it back drops only
the observation table and returns to version 2.
Migration SQL and its version update run in one transaction by
default. Do not edit applied migration files; add a new numbered file instead.

### Fresh disposable database

The following commands create a separate empty database on the same local
PostgreSQL instance. They do not reset the normal development database or volume.
Run from the repository root. Creation intentionally fails if the database name
already exists; choose a new disposable name rather than dropping unknown data.

```sh
docker compose up -d --wait postgres
docker compose exec -T postgres sh -c 'createdb --username "$POSTGRES_USER" --template template0 price_terminal_migration_check'
goose -env .env -dir backend/migrations -timeout 30s postgres "dbname=price_terminal_migration_check application_name=price_terminal_migrations connect_timeout=5" up
goose -env .env -dir backend/migrations -timeout 30s postgres "dbname=price_terminal_migration_check application_name=price_terminal_migrations connect_timeout=5" status
goose -env .env -dir backend/migrations -timeout 30s postgres "dbname=price_terminal_migration_check application_name=price_terminal_migrations connect_timeout=5" version
```

Only the target database name is overridden; host, port, user, password, and TLS
still come from the existing configuration. Repeat the same disposable-target
command with `up`, `down`, then `up` to check no-op, rollback, and reapply.
Use these read-only checks to inspect metadata and confirm four application tables:

```sh
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d price_terminal_migration_check -c "TABLE public.goose_db_version;"'
echo "SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND NOT (schemaname = 'public' AND tablename = 'goose_db_version');" | docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d price_terminal_migration_check -At'
```

After verifying a database that you created specifically for this check, remove
only that disposable database:

```sh
docker compose exec -T postgres sh -c 'dropdb --username "$POSTGRES_USER" price_terminal_migration_check'
```

Ordinary `go test ./...` in `backend/` does not run migrations and still works
with PostgreSQL stopped. Migrations are never run automatically at HTTP startup.

### Persistence integration tests

Install the pinned Goose CLI above and put it on `PATH`. Start PostgreSQL with
`docker compose up -d --wait postgres`. Load the existing `.env` into the shell
using the PowerShell or Bash commands in the Backend section (omit `go run .`).
From `backend/`, run:

```sh
go test -tags=integration -count=1 -v ./...
```

The `integration` build tag explicitly enables these tests. They require a
reachable PostgreSQL instance and a role with `CREATEDB` permission (provided by
the local Compose setup); missing prerequisites fail the integration run.
Each run creates a randomly named `price_terminal_test_*` database from
`template0`, applies the real Goose migrations, checks repeat-up/rollback/reapply,
and drops only that newly created database during cleanup. The regular
`PGDATABASE` is used only for the administrative connection; its tables and data
are not changed. An interrupted process may leave its disposable database behind.
Ordinary `go test ./...` excludes these tests and needs neither PostgreSQL nor Goose.

Listing uniqueness uses the exact `(retailer_id, url, retailer_product_id)` tuple.
An absent retailer product ID is stored as an empty string, so repeated source
identities without an ID are also rejected. Product ID is excluded to prevent
attaching the same source identity to a second product. Different variant IDs on
one URL remain distinct; URL aliases and identifiers across different URLs are
not treated as matches. No URL normalization or automatic product matching occurs.

Observation writes require a caller-supplied, globally unique collected-result ID.
Reuse it for retries; allocate a new ID for each independent collection, even if
prices are unchanged. A repeated ID returns a wrapped PostgreSQL uniqueness error
(SQLSTATE `23505`); no observation is overwritten, including if the retry payload
differs. Observations support insertion, history, and current reads; no update or
delete API exists.

History reads return one listing's observations oldest first, breaking exact-time
ties by result ID using bytewise `C` collation. Timestamps retain UTC instants and
full Go precision: `timestamptz` stores microseconds and a small remainder column
preserves sub-microsecond nanoseconds. Prices use nullable integer minor units
and one shared currency derived from present Money values. Stock-only rows have
NULL currency; NULL prices remain distinct from explicit zero. General source
and MSRP-specific evidence are stored separately.

## Collection runtime and current prices

Apply migrations first. The HTTP backend owns one collector instance. Only Listing
IDs explicitly configured in `COLLECTOR_CONFIG` are active; absent configuration
means no collection. Configuration is loaded once at startup, at most 100 Listings
and 1 MiB of JSON. All referenced Listings must already exist through the catalog
APIs. A missing Listing fails its attempt without stopping others.

For a local Fake demonstration, start the backend normally, then create this
catalog using PowerShell in another terminal (use new IDs if these already exist):

```powershell
$base = 'http://127.0.0.1:8080'
Invoke-RestMethod "$base/products" -Method Post -ContentType application/json -Body '{"id":"demo-product","name":"Demo product"}'
Invoke-RestMethod "$base/retailers" -Method Post -ContentType application/json -Body '{"id":"demo-retailer","name":"Fake retailer"}'
Invoke-RestMethod "$base/listings" -Method Post -ContentType application/json -Body '{"id":"demo-listing","product_id":"demo-product","retailer_id":"demo-retailer","url":"https://example.com/demo"}'
Invoke-RestMethod "$base/listings" -Method Post -ContentType application/json -Body '{"id":"demo-failing","product_id":"demo-product","retailer_id":"demo-retailer","url":"https://example.com/failing"}'
```

Stop the backend with Ctrl+C. In its terminal, with the same database environment
loaded and working directory `backend/`, enable the example and restart:

```powershell
$env:COLLECTOR_CONFIG = 'collection.example.json'
$env:COLLECTION_INTERVAL = '1m'
$env:COLLECTION_TIMEOUT = '10s'
$env:PRICE_MAX_AGE = '15m'
go run .
```

Those durations are the defaults and must be positive Go duration strings. Bash
can set the same variables with `export NAME=value`. The example uses Fake only:
one success and one failure. Its fixed source timestamp is intentionally preserved
on every cycle, so it can be stale; a successful collection does not make old
source evidence fresh. Edit the fixture timestamp explicitly for a fresh demo.
Amounts are integer minor units (USD cents, JPY yen); omit missing amounts and
omit currency for stock-only fixtures. MSRP requires `msrp_source` evidence.
Configuration errors fail startup; no network provider or credentials are used.

The first cycle runs immediately. Each subsequent cycle starts one interval after
the previous cycle completes. Existing batch ingestion processes Listings
sequentially with per-Listing deadlines and failure isolation. Every independent
collection gets a new result ID; there are no retries or overlapping cycles.
Run only one backend collector process. Ctrl+C/SIGTERM cancels collection, drains
HTTP requests, joins the collector, then closes the database.

Collection status is **in memory** and resets at restart: `never_attempted`,
`collecting`, `success`, or `failed` for active Listings; otherwise `inactive`.
It includes last attempt and last successful persistence times when available.
Failures use the safe code `collection_failed`, preserve the last success, and
never create a PriceObservation. Each Listing's outcome is published when its
ingestion completes, before the next Listing starts; later slow or failing Listings
do not delay its success timestamp. Collection timestamps describe runtime activity,
not the source observation time.
No schema change or durable collection-run history is introduced.

Read the two JSON endpoints:

```sh
curl http://127.0.0.1:8080/listings/demo-listing/price
curl http://127.0.0.1:8080/products/demo-product/prices
```

Both use the existing `data` / `error` envelopes. Missing Listing/Product returns
404; an existing Listing without history returns 200 with `observation: null`.
Each result contains Listing/retailer identity, exact URL, one complete observation,
and collection status. Optional price amounts and currency are omitted when absent;
explicit zero remains zero. Source, MSRP evidence, stock, and original observation
timestamp are preserved. Freshness is `missing`, `fresh`, `stale`, or `future`,
using `PRICE_MAX_AGE`; future timestamps are not considered fresh.

Latest means greatest observed timestamp (including nanoseconds), breaking exact
ties by greatest collected-result ID in bytewise `C` order. Insertion time does not
win. A latest stock-only observation does not inherit any older prices.

Product reads include all its Listings, even inactive ones, ordered by bytewise ID.
The bound is 100; larger products return 422 `too_many_listings` rather than a
misleading partial comparison. The query selects coherent rows in one SQL statement.
An empty product returns an empty array and no best price.

The deliberately conservative comparison uses **offer price, otherwise sale price**;
MSRP and retailer list price are reference values, never fallbacks. `best_price`
is present only when every Listing has a fresh, in-stock observation with that
price basis and all currencies match. Otherwise it is null and `comparison_status`
explains why (for example `missing_observation`, `not_fresh`, `not_in_stock`,
`missing_price`, or `incompatible_currencies`). Equal amounts choose the smallest
Listing ID. The best result identifies the retailer, Listing, currency, amount,
and chosen basis. This compares observed item prices only: no shipping, tax,
coupons, promotions, currency conversion, or EffectivePrice calculation. The
example's failed Listing intentionally prevents a product-wide best-price claim.

## Frontend

From the repository root, install the locked dependencies and start development:

```sh
cd frontend
npm ci
npm run dev
```

Open `http://localhost:3000`. Stop the development server with Ctrl+C.

Run lint and a production build from `frontend/`:

```sh
npm run lint
npm run build
```

To serve the completed production build:

```sh
npm start
```

Next.js generates `next-env.d.ts` and `.next/` during development/build; these and
installed dependencies are ignored by Git. Keep `package-lock.json` with the
project so `npm ci` installs the same dependency versions.
