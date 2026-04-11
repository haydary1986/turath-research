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
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(cfg.Server.AllowedOrigins))
	r.Use(middleware.RateLimit(100, time.Minute))

	// ── Handlers ─────────────────────────────────
	healthH := handler.NewHealthHandler(db, rdb)
	authH := handler.NewAuthHandler(db, cfg)

	// ── Routes ───────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Get("/health", healthH.Check)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/register", authH.Register)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWT.Secret))

			r.Get("/auth/me", authH.Me)

			// TODO: researcher routes
			// TODO: publication routes
			// TODO: organization routes
			// TODO: promotion routes
			// TODO: incentive routes
			// TODO: performance routes
			// TODO: sabbatical routes
			// TODO: conference routes
			// TODO: agreement routes
			// TODO: accreditation routes
			// TODO: workflow routes
			// TODO: AI routes
		})
	})

	return r
}
