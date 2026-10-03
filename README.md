# Product Price Tracker

A local-first Product Price Tracker built with **Next.js/TypeScript, Go, and
PostgreSQL**. It makes recorded prices, their history, and the evidence behind
conditional savings inspectable. It is a portfolio MVP, not a live retailer feed.

### What is implemented

- Catalog creation/editing, Product archiving, Listing tracking controls, global
  search, and dedicated Product/Retailer/Listing views.
- Manual price entry and bounded transactional CSV import; current prices,
  history charts, cross-Listing comparison, and recent observed-price changes.
- Promotion evidence and explicitly derived EffectivePrice scenarios; target,
  percentage-drop, and historical-low alerts with durable in-app events.
- Observation audit/invalidation and CSV export; collection attempt history,
  health overview, manual/bulk controls, and a deterministic Fake provider runtime.
- Goose migrations, PostgreSQL backup/restore instructions and integration tests,
  plus an opt-in local smoke script. See the quick start for verification commands.

### Architecture and data integrity

```text
Browser -> Next.js UI / same-origin API proxy -> Go API -> PostgreSQL
Catalog -> Manual entry / CSV import / configured Fake collection
        -> Immutable PriceObservations -> Current prices + History
                                     -> Promotion scenarios -> Derived EffectivePrice
                                     -> Observed-price alerts + Audit
```

Observed prices are evidence; EffectivePrice is a calculation with eligibility,
stacking, and cashback assumptions. Keeping them separate prevents conditional
savings from looking like guaranteed prices. Ordinary alerts use observed prices,
not EffectivePrice; CSV imports do not generate retroactive alert events.

Money uses integer minor units (USD/JPY), missing values remain distinct from zero,
and currencies are never converted or silently compared. Invalidation annotates
immutable facts; observation freshness is separate from collection attempt time.
Read endpoints are bounded, but no large-scale performance claim is made.

**Current limitation:** no real provider adapter is implemented. Price API remains
**NEEDS CLARIFICATION**, especially for normalized historical retention and public
use; see [provider research](docs/PROVIDER_RESEARCH.md). Manual/CSV observations are
the usable data path. Fake collection is opt-in development infrastructure only.
There is no auth, external alert delivery, automatic product matching, or public
production deployment configuration.

## Quick start: one local setup path (PowerShell 7)

Run commands from the repository root unless a step changes directory. This path
uses manual observations: **no real external provider is implemented or approved**.
Leave `COLLECTOR_CONFIG` empty; automatic external price collection does not work.

1. **Prerequisites:** Git, Go 1.25+, Node.js **24** (includes the TypeScript-stripping
   support used by tests), npm, PowerShell 7+, Docker with Compose v2/Linux containers.
   Start Docker Desktop. Goose installation may download Go 1.26 automatically.
2. **Environment:** copy once, then load in each backend/frontend terminal:

   ```powershell
   if (-not (Test-Path .env)) { Copy-Item .env.example .env }
   Get-Content .env | ForEach-Object {
     if ($_ -match '^([A-Z][A-Z0-9_]*)=(.*)$') {
       [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process')
     }
   }
   ```

   Required explicit backend target: `PGHOST`, `PGDATABASE`, `PGUSER`; missing
   values fail startup with the variable name (never its value). For this local
   password-authenticated database also use `PGPASSWORD` from the example.
   `PGPORT=5432` and `PGSSLMODE=disable` are explicit local settings. The public
   development password is not a production secret. `.env` is ignored; neither
   backend nor Next.js automatically loads the root file. Compose does.

   Optional: `LISTEN_ADDR` defaults to `127.0.0.1:8080`; `BACKEND_URL` defaults to
   `http://127.0.0.1:8080`. Fake-only collection settings and `PRICE_MAX_AGE` are
   optional and documented in `.env.example`. Do not put DB credentials in browser
   configuration. This MVP has no auth; keep it on a trusted/local network.
3. **Start PostgreSQL:**

   ```powershell
   docker compose up -d --wait postgres
   ```

   Data stays in the named volume. Existing-volume credentials do not change when
   `.env` changes. If 5432 is occupied, change `PGPORT` before starting.
