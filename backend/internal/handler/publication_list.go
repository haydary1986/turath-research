package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

const (
	pubListDefaultLimit = 20
	pubListMaxLimit     = 100
	pubMinYear          = 1900
	pubMaxYear          = 2100
	pubMaxTextRunes     = 200
	pubAuthorsShown     = 4
)

// publicationTypes mirrors the publication_type enum.
var publicationTypes = map[string]bool{
	"journal_article": true, "conference_paper": true, "book_chapter": true, "book": true,
	"preprint": true, "thesis": true, "patent": true, "report": true,
}

// publicationSorts maps the sort parameter to a fixed ORDER BY clause (never user text).
var publicationSorts = map[string]string{
	"newest": "p.publication_date DESC NULLS LAST, p.id",
	"oldest": "p.publication_date ASC NULLS LAST, p.id",
	"cited":  "p.citation_count DESC, p.publication_date DESC NULLS LAST, p.id",
	"title":  "lower(COALESCE(NULLIF(p.title_ar, ''), p.title_en)) ASC NULLS LAST, p.id",
}

// publicationFilter is the validated query of the publications list.
type publicationFilter struct {
	text, author, journal           string
	researcherID, collegeID, deptID string
	yearFrom, yearTo                int
	pubType, sort                   string
	page, limit                     int
}

