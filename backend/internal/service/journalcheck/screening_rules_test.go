package journalcheck

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

func TestNameKeysSplitsCompositeNames(t *testing.T) {
	data := "url,name,abbr\nhttp://a.org,Bio Accent/Bioaccent,\nhttp://b.org,Plants Journal / Journal of Plant Studies,\n" +
		"http://c.org,Amoghsiddhi Society/AES/AES Journals,\nhttp://d.org,Science and/or Art,\n"
	entries, err := ParseList(ListPredatoryPublisher, []byte(data), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	keysFor := func(name string) []string {
		keys := []string{}
		for _, entry := range entries {
			if entry.Name == name {
				keys = append(keys, entry.NameKey)
			}
		}
		return keys
	}
	if got := keysFor("Bio Accent/Bioaccent"); !slices.Equal(got, []string{"bio accent bioaccent", "bio accent", "bioaccent"}) {
		t.Errorf("composite: %v", got)
	}
	if got := keysFor("Plants Journal / Journal of Plant Studies"); !slices.Contains(got, "journal of plant studies") || !slices.Contains(got, "plants journal") {
		t.Errorf("spaced composite: %v", got)
	}
	if got := keysFor("Amoghsiddhi Society/AES/AES Journals"); slices.Contains(got, "aes") || !slices.Contains(got, "aes journals") {
		t.Errorf("short part must not become an alias: %v", got)
	}
	if got := keysFor("Science and/or Art"); len(got) != 1 {
		t.Errorf("and/or is prose, not two names: %v", got)
	}
}

func TestNameKeysUsesOnlyDistinctiveAbbreviations(t *testing.T) {
	data := "url,name,abbr\nhttp://a.org,American Academic Society,AAAS\nhttp://b.org,International Journal of Things,IJTHINGS\n"
	entries, err := ParseList(ListPredatoryJournal, []byte(data), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, entry := range entries {
		keys = append(keys, entry.NameKey)
	}
	if slices.Contains(keys, "aaas") {
		t.Errorf("a short acronym names many bodies: %v", keys)
	}
	if !slices.Contains(keys, "ijthings") {
		t.Errorf("a distinctive abbreviation should match: %v", keys)
	}
}

func TestParseListHijackedKeepsAliases(t *testing.T) {
	data := "hijacked,hijackedabbr,hijackedurl,althijackedurl,authentic,authenticabbr,authenticurl\n" +
		"Acta Bioethica/Acta Medica,,http://clone.org/,,Acta Bioethica,,http://real.org/\n"
	entries, err := ParseList(ListHijacked, []byte(data), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[2].NameKey != "acta medica" || entries[2].URLHost != "clone.org" {
		t.Errorf("entries: %+v", entries)
	}
}

func TestMatchHitsReportsACompositeRowOnce(t *testing.T) {
	rows := []RefEntry{
		{ListType: ListPredatoryJournal, NameKey: "bio accent", Name: "Bio Accent/Bioaccent", URLHost: "bad.org"},
		{ListType: ListPredatoryJournal, NameKey: "bioaccent", Name: "Bio Accent/Bioaccent", URLHost: "bad.org"},
	}
	if got := MatchHits(rows, MatchInput{InputHost: "bad.org"}); len(got) != 1 {
		t.Errorf("hits: %+v", got)
	}
}

func decideReasons(t *testing.T, in DecideInput, want string, has ...string) []string {
	t.Helper()
	in.ListsAvailable = true
	verdict, reasons := Decide(in)
	if verdict != want {
		t.Errorf("verdict %s, want %s (%v)", verdict, want, reasons)
	}
	for _, reason := range has {
		if !slices.Contains(reasons, reason) {
			t.Errorf("reasons %v lack %s", reasons, reason)
		}
	}
	return reasons
}

func TestNameMatchedAccreditationDoesNotCoverAnotherJournal(t *testing.T) {
	manual := &Manual{Classification: screening.ClassAccredited, Name: "Trusted Local Journal", MatchedBy: MatchName}
	other := evidenceWith(found("Different Predatory Review"))
	hit := ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchName, matchedKey: "different predatory review"}

	t.Run("predatory hit survives", func(t *testing.T) {
		reasons := decideReasons(t, DecideInput{Manual: manual, Hits: []ReferenceHit{hit}, Evidence: other, InputName: manual.Name},
			VerdictDanger, ReasonPredatoryName, ReasonManualUnconfirmed)
		if slices.Contains(reasons, ReasonManualOverride) || slices.Contains(reasons, ReasonManualAccredited) {
			t.Errorf("reasons: %v", reasons)
		}
	})
	t.Run("name mismatch survives", func(t *testing.T) {
		decideReasons(t, DecideInput{Manual: manual, Evidence: other, InputName: manual.Name}, VerdictCaution, ReasonNameMismatch)
	})
	t.Run("arabic entry cannot vouch for an english title", func(t *testing.T) {
		arabic := &Manual{Classification: screening.ClassAccredited, Name: "المجلة المعتمدة", MatchedBy: MatchName}
		decideReasons(t, DecideInput{Manual: arabic, Hits: []ReferenceHit{hit}, Evidence: other, InputName: arabic.Name},
			VerdictDanger, ReasonManualUnconfirmed)
	})
	t.Run("bad checksum survives a name match", func(t *testing.T) {
		decideReasons(t, DecideInput{Manual: manual, ISSNBadCheck: true, InputName: manual.Name}, VerdictCaution, ReasonISSNChecksum)
	})
	t.Run("same journal keeps the override", func(t *testing.T) {
		decideReasons(t, DecideInput{Manual: manual, Hits: []ReferenceHit{hit}, Evidence: evidenceWith(found("Trusted Local Journal")), InputName: manual.Name},
			VerdictFound, ReasonManualAccredited, ReasonManualOverride)
	})
	t.Run("no resolved journal keeps the override", func(t *testing.T) {
		decideReasons(t, DecideInput{Manual: manual, Hits: []ReferenceHit{hit}, Evidence: evidenceWith(SourceEvidence{Status: StatusNotFound}), InputName: manual.Name},
			VerdictFound, ReasonManualOverride)
	})
	t.Run("ISSN match keeps the override", func(t *testing.T) {
		byISSN := &Manual{Classification: screening.ClassAccredited, Name: "Whatever", MatchedBy: MatchISSN}
		decideReasons(t, DecideInput{Manual: byISSN, Hits: []ReferenceHit{hit}, Evidence: other}, VerdictFound, ReasonManualOverride)
	})
}

func TestNameConflictsGuardsEmptyAndShortTitles(t *testing.T) {
	long := "Journal of Totally Other Things"
	cases := []struct {
		name  string
		input string
		e     Evidence
		want  bool
	}{
		{"empty title gives nothing to compare", long, evidenceWith(found("")), false},
		{"punctuation-only title gives nothing to compare", long, evidenceWith(found("---")), false},
		{"empty title beside a real one still compares", long, Evidence{OpenAlex: found(""), Crossref: found("Energy Reports")}, true},
		{"short title inside the name is no agreement", long, evidenceWith(found("Of")), true},
		{"short input inside a title is no agreement", "Things", evidenceWith(found("Thing")), true},
		{"long containment agrees", "Energy and Buildings", evidenceWith(found("Energy and Buildings Journal")), false},
		{"exact short title agrees", "Cell", evidenceWith(found("Cell")), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nameConflicts(tc.input, tc.e); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRankHitDOAJDowngradeNeedsSameJournal(t *testing.T) {
	yes := true
	doaj := Evidence{DOAJ: SourceEvidence{Status: StatusFound, Title: "Honest Review", Publisher: "Honest Press", URL: "https://honest.org/", InDOAJ: &yes}}
	cases := []struct {
		name string
		hit  ReferenceHit
		rank int
	}{
		{"same title", ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchName, matchedKey: "honest review"}, rankCaution},
		{"typed name of another journal", ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchName, matchedKey: "typed predatory name"}, rankDanger},
		{"same publisher", ReferenceHit{List: ListPredatoryPublisher, MatchedBy: MatchPublisher, matchedKey: "honest press"}, rankCaution},
		{"another publisher", ReferenceHit{List: ListPredatoryPublisher, MatchedBy: MatchPublisher, matchedKey: "typed press"}, rankDanger},
		{"same homepage", ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchHomepage, URL: "http://honest.org/x"}, rankCaution},
		{"listed URL never downgrades", ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchURL, matchedKey: "honest review"}, rankDanger},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := rankHit(tc.hit, doaj); got != tc.rank {
				t.Errorf("rank %d, want %d", got, tc.rank)
			}
		})
	}
}

