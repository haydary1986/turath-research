package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

type WorkflowHandler struct {
	db *pgxpool.Pool
}

func NewWorkflowHandler(db *pgxpool.Pool) *WorkflowHandler {
	return &WorkflowHandler{db: db}
}

// ── Pending approvals for current user ───────────

type pendingApproval struct {
	ID           string  `json:"id"`
	WorkflowType string  `json:"workflow_type"`
	RequestID    string  `json:"request_id"`
	StepOrder    int     `json:"step_order"`
	StepName     string  `json:"step_name"`
	ApproverRole string  `json:"approver_role"`
	CreatedAt    string  `json:"created_at"`
	RequestInfo  *string `json:"request_info"` // summary of the request
}

func (h *WorkflowHandler) MyPendingApprovals(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Find pending approvals where the user is the approver OR matches the approver role
	rows, err := h.db.Query(ctx, `
		SELECT aw.id, aw.workflow_type, aw.request_id, aw.step_order,
		       aw.step_name, aw.approver_role, aw.created_at
		FROM approval_workflows aw
		WHERE aw.decision = 'pending'
		  AND (aw.approver_id = $1 OR aw.approver_role::text = $2)
		ORDER BY aw.created_at ASC
		LIMIT 50
	`, user.UserID, mapUserRoleToApproverRole(user.Role))
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch pending approvals")
		return
	}
	defer rows.Close()

	items := []pendingApproval{}
	for rows.Next() {
		var p pendingApproval
		var createdAt time.Time
		rows.Scan(&p.ID, &p.WorkflowType, &p.RequestID, &p.StepOrder, &p.StepName, &p.ApproverRole, &createdAt)
		p.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, p)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// ── Make a decision ──────────────────────────────

func (h *WorkflowHandler) MakeDecision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Decision string `json:"decision"` // approved, rejected, returned
		Comments string `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Decision != "approved" && req.Decision != "rejected" && req.Decision != "returned" {
		model.RespondError(w, http.StatusBadRequest, "Decision must be: approved, rejected, or returned")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user := authedUser(w, r)
	if user == nil {
		return
	}

	// Only the step's own approver may decide it; admin may act for anyone.
	tag, err := h.db.Exec(ctx, `
		UPDATE approval_workflows SET
			decision = $2::workflow_decision,
			comments = $3,
			approver_id = $4,
			decided_at = NOW()
		WHERE id = $1 AND decision = 'pending'
		  AND ($5 OR approver_id = $4 OR approver_role::text = $6)
	`, id, req.Decision, req.Comments, user.UserID, user.Role == middleware.RoleAdmin, mapUserRoleToApproverRole(user.Role))
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Decision failed")
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Workflow step not found or already decided")
		return
	}

	// If approved, check if there's a next step
	if req.Decision == "approved" {
		var wfType string
		var requestID string
		var currentStep int
		h.db.QueryRow(ctx, `SELECT workflow_type, request_id, step_order FROM approval_workflows WHERE id = $1`, id).
			Scan(&wfType, &requestID, &currentStep)

		// Check if all steps are approved
		var pendingCount int
		h.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM approval_workflows
			WHERE workflow_type = $1 AND request_id = $2 AND decision = 'pending'
		`, wfType, requestID).Scan(&pendingCount)

		if pendingCount == 0 {
			// All steps approved — update the request status to approved
			h.updateRequestStatus(ctx, wfType, requestID, "approved")
		}
	} else if req.Decision == "rejected" {
		// Reject the entire request
		var wfType, requestID string
		h.db.QueryRow(ctx, `SELECT workflow_type, request_id FROM approval_workflows WHERE id = $1`, id).
			Scan(&wfType, &requestID)
		h.updateRequestStatus(ctx, wfType, requestID, "rejected")
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{"decided": id, "decision": req.Decision})
}

// ── Create workflow steps for a request ──────────

