package portal

import "testing"

func TestDetectFileType(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	jpg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F'}
	pdf := []byte("%PDF-1.7\n1 0 obj")
	exe := []byte("MZ\x90\x00\x03\x00\x00\x00")
	html := []byte("<html><script>alert(1)</script></html>")

	cases := []struct {
		name    string
		data    []byte
		mime    string
		ext     string
		allowed bool
	}{
		{"png", png, "image/png", ".png", true},
		{"jpeg", jpg, "image/jpeg", ".jpg", true},
		{"pdf", pdf, "application/pdf", ".pdf", true},
		{"exe", exe, "", "", false},
		{"html", html, "", "", false},
		{"empty", nil, "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mime, ext, ok := DetectFileType(c.data)
			if ok != c.allowed || mime != c.mime || ext != c.ext {
				t.Fatalf("got (%q, %q, %v), want (%q, %q, %v)", mime, ext, ok, c.mime, c.ext, c.allowed)
			}
		})
	}
}

func TestIsImage(t *testing.T) {
	if !IsImage("image/png") || !IsImage("image/jpeg") || IsImage("application/pdf") {
		t.Fatal("IsImage misclassified")
	}
}
