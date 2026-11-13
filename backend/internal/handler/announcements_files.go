package handler

import (
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// RoleAnnouncement marks a portal file attached to an announcement.
const RoleAnnouncement = "announcement"

// UploadAnnouncementAttachment attaches (or replaces) the single PDF/image of an announcement (staff).
func (h *PortalHandler) UploadAnnouncementAttachment(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	id := chi.URLParam(r, "id")
	ctx, cancel := portalContext(r)
	defer cancel()
	if _, found, err := h.loadAnnouncement(ctx, id); err != nil || !found {
		if err != nil {
			serverError(w, "Failed to load announcement", err)
		} else {
			model.RespondError(w, http.StatusNotFound, "Announcement not found")
		}
		return
	}

	stored, ok := h.storeUpload(w, r, portal.MaxAttachmentBytes, false)
	if !ok {
		return
	}
	oldStored, err := h.swapAnnouncementFile(r, id, user.UserID, stored)
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save attachment", err)
		return
	}
	if oldStored != "" {
		h.removeStored(oldStored)
	}
	h.respondAnnouncement(w, r, id, http.StatusCreated)
}

// swapAnnouncementFile records the new file, points the announcement at it and drops
// the previous file row; it returns the previous stored name for removal from disk.
func (h *PortalHandler) swapAnnouncementFile(r *http.Request, id, userID string, stored storedFile) (string, error) {
	ctx, cancel := portalContext(r)
	defer cancel()
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var oldFile *string
	if err := tx.QueryRow(ctx, `SELECT attachment_id FROM announcements WHERE id = $1 FOR UPDATE`, id).Scan(&oldFile); err != nil {
		return "", err
	}
	var newFile string
	if err := tx.QueryRow(ctx, `
		INSERT INTO portal_files (role, original_name, stored_name, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		RoleAnnouncement, stored.originalName, stored.storedName, stored.mimeType, stored.size, userID).Scan(&newFile); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE announcements SET attachment_id = $2, updated_at = NOW() WHERE id = $1`, id, newFile); err != nil {
		return "", err
	}
	var oldStored string
	if oldFile != nil {
		if err := tx.QueryRow(ctx, `DELETE FROM portal_files WHERE id = $1 RETURNING stored_name`, *oldFile).Scan(&oldStored); err != nil && !isNoRows(err) {
			return "", err
		}
	}
	return oldStored, tx.Commit(ctx)
}

// DeleteAnnouncementAttachment removes the attachment of an announcement (staff).
func (h *PortalHandler) DeleteAnnouncementAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Announcement not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	var storedName string
	err := h.db.QueryRow(ctx, `
		DELETE FROM portal_files WHERE id = (SELECT attachment_id FROM announcements WHERE id = $1)
		RETURNING stored_name`, id).Scan(&storedName)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Attachment not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to delete attachment", err)
		return
	}
	h.removeStored(storedName)
	h.respondAnnouncement(w, r, id, http.StatusOK)
}

// DownloadPublicFile streams the attachment of a published announcement without
// authentication. Every other file, including drafts' attachments, is a 404 here.
func (h *PortalHandler) DownloadPublicFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var originalName, storedName, mimeType string
	err := h.db.QueryRow(ctx, `
		SELECT f.original_name, f.stored_name, f.mime_type
		FROM portal_files f JOIN announcements a ON a.attachment_id = f.id
		WHERE f.id = $1 AND f.role = $2 AND f.request_id IS NULL AND a.is_published`,
		fileID, RoleAnnouncement).Scan(&originalName, &storedName, &mimeType)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load file", err)
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
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": originalName}))
	// Short and revalidated: a withdrawn announcement's file must stop being served soon.
	w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("portal: file stream interrupted", "file_id", fileID, "error", err)
	}
}
