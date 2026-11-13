package handler

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
	"github.com/haydary1986/turath-research/backend/internal/service/fs003"
)

func TestResponsibilityClause(t *testing.T) {
	scopes := []activity.Scope{
		{Category: activity.CategoryInternationalAgreements},
		{Category: activity.CategoryLocalAgreements, CollegeID: "c1"},
		{Category: activity.CategoryInfrastructure},
	}
	clause, args := responsibilityClause(scopes, []any{"u1"})
	want := `(a.category = ANY($4::text[]) OR (a.category = $2 AND a.college_id = $3))`
	if clause != want {
		t.Errorf("clause = %q", clause)
	}
	wantArgs := []any{"u1", activity.CategoryLocalAgreements, "c1",
		[]string{activity.CategoryInternationalAgreements, activity.CategoryInfrastructure}}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %#v", args)
	}
	if clause, args := responsibilityClause(nil, []any{"u1"}); clause != "" || len(args) != 1 {
		t.Errorf("no responsibility adds nothing: %q %v", clause, args)
	}
}

func TestVisibleActivities(t *testing.T) {
	researcher := &middleware.UserClaims{UserID: "u1", Role: middleware.RoleResearcher}
	where, args := visibleActivities(researcher, []activity.Scope{{Category: activity.CategoryBooks, CollegeID: "c9"}})
	if where != "(a.created_by = $1 OR (a.category = $2 AND a.college_id = $3))" || len(args) != 3 {
		t.Errorf("holder sees own records and the scope: %q %v", where, args)
	}
	if where, _ := visibleActivities(researcher, nil); where != "a.created_by = $1" {
		t.Errorf("plain researcher: %q", where)
	}
	staff := &middleware.UserClaims{UserID: "s", Role: middleware.RoleScientificAffairs}
	if where, args := visibleActivities(staff, []activity.Scope{{Category: activity.CategoryBooks}}); where != "TRUE" || args != nil {
		t.Errorf("staff already see everything: %q %v", where, args)
	}
}

func TestResearchItemMapping(t *testing.T) {
	base := fs003Item{isoDate: "2026-03-15", row: fs003.Row{Title: "AI paper"}}
	cases := []struct{ indexDB, category, detail string }{
		{"local", activity.CategoryLocalResearch, "مجلة التراث"},
		{"scopus", activity.CategoryIndexedResearch, "Scopus"},
		{"wos", activity.CategoryIndexedResearch, "Web of Science"},
		{"international_unindexed", activity.CategoryInternationalResearch, "عالمية غير مستوعبة"},
	}
	for _, c := range cases {
		got, ok := researchItem(c.indexDB, "مجلة التراث", "د. أحمد", base)
		if !ok || got.category != c.category || got.row.Detail != c.detail || got.row.Notes != "الباحث: د. أحمد" {
			t.Errorf("%s: %+v", c.indexDB, got)
		}
	}
	if _, ok := researchItem("", "x", "y", base); ok {
		t.Error("an unknown index database is left out")
	}
}

func TestGroupFS003ScopesAndOrders(t *testing.T) {
	c1, c2 := "c1", "c2"
	items := []fs003Item{
		{category: activity.CategoryLocalAgreements, collegeID: &c1, isoDate: "2026-11-09", row: fs003.Row{Title: "late"}},
		{category: activity.CategoryLocalAgreements, collegeID: &c1, isoDate: "2026-11-02", row: fs003.Row{Title: "early"}},
		{category: activity.CategoryLocalAgreements, collegeID: &c2, isoDate: "2026-11-03", row: fs003.Row{Title: "other college"}},
		{category: activity.CategoryBooks, collegeID: &c1, isoDate: "2026-11-01", row: fs003.Row{Title: "book"}},
	}
	holder := activity.Actor{UserID: "h", Role: "researcher", Scopes: []activity.Scope{{Category: activity.CategoryLocalAgreements, CollegeID: c1}}}
	got := groupFS003(holder, items)
	rows := got[activity.CategoryLocalAgreements]
	if len(got) != 1 || len(rows) != 2 || rows[0].Title != "early" || rows[1].Title != "late" {
		t.Fatalf("holder export: %+v", got)
	}
	if rows[0].Date != "02/11/2026" {
		t.Errorf("date = %q", rows[0].Date)
	}
	all := groupFS003(activity.Actor{UserID: "s", Role: "vp_research"}, items)
	if len(all[activity.CategoryLocalAgreements]) != 3 || len(all[activity.CategoryBooks]) != 1 {
		t.Errorf("staff export everything: %+v", all)
	}
}

func TestNormalizeResponsibilities(t *testing.T) {
	scopes, problems := normalizeResponsibilities([]responsibilityInput{
		{Category: activity.CategoryInternationalAgreements},
		{Category: activity.CategoryInternationalAgreements, CollegeID: " "},
		{Category: activity.CategoryLocalAgreements, CollegeID: "8f14e45f-ceea-467f-a0e6-0d6a1c8f1d2b"},
		{Category: "sheet8"},
		{Category: activity.CategoryBooks, CollegeID: "not-a-uuid"},
	})
	if len(scopes) != 2 {
		t.Errorf("duplicates collapse: %+v", scopes)
	}
	if strings.Join(problems, ",") != "items.3.category:invalid,items.4.college_id:invalid" {
		t.Errorf("problems = %v", problems)
	}
	if _, problems := normalizeResponsibilities(make([]responsibilityInput, maxResponsibilities+1)); len(problems) != 1 {
		t.Error("too many items")
	}
}

func TestLoginEventFilters(t *testing.T) {
	q := url.Values{"q": {"ahmed"}, "method": {"password"}, "result": {"failure"}, "from": {"2026-11-01"}, "to": {"2026-11-13"}}
	conds, args, problems := loginEventFilters(q)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	joined := strings.Join(conds, " AND ")
	for _, part := range []string{"e.email ILIKE $1", "e.method = $2", "NOT e.success",
		"(e.created_at AT TIME ZONE 'Asia/Baghdad')::date >= $3::date", "<= $4::date"} {
		if !strings.Contains(joined, part) {
			t.Errorf("missing %q in %q", part, joined)
		}
	}
	if len(args) != 4 || args[1] != "password" || args[2] != "2026-11-01" {
		t.Errorf("args = %v", args)
	}
	_, _, problems = loginEventFilters(url.Values{"method": {"sms"}, "result": {"maybe"}, "from": {"13/11/2026"}, "user_id": {"x"}})
	if strings.Join(problems, ",") != "user_id:invalid,method:invalid,result:invalid,from:invalid_date" {
		t.Errorf("problems = %v", problems)
	}
}

func TestEveryCategoryHasAnArabicReportLabel(t *testing.T) {
	for _, c := range activity.Categories {
		if activityCategoryAr[c] == "" {
			t.Errorf("%s has no Arabic label in the annual report", c)
		}
	}
}

func TestMayLinkByEmail(t *testing.T) {
	if mayLinkByEmail(false, false) {
		t.Error("an unverified email must not claim an existing researcher record")
	}
	if !mayLinkByEmail(true, false) {
		t.Error("a Google-verified email claims the matching record")
	}
	if !mayLinkByEmail(false, true) {
		t.Error("a record created for the applicant's own request is theirs")
	}
	if verifiedEmailSQL != "u.google_sub IS NOT NULL" {
		t.Error("only Google sign-in verifies an email")
	}
}
