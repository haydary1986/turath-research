package handler

import (
	"net/url"
	"strings"
	"testing"
)

func TestParsePublicationFilter(t *testing.T) {
	f, problems := parsePublicationFilter(url.Values{
		"q": {" نانو "}, "author": {"أحمد"}, "year_from": {"2020"}, "year_to": {"2024"},
		"type": {"journal_article"}, "sort": {"cited"}, "page": {"3"}, "limit": {"50"},
		"college_id": {"3f62d3f1-e591-4ad9-b481-f91065c5a13b"},
	})
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if f.text != "نانو" || f.author != "أحمد" || f.yearFrom != 2020 || f.yearTo != 2024 ||
		f.pubType != "journal_article" || f.sort != "cited" || f.page != 3 || f.limit != 50 || f.collegeID == "" {
		t.Fatalf("filter = %+v", f)
	}
}

func TestParsePublicationFilterRejectsBadInput(t *testing.T) {
	cases := map[string]url.Values{
		"type:invalid":             {"type": {"poem"}},
		"sort:invalid":             {"sort": {"p.id; DROP TABLE x"}},
		"year:invalid":             {"year": {"20x4"}},
		"year_from:invalid":        {"year_from": {"1200"}},
		"year_to:before_year_from": {"year_from": {"2024"}, "year_to": {"2020"}},
		"college_id:invalid":       {"college_id": {"not-a-uuid"}},
		"q:too_long":               {"q": {strings.Repeat("x", pubMaxTextRunes+1)}},
	}
	for want, q := range cases {
		_, problems := parsePublicationFilter(q)
		if !contains(problems, want) {
			t.Errorf("%v: want %s, got %v", q, want, problems)
		}
	}
}

func TestPublicationFilterBindsEveryValue(t *testing.T) {
	f, _ := parsePublicationFilter(url.Values{"q": {"50%_off"}, "year": {"2023"}, "author": {"علي"}})
	where, args := f.where()
	if strings.Contains(where, "50%") || strings.Contains(where, "علي") {
		t.Fatalf("user text leaked into SQL: %s", where)
	}
	if len(args) != 4 || args[0] != `%50\%\_off%` {
		t.Fatalf("args = %#v", args)
	}
	if !strings.Contains(where, "make_date($3, 1, 1)") || !strings.Contains(where, "make_date($4 + 1, 1, 1)") {
		t.Fatalf("year range not bound: %s", where)
	}
}

func TestDefaultsAndLimits(t *testing.T) {
	f, _ := parsePublicationFilter(url.Values{"limit": {"5000"}, "page": {"-1"}})
	if f.limit != pubListDefaultLimit || f.page != 1 || f.sort != "newest" {
		t.Fatalf("defaults = %+v", f)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
