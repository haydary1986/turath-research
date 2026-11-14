package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminMailWithoutSMTP(t *testing.T) {
	h := NewAdminMailHandler(nil, "https://research.uoturath.edu.iq")

	rec := httptest.NewRecorder()
	h.Status(rec, httptest.NewRequest(http.MethodGet, "/admin/mail", nil))
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("status = %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.SendTest(rec, httptest.NewRequest(http.MethodPost, "/admin/mail/test", strings.NewReader(`{"to":"not an email"}`)))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad address: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.SendTest(rec, httptest.NewRequest(http.MethodPost, "/admin/mail/test", strings.NewReader(`{"to":"a@b.iq"}`)))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "mail_disabled") {
		t.Fatalf("disabled: %d %s", rec.Code, rec.Body.String())
	}
}
