package handler

import (
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const (
	multipartMemory  = 1 << 20
	maxOriginalRunes = 200
)

type storedFile struct {
	storedName   string
	originalName string
	mimeType     string
	size         int64
}

// storeUpload reads the multipart "file" field, sniffs its real type and writes it under a generated name.
// On failure it writes the HTTP error itself and returns ok=false.
func (h *PortalHandler) storeUpload(w http.ResponseWriter, r *http.Request, maxBytes int64, imageOnly bool) (storedFile, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+multipartMemory)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		model.RespondError(w, http.StatusRequestEntityTooLarge, "file_too_large")
		return storedFile{}, false
	}
	defer r.MultipartForm.RemoveAll() //nolint:errcheck // temp-file cleanup

	file, header, err := r.FormFile("file")
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "file is required")
		return storedFile{}, false
	}
	defer file.Close()

	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		serverError(w, "Failed to read upload", err)
		return storedFile{}, false
	}
	mimeType, ext, ok := portal.DetectFileType(head[:n])
	if !ok || (imageOnly && !portal.IsImage(mimeType)) {
		model.RespondError(w, http.StatusUnsupportedMediaType, "unsupported_file_type")
		return storedFile{}, false
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		serverError(w, "Failed to read upload", err)
		return storedFile{}, false
	}

	if err := os.MkdirAll(h.uploadDir, 0o750); err != nil {
		serverError(w, "Failed to prepare storage", err)
		return storedFile{}, false
	}
	storedName := uuid.New().String() + ext
	dst, err := os.OpenFile(filepath.Join(h.uploadDir, storedName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		serverError(w, "Failed to store file", err)
		return storedFile{}, false
	}
	written, copyErr := io.Copy(dst, io.LimitReader(file, maxBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || written > maxBytes {
		h.removeStored(storedName)
		if written > maxBytes {
			model.RespondError(w, http.StatusRequestEntityTooLarge, "file_too_large")
		} else {
			serverError(w, "Failed to store file", errors.Join(copyErr, closeErr))
		}
		return storedFile{}, false
	}
	return storedFile{storedName, sanitizeFilename(header.Filename), mimeType, written}, true
}

func (h *PortalHandler) removeStored(storedName string) {
	if err := os.Remove(filepath.Join(h.uploadDir, storedName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("portal: failed to remove stored file", "stored_name", storedName, "error", err)
	}
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	if utf8.RuneCountInString(name) > maxOriginalRunes {
		name = string([]rune(name)[:maxOriginalRunes])
	}
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

// UploadAttachment attaches a PDF/image to a request.
// Applicants upload while the request is editable; staff upload the signed form and
// other follow-up files while the request is at an active stage.
func (h *PortalHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}

	isApplicant := rec.ApplicantID == user.UserID
	isReviewer := portal.IsStaff(user.Role) && !isApplicant
	switch {
	case isApplicant && isEditable(rec.Status):
	case isReviewer && portal.IsActiveStage(rec.Status):
	default:
		model.RespondError(w, http.StatusForbidden, "Files cannot be added at this stage")
		return
	}

	// The multipart body is read by storeUpload, so the role arrives as a query parameter.
	role := r.URL.Query().Get("role")
	if !portal.AllowedAttachmentRole(rec.Kind, role) {
		model.RespondError(w, http.StatusBadRequest, "attachment role is not allowed for this request")
		return
	}
	if role == portal.RoleSignedForm && !isReviewer {
		model.RespondError(w, http.StatusForbidden, "Only reviewers may upload the signed form")
		return
	}

	var count int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM portal_files WHERE request_id = $1`, rec.ID).Scan(&count); err != nil {
		serverError(w, "Failed to check attachments", err)
		return
	}
	if count >= maxFilesPerRequest {
		model.RespondError(w, http.StatusConflict, "too_many_files")
		return
	}

	stored, ok := h.storeUpload(w, r, portal.MaxAttachmentBytes, false)
	if !ok {
		return
	}

	var info attachmentInfo
	err = h.db.QueryRow(ctx, `
		INSERT INTO portal_files (request_id, role, original_name, stored_name, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, role, original_name, mime_type, size_bytes, uploaded_by, created_at
	`, rec.ID, role, stored.originalName, stored.storedName, stored.mimeType, stored.size, user.UserID).
		Scan(&info.ID, &info.Role, &info.OriginalName, &info.MimeType, &info.SizeBytes, &info.UploadedBy, &info.CreatedAt)
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save attachment", err)
		return
	}
	model.RespondJSON(w, http.StatusCreated, info)
}

// DownloadFile streams a stored file after checking the caller may see its request.
// Template assets (letterheads, signatures) are visible to every signed-in user.
func (h *PortalHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var requestID *string
	var role, originalName, storedName, mimeType string
	err := h.db.QueryRow(ctx, `SELECT request_id, role, original_name, stored_name, mime_type FROM portal_files WHERE id = $1`, fileID).
		Scan(&requestID, &role, &originalName, &storedName, &mimeType)
	// Announcement drafts are staff business; the public reach published ones through /public/files.
	if err == nil && role == RoleAnnouncement && !portal.IsStaff(user.Role) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load file", err)
		return
	}

	if requestID != nil {
		rec, found, err := h.loadRequest(ctx, *requestID)
		if err != nil {
			serverError(w, "Failed to load request", err)
			return
		}
		if !found || !canView(user, rec) {
			model.RespondError(w, http.StatusNotFound, "File not found")
			return
		}
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
	w.Header().Set("Cache-Control", "private, max-age=300")
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("portal: file stream interrupted", "file_id", fileID, "error", err)
	}
}

// DeleteAttachment removes a request attachment. Applicants may remove their own files while the
// request is editable; reviewers may remove files only while it is at an active stage.
func (h *PortalHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	fileID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(fileID); err != nil {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var requestID *string
	var uploadedBy *string
	var storedName string
	err := h.db.QueryRow(ctx, `SELECT request_id, uploaded_by, stored_name FROM portal_files WHERE id = $1`, fileID).
		Scan(&requestID, &uploadedBy, &storedName)
	if isNoRows(err) || (err == nil && requestID == nil) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load file", err)
		return
	}
	rec, found, err := h.loadRequest(ctx, *requestID)
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "File not found")
		return
	}

	ownFileWhileEditable := rec.ApplicantID == user.UserID && isEditable(rec.Status) &&
		uploadedBy != nil && *uploadedBy == user.UserID
	reviewer := portal.IsStaff(user.Role) && rec.ApplicantID != user.UserID && portal.IsActiveStage(rec.Status)
	if !ownFileWhileEditable && !reviewer {
		model.RespondError(w, http.StatusForbidden, "File cannot be removed")
		return
	}

	if _, err := h.db.Exec(ctx, `DELETE FROM portal_files WHERE id = $1`, fileID); err != nil {
		serverError(w, "Failed to delete file", err)
		return
	}
	h.removeStored(storedName)
	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": fileID})
}
