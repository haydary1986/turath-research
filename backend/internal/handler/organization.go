package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

type OrganizationHandler struct {
	db *pgxpool.Pool
}

func NewOrganizationHandler(db *pgxpool.Pool) *OrganizationHandler {
	return &OrganizationHandler{db: db}
}

// ── Institutions ─────────────────────────────────

type institutionRow struct {
	ID     string  `json:"id"`
	NameAr string  `json:"name_ar"`
	NameEn *string `json:"name_en"`
	Domain *string `json:"domain"`
}

func (h *OrganizationHandler) ListInstitutions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `SELECT id, name_ar, name_en, domain FROM institutions ORDER BY name_ar`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch institutions")
		return
	}
	defer rows.Close()

	items := []institutionRow{}
	for rows.Next() {
		var i institutionRow
		if err := rows.Scan(&i.ID, &i.NameAr, &i.NameEn, &i.Domain); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan institution")
			return
		}
		items = append(items, i)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

func (h *OrganizationHandler) CreateInstitution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NameAr string  `json:"name_ar"`
		NameEn *string `json:"name_en"`
		Domain *string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.NameAr == "" {
		model.RespondError(w, http.StatusBadRequest, "name_ar is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	_, err := h.db.Exec(ctx, `INSERT INTO institutions (id, name_ar, name_en, domain) VALUES ($1,$2,$3,$4)`,
		id, req.NameAr, req.NameEn, req.Domain)
	if err != nil {
		model.RespondError(w, http.StatusConflict, "Institution already exists")
		return
	}
	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ── Colleges ─────────────────────────────────────

type collegeRow struct {
	ID            string  `json:"id"`
	NameAr        string  `json:"name_ar"`
	NameEn        *string `json:"name_en"`
	Code          *string `json:"code"`
	InstitutionID string  `json:"institution_id"`
}

func (h *OrganizationHandler) ListColleges(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	instID := r.URL.Query().Get("institution_id")
	query := `SELECT id, name_ar, name_en, code, institution_id FROM colleges`
	args := []any{}

	if instID != "" {
		query += ` WHERE institution_id = $1`
		args = append(args, instID)
	}
	query += ` ORDER BY name_ar`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch colleges")
		return
	}
	defer rows.Close()

	items := []collegeRow{}
	for rows.Next() {
		var c collegeRow
		if err := rows.Scan(&c.ID, &c.NameAr, &c.NameEn, &c.Code, &c.InstitutionID); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan college")
			return
		}
		items = append(items, c)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

func (h *OrganizationHandler) CreateCollege(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NameAr        string  `json:"name_ar"`
		NameEn        *string `json:"name_en"`
		Code          *string `json:"code"`
		InstitutionID string  `json:"institution_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.NameAr == "" || req.InstitutionID == "" {
		model.RespondError(w, http.StatusBadRequest, "name_ar and institution_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	_, err := h.db.Exec(ctx,
		`INSERT INTO colleges (id, name_ar, name_en, code, institution_id) VALUES ($1,$2,$3,$4,$5)`,
		id, req.NameAr, req.NameEn, req.Code, req.InstitutionID)
	if err != nil {
		model.RespondError(w, http.StatusConflict, "College already exists or invalid institution")
		return
	}
	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ── Departments ──────────────────────────────────

type departmentRow struct {
	ID        string  `json:"id"`
	NameAr    string  `json:"name_ar"`
	NameEn    *string `json:"name_en"`
	Code      *string `json:"code"`
	CollegeID string  `json:"college_id"`
}

func (h *OrganizationHandler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	collegeID := r.URL.Query().Get("college_id")
	query := `SELECT id, name_ar, name_en, code, college_id FROM departments`
	args := []any{}

	if collegeID != "" {
		query += ` WHERE college_id = $1`
		args = append(args, collegeID)
	}
	query += ` ORDER BY name_ar`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch departments")
		return
	}
	defer rows.Close()

	items := []departmentRow{}
	for rows.Next() {
		var d departmentRow
		if err := rows.Scan(&d.ID, &d.NameAr, &d.NameEn, &d.Code, &d.CollegeID); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan department")
			return
		}
		items = append(items, d)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

func (h *OrganizationHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NameAr    string  `json:"name_ar"`
		NameEn    *string `json:"name_en"`
		Code      *string `json:"code"`
		CollegeID string  `json:"college_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.NameAr == "" || req.CollegeID == "" {
		model.RespondError(w, http.StatusBadRequest, "name_ar and college_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	_, err := h.db.Exec(ctx,
		`INSERT INTO departments (id, name_ar, name_en, code, college_id) VALUES ($1,$2,$3,$4,$5)`,
		id, req.NameAr, req.NameEn, req.Code, req.CollegeID)
	if err != nil {
		model.RespondError(w, http.StatusConflict, "Department already exists or invalid college")
		return
	}
	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}
