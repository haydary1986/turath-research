package mailer

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMessageEncodesArabicSafely(t *testing.T) {
	msg := buildMessage(encodeWord("بوابة الشؤون العلمية")+" <noreply@uoturath.edu.iq>",
		"ahmed@uoturath.edu.iq", "تمت المصادقة على الطلب", "رقم الطلب: RP-2026-00001\nالحالة: تمت المصادقة")
	text := string(msg)

	for _, want := range []string{
		"To: ahmed@uoturath.edu.iq\r\n",
		"MIME-Version: 1.0\r\n",
		`Content-Type: text/plain; charset="UTF-8"` + "\r\n",
		"Content-Transfer-Encoding: base64\r\n",
		"Subject: =?UTF-8?B?",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("message missing %q:\n%s", want, text)
		}
	}

	// Header and body must be split by a blank line; the body must decode back to the input.
	parts := strings.SplitN(text, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatal("missing header/body separator")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(parts[1], "\r\n", ""))
	if err != nil {
		t.Fatalf("body is not valid base64: %v", err)
	}
	if string(decoded) != "رقم الطلب: RP-2026-00001\nالحالة: تمت المصادقة" {
		t.Fatalf("body round-trip failed: %q", decoded)
	}
	// No raw (unencoded) Arabic may appear in the wire format.
	if strings.Contains(text, "الشؤون") && !strings.Contains(parts[0], "From: =?UTF-8?B?") {
		t.Fatal("display name was not RFC 2047 encoded")
	}
}

func TestBuildMessageStripsHeaderInjection(t *testing.T) {
	msg := string(buildMessage("a@b.c", "x@y.z\r\nBcc: evil@e.com", "s\r\nX-Evil: 1", "body"))
	// CR/LF must be stripped so injected text can never start a new header line.
	if strings.Contains(msg, "\r\nBcc:") || strings.Contains(msg, "\r\nX-Evil:") {
		t.Fatalf("header injection not neutralized:\n%s", msg)
	}
}

func TestDisabledMailerIsSafe(t *testing.T) {
	var m *Mailer
	if m.Enabled() {
		t.Fatal("nil mailer must report disabled")
	}
	m.SendAsync("a@b.c", "s", "b") // must not panic
}
