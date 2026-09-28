# Product Price Tracker

A general-purpose tracker for retailer listing prices, price history, promotions,
and alerts. This repository currently contains only the application foundation:
a Go health endpoint and a minimal Next.js home page. Tracking features are planned.

## Repository layout

```text
backend/   Go HTTP server and tests (standard library only)
frontend/  Next.js App Router application with TypeScript and ESLint
docs/      Project plan, architecture, domain model, workflow, and task roadmap
```

See [the project plan](docs/PROJECT_PLAN.md) and [task roadmap](docs/TASKS.md).

## Prerequisites

- Go 1.25 or newer.
- Node.js 20.9 or newer and npm. Node.js 24 is used for local verification.

The backend and frontend run independently. No database or external services are
needed for this bootstrap.

## Backend

From the repository root:

```sh
cd backend
go run .
```

The server listens on `http://127.0.0.1:8080`. In another terminal, request
`GET /healthz`:

```sh
curl http://127.0.0.1:8080/healthz
```

Expected response: HTTP 200 with the plain-text body `ok` followed by a newline.
Stop the server with Ctrl+C.

Run tests from `backend/`:

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
