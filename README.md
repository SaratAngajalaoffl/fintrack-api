# Fintrack API (Go)

HTTP API for [Fintrack](https://github.com/SaratAngajalaoffl/fintrack). This repository is the **Go service** only; the Next.js UI lives in **[fintrack-web](https://github.com/SaratAngajalaoffl/fintrack-web)**. Docker Compose, shared docs, and submodule wiring live in the **[fintrack](https://github.com/SaratAngajalaoffl/fintrack)** meta-repo.

SQL migrations are under **`migrations/`** at the root of this repository.

## Layout

- **`cmd/api/`** — `main.go` entrypoint: config, DB pool, **run migrations**, HTTP server.
- **`internal/`** — app code not imported by other modules: `config`, `handler`, `middleware`, `migrate`, `model`, `repository`, `service`.
- **`pkg/`** — small reusable packages (`logger`, `validator`, …).
- **`docs/`** — generated **Swagger / OpenAPI 2.0** bundle (`docs.go`, `swagger.json`, `swagger.yaml`) from [swag](https://github.com/swaggo/swag); UI at **`/swagger/`** when the API is running.

## Run locally

From the **root of this repository**:

```bash
cp .env.example .env
# Set DATABASE_URL, JWT_SECRET (≥16 chars; Go signs session cookies — Next does not need this)
go run ./cmd/api
```

- **Health:** `GET http://localhost:8080/health`
- **Swagger UI:** `http://localhost:8080/swagger/index.html` (OpenAPI spec: `/swagger/doc.json`). Regenerate after changing route comments: **`make swagger`** (from **`api/`**).
- **Auth:** **`/api/auth/*`** (session cookie `fintrack_session`, bcrypt passwords, JWT OTP tickets).
- **Bank accounts, credit cards, expense categories:** CRUD-style **`/api/...`** routes (see **`web/src/configs/api-routes.ts`** in **fintrack-web**).
- **Fund buckets:** list/create plus **`/allocate`**, **`/unlock`**, **`/priority`** actions.
- Migrations apply automatically on startup unless **`SKIP_MIGRATIONS=true`** or **`1`**.

**With the Next.js app:** run this API on **:8080** and set **fintrack-web**’s **`.env.local`** from **`.env.example`** there (**`NEXT_PUBLIC_API_ORIGIN`**, optional **`API_ORIGIN`** for server/middleware). Next validates sessions by calling **`GET /api/auth/me`** on this service (no JWT secret in Next). **`getApiRoute()`** in the web repo builds URLs for browser and server-side calls. This API sends **CORS** headers for allowed origins (see **`CORS_ALLOWED_ORIGINS`** in **`.env.example`**) when the browser calls the Go origin directly.

## Data model documentation

Table and field reference (aligned with **`migrations/`**): [fintrack `docs/api/data-model.md`](https://github.com/SaratAngajalaoffl/fintrack/blob/main/docs/api/data-model.md).

## Docker / Compose

Build from this repo’s root:

- **`deploy/Dockerfile.prod`** — release image (multi-stage build, **`/migrations`** baked in). Example: `docker build -f deploy/Dockerfile.prod .`
- **`deploy/Dockerfile.dev`** — **Air** live reload (`.air.toml`); dev Compose in the meta-repo bind-mounts this tree → **`/src`**.
- **`deploy/Dockerfile.test`** — one-shot **`gotestsum`** + **`junit.xml`**. Mount the host Docker socket when running the container (integration tests use **testcontainers**). From a **meta-repo** checkout: `docker compose -f deploy/docker-compose.test.yml --profile go-tests run --rm api-go-tests` (paths assume **`api/`** exists next to **`deploy/`**).

**Build context:** repository root (module root). **`.dockerignore`** trims the context. Compose stacks are defined under **`deploy/`** in the **fintrack** meta-repo. In Compose, **`DATABASE_URL`** targets the **`postgres`** service where applicable.

## Makefile

See **`Makefile`** (`make run`, `make build`, `make test`, `make test-cover`).

- **`make test`** — `go test ./...` (unit + integration; integration uses **testcontainers** and needs **Docker**).
- **`make test-cover`** — same with **`-coverpkg=./internal/...,./pkg/...`** and writes **`coverage.out`** for `go tool cover`.
- **`make lint`** — **`golangci-lint`** when installed, else **`go vet`**. Match **golangci-lint** to your **Go** version from **`go.mod`**.

## Seeding

Database seeding is **not** run by this service. Use your own scripts (for example JSON-driven flows next to **`data/`** in the meta-repo).
