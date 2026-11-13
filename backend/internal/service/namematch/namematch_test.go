package namematch

import "testing"

// All names here are invented or common-name examples, never real staff records.
func TestMatchSameScript(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Josiah Carberry", "Josiah Carberry", true},
		{"Josiah Carberry", "carberry, josiah", true},
		{"Josiah Carberry", "J. Carberry", true},
		{"Josiah S. Carberry", "J Carberry", true},
		{"Dr. Josiah Carberry", "Prof Josiah S Carberry", true},
		{"José García", "Jose Garcia", true},
		{"Mohammed Hussein", "Muhammad Husain", true},
		{"Ali Al-Rubaie", "Ali Alrubaie", true},
		{"Abdul Rahman Kareem", "Abdulrahman Karim", true},
		{"Josiah Carberry", "Maria Lopez", false},
		{"Josiah Carberry", "Josiah Smith", false},
		{"Ali", "Ali Hassan", false},
		{"Ali A.", "Ali Ahmed", false},
		{"J. C.", "Josiah Carberry", false},
		{"", "Josiah Carberry", false},
		{"", "", false},
		{"123 !!", "Josiah Carberry", false},
	}
	for _, tc := range cases {
		if got := Match(tc.a, tc.b); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMatchAcrossScripts(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"حيدر المشهداني", "Hayder Al-Mashhadani", true},
		{"حيدر المشهداني", "Haider Almashhadani", true},
		{"د. علي حسين كاظم الربيعي", "Ali Al-Rubaie", true},
		{"علي حسين", "Ali Hussein", true},
		{"محمد عبد الرحمن", "Mohammed Abdulrahman", true},
		{"عبد الله كاظم", "Abdullah Kadhim", true},
		{"فاطمة يوسف", "Fatimah Yousif", true},
		{"زينب سعد", "Zainab Saad", true},
		{"أ.د. نور الهدى", "Noor Alhuda", true},
		{"باحث الملف", "Josiah Carberry", false},
		{"علي حسين", "Maria Lopez", false},
		{"محمد", "Mohammed Ali", false},
		{"زينب سعد", "Zainab Kareem", false},
	}
	for _, tc := range cases {
		if got := Match(tc.a, tc.b); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := Match(tc.b, tc.a); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v (reversed)", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestMatchAny(t *testing.T) {
	if !MatchAny([]string{"زينب سعد", "Zainab Saad"}, []string{"Someone Else", "Zainab S. Saad"}) {
		t.Error("any pair should do")
	}
	if MatchAny(nil, []string{"Zainab Saad"}) || MatchAny([]string{"Zainab Saad"}, nil) {
		t.Error("an empty side cannot match")
	}
}

func TestEachPartnerIsUsedOnce(t *testing.T) {
	if Match("Ali Ali", "Ali Hassan") {
		t.Error("one token must not satisfy two")
	}
}
