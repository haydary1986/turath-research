package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

type ResearcherHandler struct {
	db *pgxpool.Pool
}

func NewResearcherHandler(db *pgxpool.Pool) *ResearcherHandler {
	return &ResearcherHandler{db: db}
}

type researcherRow struct {
	ID               string  `json:"id"`
	NameAr           string  `json:"name_ar"`
	NameEn           *string `json:"name_en"`
	Email            string  `json:"email"`
	TitleAr          *string `json:"title_ar"`
	TitleEn          *string `json:"title_en"`
	DepartmentID     *string `json:"department_id"`
	HIndex           int     `json:"h_index"`
	TotalCitations   int     `json:"total_citations"`
	TotalPublications int    `json:"total_publications"`
	Status           string  `json:"status"`
}

type createResearcherReq struct {
	NameAr       string  `json:"name_ar"`
	NameEn       *string `json:"name_en"`
	Email        string  `json:"email"`
	TitleAr      *string `json:"title_ar"`
	TitleEn      *string `json:"title_en"`
	DepartmentID *string `json:"department_id"`
	OrcidID      *string `json:"orcid_id"`
	ScopusID     *string `json:"scopus_author_id"`
}

type updateResearcherReq struct {
	NameAr       *string `json:"name_ar"`
	NameEn       *string `json:"name_en"`
	Email        *string `json:"email"`
	TitleAr      *string `json:"title_ar"`
	TitleEn      *string `json:"title_en"`
	DepartmentID *string `json:"department_id"`
	Status       *string `json:"status"`
}

// List returns paginated researchers.
func (h *ResearcherHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	search := r.URL.Query().Get("q")
	deptID := r.URL.Query().Get("department_id")

	// Count total
	countQuery := `SELECT COUNT(*) FROM researchers WHERE 1=1`
	args := []any{}
	argN := 1

	if search != "" {
		countQuery += ` AND (name_ar ILIKE $` + strconv.Itoa(argN) + ` OR name_en ILIKE $` + strconv.Itoa(argN) + ` OR email ILIKE $` + strconv.Itoa(argN) + `)`
		args = append(args, "%"+search+"%")
		argN++
	}
	if deptID != "" {
		countQuery += ` AND department_id = $` + strconv.Itoa(argN)
		args = append(args, deptID)
		argN++
	}

	var total int
	if err := h.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to count researchers")
		return
	}

	// Fetch rows
	query := `SELECT id, name_ar, name_en, email, title_ar, title_en,
	                 department_id, h_index, total_citations, total_publications, status
	          FROM researchers WHERE 1=1`
	fetchArgs := []any{}
	fetchN := 1

	if search != "" {
		query += ` AND (name_ar ILIKE $` + strconv.Itoa(fetchN) + ` OR name_en ILIKE $` + strconv.Itoa(fetchN) + ` OR email ILIKE $` + strconv.Itoa(fetchN) + `)`
		fetchArgs = append(fetchArgs, "%"+search+"%")
		fetchN++
	}
	if deptID != "" {
		query += ` AND department_id = $` + strconv.Itoa(fetchN)
		fetchArgs = append(fetchArgs, deptID)
		fetchN++
	}

	query += ` ORDER BY name_ar ASC LIMIT $` + strconv.Itoa(fetchN) + ` OFFSET $` + strconv.Itoa(fetchN+1)
	fetchArgs = append(fetchArgs, limit, offset)

	rows, err := h.db.Query(ctx, query, fetchArgs...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch researchers")
		return
	}
	defer rows.Close()

	researchers := []researcherRow{}
	for rows.Next() {
		var r researcherRow
		if err := rows.Scan(&r.ID, &r.NameAr, &r.NameEn, &r.Email, &r.TitleAr, &r.TitleEn,
			&r.DepartmentID, &r.HIndex, &r.TotalCitations, &r.TotalPublications, &r.Status); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan researcher")
			return
		}
		researchers = append(researchers, r)
	}

	model.RespondPaginated(w, http.StatusOK, researchers, total, page, limit)
}

// Get returns a single researcher by ID.
func (h *ResearcherHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var res researcherRow
	err := h.db.QueryRow(ctx, `
		SELECT id, name_ar, name_en, email, title_ar, title_en,
		       department_id, h_index, total_citations, total_publications, status
		FROM researchers WHERE id = $1
	`, id).Scan(&res.ID, &res.NameAr, &res.NameEn, &res.Email, &res.TitleAr, &res.TitleEn,
		&res.DepartmentID, &res.HIndex, &res.TotalCitations, &res.TotalPublications, &res.Status)

	if err != nil {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, res)
}

// Create adds a new researcher.
func (h *ResearcherHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createResearcherReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.NameAr == "" || req.Email == "" {
		model.RespondError(w, http.StatusBadRequest, "name_ar and email are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	_, err := h.db.Exec(ctx, `
		INSERT INTO researchers (id, name_ar, name_en, email, title_ar, title_en,
		                         department_id, orcid_id, scopus_author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, req.NameAr, req.NameEn, req.Email, req.TitleAr, req.TitleEn,
		req.DepartmentID, req.OrcidID, req.ScopusID)

	if err != nil {
		model.RespondError(w, http.StatusConflict, "Researcher already exists or invalid data")
		return
	}

	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// Update modifies an existing researcher.
func (h *ResearcherHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateResearcherReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := h.db.Exec(ctx, `
		UPDATE researchers SET
			name_ar = COALESCE($2, name_ar),
			name_en = COALESCE($3, name_en),
			email = COALESCE($4, email),
			title_ar = COALESCE($5, title_ar),
			title_en = COALESCE($6, title_en),
			department_id = COALESCE($7, department_id),
			status = COALESCE($8, status)::researcher_status,
			updated_at = NOW()
		WHERE id = $1
	`, id, req.NameAr, req.NameEn, req.Email, req.TitleAr, req.TitleEn,
		req.DepartmentID, req.Status)

	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "Update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{"id": id})
}

// Delete removes a researcher.
func (h *ResearcherHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := h.db.Exec(ctx, `DELETE FROM researchers WHERE id = $1`, id)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": id})
}
