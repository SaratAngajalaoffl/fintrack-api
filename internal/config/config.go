package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime settings loaded from the environment.
type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MigrationsPath string
	SkipMigrations bool
	// JWTSecret is required when DATABASE_URL is set (session + OTP JWTs).
	JWTSecret []byte
	// CookieSecure sets Secure flag on session cookies (HTTPS). True when COOKIE_SECURE or NODE_ENV=production.
	CookieSecure bool
	// CORSAllowedOrigins lists Allowed Origins for credentialed browser fetches (see middleware.CORS). Split from CORS_ALLOWED_ORIGINS (comma-separated).
	CORSAllowedOrigins []string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	if port != "" && !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		return Config{}, err
	}

	skip := strings.EqualFold(strings.TrimSpace(os.Getenv("SKIP_MIGRATIONS")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("SKIP_MIGRATIONS")), "true")

	cookieSecure := strings.EqualFold(os.Getenv("NODE_ENV"), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "1")

	var jwtSecret []byte
	if databaseURL != "" {
		jwt := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if len(jwt) < 16 {
			return Config{}, fmt.Errorf("JWT_SECRET must be set and at least 16 characters when DATABASE_URL is set")
		}
		jwtSecret = []byte(jwt)
	}

	var corsOrigins []string
	for _, p := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			corsOrigins = append(corsOrigins, p)
		}
	}

	return Config{
		HTTPAddr:           port,
		DatabaseURL:        databaseURL,
		MigrationsPath:     filepath.Clean(migrationsPath),
		SkipMigrations:     skip,
		JWTSecret:          jwtSecret,
		CookieSecure:       cookieSecure,
		CORSAllowedOrigins: corsOrigins,
	}, nil
}

// resolveMigrationsPath returns MIGRATIONS_PATH when set; otherwise finds api/migrations
// (cwd api/ or repo root) or /migrations (Docker image).
func resolveMigrationsPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("MIGRATIONS_PATH")); v != "" {
		return filepath.Clean(v), nil
	}
	if fi, err := os.Stat("/migrations"); err == nil && fi.IsDir() {
		return "/migrations", nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getcwd: %w", err)
	}
	candidates := []string{
		filepath.Join(wd, "migrations"),
		filepath.Join(wd, "api", "migrations"),
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return filepath.Clean(p), nil
		}
	}
	return "", fmt.Errorf("migrations directory not found (expected api/migrations, or /migrations in Docker); set MIGRATIONS_PATH to override")
}
