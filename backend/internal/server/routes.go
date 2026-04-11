package server

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/handler"
	"github.com/haydary1986/turath-research/backend/internal/middleware"
)

func NewRouter(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	// ── Global middleware ─────────────────────────
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.CORS(cfg.Server.AllowedOrigins))
	r.Use(middleware.RateLimit(100, time.Minute))

	// ── Handlers ─────────────────────────────────
	healthH := handler.NewHealthHandler(db, rdb)

	// ── Routes ───────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthH.Check)

		// TODO: auth routes
		// TODO: researcher routes
		// TODO: publication routes
		// TODO: metrics routes
		// TODO: ai routes
	})

	return r
}
