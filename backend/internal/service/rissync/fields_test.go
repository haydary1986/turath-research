package rissync

import "testing"

func TestCanonicalLinkMakesSameProfileEqual(t *testing.T) {
	pairs := []struct{ field, a, b string }{
		{FieldORCID, "https://orcid.org/0000-0002-1825-0097", "0000-0002-1825-0097"},
		{FieldScholar, "https://scholar.google.com/citations?user=AbCdEf123&hl=ar", "https://scholar.google.com/citations?hl=en&user=AbCdEf123"},
		{FieldLinkedIn, "https://www.linkedin.com/in/some-one/", "https://linkedin.com/in/some-one"},
	}
	for _, p := range pairs {
		a, okA := CanonicalLink(p.field, p.a)
		b, okB := CanonicalLink(p.field, p.b)
		if !okA || !okB || a != b {
			t.Errorf("%s: %q (%v) vs %q (%v)", p.field, a, okA, b, okB)
		}
	}
	if _, ok := CanonicalLink(FieldScholar, "https://evil.example/citations?user=x"); ok {
		t.Error("a link on the wrong host must not be accepted")
	}
	if v, ok := CanonicalLink(FieldScopus, "  "); !ok || v != "" {
		t.Error("blank is a valid empty value")
	}
	if RISLinkURL(FieldORCID, "0000-0002-1825-0097") != "https://orcid.org/0000-0002-1825-0097" {
		t.Error("ORCID must be stored in RIS as a URL")
	}
}

func TestCanonicalPhoneAndName(t *testing.T) {
	if CanonicalPhone("+964 770 123 4567") != CanonicalPhone("07701234567") || CanonicalPhone("07701234567") == "" {
		t.Error("the same mobile written two ways must compare equal")
	}
	if CanonicalPhone("12345") != "" {
		t.Error("an invalid number is not synced")
	}
	if CanonicalName("  أحمد   علي ‏") != "أحمد علي" {
		t.Errorf("name = %q", CanonicalName("  أحمد   علي ‏"))
	}
}

func TestDepartmentMap(t *testing.T) {
	pc := []Org{{ID: "pc1", NameAr: "كلية الآداب"}, {ID: "pc2", NameAr: "كلية الهندسة"}}
	pd := []Org{
		{ID: "pd1", NameAr: "قسم اللغة الإنجليزية", CollegeID: "pc1"},
		{ID: "pd2", NameAr: "هندسة الحاسوب", CollegeID: "pc2"},
		{ID: "pd3", NameAr: "اللغة العربية", CollegeID: "pc1"},
	}
	rc := []Org{{ID: "rc1", NameAr: "كليه الاداب"}, {ID: "rc2", NameAr: "كلية الهندسة"}}
	rd := []Org{
		{ID: "rd1", NameAr: "اللغة الإنجليزية - آداب", CollegeID: "rc1"},
		{ID: "rd2", NameAr: "هندسة الحاسوب", CollegeID: "rc2"},
		{ID: "rd3", NameAr: "هندسة الطيران", CollegeID: "rc2"},
	}
	m := BuildDepartmentMap(pc, pd, rc, rd)
	if id, ok := m.ToRIS("pd1"); !ok || id != "rd1" {
		t.Errorf("qualifier after a dash must be ignored: %q %v", id, ok)
	}
	if id, ok := m.ToPortal("rd2"); !ok || id != "pd2" {
		t.Errorf("same name must match: %q %v", id, ok)
	}
	if _, ok := m.ToRIS("pd3"); ok {
		t.Error("a portal department with no RIS counterpart must stay unmapped")
	}
	if len(m.UnmappedRIS) != 1 || m.UnmappedRIS[0] != "هندسة الطيران" {
		t.Errorf("unmapped = %v", m.UnmappedRIS)
	}
}
