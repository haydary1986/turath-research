package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

type ResearcherHandler struct {
	db *pgxpool.Pool
}

func NewResearcherHandler(db *pgxpool.Pool) *ResearcherHandler {
	return &ResearcherHandler{db: db}
}

type researcherRow struct {
	ID                string  `json:"id"`
	NameAr            string  `json:"name_ar"`
	NameEn            *string `json:"name_en"`
	Email             string  `json:"email"`
	TitleAr           *string `json:"title_ar"`
	TitleEn           *string `json:"title_en"`
	DepartmentID      *string `json:"department_id"`
	HIndex            int     `json:"h_index"`
	TotalCitations    int     `json:"total_citations"`
	TotalPublications int     `json:"total_publications"`
	Status            string  `json:"status"`
	// NeedsIdentityReview: the last sync refused a source on a name check. Staff only.
	NeedsIdentityReview bool `json:"needs_identity_review,omitempty"`
}

const researcherColumns = `id, name_ar, name_en, email, title_ar, title_en,
	department_id, h_index, total_citations, total_publications, status, COALESCE(last_sync_error, '')`

// scanResearcher hides contact details and review state from everyone but staff.
func scanResearcher(row rowScanner, staff bool) (researcherRow, error) {
	var res researcherRow
	var syncError string
	err := row.Scan(&res.ID, &res.NameAr, &res.NameEn, &res.Email, &res.TitleAr, &res.TitleEn,
		&res.DepartmentID, &res.HIndex, &res.TotalCitations, &res.TotalPublications, &res.Status, &syncError)
	if err != nil {
		return res, err
	}
	if staff {
		res.NeedsIdentityReview = syncsvc.IsIdentityCode(syncError)
	} else {
		res.Email = ""
	}
	return res, nil
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
	OrcidID      *string `json:"orcid_id"`
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

	staff := callerIsResearchStaff(r)
	conds := ""
	args := []any{}
	if search != "" {
		args = append(args, "%"+search+"%")
		n := strconv.Itoa(len(args))
		conds += ` AND (name_ar ILIKE $` + n + ` OR name_en ILIKE $` + n
		if staff {
			conds += ` OR email ILIKE $` + n
		}
		conds += `)`
	}
	if deptID != "" {
		args = append(args, deptID)
		conds += ` AND department_id = $` + strconv.Itoa(len(args))
	}

	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers WHERE 1=1`+conds, args...).Scan(&total); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to count researchers")
		return
	}

	fetchArgs := append(append([]any{}, args...), limit, offset)
	rows, err := h.db.Query(ctx, `SELECT `+researcherColumns+` FROM researchers WHERE 1=1`+conds+
		` ORDER BY name_ar ASC LIMIT $`+strconv.Itoa(len(args)+1)+` OFFSET $`+strconv.Itoa(len(args)+2), fetchArgs...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch researchers")
		return
	}
	defer rows.Close()

	researchers := []researcherRow{}
	for rows.Next() {
		res, err := scanResearcher(rows, staff)
		if err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan researcher")
			return
		}
		researchers = append(researchers, res)
	}

	model.RespondPaginated(w, http.StatusOK, researchers, total, page, limit)
}

// Get returns a single researcher by ID.
func (h *ResearcherHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := scanResearcher(h.db.QueryRow(ctx, `SELECT `+researcherColumns+` FROM researchers WHERE id = $1`, id), callerIsResearchStaff(r))
	if err != nil {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, res)
}

// orcidFromRequest canonicalizes an optional ORCID; "" clears or omits it.
func orcidFromRequest(w http.ResponseWriter, raw *string) (string, bool) {
	if raw == nil {
		return "", true
	}
	links, problems := researchlinks.Parse(researchlinks.Input{ORCID: *raw})
	if len(problems) > 0 {
		validationError(w, problems)
		return "", false
	}
	return links.ORCID, true
}

// Create adds a new researcher.
func (h *ResearcherHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createResearcherReq
	if !decodeBody(w, r, &req) {
		return
	}
	if req.NameAr == "" || req.Email == "" {
		model.RespondError(w, http.StatusBadRequest, "name_ar and email are required")
		return
	}
	orcid, ok := orcidFromRequest(w, req.OrcidID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.db.Begin(ctx)
	if err != nil {
		serverError(w, "researcher create", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit

	id := uuid.New().String()
	code, err := claimedBy(ctx, tx, id, researchlinks.Links{ORCID: orcid})
	if err != nil {
		serverError(w, "researcher create", err)
		return
	}
	if code != "" {
		model.RespondError(w, http.StatusConflict, code)
		return
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO researchers (id, name_ar, name_en, email, title_ar, title_en,
		                         department_id, orcid_id, scopus_author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)
	`, id, req.NameAr, req.NameEn, req.Email, req.TitleAr, req.TitleEn,
		req.DepartmentID, orcid, req.ScopusID); err != nil {
		model.RespondError(w, http.StatusConflict, "Researcher already exists or invalid data")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "researcher create", err)
		return
	}

	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// applyORCIDChange runs the same checks as the researcher's own form: a taken iD is
