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
	migrationsPath := strings.TrimSpace(os.Getenv("MIGRATIONS_PATH"))
	if migrationsPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Config{}, fmt.Errorf("getcwd: %w", err)
		}
		migrationsPath = filepath.Join(wd, "..", "migrations")
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

	return Config{
		HTTPAddr:       port,
		DatabaseURL:    databaseURL,
		MigrationsPath: filepath.Clean(migrationsPath),
		SkipMigrations: skip,
		JWTSecret:      jwtSecret,
		CookieSecure:   cookieSecure,
	}, nil
}
