package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

type SearchHandler struct {
	db *pgxpool.Pool
}

func NewSearchHandler(db *pgxpool.Pool) *SearchHandler {
	return &SearchHandler{db: db}
}

type searchResult struct {
	Type  string  `json:"type"` // researcher, publication, journal
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Sub   string  `json:"sub"`
	Score float64 `json:"score"`
}

// GlobalSearch performs a unified search across researchers, publications, and journals.
func (h *SearchHandler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" || len(q) < 2 {
		model.RespondError(w, http.StatusBadRequest, "Search query must be at least 2 characters")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	pattern := "%" + q + "%"
	results := []searchResult{}

	// Search researchers (using pg_trgm similarity)
	resRows, err := h.db.Query(ctx, `
		SELECT id, name_ar, COALESCE(name_en, ''), COALESCE(title_ar, ''),
		       GREATEST(
		           similarity(name_ar, $1),
		           similarity(COALESCE(name_en, ''), $1),
		           similarity(email, $1)
		       ) AS score
		FROM researchers
		WHERE name_ar ILIKE $2 OR name_en ILIKE $2 OR email ILIKE $2
		ORDER BY score DESC
		LIMIT $3
	`, q, pattern, limit)
	if err == nil {
		defer resRows.Close()
		for resRows.Next() {
			var id, nameAr, nameEn, title string
			var score float64
			resRows.Scan(&id, &nameAr, &nameEn, &title, &score)
			display := nameAr
			if nameEn != "" {
				display += " — " + nameEn
			}
			results = append(results, searchResult{
				Type:  "researcher",
				ID:    id,
				Title: display,
				Sub:   title,
				Score: score,
			})
		}
	}

	// Search publications
	pubRows, err := h.db.Query(ctx, `
		SELECT id,
		       COALESCE(title_ar, title_en, ''),
		       COALESCE(doi, ''),
		       GREATEST(
		           similarity(COALESCE(title_ar, ''), $1),
		           similarity(COALESCE(title_en, ''), $1)
		       ) AS score
		FROM publications
		WHERE title_ar ILIKE $2 OR title_en ILIKE $2 OR doi ILIKE $2
		ORDER BY score DESC
		LIMIT $3
	`, q, pattern, limit)
	if err == nil {
		defer pubRows.Close()
		for pubRows.Next() {
			var id, title, doi string
			var score float64
			pubRows.Scan(&id, &title, &doi, &score)
			results = append(results, searchResult{
				Type:  "publication",
				ID:    id,
				Title: title,
				Sub:   doi,
				Score: score,
			})
		}
	}

	// Search journals
	jRows, err := h.db.Query(ctx, `
		SELECT id, name, COALESCE(issn, ''), COALESCE(sjr_ranking::text, ''),
		       similarity(name, $1) AS score
		FROM journals
		WHERE name ILIKE $2 OR issn ILIKE $2
		ORDER BY score DESC
		LIMIT $3
	`, q, pattern, limit/2)
	if err == nil {
		defer jRows.Close()
		for jRows.Next() {
			var id, name, issn, ranking string
			var score float64
			jRows.Scan(&id, &name, &issn, &ranking, &score)
			sub := issn
			if ranking != "" {
				sub += " (" + ranking + ")"
			}
			results = append(results, searchResult{
				Type:  "journal",
				ID:    id,
				Title: name,
				Sub:   sub,
				Score: score,
			})
		}
	}

	model.RespondJSON(w, http.StatusOK, results)
}
