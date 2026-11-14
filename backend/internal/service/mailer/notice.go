package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"strings"
)

// Notice is one branded notification email: a heading, a few lines of text and an
// optional button that opens the related page in the portal.
type Notice struct {
	Subject     string
	Heading     string
	Text        string // plain text; each line becomes a paragraph line in the HTML part
	ActionURL   string // absolute https link; empty for no button
	ActionLabel string
}

// noticeView is what the HTML template sees; html/template escapes every value.
type noticeView struct {
	Brand       string
	Org         string
	Heading     string
	Lines       []string
	ActionURL   string
	ActionLabel string
	LogoURL     string
	PortalURL   string
	SiteURL     string
}

// Brand colours of Al-Turath University.
const (
	navy    = "#001D40"
	crimson = "#890015"
)

var noticeTemplate = template.Must(template.New("notice").Parse(`<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Heading}}</title></head>
<body style="margin:0;padding:0;background:#f3f5f9;font-family:Tahoma,Arial,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f3f5f9;padding:24px 12px;">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e3e7ee;">
  <tr><td style="background:` + navy + `;padding:20px 28px;" align="right">
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr>
      <td align="right" style="color:#ffffff;">
        <div style="font-size:18px;font-weight:bold;">{{.Brand}}</div>
        <div style="font-size:13px;opacity:.85;margin-top:2px;">{{.Org}}</div>
      </td>
      {{if .LogoURL}}<td align="left" width="120"><img src="{{.LogoURL}}" alt="{{.Org}}" width="110" style="display:block;border:0;max-width:110px;height:auto;"></td>{{end}}
    </tr></table>
  </td></tr>
  <tr><td style="height:4px;background:` + crimson + `;font-size:0;line-height:0;">&nbsp;</td></tr>
  <tr><td style="padding:28px;" align="right" dir="rtl">
    <h1 style="margin:0 0 16px;font-size:20px;color:` + navy + `;">{{.Heading}}</h1>
    {{range .Lines}}<p style="margin:0 0 8px;font-size:15px;line-height:1.8;color:#2b3445;">{{.}}</p>{{end}}
    {{if .ActionURL}}
    <table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0 8px;"><tr>
      <td style="background:` + crimson + `;border-radius:8px;">
        <a href="{{.ActionURL}}" style="display:inline-block;padding:12px 28px;color:#ffffff;font-size:15px;font-weight:bold;text-decoration:none;">{{.ActionLabel}}</a>
      </td></tr></table>
    <p style="margin:8px 0 0;font-size:12px;color:#6b7486;">إن لم يعمل الزر فانسخ الرابط: <a href="{{.ActionURL}}" style="color:` + navy + `;word-break:break-all;">{{.ActionURL}}</a></p>
    {{end}}
  </td></tr>
  <tr><td style="background:#f7f8fb;border-top:1px solid #e3e7ee;padding:18px 28px;" align="right" dir="rtl">
    {{if .PortalURL}}<p style="margin:0 0 8px;font-size:13px;">
      <a href="{{.PortalURL}}" style="color:` + navy + `;font-weight:bold;text-decoration:none;">الدخول إلى البوابة</a>
      &nbsp;·&nbsp;
      <a href="{{.SiteURL}}" style="color:` + navy + `;text-decoration:none;">الموقع الرسمي للشعبة</a>
    </p>{{end}}
    <p style="margin:0;font-size:12px;line-height:1.7;color:#6b7486;">رسالة آلية من {{.Brand}} — {{.Org}}. لا ترد على هذه الرسالة؛ للتواصل استخدم البوابة.</p>
  </td></tr>
</table>
</td></tr>
</table>
</body>
</html>`))

// renderHTML returns the branded HTML part of the notice.
func (m *Mailer) renderHTML(n Notice) (string, error) {
	view := noticeView{
		Brand: m.cfg.Brand, Org: m.cfg.Org, Heading: n.Heading,
		Lines: splitLines(n.Text), ActionURL: safeURL(n.ActionURL), ActionLabel: n.ActionLabel,
	}
	if site := strings.TrimRight(m.cfg.SiteURL, "/"); strings.HasPrefix(site, "https://") {
		view.LogoURL, view.PortalURL, view.SiteURL = site+"/brand/logo-white.png", site+"/ar/portal", site+"/ar"
	}
	if view.ActionURL != "" && view.ActionLabel == "" {
		view.ActionLabel = "فتح في البوابة"
	}
	var buf bytes.Buffer
	if err := noticeTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderText is the plain-text alternative for clients that do not show HTML.
func (m *Mailer) renderText(n Notice) string {
	parts := []string{n.Heading, "", n.Text}
	if url := safeURL(n.ActionURL); url != "" {
		parts = append(parts, "", url)
	}
	parts = append(parts, "", "—", m.cfg.Brand+" — "+m.cfg.Org)
	if site := strings.TrimRight(m.cfg.SiteURL, "/"); site != "" {
		parts = append(parts, "الدخول إلى البوابة: "+site+"/ar/portal")
	}
	return strings.Join(parts, "\n")
}

func splitLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// safeURL keeps only absolute http(s) links, so nothing else can become a button.
func safeURL(u string) string {
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		return u
	}
	return ""
}

// buildNoticeMessage assembles a multipart/alternative message: plain text, then HTML.
func buildNoticeMessage(from, to, subject, text, html string) []byte {
	boundary := newBoundary()
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + encodeWord(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	for _, part := range []struct{ kind, body string }{{"text/plain", text}, {"text/html", html}} {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + part.kind + "; charset=\"UTF-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		writeBase64(&b, part.body)
	}
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func newBoundary() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "turath-boundary-0000000000"
	}
	return "turath-" + hex.EncodeToString(buf)
}
