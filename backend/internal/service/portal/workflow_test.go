package portal

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		from, to string
		want     bool
	}{
		{"draft submits to dean", KindResearchPlan, StatusDraft, StatusPendingDean, true},
		{"returned resubmits to dean", KindJournalVerification, StatusReturned, StatusPendingDean, true},
		{"submit cannot skip the dean", KindResearchPlan, StatusDraft, StatusPending, false},
		{"dean forwards to scientific affairs", KindResearchPlan, StatusPendingDean, StatusPending, true},
		{"dean returns", KindResearchPlan, StatusPendingDean, StatusReturned, true},
		{"dean cannot decide", KindResearchPlan, StatusPendingDean, StatusApproved, false},
		{"sa raises recommendation", KindResearchPlan, StatusPending, StatusPendingVP, true},
		{"sa cannot finalize alone", KindResearchPlan, StatusPending, StatusApproved, false},
		{"vp confirms approval", KindResearchPlan, StatusPendingVP, StatusApproved, true},
		{"vp confirms rejection", KindResearchPlan, StatusPendingVP, StatusRejected, true},
		{"vp sends back to sa", KindResearchPlan, StatusPendingVP, StatusPending, true},
		{"vp confirms journal verified", KindJournalVerification, StatusPendingVP, StatusVerified, true},
		{"journal flows to signatures", KindJournalVerification, StatusVerified, StatusAwaitingSignatures, true},
		{"journal completes after signatures", KindJournalVerification, StatusAwaitingSignatures, StatusCompleted, true},
		{"journal cannot skip signatures", KindJournalVerification, StatusVerified, StatusCompleted, false},
		{"letter issued after verified", KindAppreciationLetter, StatusVerified, StatusIssued, true},
		{"letter cannot issue from vp stage", KindAppreciationLetter, StatusPendingVP, StatusIssued, false},
		{"terminal state is final", KindResearchPlan, StatusApproved, StatusPending, false},
		{"unknown kind", "nope", StatusPending, StatusApproved, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanTransition(c.kind, c.from, c.to); got != c.want {
				t.Fatalf("CanTransition(%s, %s, %s) = %v, want %v", c.kind, c.from, c.to, got, c.want)
			}
		})
	}
}

func TestRequiredRole(t *testing.T) {
	cases := map[string]string{
		StatusPendingDean:        RoleCollegeDean,
		StatusPending:            RoleScientificAffairs,
		StatusPendingVP:          RoleVPResearch,
		StatusVerified:           RoleScientificAffairs,
		StatusAwaitingSignatures: RoleScientificAffairs,
		StatusDraft:              "",
		StatusApproved:           "",
	}
	for from, want := range cases {
		if got := RequiredRole(from); got != want {
			t.Errorf("RequiredRole(%s) = %q, want %q", from, got, want)
		}
	}
}

func TestFinalDecisions(t *testing.T) {
	if !IsFinalDecision(KindResearchPlan, StatusApproved) || IsFinalDecision(KindResearchPlan, StatusVerified) {
		t.Fatal("plan decisions wrong")
	}
	if !IsFinalDecision(KindJournalVerification, StatusNotEligible) || IsFinalDecision(KindJournalVerification, StatusApproved) {
		t.Fatal("journal decisions wrong")
	}
}

func TestRequiresNote(t *testing.T) {
	for _, s := range []string{StatusRejected, StatusNotEligible, StatusReturned} {
		if !RequiresNote(s) {
			t.Errorf("%s must require a note", s)
		}
	}
	for _, s := range []string{StatusApproved, StatusVerified, StatusCompleted} {
		if RequiresNote(s) {
			t.Errorf("%s must not require a note", s)
		}
	}
}

func TestIsStaff(t *testing.T) {
	for _, r := range []string{"admin", "scientific_affairs", "vp_research"} {
		if !IsStaff(r) {
			t.Errorf("%s should be staff", r)
		}
	}
	for _, r := range []string{"researcher", "viewer", "department_head", "dean_research", "college_dean", ""} {
		if IsStaff(r) {
			t.Errorf("%s should not be staff", r)
		}
	}
}

