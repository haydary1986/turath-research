// Package mailer sends plain-text notification emails over SMTP (STARTTLS).
// It is optional: with no SMTP host configured, every send is a silent no-op,
// so in-app notifications keep working on their own.
package mailer

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"time"
)

const sendTimeout = 15 * time.Second

// Config comes from SMTP_* environment variables.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string // e.g. noreply@uoturath.edu.iq
	FromName string // display name, may be Arabic
	// Branding of the HTML notices.
	Brand   string // e.g. شعبة الشؤون العلمية
	Org     string // e.g. جامعة التراث
	SiteURL string // public site root, for the logo and the footer links
}

// Mailer is safe for concurrent use. A nil *Mailer is a valid, disabled mailer.
type Mailer struct {
	cfg Config
}

// New returns nil (disabled) when no host is configured.
func New(cfg Config) *Mailer {
	if cfg.Host == "" || cfg.From == "" {
		return nil
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	if cfg.Brand == "" {
		cfg.Brand = "شعبة الشؤون العلمية"
	}
	if cfg.Org == "" {
		cfg.Org = "جامعة التراث"
	}
	return &Mailer{cfg: cfg}
}

// Enabled reports whether emails will actually be sent.
func (m *Mailer) Enabled() bool { return m != nil }

// SendNoticeAsync delivers a branded notice in the background; failures are logged, never
// surfaced. Notification email is best-effort by design — the in-app record is the source of truth.
func (m *Mailer) SendNoticeAsync(to string, n Notice) {
	if m == nil || to == "" {
		return
	}
	go func() {
		if err := m.SendNotice(to, n); err != nil {
			slog.Error("mailer: send failed", "to", to, "error", err)
		}
	}()
}

// ErrDisabled is returned by Send when no SMTP server is configured.
var ErrDisabled = fmt.Errorf("mailer: SMTP is not configured")

// SendNotice renders and delivers one branded notice, waiting for the outcome
// (bounded by sendTimeout). The admin's test message uses it to report the result.
func (m *Mailer) SendNotice(to string, n Notice) error {
	if m == nil {
		return ErrDisabled
	}
	html, err := m.renderHTML(n)
	if err != nil {
		return err
	}
	msg := buildNoticeMessage(m.fromHeader(), to, n.Subject, m.renderText(n), html)
	done := make(chan error, 1)
	go func() { done <- m.send(to, msg) }()
	select {
	case err := <-done:
		return err
	case <-time.After(sendTimeout):
		return fmt.Errorf("mailer: send timed out after %s", sendTimeout)
	}
}

// Sender returns the configured From address ("" when disabled).
func (m *Mailer) Sender() string {
	if m == nil {
		return ""
	}
	return m.cfg.From
}

// fromHeader is the From value: the encoded display name and the address.
func (m *Mailer) fromHeader() string {
	if m.cfg.FromName == "" {
		return m.cfg.From
	}
	return fmt.Sprintf("%s <%s>", encodeWord(m.cfg.FromName), m.cfg.From)
}

func (m *Mailer) send(to string, msg []byte) error {
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	// smtp.SendMail upgrades to STARTTLS whenever the server offers it.
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From,
		[]string{sanitizeHeader(to)}, msg)
}

// sanitizeHeader removes CR/LF so values from the database can never inject headers.
func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

// encodeWord RFC 2047-encodes a header word so Arabic survives every mail server.
func encodeWord(v string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(sanitizeHeader(v))) + "?="
}

// buildMessage assembles a UTF-8, base64-bodied message with CRLF line endings.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + encodeWord(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")

	writeBase64(&b, body)
	return []byte(b.String())
}

// writeBase64 writes body base64-encoded in 76-character lines.
func writeBase64(b *strings.Builder, body string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	for len(encoded) > 76 {
		b.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded + "\r\n")
}
