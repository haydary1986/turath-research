package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/publicsite"
)

// Announcements (scientific activities and circulars) live on PortalHandler so
// they share the upload helpers. Public responses carry no author or account data.

type announcementAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

// publicAnnouncement is everything an anonymous visitor may see.
type publicAnnouncement struct {
	ID          string                  `json:"id"`
	Type        string                  `json:"type"`
	Title       string                  `json:"title"`
	Body        string                  `json:"body"`
	EventDate   *string                 `json:"event_date"`
	ReferenceNo *string                 `json:"reference_no"`
	Attachment  *announcementAttachment `json:"attachment"`
	Pinned      bool                    `json:"pinned"`
	PublishedAt *time.Time              `json:"published_at"`
}

// staffAnnouncement adds the editorial state staff manage.
type staffAnnouncement struct {
	publicAnnouncement
	Published  bool      `json:"published"`
	ActivityID *string   `json:"activity_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const announcementSelect = `
	SELECT a.id, a.kind, a.title, a.body, to_char(a.event_date, 'YYYY-MM-DD'), a.reference_no,
	       f.id, f.original_name, f.mime_type, f.size_bytes,
	       a.is_pinned, a.published_at, a.is_published, a.activity_id, a.created_at, a.updated_at
	FROM announcements a
	LEFT JOIN portal_files f ON f.id = a.attachment_id`

const (
	publicOrder = ` ORDER BY a.is_pinned DESC, a.published_at DESC, a.id`
	staffOrder  = ` ORDER BY a.is_published, a.is_pinned DESC, COALESCE(a.published_at, a.created_at) DESC, a.id`
)

func scanAnnouncement(row rowScanner) (staffAnnouncement, error) {
	var a staffAnnouncement
	var fileID, fileName, fileMime *string
	var fileSize *int64
	err := row.Scan(&a.ID, &a.Type, &a.Title, &a.Body, &a.EventDate, &a.ReferenceNo,
		&fileID, &fileName, &fileMime, &fileSize,
		&a.Pinned, &a.PublishedAt, &a.Published, &a.ActivityID, &a.CreatedAt, &a.UpdatedAt)
	if err == nil && fileID != nil && fileName != nil && fileMime != nil && fileSize != nil {
		a.Attachment = &announcementAttachment{ID: *fileID, Name: *fileName, MimeType: *fileMime, SizeBytes: *fileSize}
	}
	return a, err
}

// loadAnnouncement fetches one announcement; the bool is false when it does not exist.
func (h *PortalHandler) loadAnnouncement(ctx context.Context, id string) (staffAnnouncement, bool, error) {
	if _, err := uuid.Parse(id); err != nil {
		return staffAnnouncement{}, false, nil
	}
	a, err := scanAnnouncement(h.db.QueryRow(ctx, announcementSelect+` WHERE a.id = $1`, id))
	if isNoRows(err) {
		return staffAnnouncement{}, false, nil
	}
	return a, err == nil, err
}

type announcementQuery struct {
	conds []string
	args  []any
}

func (q *announcementQuery) add(cond string, v any) {
	q.args = append(q.args, v)
	q.conds = append(q.conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(q.args))))
}

func (q *announcementQuery) where() string {
	if len(q.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(q.conds, " AND ")
}

// listAnnouncements runs a filtered, paginated query and returns the page and the total.
func (h *PortalHandler) listAnnouncements(ctx context.Context, q announcementQuery, order string, page, limit int) ([]staffAnnouncement, int, error) {
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM announcements a`+q.where(), q.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(append([]any{}, q.args...), limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, announcementSelect+q.where()+order+
		` LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []staffAnnouncement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}

// ListPublicAnnouncements returns published announcements, pinned first then newest (no authentication).
func (h *PortalHandler) ListPublicAnnouncements(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	page := clampInt(params.Get("page"), 1, 1, 100000)
	limit := clampInt(params.Get("limit"), 10, 1, 50)
	q := announcementQuery{conds: []string{"a.is_published"}}
	if t := params.Get("type"); publicsite.IsType(t) {
		q.add("a.kind = ?", t)
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	items, total, err := h.listAnnouncements(ctx, q, publicOrder, page, limit)
	if err != nil {
		serverError(w, "Failed to list announcements", err)
		return
	}
	out := make([]publicAnnouncement, 0, len(items))
	for _, a := range items {
		out = append(out, a.publicAnnouncement)
	}
	w.Header().Set("Cache-Control", "no-cache")
	model.RespondPaginated(w, http.StatusOK, out, total, page, limit)
}

// GetPublicAnnouncement returns one published announcement (no authentication).
func (h *PortalHandler) GetPublicAnnouncement(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	a, found, err := h.loadAnnouncement(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load announcement", err)
		return
	}
	if !found || !a.Published {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	model.RespondJSON(w, http.StatusOK, a.publicAnnouncement)
}

// ListAnnouncements returns every announcement, drafts included (staff).
func (h *PortalHandler) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	page := clampInt(params.Get("page"), 1, 1, 100000)
	limit := clampInt(params.Get("limit"), 20, 1, 50)
	var q announcementQuery
	if t := params.Get("type"); publicsite.IsType(t) {
		q.add("a.kind = ?", t)
	}
	switch params.Get("status") {
	case "published":
		q.conds = append(q.conds, "a.is_published")
	case "draft":
		q.conds = append(q.conds, "NOT a.is_published")
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	items, total, err := h.listAnnouncements(ctx, q, staffOrder, page, limit)
	if err != nil {
		serverError(w, "Failed to list announcements", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}

// GetAnnouncement returns one announcement, published or not (staff).
func (h *PortalHandler) GetAnnouncement(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	a, found, err := h.loadAnnouncement(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load announcement", err)
		return
	}
	if !found {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	model.RespondJSON(w, http.StatusOK, a)
}
