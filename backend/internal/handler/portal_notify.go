package handler

import (
	"context"
	"log/slog"

	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

type stageNotice struct{ titleAr, titleEn string }

var stageNotices = map[string]stageNotice{
	portal.StatusPendingDean:        {"طلب جديد بانتظار الموافقة الأولية", "New request awaiting initial approval"},
	portal.StatusPending:            {"طلب بانتظار تدقيق الشؤون العلمية", "Request awaiting scientific affairs review"},
	portal.StatusPendingVP:          {"توصية بانتظار تأكيد المساعد العلمي", "Recommendation awaiting VP confirmation"},
	portal.StatusVerified:           {"طلب مؤكَّد بانتظار إجراء الشؤون العلمية", "Verified request awaiting scientific affairs"},
	portal.StatusAwaitingSignatures: {"طلب بانتظار استكمال تواقيع اللجنة", "Request awaiting committee signatures"},
}

// stageHolderIsApplicant reports whether the applicant holds the role that acts at this stage.
// Nobody reviews their own request, so such a request can stall without an admin.
func stageHolderIsApplicant(stage, applicantRole string, applicantCollege, requestCollege *string) bool {
	required := portal.RequiredRole(stage)
	if required == "" || applicantRole != required {
		return false
	}
	if required == portal.RoleCollegeDean {
		return applicantCollege != nil && requestCollege != nil && *applicantCollege == *requestCollege
	}
	return true
}

// wakeStage alerts whoever must act now that the request reached `stage`.
func (h *PortalHandler) wakeStage(ctx context.Context, rec requestRecord, stage string) {
	notice, ok := stageNotices[stage]
	if !ok {
		return
	}
	role := portal.RequiredRole(stage)
	var college *string
	if role == portal.RoleCollegeDean {
		college = rec.CollegeID
	}
	body := "رقم الطلب: " + requestLabel(rec)
	link := requestLink(rec.ID)
	h.notifyRole(ctx, role, college, notice.titleAr, notice.titleEn, body, link)

	var applicantRole string
	var applicantCollege *string
	if err := h.db.QueryRow(ctx, `SELECT role::text, college_id FROM users WHERE id = $1`, rec.ApplicantID).
		Scan(&applicantRole, &applicantCollege); err != nil {
		slog.Error("portal: failed to load applicant role", "request_id", rec.ID, "error", err)
		return
	}
	if stageHolderIsApplicant(stage, applicantRole, applicantCollege, rec.CollegeID) {
		h.notifyRole(ctx, portal.RoleAdmin, nil,
			"طلب يحتاج تدخل مدير النظام: مقدّمه هو صاحب هذه المرحلة",
			"Request needs an admin: its applicant holds this stage",
			body, link)
	}
}