// InitiateWorkflow creates the approval workflow steps for a given request.
func (h *WorkflowHandler) InitiateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkflowType string `json:"workflow_type"`
		RequestID    string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Define steps based on workflow type
	steps := getWorkflowSteps(req.WorkflowType)
	if len(steps) == 0 {
		model.RespondError(w, http.StatusBadRequest, "Unknown workflow type")
		return
	}

	for _, step := range steps {
		h.db.Exec(ctx, `
			INSERT INTO approval_workflows (id, workflow_type, request_id, step_order, step_name, approver_role, decision)
			VALUES ($1, $2::workflow_type, $3, $4, $5, $6::approver_role, 'pending')
		`, uuid.New().String(), req.WorkflowType, req.RequestID, step.Order, step.Name, step.Role)
	}

	model.RespondJSON(w, http.StatusCreated, map[string]any{
		"workflow_type": req.WorkflowType,
		"request_id":    req.RequestID,
		"steps":         len(steps),
	})
}

// ── Workflow history for a request ───────────────

func (h *WorkflowHandler) RequestHistory(w http.ResponseWriter, r *http.Request) {
	wfType := r.URL.Query().Get("type")
	requestID := r.URL.Query().Get("request_id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, step_order, step_name, approver_role, decision, comments, decided_at, created_at
		FROM approval_workflows
		WHERE workflow_type = $1::workflow_type AND request_id = $2
		ORDER BY step_order
	`, wfType, requestID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch")
		return
	}
	defer rows.Close()

	type stepRow struct {
		ID           string  `json:"id"`
		StepOrder    int     `json:"step_order"`
		StepName     string  `json:"step_name"`
		ApproverRole string  `json:"approver_role"`
		Decision     string  `json:"decision"`
		Comments     *string `json:"comments"`
		DecidedAt    *string `json:"decided_at"`
		CreatedAt    string  `json:"created_at"`
	}
	items := []stepRow{}
	for rows.Next() {
		var s stepRow
		var decidedAt *time.Time
		var createdAt time.Time
		rows.Scan(&s.ID, &s.StepOrder, &s.StepName, &s.ApproverRole, &s.Decision, &s.Comments, &decidedAt, &createdAt)
		if decidedAt != nil {
			ts := decidedAt.Format(time.RFC3339)
			s.DecidedAt = &ts
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, s)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// ── Helpers ──────────────────────────────────────

type workflowStep struct {
	Order int
	Name  string
	Role  string
}

func getWorkflowSteps(wfType string) []workflowStep {
	switch wfType {
	case "promotion":
		return []workflowStep{
			{1, "مراجعة رئيس القسم", "department_head"},
			{2, "موافقة عميد الكلية", "college_dean"},
			{3, "مراجعة الشؤون العلمية", "scientific_affairs"},
			{4, "اعتماد المساعد العلمي", "vp_research"},
			{5, "قرار مجلس الجامعة", "council"},
		}
	case "sabbatical":
		return []workflowStep{
			{1, "مراجعة رئيس القسم", "department_head"},
			{2, "موافقة عميد الكلية", "college_dean"},
			{3, "مراجعة الشؤون العلمية", "scientific_affairs"},
			{4, "اعتماد المساعد العلمي", "vp_research"},
		}
	case "conference":
		return []workflowStep{
			{1, "مراجعة رئيس القسم", "department_head"},
			{2, "موافقة عميد الكلية", "college_dean"},
		}
	case "incentive":
		return []workflowStep{
			{1, "مراجعة الشؤون العلمية", "scientific_affairs"},
			{2, "اعتماد المساعد العلمي", "vp_research"},
		}
	default:
		return nil
	}
}

func (h *WorkflowHandler) updateRequestStatus(ctx context.Context, wfType, requestID, status string) {
	switch wfType {
	case "promotion":
		h.db.Exec(ctx, `UPDATE promotion_applications SET status = $2::promotion_status, decided_at = NOW() WHERE id = $1`, requestID, status)
	case "sabbatical":
		h.db.Exec(ctx, `UPDATE sabbatical_requests SET status = $2::sabbatical_status WHERE id = $1`, requestID, status)
	case "conference":
		h.db.Exec(ctx, `UPDATE conference_participations SET status = $2::conference_status WHERE id = $1`, requestID, status)
	}
}

func mapUserRoleToApproverRole(userRole string) string {
	switch userRole {
	case "department_head":
		return "department_head"
	case "college_dean":
		return "college_dean"
	case "scientific_affairs":
		return "scientific_affairs"
	case "dean_research":
		return "dean_research"
	case "vp_research":
		return "vp_research"
	default:
		return "none"
	}
}
