package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/publicsite"
)

// The landing-page document lives on PortalHandler so it shares the portal helpers.

type siteContentResponse struct {
	publicsite.Content
	UpdatedAt time.Time `json:"updated_at"`
}

// loadSiteContent reads and normalises the landing-page document; found is false when the row is missing.
func loadSiteContent(ctx context.Context, db *pgxpool.Pool) (siteContentResponse, bool, error) {
	var raw []byte
	var out siteContentResponse
	err := db.QueryRow(ctx, `SELECT content, updated_at FROM site_content WHERE id = 1`).Scan(&raw, &out.UpdatedAt)
	if isNoRows(err) {
		return out, false, nil
	}
	if err != nil {
		return out, false, err
	}
	content, err := decodeSiteContent(raw)
	out.Content = content
	return out, err == nil, err
}

// decodeSiteContent parses a stored document; normalising on the way out keeps lists
// as [] and strips anything a manual edit left behind.
func decodeSiteContent(raw []byte) (publicsite.Content, error) {
	var stored publicsite.Content
	if err := json.Unmarshal(raw, &stored); err != nil {
		return publicsite.Content{}, err
	}
	return publicsite.NormalizeContent(stored), nil
}

// GetSiteContent returns the public landing-page document (no authentication).
func (h *PortalHandler) GetSiteContent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()

	out, found, err := loadSiteContent(ctx, h.db)
	if err != nil {
		serverError(w, "Failed to load site content", err)
		return
	}
	if !found {
		model.RespondError(w, http.StatusNotFound, "Site content not found")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	model.RespondJSON(w, http.StatusOK, out)
}

// UpdateSiteContent replaces the landing-page document (admin and scientific affairs).
// The director's photo is managed by its own endpoints, so the stored photo id is kept.
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
	content.Director.PhotoID = ""
	raw, err := json.Marshal(content)
	if err != nil {
		serverError(w, "Failed to encode site content", err)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	var saved []byte
	var out siteContentResponse
	err = h.db.QueryRow(ctx, `
		INSERT INTO site_content (id, content, updated_by, updated_at) VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET
			content = jsonb_set(EXCLUDED.content, '{director,photo_id}',
				COALESCE(site_content.content->'director'->'photo_id', '""'::jsonb)),
			updated_by = EXCLUDED.updated_by, updated_at = NOW()
		RETURNING content, updated_at`, raw, user.UserID).Scan(&saved, &out.UpdatedAt)
	if err != nil {
		serverError(w, "Failed to save site content", err)
		return
	}
	if out.Content, err = decodeSiteContent(saved); err != nil {
		serverError(w, "Failed to read site content", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, out)
}