4. **Install pinned Goose and migrate:**

   ```powershell
   go install -tags='no_clickhouse,no_libsql,no_mssql,no_mysql,no_sqlite3,no_vertica,no_ydb' github.com/pressly/goose/v3/cmd/goose@v3.28.0
   $goBin = go env GOBIN
   if (-not $goBin) { $goBin = Join-Path (go env GOPATH) 'bin' }
   $env:Path = "$goBin;$env:Path"
   goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" up
   goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" version
   ```

   Expected latest version: **9**. Repeating `up` is safe. No migrations run at
   backend startup. See Database migrations below for status/rollback commands.
5. **Start backend** in the terminal with environment loaded:

   ```powershell
   Set-Location backend
   go run .
   ```

6. **Start frontend** in a separate terminal, loading step 2 from the repo root first:

   ```powershell
   Set-Location frontend
   npm ci
   npm run dev
   ```

   Production-mode verification instead: `npm run build` then `npm start`.
   Supply the same `BACKEND_URL` for build and start; rebuild after changing it.
7. **Verify health/readiness** from another terminal:

   ```powershell
   curl.exe --fail-with-body http://127.0.0.1:8080/healthz
   curl.exe --fail-with-body http://127.0.0.1:8080/readyz
   ```

   Expect `ok` and `ready`. An unreachable DB returns readiness 503 while liveness
   stays 200. Adjust URLs if changing `LISTEN_ADDR`.
8. **Create sample catalog:** open `http://localhost:3000/catalog`; create a Product,
   Retailer, and Listing referencing both. Use an explicit demo name and an
   `https://example.com/` source URL. Nothing is automatically seeded.
9. **Record a manual price:** open that Listing, choose Record price, enter source
   evidence, an RFC3339 time, in-stock, USD, and offer price `19.99`. Repeat at a
   later time with `17.99` for a two-point history. Blank prices remain missing;
   `0` is explicit zero. No real provider is needed.
10. **View:** Home `/`, Product dashboard `/products`, and Listing detail
    `/listings/{id}` show current data. Select ALL history to see both points.
    Optional promotions and observed-price alerts can be created on Listing detail.

### Opt-in smoke test

With the migrated backend running, run from the repo root:

```powershell
pwsh -NoProfile -File scripts/smoke.ps1
# Optional alternative local backend origin:
# pwsh -NoProfile -File scripts/smoke.ps1 -BaseUrl http://127.0.0.1:8081
```

The script checks health/readiness, creates unique `smoke-<GUID>` Product/Retailer/
Listing IDs, records one manual USD observation, then asserts current-price and
history data. It fails on unexpected status/data and does not retry writes.
**Smoke records remain**, including after partial failure; the printed prefix
identifies them. No existing records are updated and no delete API is added.
This verifies the backend flow; open its printed Listing URL to check the UI.
For a no-network script regression check, run
`pwsh -NoProfile -File scripts/smoke.test.ps1` (mock responses, not live verification).

Stop servers with Ctrl+C. `docker compose down` keeps the database volume; do not
add `--volumes` unless deliberately deleting local data.

### Routine verification

From `backend/`: `go test -count=1 ./...` and `go vet ./...`.
From `frontend/`: `npm test`, `npm run lint`, `npm run build`.
With PostgreSQL running, Goose on PATH, and environment loaded, run from `backend/`:
`go test -tags=integration -count=1 ./...`. Tests create/drop their own disposable
DBs and do not delete the regular development database.

## Manual observations and deployment configuration (Task 26)

### Listing tracking (Task 28)

Listings default to tracking enabled. `PATCH /listings/{id}/tracking` with
`{"tracking_enabled": false}` disables provider collection; send `true` to resume.
The endpoint requires an explicit boolean (400 otherwise), returns 404 for a
missing Listing and 200 with Listing ID/state on success. Listing/current-price
responses include `tracking_enabled`. Product detail provides Enable/Disable
tracking and keeps disabled Listings visible with their current data and history.

