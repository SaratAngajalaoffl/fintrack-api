package migrate_test

import (
	"context"
	"testing"

	"fintrack/api/internal/migrate"
	"fintrack/api/internal/testutil"
)

func TestRunIdempotent(t *testing.T) {
	pool, cleanup := testutil.NewPostgresPool(t)
	defer cleanup()
	ctx := context.Background()
	dir := testutil.MigrationsDir(t)
	if err := migrate.Run(ctx, pool, dir); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Run(ctx, pool, dir); err != nil {
		t.Fatal(err)
	}
}
