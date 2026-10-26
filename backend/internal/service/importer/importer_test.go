package importer

import (
	"fmt"
	"reflect"
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
