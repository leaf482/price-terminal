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
adds `price_observations`. Migration 4 adds append-only `promotions` evidence.
Migration 5 adds `price_alerts` and `price_alert_events`; the latest version is 5.
Rolling it back drops only alerts/events and returns to version 4, retaining
promotions, observations, and catalog data.
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
Use these read-only checks to inspect metadata and confirm seven application tables:

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

The catalog at `/` displays up to 20 products by ID and their current comparable
prices. Product detail pages show Listings, source links, stock, freshness, and
selectable history. Existing catalog APIs supply the data; no create/edit UI is
included. Keep the Go backend running with migrations applied.

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
