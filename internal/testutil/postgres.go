// Package testutil provides shared helpers for integration tests (Docker + Postgres).
package testutil

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// MigrationsDir returns the absolute path to api/migrations (works when tests run from any cwd).
func MigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	// internal/testutil/postgres.go -> ../../migrations
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
}

// NewPostgresPool starts a Postgres 16 container and returns a pool and cleanup (no migrations).
func NewPostgresPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("fintrack_it"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		t.Skip("postgres testcontainer (is Docker running?): ", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
	return pool, cleanup
}
