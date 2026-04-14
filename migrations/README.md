SQL migrations are maintained in a **single directory at the repository root**: **`../../migrations/`** (not here).

The Go API sets **`MIGRATIONS_PATH`** to that folder (defaulting to `../migrations` when run from **`api/`**, or `/migrations` in Docker / baked into the image).

This folder exists only so tooling that expects an `api/migrations` path has a pointer to the canonical location.
