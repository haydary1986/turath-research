package handler

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

// The latest publications of the university's academics, for the public landing page.
// Author names and affiliations of published papers are public scholarly data; nothing
// else about the researcher (email, ids, metrics) leaves the server.

const latestPublicationsCount = 8

// doiShape accepts a bare DOI ("10.<registrant>/<suffix>"), the form the sync stores.
var doiShape = regexp.MustCompile(`^10\.\d{4,9}/\S+$`)

type publicName struct {
	Ar string  `json:"ar"`
	En *string `json:"en"`
}

type publicPublication struct {
	ID      string      `json:"id"`
	TitleAr *string     `json:"title_ar"`
	TitleEn *string     `json:"title_en"`
	Journal *string     `json:"journal"`
	Year    *int        `json:"year"`
	DOI     *string     `json:"doi"`
	Author  publicName  `json:"author"`
	College *publicName `json:"college"`
}

// Newest first; only published records with a title and a linked, active researcher.
// The author shown is the first-listed university researcher on the paper.
const latestPublicationsQuery = `
	SELECT p.id, NULLIF(btrim(p.title_ar), ''), NULLIF(btrim(p.title_en), ''),
	       COALESCE(NULLIF(btrim(j.name), ''), NULLIF(btrim(p.conference_name), '')),
	       EXTRACT(YEAR FROM p.publication_date)::int, p.doi,
	       a.name_ar, NULLIF(btrim(a.name_en), ''), a.college_ar, NULLIF(btrim(a.college_en), '')
	FROM publications p
	CROSS JOIN LATERAL (
		SELECT r.name_ar, r.name_en, c.name_ar AS college_ar, c.name_en AS college_en
		FROM publication_authors pa
		JOIN researchers r ON r.id = pa.researcher_id AND r.status = 'active'
		LEFT JOIN departments d ON d.id = r.department_id
		LEFT JOIN colleges c ON c.id = d.college_id
		WHERE pa.publication_id = p.id
		ORDER BY pa.author_order
		LIMIT 1
	) a
	LEFT JOIN journals j ON j.id = p.journal_id
	WHERE p.status = 'published'
	  AND COALESCE(NULLIF(btrim(p.title_en), ''), NULLIF(btrim(p.title_ar), '')) IS NOT NULL
	ORDER BY p.publication_date DESC NULLS LAST, p.created_at DESC, p.id
	LIMIT $1`

func (h *PublicHandler) loadLatestPublications(ctx context.Context) ([]publicPublication, error) {
	rows, err := h.db.Query(ctx, latestPublicationsQuery, latestPublicationsCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []publicPublication{}
	for rows.Next() {
		var p publicPublication
		var collegeAr, collegeEn *string
		if err := rows.Scan(&p.ID, &p.TitleAr, &p.TitleEn, &p.Journal, &p.Year, &p.DOI,
			&p.Author.Ar, &p.Author.En, &collegeAr, &collegeEn); err != nil {
			return nil, err
		}
		if p.DOI != nil {
			doi := strings.ToLower(strings.TrimSpace(*p.DOI))
			if doiShape.MatchString(doi) {
				p.DOI = &doi
			} else {
				p.DOI = nil
			}
		}
		if collegeAr != nil {
			p.College = &publicName{Ar: *collegeAr, En: collegeEn}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LatestPublications returns the newest publications of university academics (no authentication).
func (h *PublicHandler) LatestPublications(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	content, _, err := loadSiteContent(ctx, h.db)
	if err != nil {
		serverError(w, "Failed to load site content", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	// Staff can hide the section; the endpoint then has nothing to show either.
	if content.HidePublications {
		model.RespondJSON(w, http.StatusOK, []publicPublication{})
		return
	}
	items, _, err := h.publications.get(ctx, h.loadLatestPublications)
	if err != nil {
		w.Header().Del("Cache-Control")
		serverError(w, "Failed to load publications", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}
