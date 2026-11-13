package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/publicsite"
)

// The landing-page document lives on PortalHandler so it shares the portal helpers.

type siteContentResponse struct {
	publicsite.Content
	UpdatedAt time.Time `json:"updated_at"`
}

// GetSiteContent returns the public landing-page document (no authentication).
func (h *PortalHandler) GetSiteContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()

	var raw []byte
	var out siteContentResponse
	err := h.db.QueryRow(ctx, `SELECT content, updated_at FROM site_content WHERE id = 1`).Scan(&raw, &out.UpdatedAt)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Site content not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load site content", err)
		return
	}
	var stored publicsite.Content
	if err := json.Unmarshal(raw, &stored); err != nil {
		serverError(w, "Failed to read site content", err)
		return
	}
	// Normalising on the way out keeps lists as [] and strips anything a manual edit left behind.
	out.Content = publicsite.NormalizeContent(stored)
	w.Header().Set("Cache-Control", "no-cache")
	model.RespondJSON(w, http.StatusOK, out)
}

// UpdateSiteContent replaces the landing-page document (admin and scientific affairs).
func (h *PortalHandler) UpdateSiteContent(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in publicsite.Content
	if !decodeBody(w, r, &in) {
		return
	}
	content := publicsite.NormalizeContent(in)
	if problems := publicsite.ValidateContent(content); len(problems) > 0 {
		validationError(w, problems)
		return
	}
	raw, err := json.Marshal(content)
	if err != nil {
		serverError(w, "Failed to encode site content", err)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	out := siteContentResponse{Content: content}
	err = h.db.QueryRow(ctx, `
		INSERT INTO site_content (id, content, updated_by, updated_at) VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content,
			updated_by = EXCLUDED.updated_by, updated_at = NOW()
		RETURNING updated_at`, raw, user.UserID).Scan(&out.UpdatedAt)
	if err != nil {
		serverError(w, "Failed to save site content", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, out)
}
