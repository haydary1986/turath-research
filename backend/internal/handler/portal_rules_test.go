package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

func TestOfferedTransitionsHidesSubmitFromEditable(t *testing.T) {
	for _, status := range []string{portal.StatusDraft, portal.StatusReturned} {
		got := offeredTransitions(requestRecord{Kind: portal.KindResearchPlan, Status: status})
		if len(got) != 0 {
			t.Errorf("%s offered %v", status, got)
		}
	}
}

func TestOfferedTransitionsAtVPStage(t *testing.T) {
	decision := portal.StatusApproved
	got := offeredTransitions(requestRecord{Kind: portal.KindResearchPlan, Status: portal.StatusPendingVP, SADecision: &decision})
	if want := []string{portal.StatusApproved, portal.StatusPending}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestStageHolderIsApplicant(t *testing.T) {
	a, b := "college-a", "college-b"
	cases := []struct {
		name             string
		stage, role      string
		applicant, owner *string
		want             bool
	}{
		{"dean of the request's college", portal.StatusPendingDean, portal.RoleCollegeDean, &a, &a, true},
		{"dean of another college", portal.StatusPendingDean, portal.RoleCollegeDean, &b, &a, false},
		{"scientific affairs", portal.StatusPending, portal.RoleScientificAffairs, nil, &a, true},
		{"vp", portal.StatusPendingVP, portal.RoleVPResearch, nil, &a, true},
		{"researcher", portal.StatusPendingDean, portal.RoleResearcher, &a, &a, false},
		{"terminal stage", portal.StatusApproved, portal.RoleAdmin, nil, nil, false},
	}
	for _, c := range cases {
		if got := stageHolderIsApplicant(c.stage, c.role, c.applicant, c.owner); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestPickAuthorOrder(t *testing.T) {
	cases := []struct {
		wanted int
		taken  []int
		want   int
	}{
		{0, nil, 1},
		{3, nil, 3},
		{1, []int{1}, 2},
		{2, []int{1, 2, 3}, 4},
		{2, []int{2, 5}, 3},
		{-4, []int{2}, 1},
	}
	for _, c := range cases {
		if got := pickAuthorOrder(c.wanted, c.taken); got != c.want {
			t.Errorf("pickAuthorOrder(%d, %v) = %d, want %d", c.wanted, c.taken, got, c.want)
		}
	}
}

func TestAuthorOrderOf(t *testing.T) {
	cases := map[string]struct {
		in   any
		want int
	}{
		"text": {" 3 ", 3}, "number": {float64(2), 2}, "fraction": {1.5, 0},
		"garbage": {"x", 0}, "zero": {"0", 0}, "absent": {nil, 0},
	}
	for name, c := range cases {
		if got := authorOrderOf(c.in); got != c.want {
			t.Errorf("%s: got %d, want %d", name, got, c.want)
		}
	}
}

func TestMayDecideActivity(t *testing.T) {
	sa := &middleware.UserClaims{UserID: "u1", Role: portal.RoleScientificAffairs}
	admin := &middleware.UserClaims{UserID: "u1", Role: portal.RoleAdmin}
	if mayDecideActivity(sa, "u1") {
		t.Error("scientific affairs must not decide its own activity")
	}
	if !mayDecideActivity(sa, "u2") {
		t.Error("scientific affairs decides others' activities")
	}
	if !mayDecideActivity(admin, "u1") {
		t.Error("admin is excepted")
	}
}

func TestDecodeOptionalBody(t *testing.T) {
	type payload struct {
		V string `json:"v"`
	}
	cases := []struct {
		body string
		ok   bool
		code int
	}{
		{"", true, http.StatusOK},
		{"   ", true, http.StatusOK},
		{`{"v":"x"}`, true, http.StatusOK},
		{`{"v":`, false, http.StatusBadRequest},
		{`not json`, false, http.StatusBadRequest},
		{`{"v":"` + strings.Repeat("a", maxJSONBody) + `"}`, false, http.StatusBadRequest},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body))
		var dst payload
		if got := decodeOptionalBody(w, r, &dst); got != c.ok || w.Code != c.code {
			t.Errorf("body len %d: ok=%v code=%d", len(c.body), got, w.Code)
		}
	}
}

func TestDiwanChangeNote(t *testing.T) {
	next := "12"
	if got := diwanChangeNote(nil, &next); !strings.Contains(got, "—") || !strings.Contains(got, "12") {
		t.Fatalf("got %q", got)
	}
}

func TestReportRange(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	valid := []struct {
		query, file string
	}{
		{"", "turath-report-2026.xlsx"},
		{"year=2024", "turath-report-2024.xlsx"},
		{"from=2026-01-01&to=2026-03-31", "turath-report-2026-01-01_2026-03-31.xlsx"},
		{"year=2025&from=2026-04-01&to=2026-06-30", "turath-report-2026-04-01_2026-06-30.xlsx"},
	}
	for _, c := range valid {
		q, _ := url.ParseQuery(c.query)
		period, problems := reportRange(q, now)
		if len(problems) != 0 || period.filename() != c.file {
			t.Errorf("%q: problems=%v file=%s", c.query, problems, period.filename())
		}
	}
	invalid := []string{
		"year=abc", "year=1999", "year=2101",
		"from=2026-01-01", "to=2026-01-01", "from=x&to=2026-01-01",
		"from=2026-02-30&to=2026-03-01", "from=2026-03-01&to=2026-01-01",
		"year=abc&from=2026-01-01&to=2026-03-31",
	}
	for _, query := range invalid {
		q, _ := url.ParseQuery(query)
		if _, problems := reportRange(q, now); len(problems) == 0 {
			t.Errorf("%q should be rejected", query)
		}
	}
}

func TestReportSheetsStayAligned(t *testing.T) {
	for _, sheet := range reportSheets() {
		for index := range sheet.transform {
			if index >= len(sheet.headers) {
				t.Errorf("%s: transform %d is past the last column", sheet.name, index)
			}
		}
		selected := sheet.query[:strings.Index(sheet.query, "FROM ")]
		// Top-level commas only: COALESCE and to_char arguments sit inside parentheses.
		depth, columns := 0, 1
		for _, ch := range selected {
			switch ch {
			case '(':
				depth++
			case ')':
				depth--
			case ',':
				if depth == 0 {
					columns++
				}
			}
		}
		if columns != len(sheet.headers) {
			t.Errorf("%s: %d columns, %d headers", sheet.name, columns, len(sheet.headers))
		}
	}
	journal := reportSheets()[0]
	if journal.headers[18] != "نسبة الاستلال" || journal.headers[20] != "حالة الطلب" {
		t.Fatalf("headers moved: %v", journal.headers)
	}
	if got := journal.transform[20]("completed"); got != "مكتملة" {
		t.Fatalf("status transform = %q", got)
	}
	if got := journal.transform[17]("extracted_phd"); got != "مستل من أطروحة دكتوراه" {
		t.Fatalf("extracted transform = %q", got)
	}
}

func TestReportDateGuards(t *testing.T) {
	for _, want := range []string{"AT TIME ZONE 'Asia/Baghdad'", "CASE", "r.status IN"} {
		if !strings.Contains(reportPublicationDate, want) {
			t.Errorf("publication date expression lacks %q", want)
		}
	}
}

func TestLongRunningContextOutlivesRequestAndWriteTimeout(t *testing.T) {
	const serverWriteTimeout = 100 * time.Millisecond
	var ctxErr error
	handler := chimw.Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parent, cancel := context.WithCancel(r.Context())
		cancel()
		ctx, done := longRunningContext(w, r.WithContext(parent), time.Minute)
		defer done()
		time.Sleep(3 * serverWriteTimeout)
		ctxErr = ctx.Err()
		_, _ = w.Write([]byte("ok"))
	}))
	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = serverWriteTimeout
	server.Start()
	defer server.Close()

	res, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("response was cut by the write timeout: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if ctxErr != nil {
		t.Fatalf("context cancelled with its parent: %v", ctxErr)
	}
}