Scheduled cycles skip disabled Listings; manual Refresh price returns 409
`tracking_disabled`. Enabling does not configure a provider: existing provider
configuration is still required. A collection already in flight may finish its
network call, but the PostgreSQL write guard serializes with the tracking update:
if disabling commits first, the collection cannot append its observation. A write
committed before disabling remains legitimate history. No historical facts,
promotions, alert configurations or triggered events are removed or rewritten.

Explicit **Record price remains allowed while tracking is disabled**, including
normal post-write alert evaluation. Tracking state does not change observation
timestamps, freshness, or existing comparison rules. The dashboard shows disabled
collection separately from failures; re-enabling permits the next configured
cycle or manual refresh. In Go, `Listing.TrackingDisabled` defaults to false and
`TrackingEnabled()` exposes its positive meaning; the database/API use the
positive `tracking_enabled` boolean.

### Product Listing comparison (Task 33)

Product detail compares Listings in sortable cards using observed offer price,
otherwise sale price. Price sorts group currencies alphabetically in both
directions, then sort amounts within each currency; missing prices come last.
Newest-observation sorting retains nanosecond precision, with missing observations
last. Retailer-name sorting is case-insensitive; ties use Listing ID. Sorting
does not change history or discard stale, out-of-stock or tracking-disabled rows.

The best observed price is the existing backend result: all Listings must have
fresh, in-stock observations with comparable prices in one currency. Otherwise
the reason no global best is available is displayed. Tracking and collection
failure remain distinct from observation freshness. Promotion evidence and
user-selected EffectivePrice scenarios stay in separate sections of each card;
conditional savings are never guaranteed and never affect observed-price sorting.

### Product archive state (Task 32)

Product detail supports Archive/Unarchive using
`PATCH /products/{id}/archive` with `{"archived":true}` or `false`.
Existing/new Products default active. The dashboard excludes archived Products
before its row limit; **Show archived** requests `/dashboard?include_archived=true`
and includes them with an Archived label. Search and filters still apply to the
bounded loaded set. Direct Product detail and catalog reads include archived
Products. Metadata edits preserve archive state.

Archiving only changes dashboard visibility. It does **not** disable Listings:
collection remains controlled by each Listing's tracking-enabled state. Listing
identities, tracking states, observations/history, promotions and alerts/events
are retained and remain accessible. Unarchive restores default visibility.

### Catalog metadata editing (Task 31)

Product detail offers **Edit Product metadata**; catalog management offers
**Edit Retailer name**. `PUT /products/{id}` replaces the descriptive fields using
`{"name":"...","brand":"...","model":"..."}`; `PUT /retailers/{id}` takes
`{"name":"..."}`. All listed fields must be explicit strings; empty strings are
allowed, while omitted/null fields are rejected. Partial updates are not supported.
IDs in request bodies and unknown fields are rejected. Success returns the updated
record in `data`; missing records return 404 and invalid bodies return 400.

Listing Product/Retailer references, URL and retailer product ID remain read-only.
Disable the old Listing if needed and create a new one for source identity changes.
Metadata updates do not alter tracking state or historical observations.

### Observation audit CSV export (Task 30)

Use **Export CSV** on a Listing in Product detail, or
`GET /listings/{id}/observations/export`. The download is UTF-8 `text/csv`, named
`listing-observations.csv`, with this ordered header:

```csv
observation_id,observed_at,source,currency,offer_price_minor,sale_price_minor,list_price_minor,msrp_minor,msrp_source,stock,valid,invalidated_at,invalidation_reason
```

This read-only audit export includes **invalidated observations**, unlike normal
price history. Original facts and provenance are unchanged; `valid=false` includes
the invalidation time/reason, while valid rows leave those fields blank. Missing
money/currency remains blank, explicit zero is `0`, and stock-only rows are kept.
Timestamps represent stored instants in UTC RFC3339 with nanosecond precision
where present. Original timestamp text in CSV-import provenance is retained.
Rows sort chronologically by observation timestamp (including nanoseconds), then
by observation ID in bytewise ascending order. Commas, quotes and newlines in
text are CSV-escaped without changing the text. Treat metadata as text when
opening it in spreadsheet software; it is not rewritten as spreadsheet formulas
or escaped with additional apostrophes.