// refused, a run in flight or a recent one blocks a new iD, and old imports go.
func applyORCIDChange(ctx context.Context, tx pgx.Tx, id, orcid string) error {
	var stored *string
	var started, finished *time.Time
	var now time.Time
	if err := tx.QueryRow(ctx, `
		SELECT orcid_id, last_sync_started_at, last_sync_finished_at, NOW() FROM researchers WHERE id = $1 FOR UPDATE
	`, id).Scan(&stored, &started, &finished, &now); err != nil {
		return err
	}
	if canonicalStoredORCID(stored) == orcid {
		return nil
	}
	code, err := claimedBy(ctx, tx, id, researchlinks.Links{ORCID: orcid})
	if err != nil {
		return err
	}
	if code != "" {
		return profileConflict(code)
	}
	if orcid != "" && !syncsvc.ORCIDChangeAllowed(started, finished, now) {
		return errORCIDChangeTooSoon
	}
	return syncsvc.ResetImported(ctx, tx, id)
}

// Update modifies an existing researcher.
func (h *ResearcherHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}
	var req updateResearcherReq
	if !decodeBody(w, r, &req) {
		return
	}
	orcid, ok := orcidFromRequest(w, req.OrcidID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := h.db.Begin(ctx)
	if err != nil {
		serverError(w, "researcher update", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit

	if req.OrcidID != nil {
		var conflict profileConflict
		switch err := applyORCIDChange(ctx, tx, id, orcid); {
		case err == nil:
		case isNoRows(err):
			model.RespondError(w, http.StatusNotFound, "Researcher not found")
			return
		case errors.As(err, &conflict):
			model.RespondError(w, http.StatusConflict, string(conflict))
			return
		case errors.Is(err, errORCIDChangeTooSoon):
			model.RespondError(w, http.StatusTooManyRequests, err.Error())
			return
		default:
			serverError(w, "researcher update", err)
			return
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE researchers SET
			name_ar = COALESCE($2, name_ar),
			name_en = COALESCE($3, name_en),
			email = COALESCE($4, email),
			title_ar = COALESCE($5, title_ar),
			title_en = COALESCE($6, title_en),
			department_id = COALESCE($7, department_id),
			status = COALESCE($8::researcher_status, status),
			orcid_attested = CASE WHEN $9 THEN orcid_attested AND orcid_id IS NOT DISTINCT FROM NULLIF($10, '') ELSE orcid_attested END,
			orcid_id = CASE WHEN $9 THEN NULLIF($10, '') ELSE orcid_id END,
			updated_at = NOW()
		WHERE id = $1
	`, id, req.NameAr, req.NameEn, req.Email, req.TitleAr, req.TitleEn,
		req.DepartmentID, req.Status, req.OrcidID != nil, orcid)

	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "Update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "researcher update", err)
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
