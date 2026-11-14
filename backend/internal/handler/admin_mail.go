package handler

import (
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/mailer"
)

// AdminMailHandler lets the admin see whether notification email is configured and send a test.
type AdminMailHandler struct {
	mail    *mailer.Mailer
	siteURL string
}

// NewAdminMailHandler builds the handler; a nil mailer reports "disabled".
func NewAdminMailHandler(m *mailer.Mailer, siteURL string) *AdminMailHandler {
	return &AdminMailHandler{mail: m, siteURL: strings.TrimRight(siteURL, "/")}
}

type mailStatus struct {
	Enabled bool   `json:"enabled"`
	Sender  string `json:"sender"`
}

// Status reports whether email is configured and from which address it is sent.
func (h *AdminMailHandler) Status(w http.ResponseWriter, _ *http.Request) {
	model.RespondJSON(w, http.StatusOK, mailStatus{Enabled: h.mail.Enabled(), Sender: h.mail.Sender()})
}

// SendTest sends one test message to the given address and reports the outcome.
// The SMTP error detail is logged, not returned, so server settings never reach the page.
func (h *AdminMailHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		To string `json:"to"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	to := strings.ToLower(strings.TrimSpace(in.To))
	if addr, err := mail.ParseAddress(to); err != nil || addr.Address != to {
		validationError(w, []string{"to:invalid"})
		return
	}
	if !h.mail.Enabled() {
		model.RespondError(w, http.StatusConflict, "mail_disabled")
		return
	}
	notice := mailer.Notice{
		Subject: "رسالة فحص — شعبة الشؤون العلمية",
		Heading: "بريد الإشعارات يعمل",
		Text: "هذه رسالة فحص من بوابة شعبة الشؤون العلمية في جامعة التراث.\n" +
			"بهذا الشكل تصل إشعارات الطلبات: عنوان الإشعار، ثم تفاصيله، ثم زر يفتح الطلب في البوابة.\n" +
			"This is a test message from the Scientific Affairs portal.",
		ActionURL: h.siteURL + "/ar/portal", ActionLabel: "الدخول إلى البوابة",
	}
	if err := h.mail.SendNotice(to, notice); err != nil {
		slog.Error("admin mail test failed", "to", to, "error", err)
		model.RespondError(w, http.StatusBadGateway, "mail_send_failed")
		return
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"sent_to": to})
}