The maximum is **10,000 observations per Listing**, including invalidated rows.
An oversized Listing returns HTTP 422 `export_limit` with no partial CSV. An empty
Listing exports the header alone; a missing Listing returns 404. This audit format
is intentionally distinct from the narrow import format below: it is not a
round-trip restore/import interface and does not export promotions or alerts.

### CSV observation import (Task 29)

Product detail includes **Import CSV observations** for each Listing. Select a
CSV file to import up to 500 observations (1 MiB maximum) atomically. Use this
exact header/order; price cells are integer minor units, not decimal prices:

```csv
observed_at,currency,offer_price_minor,sale_price_minor,list_price_minor,msrp_minor,msrp_source,stock
2026-01-02T03:04:05.123456789+05:30,USD,0,100,200,300,Manufacturer claim,in_stock
2026-01-03T00:00:00Z,,,,,,,unknown
2026-01-04T00:00:00Z,JPY,,1000,,,,out_of_stock
```

`POST /listings/{id}/observations/import` accepts the raw CSV body, with
`Content-Type: text/csv` and `X-Import-ID` (1–100 letters/digits/underscores/hyphens).
Retain this ID and identical file when retrying an uncertain request. IDs are
global; a reused ID returns 409 rather than appending duplicates. Inspect history
before assigning a fresh ID after an uncertain result. The UI retains its ID on
failure until a different file is selected; successful imports show the row count
and refresh current price/history. No automatic retry is performed.

Blank monetary cells are absent; `0` is a real amount. One row's currency applies
to every present amount, preventing mixed currencies within an observation.
USD/JPY may differ between rows without conversion. Stock-only rows may leave
currency blank. Stock must be `unknown`, `in_stock`, or `out_of_stock`; MSRP needs
`msrp_source`. RFC3339 calendar/time validation uses the manual API's backend
parser, without JavaScript timestamp conversion. The existing domain stores UTC
instants with nanosecond precision; CSV provenance also retains the original
timestamp text/offset, import ID and physical starting line of each record.

The server validates the complete bounded input before writing. Validation errors
return 400 with `error.rows` containing CSV line numbers (header is line 1); CSV
syntax/header errors and empty files are rejected. Size errors return 413. All
rows use the existing observation insert helper in one transaction; a failed row
rolls back the import. Success returns 201 with `data.imported` and `data.import_id`.
Missing Listings return 404; unexpected persistence failures return a safe 500.

Imports are explicit manual facts and remain permitted with tracking disabled.
**CSV imports never evaluate historical alerts and create no retroactive events.**
They become normal immutable observations for current/history queries, and can
serve as earlier comparison evidence for future ordinary alert evaluation.
Existing current-price ordering, invalidation and history bounds still apply.

### Single manual observation

`POST /listings/{id}/observations` accepts an immutable manual observation:

```json
{
  "id": "manual-entry-unique-id",
  "observed_at": "2026-09-30T12:00:00Z",
  "source": "Receipt or listing URL and notes",
  "stock": "in_stock",
  "offer_price": { "minor_units": 1999, "currency": "USD" }
}
```

Optional `msrp`, `retailer_list_price`, and `sale_price` use the same money object.
MSRP requires `msrp_source` identifying an explicit manufacturer-price claim.
Omit absent prices (or send null); a supplied money object requires explicit
integer `minor_units` and USD/JPY currency. Zero is valid. All present currencies
must agree; stock-only entries need no currency. Time requires RFC3339 with an
explicit offset. The server prefixes source evidence with `manual: ` and uses
the domain constructor and existing ingestion persistence/alert path. It never
refreshes observation time or provider collection status. Alert evaluation failure
is logged without undoing an accepted observation.

