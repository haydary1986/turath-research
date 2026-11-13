package journalcheck

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

var fixedNow = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

const openAlexFixture = `{"display_name":"Energy and Buildings","issn_l":"0378-7788","issn":["0378-7788","1872-6178"],
"type":"journal","is_in_doaj":false,"host_organization_name":"Elsevier BV","works_count":18169,
"homepage_url":"https://www.journals.elsevier.com/energy-and-buildings","alternate_titles":[],
"summary_stats":{"2yr_mean_citedness":8.17,"h_index":288}}`

const crossrefFixture = `{"status":"ok","message":{"title":"Energy and Buildings","publisher":"Elsevier","ISSN":["0378-7788"],
"issn-type":[{"type":"print","value":"0378-7788"}],"counts":{"total-dois":17321}}}`

const doajFixture = `{"total":1,"results":[{"bibjson":{"title":"PLoS ONE","alternative_title":"PLOS ONE",
"publisher":{"country":"US","name":"Public Library of Science (PLoS)"},"eissn":"1932-6203","pissn":null,
"ref":{"journal":"https://journals.plos.org/plosone/"}}}]}`

func TestParseListNamedRows(t *testing.T) {
	data := "\uFEFFurl,name,abbr\nhttp://www.rapidintellect.com/AEQweb/,Academic Exchange Quarterly,\n" +
		"/International%20Journal,Relative Url Journal,\njavascript:alert(1),Script Journal,\n" +
		"https://sites.google.com/site/x,Shared Host Journal,\n,,\n"
	entries, err := ParseList(ListPredatoryJournal, []byte(data), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].URLHost != "rapidintellect.com" || entries[0].NameKey != "academic exchange quarterly" {
		t.Errorf("first entry: %+v", entries[0])
	}
	if entries[1].URL != "" || entries[1].URLHost != "" {
		t.Errorf("relative URL must be dropped: %+v", entries[1])
	}
	if entries[2].URL != "" {
		t.Errorf("javascript URL must be dropped: %+v", entries[2])
	}
	if entries[3].URLHost != "" || entries[3].URL == "" {
		t.Errorf("shared host keeps the link but not the host: %+v", entries[3])
	}
}

func TestParseListPublishersTrailingColumn(t *testing.T) {
	entries, err := ParseList(ListPredatoryPublisher, []byte("url,name,abbr,\nhttp://www.5thpublisher.com/,The 5th Publisher,,\nshankargargh.org,Shankar & Sons,,\n"), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].NameKey != "5th publisher" {
		t.Errorf("leading the: %q", entries[0].NameKey)
	}
	if entries[1].URLHost != "shankargargh.org" || entries[1].NameKey != "shankar and sons" {
		t.Errorf("scheme-less row: %+v", entries[1])
	}
}

