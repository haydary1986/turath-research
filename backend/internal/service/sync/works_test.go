package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
)

func TestNormalizeDOI(t *testing.T) {
	cases := map[string]string{
		"https://doi.org/10.1016/J.Test.1": "10.1016/j.test.1",
		"http://dx.doi.org/10.1/x":         "10.1/x",
		"doi: 10.1/X":                      "10.1/x",
		" 10.1/x ":                         "10.1/x",
		"":                                 "",
		"not-a-doi":                        "",
		"10.1234":                          "",
	}
	for in, want := range cases {
		if got := normalizeDOI(in); got != want {
			t.Errorf("normalizeDOI(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMergeWorks(t *testing.T) {
	openalex := []importedWork{
		{DOI: "10.1/a", Title: "Paper A", CitationCount: 9, Journal: "J"},
		{DOI: "10.1/b", Title: "Same title"},
	}
	orcid := []importedWork{
		{DOI: "10.1/a", Title: "PAPER  a", PublicationDate: "2020-01-01"},
		{Title: "paper a"},
		{DOI: "10.1/c", Title: "Same title"},
		{Title: "No DOI"},
		{Title: "no doi"},
		{Title: "   "},
	}
	got := mergeWorks(openalex, orcid)
	if len(got) != 4 {
		t.Fatalf("want 4 works, got %d: %+v", len(got), got)
	}
	if got[0].CitationCount != 9 || got[0].PublicationDate != "2020-01-01" || got[0].Journal != "J" {
		t.Errorf("merge did not combine sources: %+v", got[0])
	}
	if got[1].DOI != "10.1/b" || got[2].DOI != "10.1/c" {
		t.Errorf("distinct DOIs with one title must stay separate: %+v", got)
	}
}

func TestMergeWorksAdoptsDOIFromLaterSource(t *testing.T) {
	got := mergeWorks([]importedWork{{Title: "T"}}, []importedWork{{DOI: "10.1/x", Title: "t"}, {DOI: "10.1/x", Title: "other wording"}})
	if len(got) != 1 || got[0].DOI != "10.1/x" {
		t.Errorf("got %+v", got)
	}
}

func TestOutcomeOf(t *testing.T) {
	down := errors.New("down")
	missing := &apiclient.StatusError{Code: 404}
	cases := []struct {
		name            string
		orcid, openalex error
		failed          int
		code            string
		succeeded       bool
	}{
		{"clean", nil, nil, 0, "", true},
		{"both down", down, down, 0, codeSourcesUnavailable, false},
		{"unknown orcid", missing, down, 0, codeORCIDNotFound, false},
		{"unknown orcid, openalex fine", missing, nil, 0, codeORCIDNotFound, true},
		{"orcid down", down, nil, 0, codeORCIDUnavailable, true},
		{"openalex down", nil, down, 0, codeOpenAlexUnavailable, true},
		{"rows failed", nil, nil, 2, codePartialSave, true},
	}
	for _, tc := range cases {
		got := outcomeOf(tc.orcid, tc.openalex, tc.failed)
		if got.Code != tc.code || got.Succeeded != tc.succeeded {
			t.Errorf("%s: got %+v", tc.name, got)
		}
	}
}

func TestIsSyncing(t *testing.T) {
	now := time.Now()
	ago := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }
	if IsSyncing(nil, nil, now) {
		t.Error("never started")
	}
	if !IsSyncing(ago(time.Minute), nil, now) {
		t.Error("first run in flight")
	}
	if !IsSyncing(ago(time.Minute), ago(time.Hour), now) {
		t.Error("later run in flight")
	}
	if IsSyncing(ago(time.Minute), ago(time.Second), now) {
		t.Error("finished")
	}
	if IsSyncing(ago(SyncTimeout+time.Minute), nil, now) {
		t.Error("crashed run must stop counting")
	}
	if SyncTimeout >= SyncCooldown {
		t.Error("runs could overlap")
	}
}

func TestSmallMappers(t *testing.T) {
	if openAlexShortID("https://openalex.org/A5023888391") != "A5023888391" {
		t.Error("short id")
	}
	if openAlexShortID("https://openalex.org/A1&x=y") != "" {
		t.Error("malformed id accepted")
	}
	if publicationType("CONFERENCE_PAPER") != "conference_paper" || publicationType("whatever") != "journal_article" {
		t.Error("publication type")
	}
	if validDate("2024-13-40") != "" || validDate("2024-02-29") != "2024-02-29" {
		t.Error("date validation")
	}
	if ar, en := splitTitle("بحث"); ar == nil || en != nil {
		t.Error("arabic title")
	}
	if ar, en := splitTitle("Paper"); ar != nil || en == nil {
		t.Error("latin title")
	}
}

func TestORCIDChangeAllowed(t *testing.T) {
	now := time.Now()
	ago := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }
	if !ORCIDChangeAllowed(nil, nil, now) {
		t.Error("never synced")
	}
	if ORCIDChangeAllowed(ago(time.Minute), nil, now) {
		t.Error("in-flight run must block a change")
	}
	if ORCIDChangeAllowed(ago(time.Minute), ago(time.Second), now) {
		t.Error("cooldown must block a change after a finished run")
	}
	if !ORCIDChangeAllowed(ago(SyncCooldown+time.Second), ago(SyncCooldown), now) {
		t.Error("cooldown over")
	}
}

func TestOrchestratorSlotsAreBounded(t *testing.T) {
	o := &Orchestrator{slots: make(chan struct{}, MaxConcurrentSyncs)}
	for range MaxConcurrentSyncs {
		if !o.tryAcquire() {
			t.Fatal("a free slot was refused")
		}
	}
	if o.tryAcquire() {
		t.Fatal("more syncs than slots")
	}
	if err := o.StartDOI("10.1/x"); !errors.Is(err, ErrBusy) {
		t.Fatalf("StartDOI = %v, want ErrBusy", err)
	}
	o.release()
	if !o.tryAcquire() {
		t.Fatal("a released slot must be reusable")
	}
}

func TestOrcidIdentityCode(t *testing.T) {
	researcher := []string{"زينب سعد كريم", "Zainab Saad"}
	if got := orcidIdentityCode(researcher, []string{"Zainab Saad"}); got != "" {
		t.Errorf("match = %q", got)
	}
	if got := orcidIdentityCode([]string{"زينب سعد"}, []string{"Zainab Saad"}); got != "" {
		t.Errorf("across scripts = %q", got)
	}
	if got := orcidIdentityCode(researcher, []string{"Josiah Carberry"}); got != CodeORCIDNameMismatch {
		t.Errorf("mismatch = %q", got)
	}
	if got := orcidIdentityCode(researcher, nil); got != CodeORCIDNameUnverified {
		t.Errorf("hidden name = %q", got)
	}
}

func TestOpenAlexTrustIgnoresAlternatives(t *testing.T) {
	orcid := []string{"Josiah Carberry"}
	if !openAlexTrusted(orcid, &apiclient.OpenAlexPerson{DisplayName: "Josiah S. Carberry"}) {
		t.Error("same person refused")
	}
	if openAlexTrusted(orcid, &apiclient.OpenAlexPerson{DisplayName: "Iain S. Burns"}) {
		t.Error("another person trusted")
	}
}

func TestOutcomeForIdentity(t *testing.T) {
	down := errors.New("down")
	missing := &apiclient.StatusError{Code: 404}
	cases := []struct {
		name      string
		fetched   fetchResult
		code      string
		succeeded bool
		complete  bool
	}{
		{"clean", fetchResult{}, "", true, true},
		{"orcid name refused", fetchResult{identityCode: CodeORCIDNameMismatch}, CodeORCIDNameMismatch, false, false},
		{"orcid name hidden", fetchResult{identityCode: CodeORCIDNameUnverified}, CodeORCIDNameUnverified, false, false},
		{"openalex refused, orcid fine", fetchResult{identityCode: CodeOpenAlexNameMismatch}, CodeOpenAlexNameMismatch, true, true},
		{"openalex refused, orcid works down", fetchResult{identityCode: CodeOpenAlexNameMismatch, orcidErr: down}, CodeOpenAlexNameMismatch, false, false},
		{"cannot read identity", fetchResult{identityUnknown: true, orcidErr: down}, codeORCIDUnavailable, false, false},
		{"unknown orcid", fetchResult{identityUnknown: true, orcidErr: missing}, codeORCIDNotFound, false, false},
		{"truncated", fetchResult{truncated: true}, codeOpenAlexTruncated, true, false},
		{"openalex down", fetchResult{openalexErr: down}, codeOpenAlexUnavailable, true, false},
	}
	for _, tc := range cases {
		got := outcomeFor(tc.fetched, 0)
		if got.Code != tc.code || got.Succeeded != tc.succeeded {
			t.Errorf("%s: got %+v", tc.name, got)
		}
		if tc.fetched.completeAnswer() != tc.complete {
			t.Errorf("%s: completeAnswer = %v", tc.name, !tc.complete)
		}
	}
}

func TestMergeWorksKeepsOpenAlexProvenance(t *testing.T) {
	got := mergeWorks([]importedWork{{DOI: "10.1/a", Title: "A", FromOpenAlex: true}}, []importedWork{{DOI: "10.1/a", Title: "A"}, {DOI: "10.1/b", Title: "B"}})
	if !got[0].FromOpenAlex || got[1].FromOpenAlex {
		t.Errorf("provenance lost: %+v", got)
	}
}