Success returns 201 with `data.id` and `data.observation`; invalid input returns
400, absent Listing 404, reused global observation ID 409, storage failure 500.
Reuse the same ID after an uncertain response; check the observation audit/history
on a conflict. A later independent entry needs a new ID. There is no update/delete.
The UI retains its request ID after failure and reloads the complete detail page
after success, refreshing current price, history, promotions, audit and alerts.
This resets history selection to its default. Backdated entries appear in history;
they do not replace a newer current observation.

`LISTEN_ADDR` defaults to `127.0.0.1:8080`; an explicit value must be host:port
with a numeric port from 1 through 65535. `BACKEND_URL` defaults to
`http://127.0.0.1:8080` and must be an HTTP(S) origin without credentials, path,
query, or fragment. Invalid values fail startup/config loading. PostgreSQL keeps
the existing `PG*` configuration. Database connection remains lazy so `/healthz`
works during an outage; `/readyz` verifies connectivity. No migrations run at startup.

Supply `BACKEND_URL` identically to `npm run build` and `npm start`: Next.js rewrites
are captured at build time while server reads use the process environment. Rebuild
when changing that target. The browser uses the existing same-origin `/api` proxy;
no database credentials or backend secrets belong in browser configuration.
Load the root `.env` in both backend and frontend shells using the commands below.
Empty `COLLECTOR_CONFIG` leaves provider collection disabled; manual entry works.
Existing interval/timeout/max-age settings are listed in `.env.example`.

This MVP has no authentication. Deploy only within a trusted access boundary;
configurable binding does not make its write APIs suitable for unrestricted public
access. Supply deployment secrets through the environment and choose PostgreSQL
TLS settings appropriate to that environment. No cloud platform is required here.

### Explicit manual demo

1. Create `.env` from `.env.example` once, start Docker, and run
   `docker compose up -d --wait postgres` from the repository root.
2. Install the documented Goose version if needed and apply migrations:
   `goose -env .env -dir backend/migrations -timeout 30s postgres "application_name=price_terminal_migrations connect_timeout=5" up`.
3. Load `.env` using the Backend section's PowerShell/Bash command and run
   `go run .` from `backend/`. In another shell load the same environment, then
   run `npm ci` and `npm run dev` from `frontend/`. For production mode use
   `npm run build` then `npm start` instead.
4. Open `http://localhost:3000/catalog`. Create a Product, Retailer, then a
   Listing referencing both with its source URL. Navigate to the Product detail.
5. Open **Record price**. Enter source evidence, stock, USD and an offer such as
   `19.99`; set an earlier observation timestamp with timezone. Repeat with a
   later time and `17.99`, then inspect current price and the ALL history range.
   Test blank prices for stock-only evidence or `0` for explicit free-price evidence.
6. Optionally create an observed-price target alert before recording a qualifying
   observation and inspect its triggered event. Promotion evidence remains a
   separate optional form and does not alter ordinary observed-price alerts.

Nothing is automatically seeded. Use clearly marked demo catalog entries and
manual evidence; this flow does not imply a real provider collected the data.

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
- Node.js 24 and npm for the documented run/test workflow.
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
adds `price_observations`. Migration 4 adds append-only `promotions` evidence.
Migration 5 adds `price_alerts` and `price_alert_events`. Migration 6 adds
`observation_invalidations` and a Product-to-Listing index. Migration 7 adds
`listings.tracking_enabled`, defaulting existing and new Listings to true. Migration 8 adds `products.archived` with default false; migration 9 adds operational `collection_attempts`; the latest version is 9.
Rolling migration 8 back removes only archive state; reapplying defaults Products to active.
Rolling migration 7 back drops only tracking state and returns to version 6 (reapply defaults tracking to enabled). Rolling migration 6 back drops the invalidation table/index and returns to version 5. Do not roll back migration 6 on
real data casually: previously excluded observations would become visible again.
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
Use these read-only checks to inspect metadata and confirm nine application tables (plus Goose metadata):

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

## Observed-price alerts

Apply migration 5 before using alerts or collecting observations with alert evaluation.
On each Product detail Listing, create a target-price, percentage-drop, or
historical-low alert, enable/disable it, and view triggered events. Use Refresh
alerts/events to fetch changes from collection; no browser polling is required.
There is no external notification delivery or user/account system.

