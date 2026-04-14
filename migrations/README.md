SQL migrations for Fintrack live in this directory: ordered `*.sql` files applied on **Go API** startup (and optionally via **`deploy/docker/scripts/run-migrations.sh`**).

**Path resolution:** the API uses **`MIGRATIONS_PATH`** when set; otherwise it looks for **`api/migrations`** relative to the current working directory (run from **`api/`** or the repo root), or **`/migrations`** in the Docker image.