// issnLookups answers per ISSN, unlike fakeLookups, so two ISSNs can differ.
type issnLookups struct {
	byISSN map[string]SourceEvidence
	asked  []string
}

func (l *issnLookups) OpenAlexByISSN(_ context.Context, issn string) (SourceEvidence, error) {
	l.asked = append(l.asked, issn)
	if evidence, ok := l.byISSN[issn]; ok {
		return evidence, nil
	}
	return SourceEvidence{}, ErrNotFound
}
func (l *issnLookups) OpenAlexByName(context.Context, string) ([]SourceEvidence, error) {
	return nil, nil
}
func (l *issnLookups) CrossrefByISSN(context.Context, string) (SourceEvidence, error) {
	return SourceEvidence{}, ErrNotFound
}
func (l *issnLookups) DOAJByISSN(context.Context, string) (SourceEvidence, error) {
	return SourceEvidence{}, ErrNotFound
}
func (l *issnLookups) FetchList(context.Context, string) ([]byte, error) { return nil, nil }

const (
	issnA = "0378-7788"
	issnB = "1872-6178"
)

func TestGatherResolvesEveryEnteredISSN(t *testing.T) {
	store := newFakeStore()
	store.refs = []RefEntry{{ListType: ListPredatoryJournal, NameKey: "clone review", Name: "Clone Review"}}
	lookups := &issnLookups{byISSN: map[string]SourceEvidence{
		issnA: {Status: StatusFound, Title: "Honest Review", Publisher: "Honest Press", ISSNs: []string{issnA}},
		issnB: {Status: StatusFound, Title: "Clone Review", Publisher: "Clone Press", ISSNs: []string{issnB}},
	}}
	res, err := newService(store, lookups).Check(context.Background(), Query{ISSN: issnA, ISSNOnline: issnB})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(lookups.asked, issnA) || !slices.Contains(lookups.asked, issnB) {
		t.Errorf("asked %v", lookups.asked)
	}
	if !slices.Contains(store.seenNames, "clone review") {
		t.Errorf("second journal was not screened: %v", store.seenNames)
	}
	if res.Verdict != VerdictDanger || !slices.Contains(res.Reasons, ReasonISSNConflict) {
		t.Errorf("verdict %s reasons %v", res.Verdict, res.Reasons)
	}
	if res.CanonicalName != "" || res.Publisher != "" {
		t.Errorf("two journals must not offer one identity: %+v", res)
	}
}

