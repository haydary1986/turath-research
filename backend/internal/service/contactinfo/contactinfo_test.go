package contactinfo

import "testing"

func TestParsePhone(t *testing.T) {
	accepted := map[string]string{
		"":                  "",
		"07701234567":       "07701234567",
		" 0770 123-4567 ":   "07701234567",
		"٠٧٧٠١٢٣٤٥٦٧":       "07701234567",
		"+9647701234567":    "07701234567",
		"009647701234567":   "07701234567",
		"\u200e07701234567": "07701234567",
	}
	for raw, want := range accepted {
		if got, ok := ParsePhone(raw); !ok || got != want {
			t.Errorf("%q: got %q %v, want %q", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"0770123456", "077012345678", "08701234567", "7701234567", "0770123456x", "+15551234567", "<script>"} {
		if got, ok := ParsePhone(raw); ok || got != "" {
			t.Errorf("%q accepted as %q", raw, got)
		}
	}
}

func TestParseImportedPhoneRestoresTheDroppedZero(t *testing.T) {
	if got, ok := ParseImportedPhone("7701234567"); !ok || got != "07701234567" {
		t.Errorf("got %q %v", got, ok)
	}
	if _, ok := ParseImportedPhone("770123456"); ok {
		t.Error("nine digits accepted")
	}
}

func TestParseEmail(t *testing.T) {
	if got, ok := ParseEmail(" Me@Example.ORG "); !ok || got != "me@example.org" {
		t.Errorf("got %q %v", got, ok)
	}
	if got, ok := ParseEmail(""); !ok || got != "" {
		t.Errorf("empty: %q %v", got, ok)
	}
	for _, raw := range []string{"nope", "a@b", "a b@c.iq", "Name <a@b.iq>", "a@b.iq,c@d.iq", "a@@b.iq", "a@b."} {
		if _, ok := ParseEmail(raw); ok {
			t.Errorf("%q accepted", raw)
		}
	}
}

func TestParseEmployeeNoAndStatus(t *testing.T) {
	if got, ok := ParseEmployeeNo("١٢٣-A"); !ok || got != "123-A" {
		t.Errorf("got %q %v", got, ok)
	}
	if _, ok := ParseEmployeeNo("a b;c"); ok {
		t.Error("separator accepted")
	}
	if got, ok := ParseAccountStatus("Needs_Review"); !ok || got != StatusNeedsReview {
		t.Errorf("got %q %v", got, ok)
	}
	if _, ok := ParseAccountStatus("active"); ok {
		t.Error("unknown status accepted")
	}
}
