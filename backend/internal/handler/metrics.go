package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

type MetricsHandler struct {
	db *pgxpool.Pool
}

func NewMetricsHandler(db *pgxpool.Pool) *MetricsHandler {
	return &MetricsHandler{db: db}
}

// ── Institution-wide dashboard KPIs ──────────────

type institutionKPIs struct {
	TotalResearchers     int     `json:"total_researchers"`
	ActiveResearchers    int     `json:"active_researchers"`
	TotalPublications    int     `json:"total_publications"`
	TotalCitations       int     `json:"total_citations"`
	AvgHIndex            float64 `json:"avg_h_index"`
	Q1Publications       int     `json:"q1_publications"`
	Q2Publications       int     `json:"q2_publications"`
	TotalJournalArticles int     `json:"total_journal_articles"`
	Q1Ratio              float64 `json:"q1_ratio"`
	PublicationsThisYear int     `json:"publications_this_year"`
	CitationsThisYear    int     `json:"citations_this_year"`
}

func (h *MetricsHandler) InstitutionKPIs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var kpis institutionKPIs

	// Researcher counts
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers`).Scan(&kpis.TotalResearchers)
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers WHERE status = 'active'`).Scan(&kpis.ActiveResearchers)

	// Publication totals
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM publications`).Scan(&kpis.TotalPublications)
	h.db.QueryRow(ctx, `SELECT COALESCE(SUM(citation_count), 0) FROM publications`).Scan(&kpis.TotalCitations)

	// Average h-index
	h.db.QueryRow(ctx, `SELECT COALESCE(AVG(h_index), 0) FROM researchers WHERE status = 'active'`).Scan(&kpis.AvgHIndex)

	// Journal ranking distribution
	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications p
		JOIN journals j ON j.id = p.journal_id
		WHERE j.sjr_ranking = 'Q1'
	`).Scan(&kpis.Q1Publications)
	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications p
		JOIN journals j ON j.id = p.journal_id
		WHERE j.sjr_ranking = 'Q2'
	`).Scan(&kpis.Q2Publications)
	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications WHERE publication_type = 'journal_article'
	`).Scan(&kpis.TotalJournalArticles)

	if kpis.TotalJournalArticles > 0 {
		kpis.Q1Ratio = float64(kpis.Q1Publications) / float64(kpis.TotalJournalArticles) * 100
	}

	// This year
	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications
		WHERE EXTRACT(YEAR FROM publication_date) = EXTRACT(YEAR FROM CURRENT_DATE)
	`).Scan(&kpis.PublicationsThisYear)

	model.RespondJSON(w, http.StatusOK, kpis)
}

// ── Researcher stats ─────────────────────────────

type researcherStats struct {
	ID                  string      `json:"id"`
	NameAr              string      `json:"name_ar"`
	HIndex              int         `json:"h_index"`
	TotalCitations      int         `json:"total_citations"`
	TotalPublications   int         `json:"total_publications"`
	PublicationsByYear  []yearCount `json:"publications_by_year"`
	CitationsByYear     []yearCount `json:"citations_by_year"`
	JournalDistribution []rankCount `json:"journal_distribution"`
	TopPublications     []topPub    `json:"top_publications"`
	CoAuthors           []coAuthor  `json:"co_authors"`
}

type yearCount struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type rankCount struct {
	Ranking string `json:"ranking"`
	Count   int    `json:"count"`
}

type topPub struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	DOI           *string `json:"doi"`
	CitationCount int     `json:"citation_count"`
	Year          *int    `json:"year"`
}

type coAuthor struct {
	ID     *string `json:"id"`
	Name   string  `json:"name"`
	Papers int     `json:"papers"`
}

func (h *MetricsHandler) ResearcherStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var stats researcherStats
	err := h.db.QueryRow(ctx, `
		SELECT id, name_ar, h_index, total_citations, total_publications
		FROM researchers WHERE id = $1
	`, id).Scan(&stats.ID, &stats.NameAr, &stats.HIndex, &stats.TotalCitations, &stats.TotalPublications)
	if err != nil {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}

	// Publications by year
	stats.PublicationsByYear = []yearCount{}
	rows, err := h.db.Query(ctx, `
		SELECT EXTRACT(YEAR FROM p.publication_date)::int AS year, COUNT(*)
		FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		WHERE pa.researcher_id = $1 AND p.publication_date IS NOT NULL
		GROUP BY year ORDER BY year
	`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var yc yearCount
			rows.Scan(&yc.Year, &yc.Count)
			stats.PublicationsByYear = append(stats.PublicationsByYear, yc)
		}
	}

	// Journal ranking distribution
	stats.JournalDistribution = []rankCount{}
	rows2, err := h.db.Query(ctx, `
		SELECT COALESCE(j.sjr_ranking::text, 'unranked') AS ranking, COUNT(*)
		FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		LEFT JOIN journals j ON j.id = p.journal_id
		WHERE pa.researcher_id = $1
		GROUP BY ranking ORDER BY ranking
	`, id)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var rc rankCount
			rows2.Scan(&rc.Ranking, &rc.Count)
			stats.JournalDistribution = append(stats.JournalDistribution, rc)
		}
	}

	// Top publications by citations
	stats.TopPublications = []topPub{}
	rows3, err := h.db.Query(ctx, `
		SELECT p.id, COALESCE(p.title_en, p.title_ar, ''), p.doi, p.citation_count,
		       EXTRACT(YEAR FROM p.publication_date)::int
		FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		WHERE pa.researcher_id = $1
		ORDER BY p.citation_count DESC
		LIMIT 10
	`, id)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var tp topPub
			rows3.Scan(&tp.ID, &tp.Title, &tp.DOI, &tp.CitationCount, &tp.Year)
			stats.TopPublications = append(stats.TopPublications, tp)
		}
	}

	// Co-authors
	stats.CoAuthors = []coAuthor{}
	rows4, err := h.db.Query(ctx, `
		SELECT pa2.researcher_id, pa2.author_name, COUNT(DISTINCT pa2.publication_id)
		FROM publication_authors pa1
		JOIN publication_authors pa2 ON pa2.publication_id = pa1.publication_id
		WHERE pa1.researcher_id = $1 AND pa2.researcher_id != $1
		GROUP BY pa2.researcher_id, pa2.author_name
		ORDER BY COUNT(DISTINCT pa2.publication_id) DESC
		LIMIT 20
	`, id)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var ca coAuthor
			rows4.Scan(&ca.ID, &ca.Name, &ca.Papers)
			stats.CoAuthors = append(stats.CoAuthors, ca)
		}
	}

	model.RespondJSON(w, http.StatusOK, stats)
}

// ── Department comparison ────────────────────────

type deptComparison struct {
	ID                string  `json:"id"`
	NameAr            string  `json:"name_ar"`
	ResearcherCount   int     `json:"researcher_count"`
	TotalPublications int     `json:"total_publications"`
	TotalCitations    int     `json:"total_citations"`
	AvgHIndex         float64 `json:"avg_h_index"`
	Q1Count           int     `json:"q1_count"`
}

func (h *MetricsHandler) DepartmentComparison(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	collegeID := r.URL.Query().Get("college_id")

	query := `
		SELECT d.id, d.name_ar,
		       COUNT(DISTINCT r.id) AS researcher_count,
		       COALESCE(SUM(r.total_publications), 0) AS total_publications,
		       COALESCE(SUM(r.total_citations), 0) AS total_citations,
		       COALESCE(AVG(r.h_index), 0) AS avg_h_index
		FROM departments d
		LEFT JOIN researchers r ON r.department_id = d.id AND r.status = 'active'
	`
	args := []any{}
	if collegeID != "" {
		query += ` WHERE d.college_id = $1`
		args = append(args, collegeID)
	}
	query += ` GROUP BY d.id, d.name_ar ORDER BY total_publications DESC`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch department metrics")
		return
	}
	defer rows.Close()

	depts := []deptComparison{}
	for rows.Next() {
		var d deptComparison
		rows.Scan(&d.ID, &d.NameAr, &d.ResearcherCount, &d.TotalPublications, &d.TotalCitations, &d.AvgHIndex)
		depts = append(depts, d)
	}

	model.RespondJSON(w, http.StatusOK, depts)
}

// ── Top researchers leaderboard ──────────────────

type leaderboardEntry struct {
	ID                string  `json:"id"`
	NameAr            string  `json:"name_ar"`
	NameEn            *string `json:"name_en"`
	DepartmentID      *string `json:"department_id"`
	HIndex            int     `json:"h_index"`
	TotalCitations    int     `json:"total_citations"`
	TotalPublications int     `json:"total_publications"`
}

func (h *MetricsHandler) TopResearchers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Scope filtering based on user role
	user := middleware.GetUser(r.Context())
	query := `
		SELECT id, name_ar, name_en, department_id, h_index, total_citations, total_publications
		FROM researchers WHERE status = 'active'
	`
	args := []any{}

	// Role-based scoping
	if user != nil {
		switch user.Role {
		case middleware.RoleDepartmentHead:
			if user.DepartmentID != "" {
				query += ` AND department_id = $1`
				args = append(args, user.DepartmentID)
			}
		case middleware.RoleCollegeDean:
			if user.CollegeID != "" {
				query += ` AND department_id IN (SELECT id FROM departments WHERE college_id = $1)`
				args = append(args, user.CollegeID)
			}
		}
	}

	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "citations":
		query += ` ORDER BY total_citations DESC`
	case "publications":
		query += ` ORDER BY total_publications DESC`
	default:
		query += ` ORDER BY h_index DESC`
	}
	query += ` LIMIT 20`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch leaderboard")
		return
	}
	defer rows.Close()

	entries := []leaderboardEntry{}
	for rows.Next() {
		var e leaderboardEntry
		rows.Scan(&e.ID, &e.NameAr, &e.NameEn, &e.DepartmentID, &e.HIndex, &e.TotalCitations, &e.TotalPublications)
		entries = append(entries, e)
	}

	model.RespondJSON(w, http.StatusOK, entries)
}

// ── CSV Export ───────────────────────────────────

func (h *MetricsHandler) ExportResearchersCSV(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT name_ar, COALESCE(name_en, ''), email, COALESCE(title_ar, ''),
		       h_index, total_citations, total_publications, status
		FROM researchers ORDER BY name_ar
	`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Export failed")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=researchers.csv")
	// BOM for Arabic in Excel
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	w.Write([]byte("الاسم بالعربي,الاسم بالإنجليزي,البريد,اللقب,h-index,الاستشهادات,الأبحاث,الحالة\n"))

	for rows.Next() {
		var nameAr, nameEn, email, title, status string
		var hIndex, citations, pubs int
		rows.Scan(&nameAr, &nameEn, &email, &title, &hIndex, &citations, &pubs, &status)
		line := fmt.Sprintf("%s,%s,%s,%s,%d,%d,%d,%s\n", nameAr, nameEn, email, title, hIndex, citations, pubs, status)
		w.Write([]byte(line))
	}
}
