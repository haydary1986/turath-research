package handler

import (
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
)

func TestPublicCheckResultHidesWatchlistInternals(t *testing.T) {
	note := "internal reason"
	in := journalcheck.Result{
		Matched: true, Classification: "predatory", Name: "Watchlisted", Note: &note, MatchedBy: "issn",
		Verdict: journalcheck.VerdictDanger, Reasons: []string{journalcheck.ReasonManualPredatory},
	}
	out := publicCheckResult(in)
	if out.Matched || out.Classification != "" || out.Name != "" || out.Note != nil || out.MatchedBy != "" {
		t.Fatalf("watchlist internals leaked: %+v", out)
	}
	if out.Verdict != journalcheck.VerdictDanger || len(out.Reasons) != 1 {
		t.Fatalf("verdict lost: %+v", out)
	}
	if !in.Matched || in.Note == nil {
		t.Fatal("input was modified")
	}
}

func TestSlugShape(t *testing.T) {
	good := []string{"abc-journal-of-science-5762630ef0", "e1e3920012"}
	bad := []string{"", "-abc", "ABC", "a/b", "a b", "a_b"}
	for _, s := range good {
		if !slugShape.MatchString(s) {
			t.Errorf("%q rejected", s)
		}
	}
	for _, s := range bad {
		if slugShape.MatchString(s) {
			t.Errorf("%q accepted", s)
		}
	}
}
