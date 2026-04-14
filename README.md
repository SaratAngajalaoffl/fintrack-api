# Fintrack API (Go)

HTTP API for Fintrack. The Next.js app lives in **`../web/`**. SQL migrations live under **`migrations/`** in this module (**`api/migrations/`**).

## Layout

- **`cmd/api/`** — `main.go` entrypoint: config, DB pool, **run migrations**, HTTP server.
- **`internal/`** — app code not imported by other modules: `config`, `handler`, `middleware`, `migrate`, `model`, `repository`, `service`.
- **`pkg/`** — small reusable packages (`logger`, `validator`, …).

## Run locally

From **`api/`**:

```bash
cp .env.example .env
# Set DATABASE_URL, JWT_SECRET (≥16 chars; same value as web for session cookies)
go run ./cmd/api
```

- **Health:** `GET http://localhost:8080/health`
- **Auth:** **`/api/auth/*`** (session cookie `fintrack_session`, bcrypt passwords, JWT OTP tickets).
- **Bank accounts, credit cards, expense categories:** CRUD-style **`/api/...`** routes (see **`web/src/configs/api-routes.ts`**).
- **Fund buckets:** list/create plus **`/allocate`**, **`/unlock`**, **`/priority`** actions.
- Migrations apply automatically on startup unless **`SKIP_MIGRATIONS=true`** or **`1`**.

**With Next.js on your machine:** run the API on **:8080** and align **`web/.env.local`** with **`web/.env.example`** (**`JWT_SECRET`**, optional **`NEXT_PUBLIC_API_ORIGIN`** / **`API_ORIGIN`**). **`getApiRoute()`** in **`web/src/configs/api-routes.ts`** builds URLs for both browser and server-side calls. This API sends **CORS** headers for allowed origins (see **`CORS_ALLOWED_ORIGINS`** in **`api/.env.example`**) when the browser calls the Go origin directly.

## Docker / Compose

In Compose, **`DATABASE_URL`** targets the `postgres` service; migrations are baked into the API image at **`/migrations`**, and dev Compose can bind-mount **`api/migrations`** there for local edits without rebuilding.

## Makefile

See **`Makefile`** (`make run`, `make build`, `make test`).

## Seeding

Database seeding is **not** run by this service. Use your own scripts (for example the JSON-driven flows documented next to **`data/`** at the repo root).
