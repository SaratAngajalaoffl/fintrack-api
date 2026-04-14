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
# Set DATABASE_URL, JWT_SECRET (≥16 chars; Go signs session cookies — Next does not need this)
go run ./cmd/api
```

- **Health:** `GET http://localhost:8080/health`
- **Auth:** **`/api/auth/*`** (session cookie `fintrack_session`, bcrypt passwords, JWT OTP tickets).
- **Bank accounts, credit cards, expense categories:** CRUD-style **`/api/...`** routes (see **`web/src/configs/api-routes.ts`**).
- **Fund buckets:** list/create plus **`/allocate`**, **`/unlock`**, **`/priority`** actions.
- Migrations apply automatically on startup unless **`SKIP_MIGRATIONS=true`** or **`1`**.

**With Next.js on your machine:** run the API on **:8080** and set **`web/.env.local`** from **`web/.env.example`** (**`NEXT_PUBLIC_API_ORIGIN`**, optional **`API_ORIGIN`** for server/middleware). Next validates sessions by calling **`GET /api/auth/me`** on this service (no JWT secret in Next). **`getApiRoute()`** in **`web/src/configs/api-routes.ts`** builds URLs for browser and server-side calls. This API sends **CORS** headers for allowed origins (see **`CORS_ALLOWED_ORIGINS`** in **`api/.env.example`**) when the browser calls the Go origin directly.

## Docker / Compose

The API image is defined in **`api/deploy/Dockerfile`**. **Build context:** **`api/`** (module root). Example: `docker build -f deploy/Dockerfile .` from **`api/`**. **`api/.dockerignore`** trims the context. Compose stacks live at the repo root in **`deploy/`** (e.g. **`deploy/docker-compose.dev.yml`**). In Compose, **`DATABASE_URL`** targets the `postgres` service; SQL files are copied from the builder stage to **`/migrations`** in the runtime image, and dev Compose can bind-mount **`api/migrations`** there for local edits without rebuilding.

## Makefile

See **`Makefile`** (`make run`, `make build`, `make test`).

## Seeding

Database seeding is **not** run by this service. Use your own scripts (for example the JSON-driven flows documented next to **`data/`** at the repo root).
