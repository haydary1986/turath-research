package handler

import "testing"

func TestMapScreeningHeaderFoldsArabic(t *testing.T) {
	header, missing := mapScreeningHeader([]string{"اسم المجلة", "ISSN", "التصنيف"})
	if len(missing) != 0 {
		t.Fatalf("missing = %v", missing)
	}
	if header[0] != "name" || header[1] != "issn" || header[2] != "classification" {
		t.Fatalf("header = %v", header)
	}
}

func TestMapScreeningHeaderReportsMissing(t *testing.T) {
	_, missing := mapScreeningHeader([]string{"ISSN"})
	if len(missing) != 2 {
		t.Fatalf("missing = %v", missing)
	}
}

func TestParseScreeningRows(t *testing.T) {
	header := map[int]string{0: "name", 1: "issn", 2: "classification"}
	valid, problems := parseScreeningRows(header, [][]string{
		{"Good", "2049-3630", "مفترسة"},
		{"", "", ""},
		{"", "2049-3630", "مفترسة"},
		{"Bad class", "", "شيء"},
		{"Bad issn", "12", "معتمدة"},
	}, 2)
	if len(valid) != 1 || valid[0].Classification != "predatory" || valid[0].Line != 2 {
		t.Fatalf("valid = %+v", valid)
	}
	want := map[int]string{4: "name_required", 5: "classification_invalid", 6: "issn_malformed"}
	if len(problems) != len(want) {
		t.Fatalf("problems = %+v", problems)
	}
	for _, p := range problems {
		if want[p.Line] != p.Code {
			t.Errorf("line %d: code %q, want %q", p.Line, p.Code, want[p.Line])
		}
	}
}

func TestScreeningRowErrorNamesLine(t *testing.T) {
	err := screeningRowError{line: 42, err: errString("boom")}
	if err.Error() != "line 42: boom" {
		t.Fatalf("got %q", err.Error())
	}
}

type errString string

func (e errString) Error() string { return string(e) }
