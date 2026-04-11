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
	researcherH := handler.NewResearcherHandler(db)
	publicationH := handler.NewPublicationHandler(db)
	orgH := handler.NewOrganizationHandler(db)
	syncH := handler.NewSyncHandler(db, cfg.CrossRef.Mailto)
	metricsH := handler.NewMetricsHandler(db)
	aiH := handler.NewAIHandler(db, cfg.DeepSeek.APIKey, cfg.DeepSeek.BaseURL)
	searchH := handler.NewSearchHandler(db)
	notifH := handler.NewNotificationHandler(db)

	// ── Routes ───────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Get("/health", healthH.Check)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/register", authH.Register)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWT.Secret))

			// Auth
			r.Get("/auth/me", authH.Me)

			// Researchers
			r.Get("/researchers", researcherH.List)
			r.Get("/researchers/{id}", researcherH.Get)
			r.Post("/researchers", researcherH.Create)
			r.Put("/researchers/{id}", researcherH.Update)
			r.Delete("/researchers/{id}", researcherH.Delete)

			// Publications
			r.Get("/publications", publicationH.List)
			r.Get("/publications/{id}", publicationH.Get)
			r.Post("/publications", publicationH.Create)
			r.Delete("/publications/{id}", publicationH.Delete)

			// Organizations
			r.Get("/institutions", orgH.ListInstitutions)
			r.Post("/institutions", orgH.CreateInstitution)
			r.Get("/colleges", orgH.ListColleges)
			r.Post("/colleges", orgH.CreateCollege)
			r.Get("/departments", orgH.ListDepartments)
			r.Post("/departments", orgH.CreateDepartment)

			// Metrics & dashboards
			r.Get("/metrics/institution", metricsH.InstitutionKPIs)
			r.Get("/metrics/researcher/{id}", metricsH.ResearcherStats)
			r.Get("/metrics/departments", metricsH.DepartmentComparison)
			r.Get("/metrics/leaderboard", metricsH.TopResearchers)
			r.Get("/export/researchers", metricsH.ExportResearchersCSV)

			// Search
			r.Get("/search", searchH.GlobalSearch)

			// Notifications
			r.Get("/notifications", notifH.ListMyNotifications)
			r.Get("/notifications/unread", notifH.UnreadCount)
			r.Put("/notifications/{id}/read", notifH.MarkRead)
			r.Put("/notifications/read-all", notifH.MarkAllRead)

			// AI insights
			r.Get("/ai/collaboration/{id}", aiH.CollaborationRecommendations)
			r.Get("/ai/trends", aiH.TrendingTopics)
			r.Get("/ai/strategy/{id}", aiH.PublicationStrategy)
			r.Get("/ai/gaps", aiH.GapAnalysis)

			// Sync routes
			r.Post("/sync/researcher/{id}", syncH.SyncResearcher)
			r.Post("/sync/doi", syncH.SyncByDOI)
			r.Get("/sync/jobs", syncH.ListSyncJobs)

			// Admin-only routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(
					middleware.RoleAdmin,
					middleware.RoleVPResearch,
					middleware.RoleDeanResearch,
					middleware.RoleScientificAffairs,
				))
				// TODO: promotion management routes
				// TODO: incentive management routes
				// TODO: performance review routes
				// TODO: accreditation routes
				// TODO: workflow management routes
			})
		})
	})

	return r
}
