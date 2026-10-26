package screening

import "testing"

func TestNormalizeISSN(t *testing.T) {
	cases := map[string]string{
		"0378-7788":      "0378-7788",
		"03787788":       "0378-7788",
		" 2049-363x ":    "2049-363X",
		"ISSN 1234-5679": "1234-5679",
		"1234-567":       "", // 7 digits
		"abcd-efgh":      "",
		"":               "",
	}
	for in, want := range cases {
		if got := NormalizeISSN(in); got != want {
			t.Errorf("NormalizeISSN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNameKey(t *testing.T) {
	a := NameKey("  Energy and Buildings ")
	b := NameKey("energy  AND buildings")
	if a == "" || a != b {
		t.Fatalf("latin keys differ: %q vs %q", a, b)
	}
	c := NameKey("المجلة العراقية للهندسة")
	d := NameKey("المجلّة العراقيه للهندسة")
	if c != d {
		t.Fatalf("arabic variants differ: %q vs %q", c, d)
	}
	if NameKey("") != "" {
		t.Fatal("empty name must give empty key")
	}
}

func TestIsClassification(t *testing.T) {
	for _, ok := range []string{"accredited", "predatory", "suspended"} {
		if !IsClassification(ok) {
			t.Errorf("%s should be valid", ok)
		}
	}
	for _, bad := range []string{"", "good", "Predatory"} {
		if IsClassification(bad) {
			t.Errorf("%s should be invalid", bad)
		}
	}
}
