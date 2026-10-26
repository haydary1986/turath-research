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
	"github.com/haydary1986/turath-research/backend/internal/service/mailer"
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
	r.Use(middleware.RateLimit(cfg.Server.RateLimitPerMin, time.Minute))

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
	sciH := handler.NewScientificAffairsHandler(db)
	accredH := handler.NewAccreditationHandler(db)
	workflowH := handler.NewWorkflowHandler(db)
	roleDashH := handler.NewRoleDashboardHandler(db)
	mail := mailer.New(mailer.Config{
		Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password, From: cfg.SMTP.From, FromName: cfg.SMTP.FromName,
	})
	portalH := handler.NewPortalHandler(db, cfg.Server.UploadDir, mail, cfg.Server.FrontendURL)
	adminUsersH := handler.NewAdminUserHandler(db)
	adminImportH := handler.NewAdminImportHandler(db)
	staffOnly := middleware.RequireRole(middleware.RoleScientificAffairs, middleware.RoleVPResearch)
	adminOnly := middleware.RequireRole(middleware.RoleAdmin)

	// ── Routes ───────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Get("/health", healthH.Check)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/register", authH.Register)
		r.Get("/auth/config", authH.Config)
		r.Post("/auth/google", authH.GoogleLogin)
		r.Get("/verify/letter/{id}", portalH.VerifyLetter)

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

			// ── Scientific Affairs ────────────────
			// Promotions
			r.Get("/promotions/eligibility/{id}", sciH.CheckEligibility)
			r.Post("/promotions", sciH.SubmitPromotionApplication)
			r.Get("/promotions", sciH.ListPromotionApplications)

			// Incentives
			r.Get("/incentives/calculate/{id}", sciH.CalculateIncentives)
			r.Get("/incentives/rules", sciH.ListIncentiveRules)

			// Performance
			r.Post("/performance/generate/{deptId}", sciH.GeneratePerformanceReviews)

			// Sabbaticals
			r.Post("/sabbaticals", sciH.SubmitSabbatical)
			r.Get("/sabbaticals", sciH.ListSabbaticals)

			// Conferences
			r.Post("/conferences", sciH.SubmitConference)

			// Agreements
			r.Get("/agreements", sciH.ListAgreements)

			// ── Accreditation & Quality ───────────
			r.Get("/accreditation/dashboard", accredH.Dashboard)
			r.Get("/accreditation/evidence", accredH.ListEvidence)
			r.Post("/accreditation/evidence", accredH.CreateEvidence)
			r.Get("/accreditation/recommendations", accredH.ListRecommendations)
			r.Put("/accreditation/recommendations/{id}", accredH.UpdateRecommendation)
			r.Get("/accreditation/kpis", accredH.ListKPIs)

			// ── Approval Workflows ────────────────
			r.Get("/workflows/pending", workflowH.MyPendingApprovals)
			r.Post("/workflows/initiate", workflowH.InitiateWorkflow)
			r.Put("/workflows/{id}/decide", workflowH.MakeDecision)
			r.Get("/workflows/history", workflowH.RequestHistory)

			// ── Role-based Dashboard ──────────────
			r.Get("/dashboard/me", roleDashH.MyDashboard)

			// ── Scientific Affairs Portal ─────────
			r.Get("/portal/stats", portalH.Stats)
			r.Get("/portal/requests", portalH.List)
			r.Post("/portal/requests", portalH.Create)
			r.Get("/portal/requests/{id}", portalH.Get)
			r.Put("/portal/requests/{id}", portalH.Update)
			r.Post("/portal/requests/{id}/submit", portalH.Submit)
			r.Post("/portal/requests/{id}/files", portalH.UploadAttachment)
			r.Get("/portal/files/{id}", portalH.DownloadFile)
			r.Delete("/portal/files/{id}", portalH.DeleteAttachment)
			r.Get("/portal/letters/{id}", portalH.GetLetter)

			// Stage actions gate themselves per role inside the handler
			// (the dean acts too, so these cannot sit behind staffOnly).
			r.Post("/portal/requests/{id}/transition", portalH.Transition)
			r.Post("/portal/requests/{id}/issue-letter", portalH.IssueLetter)

			r.Group(func(r chi.Router) {
				r.Use(staffOnly)
				r.Get("/portal/templates", portalH.ListTemplates)
				r.Post("/portal/templates", portalH.CreateTemplate)
				r.Put("/portal/templates/{id}", portalH.UpdateTemplate)
				r.Post("/portal/templates/{id}/activate", portalH.ActivateTemplate)
				r.Post("/portal/template-assets", portalH.UploadTemplateAsset)
			})

			r.Group(func(r chi.Router) {
				r.Use(adminOnly)
				r.Get("/admin/users", adminUsersH.List)
				r.Put("/admin/users/{id}", adminUsersH.Update)
				r.Post("/admin/import/researchers", adminImportH.ImportResearchers)
			})
		})
	})

	return r
}