func TestParseListHijackedExpandsAlternateURL(t *testing.T) {
	data := "hijacked,hijackedabbr,hijackedurl,althijackedurl,authentic,authenticabbr,authenticurl\n" +
		"Acta Bioethica,,http://actabioethica.org/,http://acta-two.org/,Acta Bioethica,,http://www.actabioethica.uchile.cl/\n"
	entries, err := ParseList(ListHijacked, []byte(data), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].URLHost != "acta-two.org" || entries[0].AuthenticURL == "" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestParseListRejectsUnusableFiles(t *testing.T) {
	for name, data := range map[string]string{
		"empty":          "",
		"header only":    "url,name,abbr\n",
		"changed header": "link,title\nhttp://a.org,A\n",
		"html error":     "<html>\n<body>rate limited</body>\n",
	} {
		if _, err := ParseList(ListPredatoryJournal, []byte(data), fixedNow); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestHostHandling(t *testing.T) {
	if got := URLHost("HTTPS://WWW.Example.ORG/path"); got != "example.org" {
		t.Errorf("URLHost = %q", got)
	}
	if got := SafeURL("javascript:alert(1)"); got != "" {
		t.Errorf("SafeURL = %q", got)
	}
	if got := HostCandidates("a.b.example.org"); !slices.Equal(got, []string{"a.b.example.org", "b.example.org", "example.org"}) {
		t.Errorf("HostCandidates = %v", got)
	}
	for _, host := range []string{"journals.sfu.ca", "uni.edu", "uoturath.edu.iq", "ox.ac.uk"} {
		if !isSharedHost(host) {
			t.Errorf("%s should be shared", host)
		}
	}
	if isSharedHost("scopemed.org") {
		t.Error("scopemed.org is not shared")
	}
}

func TestParseEvidence(t *testing.T) {
	oa, err := ParseOpenAlexSource([]byte(openAlexFixture))
	if err != nil {
		t.Fatal(err)
	}
	if oa.Publisher != "Elsevier BV" || *oa.HIndex != 288 || *oa.MeanCitedness != 8.17 || len(oa.ISSNs) != 2 {
		t.Errorf("openalex: %+v", oa)
	}
	cr, err := ParseCrossrefJournal([]byte(crossrefFixture))
	if err != nil {
		t.Fatal(err)
	}
	if cr.PrintISSN != "0378-7788" || *cr.TotalDOIs != 17321 {
		t.Errorf("crossref: %+v", cr)
	}
	dj, err := ParseDOAJSearch([]byte(doajFixture))
	if err != nil {
		t.Fatal(err)
	}
	if dj.OnlineISSN != "1932-6203" || dj.PrintISSN != "" || dj.Publisher == "" {
		t.Errorf("doaj: %+v", dj)
	}
	empty, err := ParseDOAJSearch([]byte(`{"total":0,"results":[]}`))
	if err != nil || empty.Status != StatusNotFound {
		t.Errorf("empty doaj: %+v %v", empty, err)
	}
	for name, parse := range map[string]func([]byte) (SourceEvidence, error){
		"openalex": ParseOpenAlexSource, "crossref": ParseCrossrefJournal, "doaj": ParseDOAJSearch,
	} {
		if _, err := parse([]byte(`{"unexpected":true}`)); err == nil {
			t.Errorf("%s: an unexpected body must be an error, not a clean answer", name)
		}
	}
}

func TestResolveByName(t *testing.T) {
	a := SourceEvidence{Status: StatusFound, Title: "Energy & Buildings", ISSNs: []string{"0378-7788"}}
	b := SourceEvidence{Status: StatusFound, Title: "Energy and Buildings", ISSNs: []string{"1111-1111"}}
	c := SourceEvidence{Status: StatusFound, Title: "Energy Reports"}
	if got := ResolveByName([]SourceEvidence{a, c}, "energy and buildings"); got.Title != a.Title {
		t.Errorf("single exact: %+v", got)
	}
	if got := ResolveByName([]SourceEvidence{a, b}, "Energy and Buildings"); got.Status != statusAmbiguous {
		t.Errorf("two exact must stay ambiguous: %+v", got)
	}
	if got := ResolveByName([]SourceEvidence{c}, "Energy"); got.Status != StatusNotFound {
		t.Errorf("partial must not resolve: %+v", got)
	}
}

func found(title string) SourceEvidence { return SourceEvidence{Status: StatusFound, Title: title} }

func evidenceWith(oa SourceEvidence) Evidence {
	return Evidence{
		OpenAlex: oa, Crossref: SourceEvidence{Status: StatusNotFound},
		DOAJ: SourceEvidence{Status: StatusNotFound}, Scopus: SourceEvidence{Status: StatusUnknown},
	}
}

func TestDecide(t *testing.T) {
	yes := true
	inDOAJ := found("J of X")
	inDOAJ.InDOAJ = &yes
	failed := SourceEvidence{Status: StatusFailed}
	allFailed := Evidence{OpenAlex: failed, Crossref: failed, DOAJ: failed}
	predName := ReferenceHit{List: ListPredatoryJournal, MatchedBy: MatchName}
	cases := []struct {
		name string
		in   DecideInput
		want string
		has  string
	}{
		{"manual predatory", DecideInput{Manual: &Manual{Classification: screening.ClassPredatory}, Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictDanger, ReasonManualPredatory},
		{"manual suspended", DecideInput{Manual: &Manual{Classification: screening.ClassSuspended}, Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictCaution, ReasonManualSuspended},
		{"accredited overrides list", DecideInput{Manual: &Manual{Classification: screening.ClassAccredited}, Hits: []ReferenceHit{predName}, Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictFound, ReasonManualOverride},
		{"accredited never excuses a clone site", DecideInput{Manual: &Manual{Classification: screening.ClassAccredited}, Hits: []ReferenceHit{{List: ListHijacked, MatchedBy: MatchURL}}, ListsAvailable: true}, VerdictDanger, ReasonHijackedURL},
		{"predatory name", DecideInput{Hits: []ReferenceHit{predName}, Evidence: evidenceWith(SourceEvidence{Status: StatusNotFound}), ListsAvailable: true}, VerdictDanger, ReasonPredatoryName},
		{"predatory name but in DOAJ", DecideInput{Hits: []ReferenceHit{predName}, Evidence: evidenceWith(inDOAJ), ListsAvailable: true}, VerdictCaution, ReasonListConflictDOAJ},
		{"predatory publisher", DecideInput{Hits: []ReferenceHit{{List: ListPredatoryPublisher, MatchedBy: MatchPublisher}}, ListsAvailable: true}, VerdictDanger, ReasonPredatoryPub},
		{"predatory host beats DOAJ", DecideInput{Hits: []ReferenceHit{{List: ListPredatoryJournal, MatchedBy: MatchURL}}, Evidence: evidenceWith(inDOAJ), ListsAvailable: true}, VerdictDanger, ReasonPredatoryURL},
		{"hijacked name is only a warning", DecideInput{Hits: []ReferenceHit{{List: ListHijacked, MatchedBy: MatchName}}, Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictCaution, ReasonHijackedName},
		{"hijacked name on authentic site", DecideInput{Hits: []ReferenceHit{{List: ListHijacked, MatchedBy: MatchName, AuthenticMatch: true}}, Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictFound, ReasonHijackedName},
		{"found", DecideInput{Evidence: evidenceWith(found("X")), ListsAvailable: true}, VerdictFound, ReasonIndexed},
		{"all sources down is not clean", DecideInput{Evidence: allFailed, ListsAvailable: true}, VerdictUnknown, ReasonSourcesFailed},
		{"found without lists is not clean", DecideInput{Evidence: evidenceWith(found("X")), ListsAvailable: false}, VerdictUnknown, ReasonListsUnavailable},
		{"nowhere is unknown", DecideInput{Evidence: evidenceWith(SourceEvidence{Status: StatusNotFound}), ListsAvailable: true}, VerdictUnknown, ReasonNoEvidence},
		{"bad checksum", DecideInput{ISSNBadCheck: true, ListsAvailable: true}, VerdictCaution, ReasonISSNChecksum},
		{"name mismatch", DecideInput{InputName: "Journal of Totally Other Things", Evidence: evidenceWith(found("Energy and Buildings")), ListsAvailable: true}, VerdictCaution, ReasonNameMismatch},
		{"arabic name never mismatches", DecideInput{InputName: "مجلة الطاقة والمباني", Evidence: evidenceWith(found("Energy and Buildings")), ListsAvailable: true}, VerdictFound, ReasonIndexed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reasons := Decide(tc.in)
			if got != tc.want {
				t.Errorf("verdict = %s, want %s (%v)", got, tc.want, reasons)
			}
			if !slices.Contains(reasons, tc.has) {
				t.Errorf("reasons %v lack %s", reasons, tc.has)
			}
		})
	}
}

func TestMatchHits(t *testing.T) {
	rows := []RefEntry{
		{ListType: ListPredatoryJournal, NameKey: "bad journal", Name: "Bad Journal", URLHost: "bad.org"},
		{ListType: ListPredatoryPublisher, NameKey: "bad press", Name: "Bad Press", URLHost: "press.org"},
		{ListType: ListHijacked, NameKey: "acta", Name: "Acta", URLHost: "clone.org", AuthenticURL: "https://acta.edu/"},
		{ListType: ListPredatoryPublisher, NameKey: "acta", Name: "Unrelated publisher"},
	}
	byName := MatchHits(rows, MatchInput{NameKeys: []string{"acta"}, InputHost: "acta.edu"})
	if len(byName) != 1 || byName[0].List != ListHijacked || !byName[0].AuthenticMatch {
		t.Errorf("journal names must not match publisher rows: %+v", byName)
	}
	byHost := MatchHits(rows, MatchInput{InputHost: "ojs.bad.org"})
	if len(byHost) != 1 || byHost[0].MatchedBy != MatchURL {
		t.Errorf("subdomain: %+v", byHost)
	}
	byPublisher := MatchHits(rows, MatchInput{PublisherKeys: []string{"bad press"}})
	if len(byPublisher) != 1 || byPublisher[0].MatchedBy != MatchPublisher {
		t.Errorf("publisher: %+v", byPublisher)
	}
	if got := MatchHits(rows, MatchInput{}); len(got) != 0 {
		t.Errorf("empty input must not match rows without a host: %+v", got)
	}
}

func TestCanonicalOnlineISSN(t *testing.T) {
	oa := SourceEvidence{Status: StatusFound, Title: "E", ISSNs: []string{"0378-7788", "1872-6178"}}
	cr := SourceEvidence{Status: StatusFound, PrintISSN: "0378-7788", ISSNs: []string{"0378-7788"}}
	if got := CanonicalFrom(Evidence{OpenAlex: oa, Crossref: cr}, "").OnlineISSN; got != "1872-6178" {
		t.Errorf("inferred from typed print: %q", got)
	}
	if got := CanonicalFrom(Evidence{OpenAlex: oa}, "").OnlineISSN; got != "" {
		t.Errorf("untyped ISSNs must not guess: %q", got)
	}
	if got := CanonicalFrom(Evidence{OpenAlex: oa}, "1872-6178").OnlineISSN; got != "0378-7788" {
		t.Errorf("entered print ISSN: %q", got)
	}
}

// ── Service with fakes ──────────────────────────

type fakeLookups struct {
	oaErr, crErr, djErr, listErr error
	oa                           SourceEvidence
	calls                        int
	listBody                     string
}

func (f *fakeLookups) OpenAlexByISSN(context.Context, string) (SourceEvidence, error) {
	f.calls++
	return f.oa, f.oaErr
}
func (f *fakeLookups) OpenAlexByName(context.Context, string) ([]SourceEvidence, error) {
	f.calls++
	return []SourceEvidence{f.oa}, f.oaErr
}
func (f *fakeLookups) CrossrefByISSN(context.Context, string) (SourceEvidence, error) {
	return SourceEvidence{}, f.crErr
}
func (f *fakeLookups) DOAJByISSN(context.Context, string) (SourceEvidence, error) {
	return SourceEvidence{}, f.djErr
}
func (f *fakeLookups) FetchList(context.Context, string) ([]byte, error) {
	return []byte(f.listBody), f.listErr
}

type fakeStore struct {
	cache     map[string][]byte
	replaced  map[string]int
	failures  map[string]string
	refs      []RefEntry
	manual    []ManualRow
	statuses  []ListStatus
	seenNames []string
}

func newFakeStore() *fakeStore {
	success := fixedNow.Add(-time.Hour)
	statuses := []ListStatus{}
	for _, listType := range ListTypes {
		statuses = append(statuses, ListStatus{ListType: listType, Count: 5, LastSuccessAt: &success})
	}
	return &fakeStore{cache: map[string][]byte{}, replaced: map[string]int{}, failures: map[string]string{}, statuses: statuses}
}

func (s *fakeStore) ReplaceList(_ context.Context, listType string, entries []RefEntry, _ time.Time) error {
	s.replaced[listType] = len(entries)
	return nil
}
func (s *fakeStore) RecordRefreshFailure(_ context.Context, listType, message string, _ time.Time) error {
	s.failures[listType] = message
	return nil
}
func (s *fakeStore) ListStatuses(context.Context) ([]ListStatus, error) { return s.statuses, nil }
func (s *fakeStore) FindReferences(_ context.Context, nameKeys, _, _ []string) ([]RefEntry, error) {
	s.seenNames = nameKeys
	return s.refs, nil
}
func (s *fakeStore) FindManual(context.Context, []string, []string) ([]ManualRow, error) {
	return s.manual, nil
}
func (s *fakeStore) CacheGet(_ context.Context, source, key string, _ time.Time) ([]byte, bool, error) {
	payload, ok := s.cache[source+"|"+key]
	return payload, ok, nil
}
func (s *fakeStore) CachePut(_ context.Context, source, key string, payload []byte, _ time.Time, _ time.Duration) error {
	s.cache[source+"|"+key] = payload
	return nil
}
func (s *fakeStore) PurgeCache(context.Context, time.Time) error { return nil }

func newService(store Store, lookups Lookups) *Service {
	svc := New(store, lookups)
	svc.now = func() time.Time { return fixedNow }
	return svc
}

func TestCheckAllSourcesDownIsUnavailableNotClean(t *testing.T) {
	boom := errors.New("timeout")
	store := newFakeStore()
	lookups := &fakeLookups{oaErr: boom, crErr: boom, djErr: boom}
	res, err := newService(store, lookups).Check(context.Background(), Query{ISSN: "0378-7788"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictUnknown || !res.Advisory {
		t.Errorf("verdict %s advisory %v", res.Verdict, res.Advisory)
	}
	if !slices.Equal(res.SourcesFailed, []string{SourceOpenAlex, SourceCrossref, SourceDOAJ}) {
		t.Errorf("sources failed: %v", res.SourcesFailed)
	}
	if res.Evidence.Scopus.Status != StatusUnknown {
		t.Errorf("scopus must stay unknown: %s", res.Evidence.Scopus.Status)
	}
	if len(store.cache) != 0 {
		t.Errorf("failures must not be cached: %v", store.cache)
	}
}

func TestCheckScreensCanonicalNameFromISSN(t *testing.T) {
	store := newFakeStore()
	store.refs = []RefEntry{{ListType: ListPredatoryJournal, NameKey: "energy and buildings", Name: "Energy and Buildings"}}
	oa, _ := ParseOpenAlexSource([]byte(openAlexFixture))
	lookups := &fakeLookups{oa: oa, crErr: ErrNotFound, djErr: ErrNotFound}
	res, err := newService(store, lookups).Check(context.Background(), Query{ISSN: "03787788"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictDanger || len(res.ReferenceHits) != 1 {
		t.Errorf("an ISSN alone must still screen the name it resolves to: %+v", res)
	}
	if res.CanonicalName != "Energy and Buildings" || res.Publisher != "Elsevier BV" || res.OnlineISSN != "1872-6178" {
		t.Errorf("canonical: %+v", res)
	}
}

func TestCheckUsesCacheAndCachedOnlyStaysOffline(t *testing.T) {
	store := newFakeStore()
	oa, _ := ParseOpenAlexSource([]byte(openAlexFixture))
	lookups := &fakeLookups{oa: oa, crErr: ErrNotFound, djErr: ErrNotFound}
	svc := newService(store, lookups)

	offline, err := svc.Check(context.Background(), Query{ISSN: "0378-7788", CachedOnly: true})
	if err != nil || lookups.calls != 0 {
		t.Fatalf("offline check made %d calls, err %v", lookups.calls, err)
	}
	if offline.Evidence.OpenAlex.Status != StatusNotChecked || offline.Verdict != VerdictUnknown || len(offline.SourcesFailed) != 0 {
		t.Errorf("offline: %+v", offline)
	}
	for range 2 {
		if _, err := svc.Check(context.Background(), Query{ISSN: "0378-7788"}); err != nil {
			t.Fatal(err)
		}
	}
	if lookups.calls != 1 {
		t.Errorf("second check must come from cache, calls = %d", lookups.calls)
	}
	cached, err := svc.Check(context.Background(), Query{ISSN: "0378-7788", CachedOnly: true})
	if err != nil || cached.Evidence.OpenAlex.Status != StatusFound {
		t.Errorf("cached offline: %+v %v", cached.Evidence.OpenAlex, err)
	}
}

func TestCheckBadChecksumSkipsNetwork(t *testing.T) {
	lookups := &fakeLookups{}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{ISSN: "0378-7789"})
	if err != nil || lookups.calls != 0 {
		t.Fatalf("calls %d err %v", lookups.calls, err)
	}
	if res.Verdict != VerdictCaution || !slices.Contains(res.Reasons, ReasonISSNChecksum) {
		t.Errorf("result: %+v", res)
	}
}

func TestCheckManualMatchesOtherISSNOfTheJournal(t *testing.T) {
	store := newFakeStore()
	store.manual = []ManualRow{{ISSN: "1872-6178", Name: "E&B", Classification: screening.ClassPredatory}}
	oa, _ := ParseOpenAlexSource([]byte(openAlexFixture))
	res, err := newService(store, &fakeLookups{oa: oa, crErr: ErrNotFound, djErr: ErrNotFound}).Check(context.Background(), Query{ISSN: "0378-7788"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.Classification != screening.ClassPredatory || res.MatchedBy != MatchISSN || res.Verdict != VerdictDanger {
		t.Errorf("result: %+v", res)
	}
}

func TestCheckEmptyQueryTouchesNothing(t *testing.T) {
	lookups := &fakeLookups{}
	res, err := newService(newFakeStore(), lookups).Check(context.Background(), Query{})
	if err != nil || lookups.calls != 0 || res.Matched || res.Verdict != VerdictUnknown {
		t.Errorf("res %+v err %v calls %d", res, err, lookups.calls)
	}
}

func TestCheckEmptyListsAreReportedAsFailed(t *testing.T) {
	store := newFakeStore()
	store.statuses = nil
	oa, _ := ParseOpenAlexSource([]byte(openAlexFixture))
	res, err := newService(store, &fakeLookups{oa: oa, crErr: ErrNotFound, djErr: ErrNotFound}).Check(context.Background(), Query{ISSN: "0378-7788"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictUnknown || !slices.Contains(res.SourcesFailed, SourceReferenceLists) {
		t.Errorf("result: %s %v", res.Verdict, res.SourcesFailed)
	}
}

func TestRefreshKeepsOldDataOnFailure(t *testing.T) {
	store := newFakeStore()
	newService(store, &fakeLookups{listErr: errors.New("offline")}).Refresh(context.Background())
	if len(store.replaced) != 0 || len(store.failures) != len(ListTypes) {
		t.Errorf("fetch failure: replaced %v failures %v", store.replaced, store.failures)
	}
	store = newFakeStore()
	newService(store, &fakeLookups{listBody: "<html>oops</html>"}).Refresh(context.Background())
	if len(store.replaced) != 0 {
		t.Errorf("bad body must not replace data: %v", store.replaced)
	}
}

func TestRefreshReplacesValidList(t *testing.T) {
	store := newFakeStore()
	newService(store, &fakeLookups{listBody: "url,name,abbr\nhttp://a.org,A Journal,\n"}).Refresh(context.Background())
	if store.replaced[ListPredatoryJournal] != 1 || store.replaced[ListPredatoryPublisher] != 1 {
		t.Errorf("replaced %v", store.replaced)
	}
	if _, ok := store.failures[ListHijacked]; !ok {
		t.Error("the hijacked list needs its own columns, so this body must fail for it alone")
	}
}

func TestUserThrottleWindows(t *testing.T) {
	now := time.Now()
	throttle := NewUserThrottle(2, time.Minute)
	throttle.now = func() time.Time { return now }
	if !throttle.Allow("a") || !throttle.Allow("a") {
		t.Fatal("within the allowance")
	}
	if throttle.Allow("a") {
		t.Fatal("over the allowance")
	}
	if !throttle.Allow("b") {
		t.Fatal("users must not share an allowance")
	}
	now = now.Add(time.Minute)
	if !throttle.Allow("a") {
		t.Fatal("a new window must allow again")
	}
	if len(throttle.users) != 1 {
		t.Errorf("expired users must be forgotten, have %d", len(throttle.users))
	}
}

func TestCheckThrottlesOnlyUncachedLookups(t *testing.T) {
	lookups := &fakeLookups{oa: SourceEvidence{Status: StatusFound, Title: "Journal"}, crErr: ErrNotFound, djErr: ErrNotFound}
	store := newFakeStore()
	service := New(store, lookups)
	asks := 0
	allow := func() bool { asks++; return asks == 1 }

	first, err := service.Check(context.Background(), Query{ISSN: "2049-3630", AllowExternal: allow})
	if err != nil || first.Throttled {
		t.Fatalf("first check: throttled=%v err=%v", first.Throttled, err)
	}
	if asks != 1 {
		t.Errorf("one check must spend one unit across all sources, spent %d", asks)
	}

	calls := lookups.calls
	repeat, err := service.Check(context.Background(), Query{ISSN: "2049-3630", AllowExternal: allow})
	if err != nil || repeat.Throttled || repeat.Evidence.OpenAlex.Status != StatusFound || asks != 1 {
		t.Fatalf("a cached repeat must cost nothing: %+v asks=%d err=%v", repeat, asks, err)
	}

	denied, err := service.Check(context.Background(), Query{ISSN: "0378-7788", AllowExternal: allow})
	if err != nil {
		t.Fatal(err)
	}
	if !denied.Throttled || denied.Evidence.OpenAlex.Status != StatusNotChecked {
		t.Fatalf("over the allowance: %+v", denied)
	}
	if lookups.calls != calls {
		t.Errorf("a throttled check reached a source")
	}
}

func TestRefreshKeepsLargeListAgainstTinyReplacement(t *testing.T) {
	cases := []struct {
		current, incoming int
		suspicious        bool
	}{
		{0, 1, false}, {99, 1, false}, {100, 49, true}, {100, 50, false}, {5000, 2400, true}, {5000, 5001, false},
	}
	for _, tc := range cases {
		if got := suspiciouslySmall(tc.current, tc.incoming); got != tc.suspicious {
			t.Errorf("suspiciouslySmall(%d, %d) = %v", tc.current, tc.incoming, got)
		}
	}

	store := newFakeStore()
	store.statuses = []ListStatus{{ListType: ListPredatoryJournal, Count: 5000}}
	lookups := &fakeLookups{listBody: "url,name\nhttp://a.org,Only Journal\n"}
	New(store, lookups).Refresh(context.Background())
	if _, replaced := store.replaced[ListPredatoryJournal]; replaced {
		t.Error("a tiny file replaced a large list")
	}
	if store.failures[ListPredatoryJournal] == "" {
		t.Error("the refusal must be recorded as a failed refresh")
	}
}

func TestParseListRejectsOversizedFiles(t *testing.T) {
	var body strings.Builder
	body.WriteString("url,name\n")
	for i := 0; i <= maxListRows; i++ {
		body.WriteString("http://a.org,J\n")
	}
	if _, err := ParseList(ListPredatoryJournal, []byte(body.String()), time.Now()); err == nil {
		t.Error("a file over the row cap must be refused before it can replace a list")
	}
}
