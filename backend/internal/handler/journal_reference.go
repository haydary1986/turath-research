package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

// Read-only browsing of the downloaded predatory/hijacked reference lists, so
// every member can look a journal or publisher up before choosing where to publish.

var referenceListTypes = map[string]struct{}{
	"predatory_journal": {}, "predatory_publisher": {}, "hijacked": {},
}

type referenceEntry struct {
	Name          string    `json:"name"`
	URL           *string   `json:"url"`
	ListType      string    `json:"list_type"`
	AuthenticName *string   `json:"authentic_name"`
	AuthenticURL  *string   `json:"authentic_url"`
	Source        string    `json:"source"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// ListReferenceEntries pages through one reference list, optionally filtered by name or URL.
func (h *PortalHandler) ListReferenceEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	listType := q.Get("type")
	if _, ok := referenceListTypes[listType]; !ok {
		model.RespondError(w, http.StatusBadRequest, "type must be predatory_journal, predatory_publisher or hijacked")
		return
	}
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 50, 1, 100)

	where := " WHERE list_type = $1"
	args := []any{listType}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		if len([]rune(text)) > 200 {
			model.RespondError(w, http.StatusBadRequest, "query is too long")
			return
		}
		args = append(args, "%"+likeEscaper.Replace(text)+"%")
		where += " AND (name ILIKE $2 OR url ILIKE $2 OR authentic_name ILIKE $2)"
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM journal_reference_list`+where, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count list", err)
		return
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, `
		SELECT name, url, list_type, authentic_name, authentic_url, source, fetched_at
		FROM journal_reference_list`+where+
		` ORDER BY name LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	defer rows.Close()
	items := []referenceEntry{}
	for rows.Next() {
		var e referenceEntry
		if err := rows.Scan(&e.Name, &e.URL, &e.ListType, &e.AuthenticName, &e.AuthenticURL, &e.Source, &e.FetchedAt); err != nil {
			serverError(w, "Failed to read list", err)
			return
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