// parsePublicationFilter validates the query string; problems are "field:code" entries.
func parsePublicationFilter(q url.Values) (publicationFilter, []string) {
	f := publicationFilter{sort: "newest", page: 1, limit: pubListDefaultLimit}
	var problems []string

	text := func(key string) string {
		v := strings.TrimSpace(q.Get(key))
		if len([]rune(v)) > pubMaxTextRunes {
			problems = append(problems, key+":too_long")
			return ""
		}
		return v
	}
	f.text, f.author, f.journal = text("q"), text("author"), text("journal")

	for key, dst := range map[string]*string{"researcher_id": &f.researcherID, "college_id": &f.collegeID, "department_id": &f.deptID} {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			if _, err := uuid.Parse(v); err != nil {
				problems = append(problems, key+":invalid")
			} else {
				*dst = v
			}
		}
	}

	year := func(key string) int {
		v := strings.TrimSpace(q.Get(key))
		if v == "" {
			return 0
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < pubMinYear || n > pubMaxYear {
			problems = append(problems, key+":invalid")
			return 0
		}
		return n
	}
	if y := year("year"); y != 0 {
		f.yearFrom, f.yearTo = y, y
	} else {
		f.yearFrom, f.yearTo = year("year_from"), year("year_to")
	}
	if f.yearFrom != 0 && f.yearTo != 0 && f.yearFrom > f.yearTo {
		problems = append(problems, "year_to:before_year_from")
	}

	if t := q.Get("type"); t != "" {
		if !publicationTypes[t] {
			problems = append(problems, "type:invalid")
		} else {
			f.pubType = t
		}
	}
	if s := q.Get("sort"); s != "" {
		if _, ok := publicationSorts[s]; !ok {
			problems = append(problems, "sort:invalid")
		} else {
			f.sort = s
		}
	}
	if n, err := strconv.Atoi(q.Get("page")); err == nil && n > 0 {
		f.page = n
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= pubListMaxLimit {
		f.limit = n
	}
	return f, problems
}

// likePattern escapes LIKE wildcards so user text matches literally.
func likePattern(text string) string {
	return "%" + likeEscaper.Replace(text) + "%"
}

// where builds the WHERE clause and its arguments; every value is a bind parameter.
func (f publicationFilter) where() (string, []any) {
	conds := []string{"TRUE"}
	args := []any{}
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	// Authors of the publication who are (or match) the university's researchers.
	const byAuthor = `EXISTS (SELECT 1 FROM publication_authors pa LEFT JOIN researchers r ON r.id = pa.researcher_id
		LEFT JOIN departments d ON d.id = r.department_id WHERE pa.publication_id = p.id AND `

	if f.text != "" {
		add("(p.title_ar ILIKE ? OR p.title_en ILIKE ? OR p.doi ILIKE ?)", likePattern(f.text))
	}
	if f.author != "" {
		add(byAuthor+"(pa.author_name ILIKE ? OR r.name_ar ILIKE ? OR r.name_en ILIKE ?))", likePattern(f.author))
	}
	if f.journal != "" {
		add("(p.metadata->>'journal_name' ILIKE ? OR p.conference_name ILIKE ?)", likePattern(f.journal))
	}
	if f.researcherID != "" {
		add(byAuthor+"pa.researcher_id = ?)", f.researcherID)
	}
	if f.deptID != "" {
		add(byAuthor+"r.department_id = ?)", f.deptID)
	}
	if f.collegeID != "" {
		add(byAuthor+"d.college_id = ?)", f.collegeID)
	}
	if f.yearFrom != 0 {
		add("p.publication_date >= make_date(?, 1, 1)", f.yearFrom)
	}
	if f.yearTo != 0 {
		add("p.publication_date < make_date(? + 1, 1, 1)", f.yearTo)
	}
	if f.pubType != "" {
		add("p.publication_type = ?::publication_type", f.pubType)
	}
	return strings.Join(conds, " AND "), args
}

type publicationAuthor struct {
	Name         string  `json:"name"`
	ResearcherID *string `json:"researcher_id"`
	NameEn       *string `json:"name_en,omitempty"`
}

type publicationListRow struct {
	publicationRow
	JournalName  *string             `json:"journal_name"`
	Authors      []publicationAuthor `json:"authors"`
	AuthorsCount int                 `json:"authors_count"`
}

// List returns publications matching the filters, newest first by default.
func (h *PublicationHandler) List(w http.ResponseWriter, r *http.Request) {
	f, problems := parsePublicationFilter(r.URL.Query())
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	where, args := f.where()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM publications p WHERE `+where, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count publications", err)
		return
	}

	n := len(args)
	query := `
		SELECT p.id, p.doi, p.title_ar, p.title_en, p.publication_date::text, p.publication_type, p.journal_id,
		       p.status, p.citation_count, p.language,
		       COALESCE(NULLIF(j.name, ''), NULLIF(p.metadata->>'journal_name', ''), NULLIF(p.conference_name, '')),
		       COALESCE((SELECT json_agg(json_build_object(
		                     'name', COALESCE(NULLIF(r.name_ar, ''), a.author_name),
		                     'name_en', r.name_en, 'researcher_id', a.researcher_id) ORDER BY a.author_order)
		                 FROM (SELECT * FROM publication_authors WHERE publication_id = p.id
		                       ORDER BY researcher_id IS NULL, author_order LIMIT ` + strconv.Itoa(pubAuthorsShown) + `) a
		                 LEFT JOIN researchers r ON r.id = a.researcher_id), '[]'),
		       (SELECT COUNT(*) FROM publication_authors WHERE publication_id = p.id)
		FROM publications p LEFT JOIN journals j ON j.id = p.journal_id
		WHERE ` + where + `
		ORDER BY ` + publicationSorts[f.sort] + `
		LIMIT $` + strconv.Itoa(n+1) + ` OFFSET $` + strconv.Itoa(n+2)
	rows, err := h.db.Query(ctx, query, append(args, f.limit, (f.page-1)*f.limit)...)
	if err != nil {
		serverError(w, "Failed to fetch publications", err)
		return
	}
	defer rows.Close()

	pubs := []publicationListRow{}
	for rows.Next() {
		var p publicationListRow
		var authors []byte
		if err := rows.Scan(&p.ID, &p.DOI, &p.TitleAr, &p.TitleEn, &p.PublicationDate, &p.PublicationType,
			&p.JournalID, &p.Status, &p.CitationCount, &p.Language, &p.JournalName, &authors, &p.AuthorsCount); err != nil {
			serverError(w, "Failed to scan publication", err)
			return
		}
		if err := json.Unmarshal(authors, &p.Authors); err != nil {
			serverError(w, "Failed to read authors", err)
			return
		}
		pubs = append(pubs, p)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to fetch publications", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, pubs, total, f.page, f.limit)
}

type facetCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type publicationFacets struct {
	Years []facetCount `json:"years"`
	Types []facetCount `json:"types"`
}

// Facets lists the years and types present, with counts, for the filter menus.
func (h *PublicationHandler) Facets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	load := func(sql string) ([]facetCount, error) {
		rows, err := h.db.Query(ctx, sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []facetCount{}
		for rows.Next() {
			var f facetCount
			if err := rows.Scan(&f.Value, &f.Count); err != nil {
				return nil, err
			}
			out = append(out, f)
		}
		return out, rows.Err()
	}
	years, err := load(`SELECT EXTRACT(YEAR FROM publication_date)::int::text, COUNT(*) FROM publications
		WHERE publication_date IS NOT NULL GROUP BY 1 ORDER BY 1 DESC`)
	if err != nil {
		serverError(w, "Failed to load years", err)
		return
	}
	types, err := load(`SELECT publication_type::text, COUNT(*) FROM publications GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		serverError(w, "Failed to load types", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, publicationFacets{Years: years, Types: types})
}
