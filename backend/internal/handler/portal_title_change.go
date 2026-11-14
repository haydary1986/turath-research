package handler

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// eventTitleChanged marks, in a research plan's history, that an approved title-change request renamed it.
const eventTitleChanged = "title_changed"

// applyTitleChange copies the approved new title onto the linked research plan and records the
// old title in the plan's history. A request without a linked plan (research registered before
// the portal) changes nothing else: the approval itself is the official record.
func applyTitleChange(ctx context.Context, tx pgx.Tx, rec requestRecord, actorID string) error {
	if rec.RelatedRequestID == nil {
		return nil
	}
	var oldAr, oldEn *string
	var planStatus string
	err := tx.QueryRow(ctx, `
		SELECT title_ar, title_en, status FROM portal_requests
		WHERE id = $1 AND kind = $2 AND applicant_id = $3
		FOR UPDATE
	`, *rec.RelatedRequestID, portal.KindResearchPlan, rec.ApplicantID).Scan(&oldAr, &oldEn, &planStatus)
	if isNoRows(err) {
		return nil // the plan is gone or no longer the applicant's: keep the approval, change nothing
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE portal_requests
		SET title_ar = COALESCE($2, title_ar), title_en = COALESCE($3, title_en), updated_at = NOW()
		WHERE id = $1
	`, *rec.RelatedRequestID, rec.TitleAr, rec.TitleEn); err != nil {
		return err
	}
	return recordEvent(ctx, tx, *rec.RelatedRequestID, actorID, &planStatus, eventTitleChanged,
		titleChangeNote(rec, oldAr, oldEn))
}

// titleChangeNote explains the rename in the plan's history: which request authorised it and the previous title.
func titleChangeNote(rec requestRecord, oldAr, oldEn *string) string {
	lines := []string{"بموجب طلب تعديل العنوان: " + requestLabel(rec)}
	var previous []string
	for _, title := range []*string{oldAr, oldEn} {
		if title != nil && strings.TrimSpace(*title) != "" {
			previous = append(previous, strings.TrimSpace(*title))
		}
	}
	if len(previous) > 0 {
		lines = append(lines, "العنوان السابق: "+strings.Join(previous, " / "))
	}
	return strings.Join(lines, "\n")
}
