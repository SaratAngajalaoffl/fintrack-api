package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fintrack/api/internal/config"
	"fintrack/api/internal/handler"
	"fintrack/api/internal/migrate"
	"fintrack/api/internal/middleware"
	applogger "fintrack/api/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	applogger.Init()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		ctx := context.Background()
		p, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("database connect", "error", err)
			os.Exit(1)
		}
		defer p.Close()
		pool = p
	} else if !cfg.SkipMigrations {
		slog.Error("DATABASE_URL is required unless SKIP_MIGRATIONS is set")
		os.Exit(1)
	}

	if pool != nil && !cfg.SkipMigrations {
		if !migrate.DirExists(cfg.MigrationsPath) {
			slog.Error("migrations directory missing or empty", "path", cfg.MigrationsPath)
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := migrate.Run(ctx, pool, cfg.MigrationsPath); err != nil {
			slog.Error("migrations failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations complete", "dir", cfg.MigrationsPath)
	}

	mux := handler.NewMux(handler.Deps{
		DB:           pool,
		JWTSecret:    cfg.JWTSecret,
		CookieSecure: cfg.CookieSecure,
	})
	h := middleware.Chain(mux, middleware.CORS(cfg.CORSAllowedOrigins), middleware.Logging)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
