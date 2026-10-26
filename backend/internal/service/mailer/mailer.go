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
	return &Mailer{cfg: cfg}
}

// Enabled reports whether emails will actually be sent.
func (m *Mailer) Enabled() bool { return m != nil }

// SendAsync delivers in the background; failures are logged, never surfaced.
// Notification email is best-effort by design — the in-app record is the source of truth.
func (m *Mailer) SendAsync(to, subject, body string) {
	if m == nil || to == "" {
		return
	}
	go func() {
		done := make(chan error, 1)
		go func() { done <- m.send(to, subject, body) }()
		select {
		case err := <-done:
			if err != nil {
				slog.Error("mailer: send failed", "to", to, "error", err)
			}
		case <-time.After(sendTimeout):
			slog.Error("mailer: send timed out", "to", to)
		}
	}()
}

func (m *Mailer) send(to, subject, body string) error {
	from := m.cfg.From
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", encodeWord(m.cfg.FromName), m.cfg.From)
	}
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	// smtp.SendMail upgrades to STARTTLS whenever the server offers it.
	return smtp.SendMail(m.cfg.Host+":"+m.cfg.Port, auth, m.cfg.From,
		[]string{sanitizeHeader(to)}, buildMessage(from, to, subject, body))
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

	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	for len(encoded) > 76 {
		b.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded + "\r\n")
	return []byte(b.String())
}