func TestGatherSameJournalUnderBothISSNsIsNoConflict(t *testing.T) {
	both := SourceEvidence{Status: StatusFound, Title: "Honest Review", ISSNs: []string{issnA, issnB}}
	lookups := &issnLookups{byISSN: map[string]SourceEvidence{issnA: both, issnB: both}}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{ISSN: issnA, ISSNOnline: issnB})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictFound || slices.Contains(res.Reasons, ReasonISSNConflict) || res.CanonicalName != "Honest Review" {
		t.Errorf("result: %+v", res)
	}
}

func TestGatherSecondISSNFillsAnUnansweredFirst(t *testing.T) {
	lookups := &issnLookups{byISSN: map[string]SourceEvidence{issnB: {Status: StatusFound, Title: "Honest Review", ISSNs: []string{issnB}}}}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{ISSN: issnA, ISSNOnline: issnB})
	if err != nil {
		t.Fatal(err)
	}
	if res.Evidence.OpenAlex.Status != StatusFound || res.Verdict != VerdictFound {
		t.Errorf("result: %+v", res)
	}
}

func TestConflictingISSNsPickTheWorstManualEntry(t *testing.T) {
	store := newFakeStore()
	store.manual = []ManualRow{
		{ISSN: issnA, Name: "Honest Review", Classification: screening.ClassAccredited},
		{ISSN: issnB, Name: "Clone Review", Classification: screening.ClassPredatory},
	}
	lookups := &issnLookups{byISSN: map[string]SourceEvidence{
		issnA: {Status: StatusFound, Title: "Honest Review", ISSNs: []string{issnA}},
		issnB: {Status: StatusFound, Title: "Clone Review", ISSNs: []string{issnB}},
	}}
	res, err := newService(store, lookups).Check(context.Background(), Query{ISSN: issnA, ISSNOnline: issnB})
	if err != nil {
		t.Fatal(err)
	}
	if res.Classification != screening.ClassPredatory || res.Verdict != VerdictDanger {
		t.Errorf("result: %+v", res)
	}
}

func TestFullNameNotFoundReportsNoEvidence(t *testing.T) {
	lookups := &fakeLookups{oa: found("Something Else Entirely")}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{Name: "Journal of Unlisted Studies"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(res.Reasons, ReasonNoEvidence) || slices.Contains(res.Reasons, ReasonNotChecked) || res.Verdict != VerdictUnknown {
		t.Errorf("result: %+v", res)
	}
}

func TestNameSearchMinimum(t *testing.T) {
	atMinimum := strings.Repeat("c", MinNameSearchRunes)
	lookups := &fakeLookups{oa: found(atMinimum)}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{Name: atMinimum})
	if err != nil || res.Evidence.OpenAlex.Status != StatusFound {
		t.Errorf("minimum-length name must be searched: %+v %v", res.Evidence.OpenAlex, err)
	}

	short := &fakeLookups{}
	res, err = newService(newFakeStore(), short).Check(context.Background(), Query{Name: atMinimum[1:]})
	if err != nil || short.calls != 0 {
		t.Fatalf("calls %d err %v", short.calls, err)
	}
	if !slices.Contains(res.Reasons, ReasonNotChecked) {
		t.Errorf("too-short name is incomplete input: %v", res.Reasons)
	}
}

func TestRejectedURLAddsReason(t *testing.T) {
	for raw, rejected := range map[string]bool{"javascript:alert(1)": true, "not a url": true, "https://ok.org/": false} {
		res, err := newService(newFakeStore(), &fakeLookups{}).Check(context.Background(), Query{URL: raw})
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Contains(res.Reasons, ReasonURLRejected); got != rejected {
			t.Errorf("%q: reason present = %v, reasons %v", raw, got, res.Reasons)
		}
	}
}
