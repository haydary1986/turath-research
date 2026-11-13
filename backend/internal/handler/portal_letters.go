package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const maxDiwanRunes = 100

type letterRecord struct {
	ID               string    `json:"id"`
	RequestID        string    `json:"request_id"`
	LetterNo         string    `json:"letter_no"`
	DiwanNo          *string   `json:"diwan_no"`
	IssuedDate       string    `json:"issued_date"`
	RecipientName    string    `json:"recipient_name"`
	CollegeName      string    `json:"college_name"`
	DepartmentName   string    `json:"department_name"`
	HeaderLine1      string    `json:"header_line_1"`
	HeaderLine2      string    `json:"header_line_2"`
	TitleText        string    `json:"title_text"`
	BodyText         string    `json:"body_text"`
	ClosingText      string    `json:"closing_text"`
	SignerName       string    `json:"signer_name"`
	SignerTitle      string    `json:"signer_title"`
	BackgroundFileID *string   `json:"background_file_id"`
	SignatureFileID  *string   `json:"signature_file_id"`
	CreatedAt        time.Time `json:"created_at"`
}

const letterSelect = `
	SELECT id, request_id, letter_no, diwan_no, to_char(issued_date, 'YYYY-MM-DD'), recipient_name, college_name,
	       department_name, header_line_1, header_line_2, title_text, body_text, closing_text,
	       signer_name, signer_title, background_file_id, signature_file_id, created_at
	FROM issued_letters`

func scanLetter(row rowScanner) (letterRecord, error) {
	var l letterRecord
	err := row.Scan(&l.ID, &l.RequestID, &l.LetterNo, &l.DiwanNo, &l.IssuedDate, &l.RecipientName, &l.CollegeName,
		&l.DepartmentName, &l.HeaderLine1, &l.HeaderLine2, &l.TitleText, &l.BodyText, &l.ClosingText,
		&l.SignerName, &l.SignerTitle, &l.BackgroundFileID, &l.SignatureFileID, &l.CreatedAt)
	return l, err
}

func valueOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// letterVariables maps a request to the placeholders a template may use.
func letterVariables(rec requestRecord) (map[string]string, error) {
	var details map[string]any
	if err := json.Unmarshal(rec.Details, &details); err != nil {
		return nil, err
	}
	detail := func(key string) string {
		s, _ := details[key].(string)
		return s
	}
	published := detail("publication_date")
	if t, err := time.Parse("2006-01-02", published); err == nil {
		published = t.Format("02/01/2006")
	}
	return map[string]string{
		"researcher_name":  rec.ApplicantName,
		"college":          valueOr(rec.CollegeName, ""),
		"department":       valueOr(rec.DepartmentName, ""),
		"research_title":   valueOr(rec.TitleEn, valueOr(rec.TitleAr, "")),
		"journal_name":     detail("journal_name"),
		"index_db":         portal.IndexDBLabel(detail("index_db")),
		"quartile":         detail("quartile"),
		"publication_date": published,
	}, nil
}

// IssueLetter creates the official appreciation letter for a verified request.
// The rendered text and the template's look are snapshotted; only the registry number may change afterwards.
func (h *PortalHandler) IssueLetter(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var in struct {
		DiwanNo string `json:"diwan_no"`
	}
	if !decodeOptionalBody(w, r, &in) {
		return
	}
	in.DiwanNo = strings.TrimSpace(in.DiwanNo)
	if utf8.RuneCountInString(in.DiwanNo) > maxDiwanRunes {
		validationError(w, []string{"diwan_no:too_long"})
		return
	}

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}
	if rec.Kind == portal.KindAppreciationLetter && rec.Status == portal.StatusIssued {
		model.RespondError(w, http.StatusConflict, "Letter was already issued")
		return
	}
	if !mayActOn(user, rec) {
		model.RespondError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	if rec.Kind != portal.KindAppreciationLetter || rec.Status != portal.StatusVerified {
		model.RespondError(w, http.StatusConflict, "Letter can only be issued for a verified appreciation request")
		return
	}

	tpl, err := scanTemplate(h.db.QueryRow(ctx, templateSelect+` WHERE kind = $1 AND is_active`, portal.KindAppreciationLetter))
	if isNoRows(err) {
		model.RespondError(w, http.StatusConflict, "no_active_template")
		return
	}
	if err != nil {
		serverError(w, "Failed to load template", err)
		return
	}
	vars, err := letterVariables(rec)
	if err != nil {
		serverError(w, "Failed to read request details", err)
		return
	}

	letterID, err := h.createLetter(ctx, rec, tpl, vars, user.UserID, blankToNil(in.DiwanNo))
	if err != nil {
		if errors.Is(err, errStatusChanged) || isUniqueViolation(err) {
			model.RespondError(w, http.StatusConflict, "Letter was already issued")
			return
		}
		serverError(w, "Failed to issue letter", err)
		return
	}

	h.notifyUser(ctx, rec.ApplicantID, statusTitlesAr[portal.StatusIssued], "Appreciation letter issued",
		"رقم الطلب: "+requestLabel(rec), "success", "/portal/letters/"+letterID)
	letter, err := scanLetter(h.db.QueryRow(ctx, letterSelect+` WHERE id = $1`, letterID))
	if err != nil {
		serverError(w, "Failed to load letter", err)
		return
	}
	model.RespondJSON(w, http.StatusCreated, letter)
}

