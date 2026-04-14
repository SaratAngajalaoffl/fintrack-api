# Fintrack API (Go)

HTTP API for Fintrack. The Next.js app lives in **`../web/`**. SQL migrations remain the **single source of truth** at the repository root: **`../migrations/`** (not duplicated under `api/`).

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
- **Auth:** all **`/api/auth/*`** routes (session cookie `fintrack_session`, bcrypt passwords, JWT OTP tickets).
- **Bank accounts:** **`/api/bank-accounts`** (`GET`/`POST` list + create, **`/{id}`** `GET`/`PATCH`/`DELETE`) — same JSON shapes as the former Next Route Handlers.
- Migrations apply automatically on startup unless **`SKIP_MIGRATIONS=true`** or **`1`**.

**With Next.js on your machine:** run the API on **:8080** and set **`web/.env.local`** (see **`web/.env.example`**) so **`API_ORIGIN=http://127.0.0.1:8080`** and **`JWT_SECRET`** matches this service. **`next.config.ts`** rewrites same-origin **`/api/auth/*`** and **`/api/bank-accounts`** to the Go server.

## Docker / Compose

In Compose, **`DATABASE_URL`** targets the `postgres` service and **`MIGRATIONS_PATH=/migrations`** with the repo’s `migrations/` directory mounted read-only.

## Makefile

See **`Makefile`** (`make run`, `make build`, `make test`).

## Seeding

Database seeding is **not** run by this service. Use your own scripts (for example the JSON-driven flows documented next to **`data/`** at the repo root).
