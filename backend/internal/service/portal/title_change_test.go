package portal

import (
	"strings"
	"testing"
)

func TestTitleChangeWorkflow(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		want     bool
	}{
		{"submit goes straight to scientific affairs", StatusDraft, StatusPending, true},
		{"resubmit after return", StatusReturned, StatusPending, true},
		{"no dean stage", StatusDraft, StatusPendingDean, false},
		{"sa approves directly", StatusPending, StatusApproved, true},
		{"sa rejects directly", StatusPending, StatusRejected, true},
		{"sa returns", StatusPending, StatusReturned, true},
		{"no vp stage", StatusPending, StatusPendingVP, false},
		{"approved is final", StatusApproved, StatusPending, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanTransition(KindTitleChange, c.from, c.to); got != c.want {
				t.Fatalf("CanTransition(title_change, %s, %s) = %v, want %v", c.from, c.to, got, c.want)
			}
		})
	}
	if !IsKind(KindTitleChange) {
		t.Fatal("title_change must be a known kind")
	}
	if got := FormatRequestNo(KindTitleChange, 2026, 7); got != "TC-2026-00007" {
		t.Fatalf("request no = %s", got)
	}
}

func TestFirstStage(t *testing.T) {
	if FirstStage(KindTitleChange) != StatusPending {
		t.Fatal("title change starts at scientific affairs")
	}
	for _, kind := range []string{KindResearchPlan, KindJournalVerification, KindAppreciationLetter} {
		if FirstStage(kind) != StatusPendingDean {
			t.Fatalf("%s must start at the dean", kind)
		}
		if !CanTransition(kind, StatusDraft, FirstStage(kind)) {
			t.Fatalf("%s cannot reach its first stage", kind)
		}
	}
	if !CanTransition(KindTitleChange, StatusDraft, FirstStage(KindTitleChange)) {
		t.Fatal("title change cannot reach its first stage")
	}
}

func TestValidateTitleChange(t *testing.T) {
	valid := map[string]any{"old_title": "عنوان قديم", "reason": "توصية اللجنة العلمية"}
	if errs := ValidateDetails(KindTitleChange, valid); len(errs) != 0 {
		t.Fatalf("valid input rejected: %v", errs)
	}
	errs := ValidateDetails(KindTitleChange, map[string]any{})
	if !contains(errs, "old_title:required") || !contains(errs, "reason:required") {
		t.Fatalf("missing fields not reported: %v", errs)
	}
	long := map[string]any{"old_title": "x", "reason": strings.Repeat("س", maxReasonRunes+1)}
	if !contains(ValidateDetails(KindTitleChange, long), "reason:too_long") {
		t.Fatal("over-long reason accepted")
	}
}

func TestValidatePlanCoResearchers(t *testing.T) {
	base := func(extra map[string]any) map[string]any {
		d := map[string]any{"expected_completion": "2027-01-01"}
		for k, v := range extra {
			d[k] = v
		}
		return d
	}
	cases := []struct {
		name    string
		details map[string]any
		want    string // expected error, "" for none
	}{
		{"individual alone", base(map[string]any{"work_type": "individual"}), ""},
		{"joint with one", base(map[string]any{"work_type": "joint", "co_researcher_1": "أحمد علي"}), ""},
		{"joint with three", base(map[string]any{"work_type": "joint", "co_researcher_1": "أ", "co_researcher_2": "ب", "co_researcher_3": "ج"}), ""},
		{"joint without names", base(map[string]any{"work_type": "joint"}), "co_researchers:required"},
		{"joint with blank names", base(map[string]any{"work_type": "joint", "co_researcher_1": "  "}), "co_researchers:required"},
		{"individual with names", base(map[string]any{"work_type": "individual", "co_researcher_2": "ب"}), "co_researchers:not_allowed"},
		{"fourth name", base(map[string]any{"work_type": "joint", "co_researcher_1": "أ", "co_researcher_4": "د"}), "co_researchers:too_many"},
		{"name too long", base(map[string]any{"work_type": "joint", "co_researcher_1": strings.Repeat("ا", maxCoResearcherRunes+1)}), "co_researcher_1:too_long"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs := ValidateDetails(KindResearchPlan, c.details)
			if c.want == "" && len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if c.want != "" && !contains(errs, c.want) {
				t.Fatalf("want %s, got %v", c.want, errs)
			}
		})
	}
}

func TestCoResearchers(t *testing.T) {
	got := CoResearchers(map[string]any{"co_researcher_1": " أ ", "co_researcher_2": "", "co_researcher_3": "ج"})
	if len(got) != 2 || got[0] != "أ" || got[1] != "ج" {
		t.Fatalf("CoResearchers = %v", got)
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