func (h *PortalHandler) createLetter(ctx context.Context, rec requestRecord, tpl templateRecord, vars map[string]string, actorID string, diwanNo *string) (string, error) {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	year := time.Now().Year()
	seq, err := nextSequence(ctx, tx, "letter", year)
	if err != nil {
		return "", err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO issued_letters
			(request_id, template_id, letter_year, letter_seq, letter_no, diwan_no, recipient_name, college_name, department_name,
			 header_line_1, header_line_2, title_text, body_text, closing_text, signer_name, signer_title,
			 background_file_id, signature_file_id, issued_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id
	`, rec.ID, tpl.ID, year, seq, portal.FormatLetterNo(seq, year), diwanNo,
		vars["researcher_name"], vars["college"], vars["department"],
		tpl.HeaderLine1, tpl.HeaderLine2, tpl.TitleText, portal.RenderBody(tpl.BodyText, vars), tpl.ClosingText,
		tpl.SignerName, tpl.SignerTitle, tpl.BackgroundFileID, tpl.SignatureFileID, actorID).Scan(&id)
	if err != nil {
		return "", err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE portal_requests SET status = 'issued', decided_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'verified'
	`, rec.ID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", errStatusChanged
	}
	from := rec.Status
	if err := recordEvent(ctx, tx, rec.ID, actorID, &from, portal.StatusIssued, ""); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// UpdateLetterDiwanNo sets the registry number, which the diwan may assign after printing.
// It is the only editable field of an issued letter. An absent diwan_no changes nothing,
// so an empty body can never wipe a recorded number; an explicit "" clears it.
func (h *PortalHandler) UpdateLetterDiwanNo(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	var in struct {
		DiwanNo *string `json:"diwan_no"`
	}
	if !decodeOptionalBody(w, r, &in) {
		return
	}
	var next *string
	if in.DiwanNo != nil {
		next = blankToNil(*in.DiwanNo)
		if next != nil && utf8.RuneCountInString(*next) > maxDiwanRunes {
			validationError(w, []string{"diwan_no:too_long"})
			return
		}
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	if in.DiwanNo != nil {
		err := h.saveDiwanNo(ctx, id, user.UserID, next)
		if isNoRows(err) {
			model.RespondError(w, http.StatusNotFound, "Letter not found")
			return
		}
		if err != nil {
			serverError(w, "Failed to update letter", err)
			return
		}
	}
	letter, err := scanLetter(h.db.QueryRow(ctx, letterSelect+` WHERE id = $1`, id))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load letter", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, letter)
}

func (h *PortalHandler) saveDiwanNo(ctx context.Context, letterID, actorID string, next *string) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var requestID string
	var current *string
	if err := tx.QueryRow(ctx, `SELECT request_id, diwan_no FROM issued_letters WHERE id = $1 FOR UPDATE`, letterID).
		Scan(&requestID, &current); err != nil {
		return err
	}
	if valueOr(current, "") == valueOr(next, "") {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE issued_letters SET diwan_no = $2 WHERE id = $1`, letterID, next); err != nil {
		return err
	}
	from := portal.StatusIssued
	if err := recordEvent(ctx, tx, requestID, actorID, &from, portal.StatusIssued, diwanChangeNote(current, next)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func diwanChangeNote(previous, next *string) string {
	return "تعديل رقم صادرة الديوان: " + valueOr(previous, "—") + " ← " + valueOr(next, "—")
}

// GetLetter returns an issued letter to anyone who may see its request.
func (h *PortalHandler) GetLetter(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	letter, err := scanLetter(h.db.QueryRow(ctx, letterSelect+` WHERE id = $1`, id))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load letter", err)
		return
	}
	rec, found, err := h.loadRequest(ctx, letter.RequestID)
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	model.RespondJSON(w, http.StatusOK, letter)
}

// VerifyLetter is public: it confirms that a letter id (printed on the letter) is genuine
// and reveals only the minimum needed to compare it with the paper copy.
func (h *PortalHandler) VerifyLetter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	var out struct {
		LetterNo      string  `json:"letter_no"`
		DiwanNo       *string `json:"diwan_no"`
		IssuedDate    string  `json:"issued_date"`
		RecipientName string  `json:"recipient_name"`
		SignerTitle   string  `json:"signer_title"`
	}
	err := h.db.QueryRow(ctx, `
		SELECT letter_no, diwan_no, to_char(issued_date, 'YYYY-MM-DD'), recipient_name, signer_title
		FROM issued_letters WHERE id = $1
	`, id).Scan(&out.LetterNo, &out.DiwanNo, &out.IssuedDate, &out.RecipientName, &out.SignerTitle)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Letter not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to verify letter", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, out)
}
