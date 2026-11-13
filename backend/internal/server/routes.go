package server

import (
	"context"
	"errors"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/handler"
	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/service/identity"
	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
	"github.com/haydary1986/turath-research/backend/internal/service/mailer"
	"github.com/haydary1986/turath-research/backend/internal/service/presence"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

func NewRouter(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, journals *journalcheck.Service) *chi.Mux {
	orchestrator := syncsvc.NewOrchestrator(db, cfg.CrossRef.Mailto, cfg.OpenAlex.APIKey)
	return newRouter(cfg, db, rdb, journals, orchestrator, newBulkSync(cfg, db, orchestrator))
}

func newBulkSync(cfg *config.Config, db *pgxpool.Pool, orchestrator *syncsvc.Orchestrator) *syncsvc.BulkRunner {
	return syncsvc.NewBulkRunner(db, orchestrator, syncsvc.BulkConfig{
		Delay: cfg.BulkSync.Delay, DailyLimit: cfg.BulkSync.DailyLimit,
	})
}

func newRouter(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, journals *journalcheck.Service,
	orchestrator *syncsvc.Orchestrator, bulkSync *syncsvc.BulkRunner) *chi.Mux {
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
	syncH := handler.NewSyncHandler(db, orchestrator)
	researchProfileH := handler.NewResearchProfileHandler(db, orchestrator)
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
	portalH := handler.NewPortalHandler(db, cfg.Server.UploadDir, mail, cfg.Server.FrontendURL, journals)
	// Role and organization come from the database (cached ~60 s per user), not the JWT.
	identities := identity.NewCache(identityLoader(db), identity.DefaultTTL, time.Now)
	adminUsersH := handler.NewAdminUserHandler(db, identities.Invalidate, cfg.Google.AllowedDomain)
	adminImportH := handler.NewAdminImportHandler(db)
	bulkSyncH := handler.NewBulkSyncHandler(bulkSync)
	responsibilityH := handler.NewResponsibilityHandler(db, identities.Invalidate)
	auditH := handler.NewAuditHandler(db)
	lastSeen := presence.NewTracker(lastSeenToucher(db), presence.DefaultInterval)
	staffOnly := middleware.RequireRole(middleware.RoleScientificAffairs, middleware.RoleVPResearch)
	adminOnly := middleware.RequireRole(middleware.RoleAdmin)
	// RequireRole always admits admin, so this is admin and scientific affairs.
	registryManagers := middleware.RequireRole(middleware.RoleScientificAffairs)

	// ── Routes ───────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// Public
		r.Get("/health", healthH.Check)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/register", authH.Register)
		r.Get("/auth/config", authH.Config)
		r.Post("/auth/google", authH.GoogleLogin)
		r.Get("/verify/letter/{id}", portalH.VerifyLetter)

		// Public site: landing-page content and published announcements only.
		r.Get("/public/site", portalH.GetSiteContent)
		r.Get("/public/announcements", portalH.ListPublicAnnouncements)
		r.Get("/public/announcements/{id}", portalH.GetPublicAnnouncement)
		r.Get("/public/files/{id}", portalH.DownloadPublicFile)

		// Protected
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(cfg.JWT.Secret))
			r.Use(middleware.RefreshIdentity(identityLookup(identities)))
			// users.last_seen_at: throttled per user and written in the background.
			r.Use(middleware.TrackPresence(lastSeen.Seen))

			// Auth
			r.Get("/auth/me", authH.Me)

			// The signed-in user's own research profile
			r.Get("/me/responsibilities", responsibilityH.Mine)
			r.Get("/me/research-profile", researchProfileH.Get)
			r.Put("/me/research-profile", researchProfileH.Update)
			r.Post("/me/research-profile/sync", researchProfileH.Sync)

			// Researchers
			r.Get("/researchers", researcherH.List)
			r.Get("/researchers/{id}", researcherH.Get)
			// The handler admits the researcher and staff.
			r.Get("/researchers/{id}/contact", researcherH.Contact)

			// Publications
			r.Get("/publications", publicationH.List)
			r.Get("/publications/{id}", publicationH.Get)

			// Organizations
			r.Get("/institutions", orgH.ListInstitutions)
			r.Get("/colleges", orgH.ListColleges)
			r.Get("/departments", orgH.ListDepartments)

			// Metrics & dashboards
			r.Get("/metrics/institution", metricsH.InstitutionKPIs)
			r.Get("/metrics/researcher/{id}", metricsH.ResearcherStats)
			r.Get("/metrics/departments", metricsH.DepartmentComparison)
			r.Get("/metrics/leaderboard", metricsH.TopResearchers)

			// Search
			r.Get("/search", searchH.GlobalSearch)

			// Notifications
			r.Get("/notifications", notifH.ListMyNotifications)
			r.Get("/notifications/unread", notifH.UnreadCount)
			r.Put("/notifications/{id}/read", notifH.MarkRead)
			r.Put("/notifications/read-all", notifH.MarkAllRead)

			// Promotions, incentives and leave act on one researcher: the handlers
			// admit that researcher and staff.
			r.Get("/promotions/eligibility/{id}", sciH.CheckEligibility)
			r.Post("/promotions", sciH.SubmitPromotionApplication)
			r.Get("/incentives/calculate/{id}", sciH.CalculateIncentives)
			r.Get("/incentives/rules", sciH.ListIncentiveRules)
			r.Post("/sabbaticals", sciH.SubmitSabbatical)
			r.Post("/conferences", sciH.SubmitConference)

			// Approval workflows: the pending list is scoped to the caller and a
			// decision is gated to the step's approver in the handler.
			r.Get("/workflows/pending", workflowH.MyPendingApprovals)
			r.Put("/workflows/{id}/decide", workflowH.MakeDecision)

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

			// ── Scientific activities ─────────────
			// Category rights (institutional categories, responsibilities) are checked per
			// request in the handlers; the FS 003 export admits staff and responsibility holders.
			r.Get("/activities", portalH.ListActivities)
			r.Get("/activities/fs003.xlsx", portalH.ExportFS003)
			r.Post("/activities", portalH.CreateActivity)
			r.Put("/activities/{id}", portalH.UpdateActivity)

			// Automatic journal check (request forms, request detail, journals page)
			r.Get("/screening/check", portalH.CheckJournal)
			r.Get("/portal/form-templates", portalH.ListFormTemplates)
			r.Get("/screening/status", portalH.JournalListStatus)
			r.Get("/screening/reference-list", portalH.ListReferenceEntries)

			r.Group(func(r chi.Router) {
				r.Use(staffOnly)
				r.Get("/researchers/{id}/links", researchProfileH.StaffLinks)

				// Each cache miss spends AI budget; the screens are staff-only too.
				r.Get("/ai/collaboration/{id}", aiH.CollaborationRecommendations)
				r.Get("/ai/trends", aiH.TrendingTopics)
				r.Get("/ai/strategy/{id}", aiH.PublicationStrategy)
				r.Get("/ai/gaps", aiH.GapAnalysis)

				// Institution-wide views of other people's applications and evidence.
				r.Get("/promotions", sciH.ListPromotionApplications)
				r.Get("/sabbaticals", sciH.ListSabbaticals)
				r.Get("/agreements", sciH.ListAgreements)
				r.Get("/accreditation/dashboard", accredH.Dashboard)
				r.Get("/accreditation/evidence", accredH.ListEvidence)
				r.Get("/accreditation/recommendations", accredH.ListRecommendations)
				r.Get("/accreditation/kpis", accredH.ListKPIs)
				r.Get("/workflows/history", workflowH.RequestHistory)

				r.Get("/portal/templates", portalH.ListTemplates)
				r.Post("/portal/templates", portalH.CreateTemplate)
				r.Put("/portal/templates/{id}", portalH.UpdateTemplate)
				r.Post("/portal/templates/{id}/activate", portalH.ActivateTemplate)
				r.Post("/portal/template-assets", portalH.UploadTemplateAsset)
				r.Post("/portal/form-templates/{kind}", portalH.UploadFormTemplate)
				r.Delete("/portal/form-templates/{kind}", portalH.DeleteFormTemplate)
				r.Post("/activities/{id}/decide", portalH.DecideActivity)
				r.Get("/screening", portalH.ListScreening)
				r.Post("/screening", portalH.UpsertScreening)
				r.Post("/screening/import", portalH.ImportScreening)
				r.Put("/screening/{id}", portalH.UpsertScreening)
				r.Delete("/screening/{id}", portalH.DeleteScreening)
				r.Get("/reports/annual.xlsx", portalH.AnnualReport)
			})

			r.Group(func(r chi.Router) {
				r.Use(registryManagers)
				r.Put("/portal/letters/{id}/diwan-no", portalH.UpdateLetterDiwanNo)

				// The public site speaks for the department: its content and announcements.
				r.Put("/site", portalH.UpdateSiteContent)
				r.Get("/announcements", portalH.ListAnnouncements)
				r.Post("/announcements", portalH.CreateAnnouncement)
				r.Get("/announcements/{id}", portalH.GetAnnouncement)
				r.Put("/announcements/{id}", portalH.UpdateAnnouncement)
				r.Delete("/announcements/{id}", portalH.DeleteAnnouncement)
				r.Post("/announcements/{id}/publish", portalH.PublishAnnouncement)
				r.Post("/announcements/{id}/pin", portalH.PinAnnouncement)
				r.Post("/announcements/{id}/attachment", portalH.UploadAnnouncementAttachment)
				r.Delete("/announcements/{id}/attachment", portalH.DeleteAnnouncementAttachment)
				r.Post("/activities/{id}/announce", portalH.AnnounceActivity)

				// Changing a researcher's email hands over their profile, and these
				// tables are shared by every user of the system.
				r.Post("/researchers", researcherH.Create)
				r.Put("/researchers/{id}", researcherH.Update)
				r.Delete("/researchers/{id}", researcherH.Delete)
				r.Post("/publications", publicationH.Create)
				r.Delete("/publications/{id}", publicationH.Delete)
				r.Post("/institutions", orgH.CreateInstitution)
				r.Post("/colleges", orgH.CreateCollege)
				r.Post("/departments", orgH.CreateDepartment)
				r.Get("/export/researchers", metricsH.ExportResearchersCSV)

				// Syncs spend upstream quota and rewrite figures for someone else.
				r.Post("/sync/researcher/{id}", syncH.SyncResearcher)
				r.Post("/sync/doi", syncH.SyncByDOI)
				r.Get("/sync/jobs", syncH.ListSyncJobs)

				r.Post("/performance/generate/{deptId}", sciH.GeneratePerformanceReviews)
				r.Post("/accreditation/evidence", accredH.CreateEvidence)
				r.Put("/accreditation/recommendations/{id}", accredH.UpdateRecommendation)
				r.Post("/workflows/initiate", workflowH.InitiateWorkflow)
			})

			r.Group(func(r chi.Router) {
				r.Use(adminOnly)
				r.Get("/admin/users", adminUsersH.List)
				r.Post("/admin/users", adminUsersH.Create)
				r.Put("/admin/users/{id}", adminUsersH.Update)
				r.Get("/admin/users/{id}/responsibilities", responsibilityH.List)
				r.Put("/admin/users/{id}/responsibilities", responsibilityH.Replace)
				r.Get("/admin/login-events", auditH.ListLoginEvents)
				r.Get("/admin/accounts", auditH.ListAccounts)
				r.Get("/admin/accounts/summary", auditH.AccountsSummary)
				r.Post("/admin/import/researchers", adminImportH.ImportResearchers)
				r.Get("/admin/sync/orcid", bulkSyncH.Progress)
				r.Post("/admin/sync/orcid", bulkSyncH.Start)
				r.Post("/admin/sync/orcid/stop", bulkSyncH.Stop)
			})
		})
	})

	return r
}

// lastSeenToucher writes users.last_seen_at; without a database (route tests) nothing is tracked.
func lastSeenToucher(db *pgxpool.Pool) presence.Toucher {
	if db == nil {
		return nil
	}
	return func(ctx context.Context, userID string) error {
		_, err := db.Exec(ctx, `UPDATE users SET last_seen_at = NOW() WHERE id = $1`, userID)
		return err
	}
}

// identityLoader reads a user's current role and organization; without a database
// (route tests) there is nothing to load and the signed claims are used.
func identityLoader(db *pgxpool.Pool) identity.Loader {
	if db == nil {
		return nil
	}
	return func(ctx context.Context, userID string) (identity.Current, bool, error) {
		var cur identity.Current
		err := db.QueryRow(ctx, `
			SELECT role::text, COALESCE(institution_id::text, ''), COALESCE(college_id::text, ''),
			       COALESCE(department_id::text, '')
			FROM users WHERE id = $1`, userID).
			Scan(&cur.Role, &cur.InstitutionID, &cur.CollegeID, &cur.DepartmentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return cur, false, nil
		}
		return cur, err == nil, err
	}
}

func identityLookup(cache *identity.Cache) middleware.IdentityLookup {
	if cache == nil {
		return nil
	}
	return cache.Get
}
