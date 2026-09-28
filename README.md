# Product Price Tracker

A general-purpose tracker for retailer listing prices, price history, promotions,
and alerts. This repository currently contains a Go backend with liveness and
database readiness checks, local PostgreSQL, and a minimal Next.js home page.
Tracking features are planned; no application schema or tables exist yet.

## Repository layout

```text
backend/            Go HTTP server, PostgreSQL connection setup, and tests
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
database. No migrations or application tables are created.

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
