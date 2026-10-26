package handler

import (
	"context"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const (
	maxTemplateLine = 255
	maxTemplateBody = 5000
)

type templateRecord struct {
	ID               string    `json:"id"`
	Kind             string    `json:"kind"`
	Name             string    `json:"name"`
	HeaderLine1      string    `json:"header_line_1"`
	HeaderLine2      string    `json:"header_line_2"`
	TitleText        string    `json:"title_text"`
	BodyText         string    `json:"body_text"`
	ClosingText      string    `json:"closing_text"`
	SignerName       string    `json:"signer_name"`
	SignerTitle      string    `json:"signer_title"`
	BackgroundFileID *string   `json:"background_file_id"`
	SignatureFileID  *string   `json:"signature_file_id"`
	IsActive         bool      `json:"is_active"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const templateColumns = `id, kind, name, header_line_1, header_line_2, title_text, body_text, closing_text,
	signer_name, signer_title, background_file_id, signature_file_id, is_active, updated_at`

const templateSelect = `SELECT ` + templateColumns + ` FROM document_templates`

func scanTemplate(row rowScanner) (templateRecord, error) {
	var t templateRecord
	err := row.Scan(&t.ID, &t.Kind, &t.Name, &t.HeaderLine1, &t.HeaderLine2, &t.TitleText, &t.BodyText,
		&t.ClosingText, &t.SignerName, &t.SignerTitle, &t.BackgroundFileID, &t.SignatureFileID, &t.IsActive, &t.UpdatedAt)
	return t, err
}

type templateInput struct {
	Name             string `json:"name"`
	HeaderLine1      string `json:"header_line_1"`
	HeaderLine2      string `json:"header_line_2"`
	TitleText        string `json:"title_text"`
	BodyText         string `json:"body_text"`
	ClosingText      string `json:"closing_text"`
	SignerName       string `json:"signer_name"`
	SignerTitle      string `json:"signer_title"`
	BackgroundFileID string `json:"background_file_id"`
	SignatureFileID  string `json:"signature_file_id"`
}

// problems returns "field:code" entries for the input.
func (in templateInput) problems() []string {
	var out []string
	if in.Name == "" {
		out = append(out, "name:required")
	}
	if in.BodyText == "" {
		out = append(out, "body_text:required")
	}
	if utf8.RuneCountInString(in.BodyText) > maxTemplateBody {
		out = append(out, "body_text:too_long")
	}
	lines := []struct{ field, value string }{
		{"name", in.Name}, {"header_line_1", in.HeaderLine1}, {"header_line_2", in.HeaderLine2},
		{"title_text", in.TitleText}, {"closing_text", in.ClosingText},
		{"signer_name", in.SignerName}, {"signer_title", in.SignerTitle},
	}
	for _, l := range lines {
		if utf8.RuneCountInString(l.value) > maxTemplateLine {
			out = append(out, l.field+":too_long")
		}
	}
	if len(portal.UnknownPlaceholders(in.BodyText)) > 0 {
		out = append(out, "body_text:unknown_placeholder")
	}
	return out
}

// assetsExist checks that each id refers to an uploaded template asset (a file not tied to a request).
func (h *PortalHandler) assetsExist(ctx context.Context, ids ...*string) (bool, error) {
	for _, id := range ids {
		if id == nil {
			continue
		}
		var ok bool
		if err := h.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM portal_files WHERE id = $1 AND request_id IS NULL)`, *id).Scan(&ok); err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// ListTemplates returns every letter template, the active one first.
func (h *PortalHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()

	rows, err := h.db.Query(ctx, templateSelect+` ORDER BY is_active DESC, updated_at DESC`)
	if err != nil {
		serverError(w, "Failed to list templates", err)
		return
	}
	defer rows.Close()
	items := []templateRecord{}
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			serverError(w, "Failed to read templates", err)
			return
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read templates", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// parseTemplateInput decodes, validates and resolves asset ids. It writes the error response itself.
func (h *PortalHandler) parseTemplateInput(ctx context.Context, w http.ResponseWriter, r *http.Request) (templateInput, *string, *string, bool) {
	var in templateInput
	if !decodeBody(w, r, &in) {
		return in, nil, nil, false
	}
	if problems := in.problems(); len(problems) > 0 {
		validationError(w, problems)
		return in, nil, nil, false
	}
	background, ok1 := idOrNil(in.BackgroundFileID)
	signature, ok2 := idOrNil(in.SignatureFileID)
	if !ok1 || !ok2 {
		model.RespondError(w, http.StatusBadRequest, "invalid identifier")
		return in, nil, nil, false
	}
	exist, err := h.assetsExist(ctx, background, signature)
	if err != nil {
		serverError(w, "Failed to check assets", err)
		return in, nil, nil, false
	}
	if !exist {
		model.RespondError(w, http.StatusBadRequest, "asset_not_found")
		return in, nil, nil, false
	}
	return in, background, signature, true
}

// CreateTemplate stores a new (inactive) letter template.
func (h *PortalHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	in, background, signature, ok := h.parseTemplateInput(ctx, w, r)
	if !ok {
		return
	}
	t, err := scanTemplate(h.db.QueryRow(ctx, `
		INSERT INTO document_templates
			(name, header_line_1, header_line_2, title_text, body_text, closing_text,
			 signer_name, signer_title, background_file_id, signature_file_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+templateColumns,
		in.Name, in.HeaderLine1, in.HeaderLine2, in.TitleText, in.BodyText, in.ClosingText,
		in.SignerName, in.SignerTitle, background, signature, user.UserID))
	if err != nil {
		serverError(w, "Failed to create template", err)
		return
	}
	model.RespondJSON(w, http.StatusCreated, t)
}

// UpdateTemplate edits a template. Letters already issued keep their own snapshot.
func (h *PortalHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Template not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	in, background, signature, ok := h.parseTemplateInput(ctx, w, r)
	if !ok {
		return
	}
	t, err := scanTemplate(h.db.QueryRow(ctx, `
		UPDATE document_templates SET
			name = $2, header_line_1 = $3, header_line_2 = $4, title_text = $5, body_text = $6,
			closing_text = $7, signer_name = $8, signer_title = $9,
			background_file_id = $10, signature_file_id = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING `+templateColumns,
		id, in.Name, in.HeaderLine1, in.HeaderLine2, in.TitleText, in.BodyText, in.ClosingText,
		in.SignerName, in.SignerTitle, background, signature))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Template not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to update template", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, t)
}

// ActivateTemplate makes one template the active one for its kind.
func (h *PortalHandler) ActivateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Template not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	tx, err := h.db.Begin(ctx)
	if err != nil {
		serverError(w, "Failed to activate template", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var kind string
	if err := tx.QueryRow(ctx, `SELECT kind FROM document_templates WHERE id = $1`, id).Scan(&kind); err != nil {
		if isNoRows(err) {
			model.RespondError(w, http.StatusNotFound, "Template not found")
			return
		}
		serverError(w, "Failed to activate template", err)
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE document_templates SET is_active = FALSE WHERE kind = $1 AND is_active`, kind); err != nil {
		serverError(w, "Failed to activate template", err)
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE document_templates SET is_active = TRUE, updated_at = NOW() WHERE id = $1`, id); err != nil {
		serverError(w, "Failed to activate template", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "Failed to activate template", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"activated": id})
}

// UploadTemplateAsset stores a letterhead/background or signature image for use in templates.
func (h *PortalHandler) UploadTemplateAsset(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	role := "template_" + r.URL.Query().Get("role")
	if role != "template_background" && role != "template_signature" {
		model.RespondError(w, http.StatusBadRequest, "role must be background or signature")
		return
	}
	stored, ok := h.storeUpload(w, r, portal.MaxAssetBytes, true)
	if !ok {
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	var info attachmentInfo
	err := h.db.QueryRow(ctx, `
		INSERT INTO portal_files (role, original_name, stored_name, mime_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, role, original_name, mime_type, size_bytes, uploaded_by, created_at
	`, role, stored.originalName, stored.storedName, stored.mimeType, stored.size, user.UserID).
		Scan(&info.ID, &info.Role, &info.OriginalName, &info.MimeType, &info.SizeBytes, &info.UploadedBy, &info.CreatedAt)
	if err != nil {
		h.removeStored(stored.storedName)
		serverError(w, "Failed to save asset", err)
		return
	}
	model.RespondJSON(w, http.StatusCreated, info)
}
