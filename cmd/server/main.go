package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"

	// Import pgx stdlib driver for goose
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bangun-ekosistem/service-api/internal/config"
	"github.com/bangun-ekosistem/service-api/internal/pkg/cache"
	"github.com/bangun-ekosistem/service-api/internal/pkg/pagination"
	"github.com/bangun-ekosistem/service-api/internal/scheduler"
	"github.com/bangun-ekosistem/service-api/internal/server"
	"github.com/bangun-ekosistem/service-api/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize pagination defaults from config
	pagination.Init(cfg.DefaultPerPage, cfg.MaxPerPage)

	// Connect to PostgreSQL with pgx pool
	ctx := context.Background()
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to parse database URL", "error", err)
		os.Exit(1)
	}

	// Tune connection pool for production
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 10
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	// Run goose migrations
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Optional: connect to Redis (for rate limiting, etc.)
	var redisClient *redis.Client
	if cfg.RedisURL != "" {
		rc, err := cache.NewRedisClient(cfg.RedisURL)
		if err != nil {
			slog.Warn("failed to parse REDIS_URL, continuing without Redis", "error", err)
		} else if cache.IsAvailable(rc) {
			redisClient = rc
			slog.Info("connected to redis")
		} else {
			slog.Warn("redis not reachable, continuing without Redis")
			_ = rc.Close()
		}
	} else {
		slog.Info("REDIS_URL not set, rate limiting will use in-memory backend")
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// Create and start server
	srv := server.NewServer(cfg, pool, redisClient)

	// Start daily summary + low stock alert scheduler
	notifSvc := service.NewNotificationService(pool, cfg)
	inventorySvc := service.NewInventoryService(pool)
	sched := scheduler.New(notifSvc, inventorySvc)
	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	sched.Start(schedCtx)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "port", cfg.Port)

	<-quit
	slog.Info("received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

func runMigrations(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	slog.Info("running database migrations")
	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}
	slog.Info("migrations completed")

	return nil
}
