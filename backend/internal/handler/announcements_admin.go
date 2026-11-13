package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
	"github.com/haydary1986/turath-research/backend/internal/service/publicsite"
)

// parseAnnouncement decodes, normalises and validates the body; it writes the error response itself.
func parseAnnouncement(w http.ResponseWriter, r *http.Request) (publicsite.AnnouncementInput, bool) {
	var raw publicsite.AnnouncementInput
	if !decodeBody(w, r, &raw) {
		return raw, false
	}
	in := publicsite.NormalizeAnnouncement(raw)
	if problems := publicsite.ValidateAnnouncement(in); len(problems) > 0 {
		validationError(w, problems)
		return in, false
	}
	return in, true
}

// respondAnnouncement reloads the announcement so every response has the same shape.
func (h *PortalHandler) respondAnnouncement(w http.ResponseWriter, r *http.Request, id string, status int) {
	ctx, cancel := portalContext(r)
	defer cancel()
	a, found, err := h.loadAnnouncement(ctx, id)
	if err != nil || !found {
		serverError(w, "Failed to load announcement", err)
		return
	}
	model.RespondJSON(w, status, a)
}

// CreateAnnouncement stores a new, unpublished announcement (staff).
func (h *PortalHandler) CreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	in, ok := parseAnnouncement(w, r)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	var id string
	err := h.db.QueryRow(ctx, `
		INSERT INTO announcements (kind, title, body, event_date, reference_no, created_by)
		VALUES ($1, $2, $3, NULLIF($4, '')::date, NULLIF($5, ''), $6)
		RETURNING id`, in.Type, in.Title, in.Body, in.EventDate, in.ReferenceNo, user.UserID).Scan(&id)
	if err != nil {
		serverError(w, "Failed to save announcement", err)
		return
	}
	h.respondAnnouncement(w, r, id, http.StatusCreated)
}

// UpdateAnnouncement replaces the text of an announcement (staff).
func (h *PortalHandler) UpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	in, ok := parseAnnouncement(w, r)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	tag, err := h.db.Exec(ctx, `
		UPDATE announcements SET kind = $2, title = $3, body = $4,
			event_date = NULLIF($5, '')::date, reference_no = NULLIF($6, ''), updated_at = NOW()
		WHERE id = $1`, id, in.Type, in.Title, in.Body, in.EventDate, in.ReferenceNo)
	if err != nil {
		serverError(w, "Failed to save announcement", err)
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	h.respondAnnouncement(w, r, id, http.StatusOK)
}

type announcementFlag struct {
	Published *bool `json:"published"`
	Pinned    *bool `json:"pinned"`
}

// PublishAnnouncement publishes or withdraws an announcement (staff). The first
// publication date is kept when an announcement is withdrawn and published again.
func (h *PortalHandler) PublishAnnouncement(w http.ResponseWriter, r *http.Request) {
	h.setAnnouncementFlag(w, r, func(f announcementFlag) (string, *bool) {
		return `is_published = $2, published_at = CASE WHEN $2 THEN COALESCE(published_at, NOW()) ELSE published_at END`, f.Published
	}, "published")
}

// PinAnnouncement pins or unpins an announcement (staff).
func (h *PortalHandler) PinAnnouncement(w http.ResponseWriter, r *http.Request) {
	h.setAnnouncementFlag(w, r, func(f announcementFlag) (string, *bool) {
		return `is_pinned = $2`, f.Pinned
	}, "pinned")
}

func (h *PortalHandler) setAnnouncementFlag(w http.ResponseWriter, r *http.Request,
	pick func(announcementFlag) (string, *bool), field string) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	var in announcementFlag
	if !decodeBody(w, r, &in) {
		return
	}
	set, value := pick(in)
	if value == nil {
		validationError(w, []string{field + ":" + publicsite.CodeRequired})
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	tag, err := h.db.Exec(ctx, `UPDATE announcements SET `+set+`, updated_at = NOW() WHERE id = $1`, id, *value)
	if err != nil {
		serverError(w, "Failed to update announcement", err)
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	h.respondAnnouncement(w, r, id, http.StatusOK)
}

// DeleteAnnouncement removes an announcement and its attachment (staff).
func (h *PortalHandler) DeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		serverError(w, "Failed to delete announcement", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var attachmentID *string
	err = tx.QueryRow(ctx, `DELETE FROM announcements WHERE id = $1 RETURNING attachment_id`, id).Scan(&attachmentID)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	var storedName string
	if err == nil && attachmentID != nil {
		err = tx.QueryRow(ctx, `DELETE FROM portal_files WHERE id = $1 RETURNING stored_name`, *attachmentID).Scan(&storedName)
		if isNoRows(err) {
			err = nil
		}
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		serverError(w, "Failed to delete announcement", err)
		return
	}
	if storedName != "" {
		h.removeStored(storedName)
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

// AnnounceActivity prefills an unpublished announcement from an approved scientific
// activity (staff). Asking again returns the announcement already made for it.
func (h *PortalHandler) AnnounceActivity(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	activityID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(activityID); err != nil {
		model.RespondError(w, http.StatusNotFound, "Activity not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var existing string
	err := h.db.QueryRow(ctx, `SELECT id FROM announcements WHERE activity_id = $1`, activityID).Scan(&existing)
	if err == nil {
		h.respondAnnouncement(w, r, existing, http.StatusOK)
		return
	}
	if !isNoRows(err) {
		serverError(w, "Failed to check announcements", err)
		return
	}

	var status, title, date string
	var description, presenter, location *string
	err = h.db.QueryRow(ctx, `
		SELECT status, title_ar, to_char(activity_date, 'YYYY-MM-DD'), description, presenter, location
		FROM activities WHERE id = $1`, activityID).Scan(&status, &title, &date, &description, &presenter, &location)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Activity not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load activity", err)
		return
	}
	if status != activity.StatusApproved {
		model.RespondError(w, http.StatusConflict, "activity_not_approved")
		return
	}

	in := publicsite.NormalizeAnnouncement(publicsite.AnnouncementInput{
		Type:      publicsite.TypeActivity,
		Title:     truncateRunes(title, publicsite.MaxTitle),
		Body:      truncateRunes(activityAnnouncementBody(description, presenter, location), publicsite.MaxBody),
		EventDate: date,
	})
	var id string
	err = h.db.QueryRow(ctx, `
		INSERT INTO announcements (kind, title, body, event_date, activity_id, created_by)
		VALUES ($1, $2, $3, NULLIF($4, '')::date, $5, $6)
		ON CONFLICT (activity_id) DO UPDATE SET updated_at = announcements.updated_at
		RETURNING id`, in.Type, in.Title, in.Body, in.EventDate, activityID, user.UserID).Scan(&id)
	if err != nil {
		serverError(w, "Failed to save announcement", err)
		return
	}
	h.respondAnnouncement(w, r, id, http.StatusCreated)
}

// activityAnnouncementBody drafts the announcement text from the activity record.
func activityAnnouncementBody(description, presenter, location *string) string {
	parts := []string{}
	if description != nil && strings.TrimSpace(*description) != "" {
		parts = append(parts, strings.TrimSpace(*description))
	}
	var details []string
	if presenter != nil && strings.TrimSpace(*presenter) != "" {
		details = append(details, "المحاضر: "+strings.TrimSpace(*presenter))
	}
	if location != nil && strings.TrimSpace(*location) != "" {
		details = append(details, "المكان: "+strings.TrimSpace(*location))
	}
	if len(details) > 0 {
		parts = append(parts, strings.Join(details, "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}
