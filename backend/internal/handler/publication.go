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

type PublicationHandler struct {
	db *pgxpool.Pool
}

func NewPublicationHandler(db *pgxpool.Pool) *PublicationHandler {
	return &PublicationHandler{db: db}
}

type publicationRow struct {
	ID              string  `json:"id"`
	DOI             *string `json:"doi"`
	TitleAr         *string `json:"title_ar"`
	TitleEn         *string `json:"title_en"`
	PublicationDate *string `json:"publication_date"`
	PublicationType string  `json:"publication_type"`
	JournalID       *string `json:"journal_id"`
	Status          string  `json:"status"`
	CitationCount   int     `json:"citation_count"`
	Language        string  `json:"language"`
}

type createPublicationReq struct {
	DOI             *string `json:"doi"`
	TitleAr         *string `json:"title_ar"`
	TitleEn         *string `json:"title_en"`
	AbstractAr      *string `json:"abstract_ar"`
	AbstractEn      *string `json:"abstract_en"`
	PublicationDate *string `json:"publication_date"`
	PublicationType *string `json:"publication_type"`
	JournalID       *string `json:"journal_id"`
	ConferenceName  *string `json:"conference_name"`
	Volume          *string `json:"volume"`
	Issue           *string `json:"issue"`
	Pages           *string `json:"pages"`
	URL             *string `json:"url"`
	Language        *string `json:"language"`
}

// List returns paginated publications.
func (h *PublicationHandler) List(w http.ResponseWriter, r *http.Request) {
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
	year := r.URL.Query().Get("year")
	pubType := r.URL.Query().Get("type")

	countQuery := `SELECT COUNT(*) FROM publications WHERE 1=1`
	args := []any{}
	argN := 1

	if search != "" {
		countQuery += ` AND (title_ar ILIKE $` + strconv.Itoa(argN) + ` OR title_en ILIKE $` + strconv.Itoa(argN) + `)`
		args = append(args, "%"+search+"%")
		argN++
	}
	if year != "" {
		countQuery += ` AND EXTRACT(YEAR FROM publication_date) = $` + strconv.Itoa(argN)
		args = append(args, year)
		argN++
	}
	if pubType != "" {
		countQuery += ` AND publication_type = $` + strconv.Itoa(argN) + `::publication_type`
		args = append(args, pubType)
		argN++
	}

	var total int
	if err := h.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to count publications")
		return
	}

	query := `SELECT id, doi, title_ar, title_en,
	                 publication_date::text, publication_type, journal_id,
	                 status, citation_count, language
	          FROM publications WHERE 1=1`
	fetchArgs := []any{}
	fetchN := 1

	if search != "" {
		query += ` AND (title_ar ILIKE $` + strconv.Itoa(fetchN) + ` OR title_en ILIKE $` + strconv.Itoa(fetchN) + `)`
		fetchArgs = append(fetchArgs, "%"+search+"%")
		fetchN++
	}
	if year != "" {
		query += ` AND EXTRACT(YEAR FROM publication_date) = $` + strconv.Itoa(fetchN)
		fetchArgs = append(fetchArgs, year)
		fetchN++
	}
	if pubType != "" {
		query += ` AND publication_type = $` + strconv.Itoa(fetchN) + `::publication_type`
		fetchArgs = append(fetchArgs, pubType)
		fetchN++
	}

	query += ` ORDER BY publication_date DESC NULLS LAST LIMIT $` + strconv.Itoa(fetchN) + ` OFFSET $` + strconv.Itoa(fetchN+1)
	fetchArgs = append(fetchArgs, limit, offset)

	rows, err := h.db.Query(ctx, query, fetchArgs...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch publications")
		return
	}
	defer rows.Close()

	pubs := []publicationRow{}
	for rows.Next() {
		var p publicationRow
		if err := rows.Scan(&p.ID, &p.DOI, &p.TitleAr, &p.TitleEn,
			&p.PublicationDate, &p.PublicationType, &p.JournalID,
			&p.Status, &p.CitationCount, &p.Language); err != nil {
			model.RespondError(w, http.StatusInternalServerError, "Failed to scan publication")
			return
		}
		pubs = append(pubs, p)
	}

	model.RespondPaginated(w, http.StatusOK, pubs, total, page, limit)
}

// Get returns a single publication with authors.
func (h *PublicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var p struct {
		publicationRow
		AbstractAr     *string `json:"abstract_ar"`
		AbstractEn     *string `json:"abstract_en"`
		ConferenceName *string `json:"conference_name"`
		Volume         *string `json:"volume"`
		Issue          *string `json:"issue"`
		Pages          *string `json:"pages"`
		URL            *string `json:"url"`
	}

	err := h.db.QueryRow(ctx, `
		SELECT id, doi, title_ar, title_en, publication_date::text,
		       publication_type, journal_id, status, citation_count, language,
		       abstract_ar, abstract_en, conference_name, volume, issue, pages, url
		FROM publications WHERE id = $1
	`, id).Scan(&p.ID, &p.DOI, &p.TitleAr, &p.TitleEn, &p.PublicationDate,
		&p.PublicationType, &p.JournalID, &p.Status, &p.CitationCount, &p.Language,
		&p.AbstractAr, &p.AbstractEn, &p.ConferenceName, &p.Volume, &p.Issue, &p.Pages, &p.URL)

	if err != nil {
		model.RespondError(w, http.StatusNotFound, "Publication not found")
		return
	}

	// Fetch authors
	authorRows, err := h.db.Query(ctx, `
		SELECT pa.author_name, pa.author_order, pa.is_corresponding, pa.researcher_id
		FROM publication_authors pa
		WHERE pa.publication_id = $1
		ORDER BY pa.author_order
	`, id)
	if err == nil {
		defer authorRows.Close()
		type author struct {
			Name            string  `json:"name"`
			Order           int     `json:"order"`
			IsCorresponding bool    `json:"is_corresponding"`
			ResearcherID    *string `json:"researcher_id"`
		}
		authors := []author{}
		for authorRows.Next() {
			var a author
			authorRows.Scan(&a.Name, &a.Order, &a.IsCorresponding, &a.ResearcherID)
			authors = append(authors, a)
		}

		result := map[string]any{
			"publication": p,
			"authors":     authors,
		}
		model.RespondJSON(w, http.StatusOK, result)
		return
	}

	model.RespondJSON(w, http.StatusOK, p)
}

// Create adds a new publication.
func (h *PublicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPublicationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.TitleAr == nil && req.TitleEn == nil {
		model.RespondError(w, http.StatusBadRequest, "At least one title (title_ar or title_en) is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	pubType := "journal_article"
	if req.PublicationType != nil {
		pubType = *req.PublicationType
	}
	lang := "ar"
	if req.Language != nil {
		lang = *req.Language
	}

	_, err := h.db.Exec(ctx, `
		INSERT INTO publications (id, doi, title_ar, title_en, abstract_ar, abstract_en,
		                          publication_date, publication_type, journal_id,
		                          conference_name, volume, issue, pages, url, language)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::publication_type,$9,$10,$11,$12,$13,$14,$15)
	`, id, req.DOI, req.TitleAr, req.TitleEn, req.AbstractAr, req.AbstractEn,
		req.PublicationDate, pubType, req.JournalID,
		req.ConferenceName, req.Volume, req.Issue, req.Pages, req.URL, lang)

	if err != nil {
		model.RespondError(w, http.StatusConflict, "Publication already exists or invalid data")
		return
	}

	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// Delete removes a publication.
func (h *PublicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := h.db.Exec(ctx, `DELETE FROM publications WHERE id = $1`, id)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Publication not found")
		return
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": id})
}
