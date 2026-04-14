# Fintrack API (Go)

HTTP API service for Fintrack. The Next.js frontend lives in **`../web/`**.

- **Run locally:** from this directory, `go run ./cmd/fintrack` (listens on `:8080`; override with `PORT`).
- **Health:** `GET http://localhost:8080/health` returns `{"status":"ok"}`.

Shared SQL migrations remain at the repository root: **`../migrations/`**.
