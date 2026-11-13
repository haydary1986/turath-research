package handler

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const (
	// RoleDirectorPhoto marks the portal file shown beside the director's message.
	RoleDirectorPhoto = "site_director_photo"
	// A portrait needs far less than a letterhead; keep public downloads light.
	directorPhotoMaxBytes = 3 << 20
)

// UploadDirectorPhoto stores (or replaces) the director's photo: images only (staff).
func (h *PortalHandler) UploadDirectorPhoto(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	stored, ok := h.storeUpload(w, r, directorPhotoMaxBytes, true)
	if !ok {
		return
	}
	oldStored, err := h.swapDirectorPhoto(r, &stored, user.UserID)
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save photo", err)
		return
	}
	if oldStored != "" {
		h.removeStored(oldStored)
	}
	h.respondSiteContent(w, r, http.StatusCreated)
}

// DeleteDirectorPhoto removes the director's photo (staff).
func (h *PortalHandler) DeleteDirectorPhoto(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	oldStored, err := h.swapDirectorPhoto(r, nil, user.UserID)
	if err != nil {
		serverError(w, "Failed to remove photo", err)
		return
	}
	if oldStored != "" {
		h.removeStored(oldStored)
	}
	h.respondSiteContent(w, r, http.StatusOK)
}

// swapDirectorPhoto points the document at the new file (nil clears it) and drops the
// previous file row; it returns the previous stored name for removal from disk.
func (h *PortalHandler) swapDirectorPhoto(r *http.Request, stored *storedFile, userID string) (string, error) {
	ctx, cancel := portalContext(r)
	defer cancel()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var oldID string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(content->'director'->>'photo_id', '') FROM site_content WHERE id = 1 FOR UPDATE`).Scan(&oldID); err != nil {
		return "", err
	}
	newID := ""
	if stored != nil {
		if err := tx.QueryRow(ctx, `
			INSERT INTO portal_files (role, original_name, stored_name, mime_type, size_bytes, uploaded_by)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			RoleDirectorPhoto, stored.originalName, stored.storedName, stored.mimeType, stored.size, userID).Scan(&newID); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE site_content SET content = jsonb_set(
			CASE WHEN jsonb_typeof(content->'director') = 'object' THEN content
			     ELSE jsonb_set(content, '{director}', '{}'::jsonb) END,
			'{director,photo_id}', to_jsonb($1::text)), updated_by = $2, updated_at = NOW()
		WHERE id = 1`, newID, userID); err != nil {
		return "", err
	}
	var oldStored string
	if _, perr := uuid.Parse(oldID); perr == nil {
		err := tx.QueryRow(ctx, `DELETE FROM portal_files WHERE id = $1 AND role = $2 RETURNING stored_name`,
			oldID, RoleDirectorPhoto).Scan(&oldStored)
		if err != nil && !isNoRows(err) {
			return "", err
		}
	}
	return oldStored, tx.Commit(ctx)
}

func (h *PortalHandler) respondSiteContent(w http.ResponseWriter, r *http.Request, status int) {
	ctx, cancel := portalContext(r)
	defer cancel()
	out, found, err := loadSiteContent(ctx, h.db)
	if err != nil || !found {
		serverError(w, "Failed to load site content", err)
		return
	}
	model.RespondJSON(w, status, out)
}

// DownloadDirectorPhoto serves the director's current photo without authentication.
// Only the file the document points at is served; replaced photos are a 404.
func (h *PortalHandler) DownloadDirectorPhoto(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var storedName, mimeType string
	err := h.db.QueryRow(ctx, `
		SELECT f.stored_name, f.mime_type
		FROM portal_files f JOIN site_content s ON s.id = 1 AND s.content->'director'->>'photo_id' = f.id::text
		WHERE f.id = $1 AND f.role = $2 AND f.request_id IS NULL`,
		fileID, RoleDirectorPhoto).Scan(&storedName, &mimeType)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load file", err)
		return
	}
	if !portal.IsImage(mimeType) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}

	f, err := os.Open(filepath.Join(h.uploadDir, filepath.Base(storedName)))
	if err != nil {
		slog.Error("portal: stored file missing", "file_id", fileID, "error", err)
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", "inline")
	// A new upload gets a new id, so the URL of a photo never changes content.
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("portal: file stream interrupted", "file_id", fileID, "error", err)
	}
}