All alert APIs return the existing `{ "data": ... }` envelope:

- `POST /listings/{id}/alerts`: create; returns 201.
- `GET /listings/{id}/alerts`: all configured alerts, ordered by ID, at most 100.
- `PATCH /listings/{id}/alerts/{alertID}` with `{"enabled":false}` or
  `{"enabled":true}`: change enabled state; returns 200.
- `GET /listings/{id}/alert-events`: `{events: [...], truncated: boolean}`,
  newest 100 events, ordered by trigger time then alert/observation IDs.

Example POST bodies (identities are assigned by the server):

```json
{"kind":"target","currency":"USD","threshold_minor_units":8000,"require_in_stock":true}
```

```json
{"kind":"drop","currency":"JPY","drop_basis_points":1000,"require_in_stock":false}
```

```json
{"kind":"historical_low","currency":"USD"}
```

Currency is required for every alert. Target amounts use integer minor units;
explicit zero is valid, missing/null thresholds are invalid. Drop thresholds use
1–10000 basis points (0.01–100%). `enabled` and `require_in_stock` default to true.
Other types' numeric parameters must be absent. Invalid input returns 400,
missing Listing/alert 404, the 100-alert-per-Listing limit 409 (including disabled
alerts), and database failures a generic 500. No delete or condition-edit API is
provided; enable/disable preserves event meaning.

The price basis is **observed offer price, otherwise sale price**, as in current
prices/history. No reference prices, coupons, promotions, or EffectivePrice are
used. Each alert compares only its currency and, when requested, in-stock
observations. Target price is inclusive (`price <= threshold`). A drop uses the
immediately previous comparable observation and exact percentage arithmetic;
a historical low is strictly below the minimum of earlier comparable observations.
Earlier means observation timestamp (including nanoseconds), then bytewise result
ID for equal times. The current observation is excluded. Without an earlier
comparable value, drop/low cannot trigger; a zero previous price cannot define a
percentage drop. Stock-only/missing prices do not trigger.

Ingestion evaluates enabled alerts after a successful new observation write, in
a separate transaction. Events preserve observed value/basis, observation identity
and time, trigger time, and previous/minimum comparison value when applicable.
The same alert/observation pair produces at most one event; later qualifying
observations may produce new events even with identical prices. Creation or
re-enabling does not evaluate old observations. Future-dated observations are
excluded; accepted past observations have no implicit age cutoff. Comparisons use
earlier data available when evaluated, with no retroactive event rewriting.

Evaluation failures are logged with observation identity and do not invalidate
the committed observation or collection success. This MVP does not automatically
retry evaluation, so a failure can leave an observation without alert events.
There are no delivery attempts, email, SMS, push, webhooks, or notification queues.

## Frontend

For observation invalidation, operational diagnostics, and Docker backup/restore,
see [Reliability and recovery](docs/RELIABILITY.md).

Open **Manage catalog** (`/catalog`) to create Products, Retailers, and linked
Listings using the existing catalog APIs. Product/Retailer suggestions and browse
lists are bounded to 100 records; exact IDs can also be entered. Listings are
browsed by Product and retain their exact URL and optional retailer product ID.
There are no edit/delete operations.

Product detail separates source observation time from process-local collection
attempt/success times. **Refresh price** sends `POST /listings/{id}/collect`,
which synchronously uses that Listing's existing configured provider and ingestion
path. It returns 200 on success, 404 for a missing Listing, 409 if that Listing is
already collecting, 503 if no provider is configured, or a generic 502 on failure.
Failures retain previous observations and their timestamps. Both manual and
scheduled attempts share a per-Listing guard; overlapping attempts are not queued.
**Reload status** reads fresh server state without collecting.

Creating a Listing does not configure a provider. Collection still uses the
startup `COLLECTOR_CONFIG` Fake fixtures described above; restart the backend after
changing that configuration. A successful Fake collection preserves its fixture's
observation time. Collection status resets on backend restart.

### Promotion evidence and derived scenarios