func TestNormalizeDOI(t *testing.T) {
	cases := map[string]string{
		"10.1016/J.Enbuild.2024.114567":                    "10.1016/j.enbuild.2024.114567",
		"  https://doi.org/10.1016/j.enbuild.2024.114567 ": "10.1016/j.enbuild.2024.114567",
		"doi:10.1000/XYZ":                                  "10.1000/xyz",
		"":                                                 "",
	}
	for in, want := range cases {
		if got := NormalizeDOI(in); got != want {
			t.Errorf("NormalizeDOI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateDetails(t *testing.T) {
	t.Run("plan needs expected completion date", func(t *testing.T) {
		errs := ValidateDetails(KindResearchPlan, map[string]any{"work_type": "individual"})
		if len(errs) == 0 {
			t.Fatal("expected errors")
		}
	})
	t.Run("plan valid", func(t *testing.T) {
		errs := ValidateDetails(KindResearchPlan, map[string]any{
			"work_type":           "joint",
			"expected_completion": "2027-06-30",
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
	})
	t.Run("plan rejects bad work type", func(t *testing.T) {
		errs := ValidateDetails(KindResearchPlan, map[string]any{
			"work_type":           "group",
			"expected_completion": "2027-06-30",
		})
		if len(errs) == 0 {
			t.Fatal("expected errors")
		}
	})
	t.Run("journal rejects unknown index db", func(t *testing.T) {
		errs := ValidateDetails(KindJournalVerification, map[string]any{
			"journal_name": "X", "publisher": "Y", "index_db": "bing", "publication_date": "2024-03-15",
		})
		if len(errs) == 0 {
			t.Fatal("expected errors")
		}
	})
	t.Run("journal valid", func(t *testing.T) {
		errs := ValidateDetails(KindJournalVerification, map[string]any{
			"journal_name": "Energy and Buildings", "publisher": "Elsevier",
			"index_db": "scopus", "publication_date": "2024-03-15", "issn": "0378-7788",
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
	})
	t.Run("journal rejects malformed issn", func(t *testing.T) {
		errs := ValidateDetails(KindJournalVerification, map[string]any{
			"journal_name": "X", "publisher": "Y", "index_db": "local",
			"publication_date": "2024-03-15", "issn": "12345",
		})
		if len(errs) == 0 {
			t.Fatal("expected errors")
		}
	})
	t.Run("rejects malformed date", func(t *testing.T) {
		errs := ValidateDetails(KindJournalVerification, map[string]any{
			"journal_name": "X", "publisher": "Y", "index_db": "local", "publication_date": "15/03/2024",
		})
		if len(errs) == 0 {
			t.Fatal("expected errors")
		}
	})
	t.Run("unknown kind", func(t *testing.T) {
		if len(ValidateDetails("nope", nil)) == 0 {
			t.Fatal("expected errors")
		}
	})
}

func TestRequiredAttachments(t *testing.T) {
	if got := RequiredAttachments(KindAppreciationLetter, false); len(got) != 2 {
		t.Fatalf("letter without related request needs 2 attachments, got %v", got)
	}
	got := RequiredAttachments(KindAppreciationLetter, true)
	if len(got) != 1 || got[0] != RoleCommitteeApproval {
		t.Fatalf("letter with related verification needs only committee approval, got %v", got)
	}
	if len(RequiredAttachments(KindResearchPlan, false)) != 0 {
		t.Fatal("plan has no required attachments")
	}
}

func TestAllowedAttachmentRole(t *testing.T) {
	if !AllowedAttachmentRole(KindJournalVerification, RoleSignedForm) {
		t.Error("journal verification accepts signed form")
	}
	if AllowedAttachmentRole(KindResearchPlan, RoleCommitteeApproval) {
		t.Error("plan must not accept committee approval")
	}
	if AllowedAttachmentRole(KindResearchPlan, "random") {
		t.Error("unknown role must be rejected")
	}
}

func TestFormatRequestNo(t *testing.T) {
	if got := FormatRequestNo(KindResearchPlan, 2026, 124); got != "RP-2026-00124" {
		t.Fatalf("got %s", got)
	}
	if got := FormatRequestNo(KindJournalVerification, 2026, 1); got != "JV-2026-00001" {
		t.Fatalf("got %s", got)
	}
	if got := FormatRequestNo(KindAppreciationLetter, 2026, 99999); got != "AR-2026-99999" {
		t.Fatalf("got %s", got)
	}
}
