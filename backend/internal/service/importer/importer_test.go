package importer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestMapHeaderAcceptsArabicAndEnglishAliases(t *testing.T) {
	header, missing := MapHeader([]string{"الاسم", "الاسم بالإنجليزية", "البريد الإلكتروني", "اللقب العلمي", "الكلية", "القسم", "ORCID"})
	if len(missing) != 0 {
		t.Fatalf("unexpected missing: %v", missing)
	}
	want := map[int]string{0: "name_ar", 1: "name_en", 2: "email", 3: "title_ar", 4: "college", 5: "department", 6: "orcid"}
	if !reflect.DeepEqual(header, want) {
		t.Fatalf("got %v, want %v", header, want)
	}

	header, missing = MapHeader([]string{"Name (Arabic)", "Email"})
	if len(missing) != 0 || header[0] != "name_ar" || header[1] != "email" {
		t.Fatalf("english aliases failed: %v %v", header, missing)
	}
}

func TestMapHeaderReportsMissingRequired(t *testing.T) {
	_, missing := MapHeader([]string{"الاسم", "الكلية"})
	if !reflect.DeepEqual(missing, []string{"email"}) {
		t.Fatalf("got %v", missing)
	}
	_, missing = MapHeader(nil)
	if len(missing) != 2 {
		t.Fatalf("got %v", missing)
	}
}

func TestParseRows(t *testing.T) {
	header, _ := MapHeader([]string{"الاسم", "البريد الالكتروني", "القسم"})
	rows := [][]string{
		{"  د. أحمد محمد  ", " Ahmed@UOTurath.edu.iq ", "هندسة الحاسوب"},
		{"", "x@uoturath.edu.iq", ""},                 // missing name
		{"د. سارة", "not-an-email", ""},               // bad email
		{"د. أحمد مكرر", "ahmed@uoturath.edu.iq", ""}, // duplicate email (case-insensitive)
		{"", "", ""}, // blank row → skipped silently
		{"د. مريم", "mariam@uoturath.edu.iq", "الصيدلة"},
	}
	valid, problems := ParseRows(header, rows, 2)

	if len(valid) != 2 {
		t.Fatalf("want 2 valid rows, got %d: %+v", len(valid), valid)
	}
	if valid[0].NameAr != "د. أحمد محمد" || valid[0].Email != "ahmed@uoturath.edu.iq" || valid[0].Department != "هندسة الحاسوب" {
		t.Fatalf("row not normalized: %+v", valid[0])
	}
	if valid[0].Line != 2 || valid[1].Line != 7 {
		t.Fatalf("line numbers wrong: %d, %d", valid[0].Line, valid[1].Line)
	}

	codes := map[int]string{}
	for _, p := range problems {
		codes[p.Line] = p.Code
	}
	want := map[int]string{3: "name_required", 4: "invalid_email", 5: "duplicate_email"}
	if !reflect.DeepEqual(codes, want) {
		t.Fatalf("problems = %v, want %v", codes, want)
	}
}

func TestParseRowsLimit(t *testing.T) {
	header, _ := MapHeader([]string{"الاسم", "البريد"})
	rows := make([][]string, MaxRows+5)
	for i := range rows {
		rows[i] = []string{"اسم", fmt.Sprintf("u%d@x.iq", i)}
	}
	_, problems := ParseRows(header, rows, 1)
	found := false
	for _, p := range problems {
		if p.Code == "too_many_rows" {
			found = true
		}
	}
	if !found {
		t.Fatal("row limit not enforced")
	}
}

