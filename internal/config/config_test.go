package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// chdirAPIRoot sets cwd to the api/ module root so resolveMigrationsPath finds api/migrations.
func chdirAPIRoot(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	t.Chdir(apiRoot)
}

func TestLoadWithoutDatabaseURL(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("SKIP_MIGRATIONS", "")
	t.Setenv("PORT", "9999")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("NODE_ENV", "")
	t.Setenv("COOKIE_SECURE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "" {
		t.Fatal("expected empty database URL")
	}
	if len(cfg.JWTSecret) != 0 {
		t.Fatal("expected no jwt secret")
	}
	if !strings.HasPrefix(cfg.HTTPAddr, ":") || cfg.HTTPAddr != ":9999" {
		t.Fatalf("addr %q", cfg.HTTPAddr)
	}
}

func TestLoadWithDatabaseRequiresJWT(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", "short")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short JWT")
	}
}

func TestLoadWithDatabaseAndJWT(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", "sixteencharslong")
	t.Setenv("CORS_ALLOWED_ORIGINS", " http://a.test , http://b.test ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.JWTSecret) < 16 {
		t.Fatal("jwt secret")
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("origins %v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadSkipMigrations(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", "sixteencharslong")
	t.Setenv("SKIP_MIGRATIONS", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.SkipMigrations {
		t.Fatal("expected skip")
	}
}

func TestLoadCookieSecure(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("COOKIE_SECURE", "1")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.CookieSecure {
		t.Fatal("expected cookie secure")
	}
}

func TestLoadPortDefault(t *testing.T) {
	chdirAPIRoot(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("got %q", cfg.HTTPAddr)
	}
}

func TestResolveMigrationsPathFromEnv(t *testing.T) {
	chdirAPIRoot(t)
	tmp := t.TempDir()
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "001_x.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	old, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cfg.MigrationsPath, "migrations") {
		t.Fatalf("path %q", cfg.MigrationsPath)
	}
}
