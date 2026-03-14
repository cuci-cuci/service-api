package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/bangun-ekosistem/service-api/internal/config"
)

type Server struct {
	Router      *chi.Mux
	Config      *config.Config
	DB          *pgxpool.Pool
	Validate    *validator.Validate
	RedisClient *redis.Client // nil when Redis is not configured
	server      *http.Server
}

func NewServer(cfg *config.Config, db *pgxpool.Pool, redisClient *redis.Client) *Server {
	s := &Server{
		Router:      chi.NewRouter(),
		Config:      cfg,
		DB:          db,
		Validate:    validator.New(),
		RedisClient: redisClient,
	}

	s.RegisterRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.Router,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSecs) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSecs) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	slog.Info("starting server", "port", s.Config.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	return s.server.Shutdown(ctx)
}
