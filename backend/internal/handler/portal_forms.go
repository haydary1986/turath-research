package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// Official blank forms: scientific affairs uploads one PDF per request kind and
// every signed-in member can download it, fill it and have it signed before applying.

type formTemplate struct {
	Kind         string    `json:"kind"`
	FileID       string    `json:"file_id"`
	OriginalName string    `json:"original_name"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ListFormTemplates returns the official form available for each kind.
func (h *PortalHandler) ListFormTemplates(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		SELECT t.kind, t.file_id, f.original_name, t.updated_at
		FROM form_templates t JOIN portal_files f ON f.id = t.file_id
		ORDER BY t.kind`)
	if err != nil {
		serverError(w, "Failed to list forms", err)
		return
	}
	defer rows.Close()
	items := []formTemplate{}
	for rows.Next() {
		var f formTemplate
		if err := rows.Scan(&f.Kind, &f.FileID, &f.OriginalName, &f.UpdatedAt); err != nil {
			serverError(w, "Failed to read forms", err)
			return
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read forms", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// UploadFormTemplate replaces the official PDF form of one kind (staff only).
func (h *PortalHandler) UploadFormTemplate(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	kind := chi.URLParam(r, "kind")
	if !portal.IsKind(kind) {
		model.RespondError(w, http.StatusNotFound, "Unknown form")
		return
	}
	stored, ok := h.storeUpload(w, r, portal.MaxAttachmentBytes, false)
	if !ok {
		return
	}
	if stored.mimeType != "application/pdf" {
		h.removeStored(stored.storedName)
		model.RespondError(w, http.StatusUnsupportedMediaType, "unsupported_file_type")
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save form", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var oldFileID *string
	if err := tx.QueryRow(ctx, `SELECT file_id FROM form_templates WHERE kind = $1`, kind).Scan(&oldFileID); err != nil && !isNoRows(err) {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save form", err)
		return
	}
	var out formTemplate
	err = tx.QueryRow(ctx, `
		WITH f AS (
			INSERT INTO portal_files (role, original_name, stored_name, mime_type, size_bytes, uploaded_by)
			VALUES ('form_template', $2, $3, $4, $5, $6)
			RETURNING id, original_name
		), t AS (
			INSERT INTO form_templates (kind, file_id, uploaded_by)
			SELECT $1, id, $6 FROM f
			ON CONFLICT (kind) DO UPDATE SET file_id = EXCLUDED.file_id,
				uploaded_by = EXCLUDED.uploaded_by, updated_at = NOW()
			RETURNING kind, file_id, updated_at
		)
		SELECT t.kind, t.file_id, f.original_name, t.updated_at FROM t, f
	`, kind, stored.originalName, stored.storedName, stored.mimeType, stored.size, user.UserID).
		Scan(&out.Kind, &out.FileID, &out.OriginalName, &out.UpdatedAt)
	if err == nil && oldFileID != nil {
		_, err = tx.Exec(ctx, `DELETE FROM portal_files WHERE id = $1`, *oldFileID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save form", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, out)
}

// DeleteFormTemplate withdraws the official form of one kind (staff only).
func (h *PortalHandler) DeleteFormTemplate(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	ctx, cancel := portalContext(r)
	defer cancel()
	var storedName string
	err := h.db.QueryRow(ctx, `
		DELETE FROM portal_files WHERE id = (SELECT file_id FROM form_templates WHERE kind = $1)
		RETURNING stored_name`, kind).Scan(&storedName)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Form not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to delete form", err)
		return
	}
	h.removeStored(storedName)
	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": kind})
}
