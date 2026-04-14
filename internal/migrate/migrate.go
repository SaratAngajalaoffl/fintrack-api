package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run applies pending *.sql files in lexicographic order, recording each filename in schema_migrations
// (same behavior as deploy/docker/scripts/run-migrations.sh).
func Run(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	for _, name := range names {
		var applied bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %q: %w", name, err)
		}
		if applied {
			slog.Info("migration skip (already applied)", "file", name)
			continue
		}

		path := filepath.Join(migrationsDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}

		sql := strings.TrimSpace(string(body))
		if sql == "" {
			return fmt.Errorf("empty migration file: %s", name)
		}

		// Simple Query protocol supports multiple statements per round-trip.
		pgc := conn.Conn().PgConn()
		result := pgc.Exec(ctx, sql)
		_, err = result.ReadAll()
		if err != nil {
			return fmt.Errorf("apply migration %q: %w", name, err)
		}

		if _, err := conn.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`,
			name,
		); err != nil {
			return fmt.Errorf("record migration %q: %w", name, err)
		}

		slog.Info("migration applied", "file", name)
	}

	return nil
}

// DirExists reports whether path is a directory that contains at least one .sql file.
func DirExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return false
	}
	var hasSQL bool
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			hasSQL = true
			return fs.SkipAll
		}
		return nil
	})
	return hasSQL
}