func TestParseRowsCanonicalizesORCID(t *testing.T) {
	header := map[int]string{0: "name_ar", 1: "email", 2: "orcid"}
	valid, problems := ParseRows(header, [][]string{
		{"باحث", "a@test.iq", "https://orcid.org/0000-0002-1694-233x"},
		{"باحث", "b@test.iq", "not-an-orcid"},
		{"باحث", "c@test.iq", ""},
	}, 2)
	if len(valid) != 2 || valid[0].ORCID != "0000-0002-1694-233X" || valid[1].ORCID != "" {
		t.Errorf("valid = %+v", valid)
	}
	if len(problems) != 1 || problems[0].Code != "invalid_orcid" || problems[0].Line != 3 {
		t.Errorf("problems = %+v", problems)
	}
}

func TestMapHeaderKnowsTheContactAndLinkColumns(t *testing.T) {
	cells := []string{"name_ar", "email", "phone", "phone_secondary", "recovery_email", "google_scholar_url", "scopus_url",
		"researchgate_url", "linkedin_url", "web_of_science_url", "orcid", "employee_no", "title_ar", "name_en",
		"college", "department", "account_status"}
	header, missing := MapHeader(cells)
	if len(missing) != 0 {
		t.Fatalf("missing: %v", missing)
	}
	for i, cell := range cells {
		if header[i] != cell {
			t.Errorf("%q mapped to %q", cell, header[i])
		}
	}
}

func TestParseRowsNormalizesOptionalCells(t *testing.T) {
	header, _ := MapHeader([]string{"name_ar", "email", "phone", "phone_secondary", "recovery_email",
		"web_of_science_url", "google_scholar_url", "employee_no", "account_status"})
	valid, problems := ParseRows(header, [][]string{{
		"باحث", "a@test.iq", "7701234567", "+964 780 123 4567", "Backup@Example.org",
		"https://www.webofscience.com/wos/author/record/123456?x=1", "https://scholar.google.com/citations?user=AbC-12_xyzAAJ&hl=ar",
		"E-77", "needs_review",
	}}, 2)
	if len(problems) != 0 || len(valid) != 1 {
		t.Fatalf("valid=%d problems=%+v", len(valid), problems)
	}
	row := valid[0]
	if row.Phone != "07701234567" || row.PhoneSecondary != "07801234567" || row.RecoveryEmail != "backup@example.org" {
		t.Errorf("contact not normalized: %q %q %q", row.Phone, row.PhoneSecondary, row.RecoveryEmail)
	}
	if row.Links.WebOfScienceURL != "https://www.webofscience.com/wos/author/record/123456" || row.Links.GoogleScholarID != "AbC-12_xyzAAJ" {
		t.Errorf("links not canonical: %+v", row.Links)
	}
	if row.EmployeeNo != "E-77" || row.AccountStatus != "needs_review" {
		t.Errorf("row = %+v", row)
	}
}

func TestParseRowsReportsEveryBadOptionalCell(t *testing.T) {
	header, _ := MapHeader([]string{"name_ar", "email", "phone", "recovery_email", "scopus_url", "web_of_science_url", "account_status"})
	valid, problems := ParseRows(header, [][]string{
		{"باحث", "a@test.iq", "12345", "nope", "https://evil.example/x", "https://webofscience.com.evil.org/wos/author/record/123", "active"},
		{"باحث", "a@test.iq", "", "", "", "", ""},
	}, 2)
	if len(valid) != 1 || valid[0].Line != 3 {
		t.Fatalf("a rejected row must not claim its email: %+v", valid)
	}
	got := map[string]bool{}
	for _, p := range problems {
		got[p.Code] = p.Line == 2
	}
	for _, code := range []string{"invalid_phone", "invalid_recovery_email", "invalid_scopus_url", "invalid_web_of_science_url", "invalid_account_status"} {
		if !got[code] {
			t.Errorf("missing %s in %+v", code, problems)
		}
	}
}

func TestPreviewCarriesNoContactData(t *testing.T) {
	data, err := json.Marshal(Row{Phone: "07701234567", PhoneSecondary: "07801234567", RecoveryEmail: "x@y.iq", EmployeeNo: "E-1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"07701234567", "07801234567", "x@y.iq", "E-1"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("preview leaks %s", secret)
		}
	}
}
