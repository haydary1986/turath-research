package mailer

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

func testMailer() *Mailer {
	return New(Config{Host: "smtp.example", From: "no_reply@uoturath.edu.iq",
		FromName: "شعبة الشؤون العلمية — جامعة التراث", SiteURL: "https://research.uoturath.edu.iq/"})
}

func TestNoticeHTMLIsBrandedAndEscaped(t *testing.T) {
	html, err := testMailer().renderHTML(Notice{
		Heading: "تمت المصادقة على الطلب", Text: "رقم الطلب: RP-1\n<script>alert(1)</script>",
		ActionURL: "https://research.uoturath.edu.iq/ar/portal/requests/1", ActionLabel: "مراجعة الطلب",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`dir="rtl"`, "شعبة الشؤون العلمية", "جامعة التراث", "تمت المصادقة على الطلب",
		`href="https://research.uoturath.edu.iq/ar/portal/requests/1"`, "مراجعة الطلب",
		`src="https://research.uoturath.edu.iq/brand/logo-white.png"`,
		`href="https://research.uoturath.edu.iq/ar/portal"`, "الدخول إلى البوابة",
		"&lt;script&gt;",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Fatal("body text was not escaped")
	}
}

func TestNoticeDropsUnsafeActionLink(t *testing.T) {
	html, _ := testMailer().renderHTML(Notice{Heading: "x", ActionURL: "javascript:alert(1)", ActionLabel: "اضغط"})
	if strings.Contains(html, "javascript:") || strings.Contains(html, "اضغط") {
		t.Fatal("unsafe link became a button")
	}
}

func TestNoticeMessageIsMultipart(t *testing.T) {
	m := testMailer()
	n := Notice{Subject: "فحص", Heading: "فحص", Text: "سطر"}
	html, _ := m.renderHTML(n)
	msg := string(buildNoticeMessage(m.fromHeader(), "a@b.iq", n.Subject, m.renderText(n), html))
	if !strings.Contains(msg, "Content-Type: multipart/alternative; boundary=") ||
		!strings.Contains(msg, `Content-Type: text/plain; charset="UTF-8"`) ||
		!strings.Contains(msg, `Content-Type: text/html; charset="UTF-8"`) {
		t.Fatalf("not multipart/alternative:\n%s", msg[:300])
	}
	if !strings.Contains(msg, "From: =?UTF-8?B?") {
		t.Fatal("display name not encoded")
	}
	boundary := regexp.MustCompile(`boundary="([^"]+)"`).FindStringSubmatch(msg)[1]
	if !strings.HasSuffix(msg, "--"+boundary+"--\r\n") {
		t.Fatal("closing boundary missing")
	}
	// The plain part names the brand and links the portal.
	parts := strings.Split(msg, "--"+boundary)
	body := parts[1][strings.Index(parts[1], "\r\n\r\n")+4:]
	text, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
	if err != nil || !strings.Contains(string(text), "شعبة الشؤون العلمية") || !strings.Contains(string(text), "/ar/portal") {
		t.Fatalf("plain part = %q, %v", text, err)
	}
}