Apply migration 4 before using promotion features. Evidence is appended with a
unique caller-supplied `id`; retries return 409 rather than updating old evidence.
There is no update/delete API or automatic promotion discovery. A new evidence
snapshot gets a new ID. Records retain Listing FK, provenance, nanosecond timestamps,
optional validity, kind/value, eligibility requirement, stacking state, and terms.
Old snapshots are retained; do not select two snapshots of the same offer as two
different discounts. No automatic offer matching or supersession is inferred.

`POST /listings/{id}/promotions` accepts this body (example evidence only):

```json
{
  "id": "coupon-evidence-1",
  "source": "https://example.com/official-offer-terms",
  "observed_at": "2026-09-29T12:00:00Z",
  "starts_at": "2026-09-29T00:00:00Z",
  "ends_at": "2026-10-01T00:00:00Z",
  "kind": "fixed",
  "amount": {"minor_units": 500, "currency": "USD"},
  "requirement": "other",
  "stacking": "unknown",
  "terms": "Apply coupon SAVE5; confirm the source eligibility terms."
}
```

Kinds are `fixed` (coupon/instant amount), `percentage`, `cashback` (fixed rebate),
and `membership` (fixed immediate amount with `requirement: membership`). Monetary
kinds require `amount`; percentages require `basis_points` from 1 to 10000 and
omit `amount` (1250 means 12.50%). Terms must preserve coupon activation and any
conditions. Requirements are `none`, `membership`, `other`, or `unknown`; stacking
is `allowed`, `disallowed`, or `unknown`. Only record `none`/`allowed` when evidence
supports them. Missing or invalid states are rejected, never silently defaulted.

`GET /listings/{id}/promotions` returns up to 100 newest relevant evidence snapshots,
with a `truncated` flag. Relevant means observed by request time and not explicitly
expired; future-start offers remain visible. Start is inclusive; end is exclusive.
Unknown bounds are preserved. No deduced "best promotion" or stacking occurs.

An explicit scenario uses:

```text
GET /listings/{id}/effective-price?scenario=selected&promotion_id=coupon-evidence-1&member=unknown&eligible=yes
```

Repeat `promotion_id` to select a second record. At most one immediate discount
and one cashback may be used; two immediate discounts or two cashbacks are unsupported.
Both records must explicitly allow stacking when two are selected. `member` and
`eligible` are caller assumptions (`yes`, `no`, `unknown`; omitted means unknown),
not stored membership profiles. `eligible` confirms all other recorded conditions,
including coupon activation. Unknown source requirements remain conditional even
if the caller says yes. Membership also needs a yes membership assumption.

The base is the latest coherent observed offer price, otherwise sale price. No
reference-price fallback occurs. Percentage discount = base × basis points / 10000,
rounded to the nearest minor unit with halves up using overflow-safe integer math.
Discounts above the base, cashback above payable, currency mismatches, mismatched
Listings, duplicate selections, future evidence, expired/not-yet-active offers,
and unsupported combinations are unavailable. Valid discounts may yield zero.

Results contain `status`, `reason`, rule version, full source observation, selected
evidence, scenario assumptions, calculation time, base, immediate discount, immediate
payable, potential cashback, potential net, and exclusions. Conditional/unavailable
results withhold derived totals. Missing conditions do not become a confident estimate.
HTTP 200 carries available/conditional/unavailable evaluations; malformed scenarios
return 400, missing Listing/evidence 404, and internal failures generic 500 responses.
Evidence creation returns 201, duplicate IDs 409, and invalid evidence 400.

Cashback never reduces immediate payable. Tax/shipping are excluded. Selected
discounts are assumed additional to the observed price; no live verification or
freshness refresh occurs, and recorded validity bounds are not proof an offer is
still available. The UI shows the observation time/stock and clearly labels derived
scenario estimates and potential cashback as not guaranteed. Existing observed
price, comparison, and history APIs remain unchanged.

### Catalog and history

The Product dashboard at `/products` displays up to 20 products by ID and their current comparable
prices. Product detail pages show Listings, source links, stock, freshness, and
selectable history. Existing catalog APIs supply the data; catalog create/edit UI is available at `/catalog`. Keep the Go backend running with migrations applied.

