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

- **`api/deploy/Dockerfile.prod`** — release image (multi-stage build, **`/migrations`** baked in). Example: `docker build -f deploy/Dockerfile.prod .` from **`api/`**.
- **`api/deploy/Dockerfile.dev`** — **Air** live reload (`api/.air.toml`); **`deploy/docker-compose.dev.yml`** bind-mounts **`api/`** → **`/src`**.
- **`api/deploy/Dockerfile.test`** — one-shot **`gotestsum`** + **`junit.xml`**. Mount the host Docker socket when running the container (integration tests use **testcontainers**). From the repo root: `docker compose -f deploy/docker-compose.test.yml --profile go-tests run --rm api-go-tests` (same path as **`.github/workflows/ci.yml`**). **`test-summary/action`** reads **`api/junit.xml`** after Compose (**`api/`** bind-mounted to **`/src`**).

**Build context:** **`api/`** (module root). **`api/.dockerignore`** trims the context. Compose stacks live under **`deploy/`**. In Compose, **`DATABASE_URL`** targets the **`postgres`** service where applicable.

## Makefile

See **`Makefile`** (`make run`, `make build`, `make test`, `make test-cover`).

- **`make test`** — `go test ./...` (unit + integration; integration uses **testcontainers** and needs **Docker**).
- **`make test-cover`** — same with **`-coverpkg=./internal/...,./pkg/...`** and writes **`coverage.out`** for `go tool cover`.
- **`make lint`** — **`golangci-lint`** when installed, else **`go vet`**. With **Go 1.25** in **`go.mod`**, install **golangci-lint v2** (v1.x is built with Go 1.24 and errors on 1.25 projects).

## Seeding

Database seeding is **not** run by this service. Use your own scripts (for example the JSON-driven flows documented next to **`data/`** at the repo root).
