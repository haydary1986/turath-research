package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/server"
)

func main() {
	// Load .env in development (silently ignored if missing).
	_ = godotenv.Load()

	// Structured logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Connect to PostgreSQL ────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	slog.Info("connected to PostgreSQL")

	// ── Connect to Redis ─────────────────────────
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("redis parse URL: %w", err)
	}

	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	slog.Info("connected to Redis")

	// ── Run migrations if enabled ────────────────
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		slog.Info("running database migrations")
		if err := runMigrations(cfg.Database.URL); err != nil {
			slog.Warn("migrations", "error", err)
		}
	}

	// ── Start HTTP server ────────────────────────
	return server.Run(cfg, dbpool, rdb)
}

func runMigrations(dbURL string) error {
	migrate, err := newMigrate(dbURL)
	if err != nil {
		return err
	}
	defer migrate.Close()

	if err := migrate.Up(); err != nil && err.Error() != "no change" {
		return fmt.Errorf("migrate up: %w", err)
	}

	slog.Info("migrations complete")
	return nil
}