Set `BACKEND_URL` in the frontend process environment to override
`http://127.0.0.1:8080` (no trailing slash). Restart/rebuild Next.js after changing
it. Server-rendered pages fetch the backend directly; browser history requests
use Next.js `/api` forwarding, so no backend CORS changes are needed.

`GET /listings/{id}/history?range=1M` defaults to `1M`. Supported ranges are `1D`,
`1W`, `1M`, `3M`, `1Y`, and `ALL`: rolling 1/7/30/90/365 days ending at request
time, not calendar months/years. Bounds are inclusive, with nanosecond precision;
future observations are excluded. The newest 1,000 matching observations are
returned chronologically, breaking timestamp ties by bytewise result ID. A
`truncated` flag signals older omitted rows; use a narrower range. No schema change,
aggregation, downsampling, or interpolation is performed.

Historical metadata uses offer price, otherwise sale price, including explicit zero.
The low covers priced observations in the selected interval, regardless of historical
stock. Change uses the first and last actual observations, not the nearest available
prices or interpolated range endpoints. It requires two distinct timestamps, priced
endpoints, a nonzero starting amount, and one comparable currency throughout the
priced history. Change is an exact rational percentage rounded to two decimals
(half away from zero), returned as a string. Mixed currencies or a truncated window
suppress both summaries; missing endpoints suppress change but can still allow a low.
These historical statistics do not claim present availability or an effective price.

The SVG history chart groups currencies separately. Only adjacent priced observations
with the same price basis and at most a 15-minute gap are connected as visual guides.
Missing prices and longer gaps break lines. No synthetic observations are added;
individual marks and the expandable observation list preserve actual evidence.
Money formatting uses integer/BigInt arithmetic for USD and JPY. API amounts outside
JavaScript's exact safe-integer range are rejected visibly rather than rounded.
No chart, state-management, or UI dependency was added.

With Node.js 24, run frontend parser/formatting/history tests using `npm test` from
`frontend/`. Loading, empty, and retryable error states are provided by the pages
and history view.

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


### Collection attempt audit

Listing detail reads `GET /listings/{id}/collection-attempts`: the newest 50
completed attempts ordered by start time descending, then attempt ID descending.
Scheduled and manual Refresh price collection use the same recorder. Successful
records reference the immutable observation; alert-evaluation failure does not
change collection success. Failure summaries are fixed codes only.

Disabled/unconfigured/busy requests, preflight lookup failures, and cancellation
before ingestion starts are rejected without an attempt row. A tracking-disable
race after ingestion starts records `unavailable`. Started cancellations/timeouts
record `cancelled`; provider/normalization and persistence failures remain distinct.
Manual observation entry and CSV import are not collection attempts.

Attempt insertion is best-effort after ingestion, using a separate two-second
context even on cancellation. Failure emits `collection_attempt_record_failed`
without raw errors and never rolls back a valid observation or causes a retry.
A process crash before recording, or an unavailable database, can therefore leave
a gap in this operational audit. There are no retries or durable in-flight records.
Process-local latest status still resets on restart; recorded attempt history does
not. Neither attempt time nor outcome changes observation freshness.


### Bulk manual collection

Collection health supports selecting up to 20 Listings and Refresh selected.
`POST /collection/refresh` accepts `{"listing_ids":["listing-a","listing-b"]}`.
The raw array must contain 1–20 nonblank IDs; duplicates are collected once in
first-occurrence order. A valid batch returns HTTP 200 with per-Listing success,
failure, unavailable, or cancelled outcomes and safe codes; successes include
an observation ID. Disabled/unconfigured/busy Listings return unavailable.

Work is sequential through the existing manual collection runtime, including
per-Listing deadlines and manual CollectionAttempt records. Cancellation stops
unstarted work; committed observations survive other failures. Rejected requests
and unstarted work create no attempt records. There are no automatic retries.
A disconnected client may not receive results even though some writes succeeded;
reload health and inspect attempt history before explicitly retrying. Selections
persist across filters; only selected IDs are sent. Health reloads after a batch.
