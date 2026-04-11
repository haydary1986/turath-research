package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

type RoleDashboardHandler struct {
	db *pgxpool.Pool
}

func NewRoleDashboardHandler(db *pgxpool.Pool) *RoleDashboardHandler {
	return &RoleDashboardHandler{db: db}
}

// MyDashboard returns a role-specific dashboard for the current user.
func (h *RoleDashboardHandler) MyDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch user.Role {
	case middleware.RoleResearcher:
		h.researcherDashboard(ctx, w, user)
	case middleware.RoleDepartmentHead:
		h.departmentHeadDashboard(ctx, w, user)
	case middleware.RoleCollegeDean:
		h.collegeDeanDashboard(ctx, w, user)
	case middleware.RoleScientificAffairs, middleware.RoleDeanResearch:
		h.scientificAffairsDashboard(ctx, w, user)
	case middleware.RoleVPResearch, middleware.RoleAdmin:
		h.vpResearchDashboard(ctx, w, user)
	default:
		model.RespondJSON(w, http.StatusOK, map[string]string{"role": user.Role, "message": "No dashboard configured"})
	}
}

func (h *RoleDashboardHandler) researcherDashboard(ctx context.Context, w http.ResponseWriter, user *middleware.UserClaims) {
	var resID *string
	h.db.QueryRow(ctx, `SELECT researcher_id FROM users WHERE id = $1`, user.UserID).Scan(&resID)

	dash := map[string]any{"role": "researcher"}

	if resID != nil {
		var hIndex, totalPubs, totalCit int
		h.db.QueryRow(ctx, `SELECT h_index, total_publications, total_citations FROM researchers WHERE id = $1`, *resID).
			Scan(&hIndex, &totalPubs, &totalCit)
		dash["h_index"] = hIndex
		dash["total_publications"] = totalPubs
		dash["total_citations"] = totalCit

		// Pending tasks
		var pendingApprovals int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM approval_workflows WHERE approver_id = $1 AND decision = 'pending'`, user.UserID).Scan(&pendingApprovals)
		dash["pending_approvals"] = pendingApprovals

		// Unread notifications
		var unread int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, user.UserID).Scan(&unread)
		dash["unread_notifications"] = unread
	}

	model.RespondJSON(w, http.StatusOK, dash)
}

func (h *RoleDashboardHandler) departmentHeadDashboard(ctx context.Context, w http.ResponseWriter, user *middleware.UserClaims) {
	dash := map[string]any{"role": "department_head"}

	if user.DepartmentID != "" {
		var resCount, totalPubs, totalCit int
		var avgH float64
		h.db.QueryRow(ctx, `
			SELECT COUNT(*), COALESCE(SUM(total_publications),0), COALESCE(SUM(total_citations),0), COALESCE(AVG(h_index),0)
			FROM researchers WHERE department_id = $1 AND status = 'active'
		`, user.DepartmentID).Scan(&resCount, &totalPubs, &totalCit, &avgH)

		dash["researchers"] = resCount
		dash["total_publications"] = totalPubs
		dash["total_citations"] = totalCit
		dash["avg_h_index"] = avgH

		// Pending approvals for this department head
		var pending int
		h.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM approval_workflows
			WHERE approver_role = 'department_head' AND decision = 'pending'
		`).Scan(&pending)
		dash["pending_approvals"] = pending
	}

	model.RespondJSON(w, http.StatusOK, dash)
}

func (h *RoleDashboardHandler) collegeDeanDashboard(ctx context.Context, w http.ResponseWriter, user *middleware.UserClaims) {
	dash := map[string]any{"role": "college_dean"}

	if user.CollegeID != "" {
		var deptCount, resCount, totalPubs int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM departments WHERE college_id = $1`, user.CollegeID).Scan(&deptCount)
		h.db.QueryRow(ctx, `
			SELECT COUNT(*), COALESCE(SUM(r.total_publications),0)
			FROM researchers r
			JOIN departments d ON d.id = r.department_id
			WHERE d.college_id = $1 AND r.status = 'active'
		`, user.CollegeID).Scan(&resCount, &totalPubs)

		dash["departments"] = deptCount
		dash["researchers"] = resCount
		dash["total_publications"] = totalPubs

		var pending int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM approval_workflows WHERE approver_role = 'college_dean' AND decision = 'pending'`).Scan(&pending)
		dash["pending_approvals"] = pending
	}

	model.RespondJSON(w, http.StatusOK, dash)
}

func (h *RoleDashboardHandler) scientificAffairsDashboard(ctx context.Context, w http.ResponseWriter, user *middleware.UserClaims) {
	dash := map[string]any{"role": "scientific_affairs"}

	// Counts across all operations
	var pendingPromotions, pendingIncentives, pendingSabbaticals, pendingConferences int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM promotion_applications WHERE status NOT IN ('draft','approved','rejected')`).Scan(&pendingPromotions)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM publication_incentives WHERE status = 'pending_review'`).Scan(&pendingIncentives)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM sabbatical_requests WHERE status NOT IN ('draft','approved','rejected','completed')`).Scan(&pendingSabbaticals)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM conference_participations WHERE status NOT IN ('draft','approved','rejected','completed')`).Scan(&pendingConferences)

	dash["pending_promotions"] = pendingPromotions
	dash["pending_incentives"] = pendingIncentives
	dash["pending_sabbaticals"] = pendingSabbaticals
	dash["pending_conferences"] = pendingConferences

	var totalAgreements, activeAgreements int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM agreements`).Scan(&totalAgreements)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM agreements WHERE status = 'active'`).Scan(&activeAgreements)
	dash["total_agreements"] = totalAgreements
	dash["active_agreements"] = activeAgreements

	// Workflow approvals pending for scientific_affairs role
	var pendingApprovals int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM approval_workflows WHERE approver_role = 'scientific_affairs' AND decision = 'pending'`).Scan(&pendingApprovals)
	dash["pending_approvals"] = pendingApprovals

	model.RespondJSON(w, http.StatusOK, dash)
}

func (h *RoleDashboardHandler) vpResearchDashboard(ctx context.Context, w http.ResponseWriter, user *middleware.UserClaims) {
	dash := map[string]any{"role": "vp_research"}

	// Institution-wide KPIs
	var totalRes, totalPubs, totalCit int
	var avgH float64
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers WHERE status = 'active'`).Scan(&totalRes)
	h.db.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(citation_count),0) FROM publications`).Scan(&totalPubs, &totalCit)
	h.db.QueryRow(ctx, `SELECT COALESCE(AVG(h_index),0) FROM researchers WHERE status = 'active'`).Scan(&avgH)

	dash["total_researchers"] = totalRes
	dash["total_publications"] = totalPubs
	dash["total_citations"] = totalCit
	dash["avg_h_index"] = avgH

	// Pending VP approvals
	var pending int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM approval_workflows WHERE approver_role = 'vp_research' AND decision = 'pending'`).Scan(&pending)
	dash["pending_approvals"] = pending

	// Accreditation summary
	var openRecs, overdueRecs int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM accreditation_recommendations WHERE status IN ('open','in_progress')`).Scan(&openRecs)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM accreditation_recommendations WHERE status = 'overdue'`).Scan(&overdueRecs)
	dash["open_recommendations"] = openRecs
	dash["overdue_recommendations"] = overdueRecs

	// Colleges count
	var colleges int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM colleges`).Scan(&colleges)
	dash["colleges"] = colleges

	model.RespondJSON(w, http.StatusOK, dash)
}
