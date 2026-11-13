package journalcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

// MinNameSearchRunes is the shortest folded name worth searching for: short enough
// for real titles like "Cell" and "JAMA". The frontend keeps the same minimum.
const MinNameSearchRunes = 4

const (
	foundTTL          = 30 * 24 * time.Hour
	notFoundTTL       = 7 * 24 * time.Hour
	refreshInterval   = 24 * time.Hour
	refreshTimeout    = 90 * time.Second
	sourceTimeout     = 6 * time.Second
	listsStaleAfter   = 72 * time.Hour
	MatchISSN         = "issn"
	maxRefreshErrSize = 300
	// A list this large is worth protecting from a broken upstream file.
	shrinkGuardMinRows = 100
)

// ErrNotFound is what a Lookups method returns when a source has no such journal.
var ErrNotFound = errors.New("journal not found at source")

// Lookups reaches the external sources. Any error other than ErrNotFound means
// the source could not answer.
type Lookups interface {
	OpenAlexByISSN(ctx context.Context, issn string) (SourceEvidence, error)
	OpenAlexByName(ctx context.Context, name string) ([]SourceEvidence, error)
	CrossrefByISSN(ctx context.Context, issn string) (SourceEvidence, error)
	DOAJByISSN(ctx context.Context, issn string) (SourceEvidence, error)
	FetchList(ctx context.Context, listType string) ([]byte, error)
}

// ManualRow is a candidate from the staff watchlist.
type ManualRow struct {
	ISSN           string
	NameKey        string
	Name           string
	Classification string
	Note           *string
}

// Store persists lists, refresh state and cached lookups.
type Store interface {
	ReplaceList(ctx context.Context, listType string, entries []RefEntry, at time.Time) error
	RecordRefreshFailure(ctx context.Context, listType, message string, at time.Time) error
	ListStatuses(ctx context.Context) ([]ListStatus, error)
	FindReferences(ctx context.Context, nameKeys, publisherKeys, hosts []string) ([]RefEntry, error)
	FindManual(ctx context.Context, issns, nameKeys []string) ([]ManualRow, error)
	CacheGet(ctx context.Context, source, key string, now time.Time) ([]byte, bool, error)
	CachePut(ctx context.Context, source, key string, payload []byte, now time.Time, ttl time.Duration) error
	PurgeCache(ctx context.Context, now time.Time) error
}

// Query is what a caller knows about a journal.
type Query struct {
	ISSN       string
	ISSNOnline string
	Name       string
	Publisher  string
	URL        string
	// CachedOnly answers without network calls, for page loads that must not wait.
	CachedOnly bool
	// AllowExternal, when set, is asked before the first uncached lookup of a check;
	// false answers from local data only. It costs one unit per check, not per source.
	AllowExternal func() bool
}

func (q Query) empty() bool {
	return strings.TrimSpace(q.ISSN+q.ISSNOnline+q.Name+q.Publisher+q.URL) == ""
}

// Service runs checks and keeps the reference lists fresh.
type Service struct {
	store   Store
	lookups Lookups
	now     func() time.Time
}

// New builds a Service.
func New(store Store, lookups Lookups) *Service {
	return &Service{store: store, lookups: lookups, now: time.Now}
}

// Run refreshes the lists at startup and every 24h until ctx ends.
func (s *Service) Run(ctx context.Context) {
	s.Refresh(ctx)
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Refresh(ctx)
		}
	}
}

// Refresh updates each list independently; a list that fails keeps its old rows.
func (s *Service) Refresh(ctx context.Context) {
	for _, listType := range ListTypes {
		if err := s.refreshList(ctx, listType); err != nil {
			slog.Warn("journal reference list refresh failed, keeping old data", "list", listType, "error", err)
			message := err.Error()
			if len(message) > maxRefreshErrSize {
				message = strings.ToValidUTF8(message[:maxRefreshErrSize], "")
			}
			recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			if recErr := s.store.RecordRefreshFailure(recordCtx, listType, message, s.now()); recErr != nil {
				slog.Error("failed to record list refresh failure", "list", listType, "error", recErr)
			}
			cancel()
		}
	}
	if err := s.store.PurgeCache(ctx, s.now()); err != nil {
		slog.Warn("journal lookup cache purge failed", "error", err)
	}
}

func (s *Service) refreshList(ctx context.Context, listType string) error {
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	data, err := s.lookups.FetchList(ctx, listType)
	if err != nil {
		return err
	}
	at := s.now()
	entries, err := ParseList(listType, data, at)
	if err != nil {
		return err
	}
	if err := s.guardShrink(ctx, listType, len(entries)); err != nil {
		return err
	}
	if err := s.store.ReplaceList(ctx, listType, entries, at); err != nil {
		return err
	}
	slog.Info("journal reference list refreshed", "list", listType, "rows", len(entries))
	return nil
}

// guardShrink refuses a refresh that would more than halve a large list. Real lists
// grow slowly, so that shape means a truncated or replaced upstream file.
func (s *Service) guardShrink(ctx context.Context, listType string, incoming int) error {
	statuses, err := s.store.ListStatuses(ctx)
	if err != nil {
		return fmt.Errorf("read current list size: %w", err)
	}
	for _, status := range statuses {
		if status.ListType == listType && suspiciouslySmall(status.Count, incoming) {
			return fmt.Errorf("list shrank from %d to %d rows", status.Count, incoming)
		}
	}
	return nil
}

func suspiciouslySmall(current, incoming int) bool {
	return current >= shrinkGuardMinRows && incoming*2 < current
}

// Statuses reports list counts and refresh times.
func (s *Service) Statuses(ctx context.Context) ([]ListStatus, error) {
	return s.store.ListStatuses(ctx)
}

// Check gathers the evidence and combines it. It returns an error only when the
// local database fails; an unreachable external source is reported in the result.
func (s *Service) Check(ctx context.Context, q Query) (Result, error) {
	result := Result{
		Advisory: true, Verdict: VerdictUnknown, Reasons: []string{},
		ReferenceHits: []ReferenceHit{}, ISSNs: []string{}, SourcesFailed: []string{},
		Evidence: notCheckedEvidence(),
	}
	if q.empty() {
		return result, nil
	}

	printISSN := screening.NormalizeISSN(q.ISSN)
	onlineISSN := screening.NormalizeISSN(q.ISSNOnline)
	malformed := (strings.TrimSpace(q.ISSN) != "" && printISSN == "") ||
		(strings.TrimSpace(q.ISSNOnline) != "" && onlineISSN == "")
	inputISSNs := normalizeISSNs([]string{q.ISSN, q.ISSNOnline})
	valid, badCheck := splitByChecksum(inputISSNs)

	access := newExternalAccess(q)
	gathered := s.gather(ctx, q, valid, access)
	evidence := gathered.evidence
	canonical := CanonicalFrom(evidence, printISSN)

	titles := titlesOf(gathered.found)
	nameKeys := refKeys(append(append([]string{q.Name}, titles...), canonical.Name))
	publisherKeys := []string{q.Publisher}
	homepages := []string{}
	foundISSNs := []string{}
	for _, source := range gathered.found {
		publisherKeys = append(publisherKeys, source.Publisher)
		homepages = append(homepages, URLHost(source.URL))
		foundISSNs = append(foundISSNs, source.ISSNs...)
	}
	publisherKeys = refKeys(publisherKeys)
	inputHost := URLHost(q.URL)

	hosts := HostCandidates(inputHost)
	for _, homepage := range homepages {
		hosts = append(hosts, HostCandidates(homepage)...)
	}
	rows, err := s.store.FindReferences(ctx, nameKeys, publisherKeys, hosts)
	if err != nil {
		return result, err
	}
	statuses, err := s.store.ListStatuses(ctx)
	if err != nil {
		return result, err
	}
	manual, err := s.findManual(ctx, q.Name, slices.Concat(inputISSNs, foundISSNs), titles, gathered.issnConflict)
	if err != nil {
		return result, err
	}

	hits := MatchHits(rows, MatchInput{NameKeys: nameKeys, PublisherKeys: publisherKeys, InputHost: inputHost, HomepageHosts: homepages})
	refreshed, available, stale := summarizeLists(statuses, s.now())

	result.Verdict, result.Reasons = Decide(DecideInput{
		Manual: manual, Hits: hits, Evidence: evidence, InputName: q.Name,
		ISSNMalformed: malformed, ISSNBadCheck: badCheck, NameAmbiguous: gathered.ambiguous,
		NameNotFound: gathered.nameNotFound, ISSNConflict: gathered.issnConflict,
		URLRejected:    strings.TrimSpace(q.URL) != "" && inputHost == "",
		ListsAvailable: available, ListsStale: stale,
	})
	// A rejected name match must not reach the screen as an "accredited" banner.
	rejected := manual != nil && manual.Classification == screening.ClassAccredited &&
		!accreditationApplies(manual, evidence, gathered.issnConflict)
	if manual != nil && !rejected {
		result.Matched, result.Classification, result.Name = true, manual.Classification, manual.Name
		result.Note, result.MatchedBy = manual.Note, manual.MatchedBy
	}
	result.ReferenceHits = hits
	result.Evidence = evidence
	result.ISSNs = canonical.ISSNs
	// Two journals have no single identity to offer back to the form.
	if !gathered.issnConflict {
		result.CanonicalName, result.Publisher, result.OnlineISSN = canonical.Name, canonical.Publisher, canonical.OnlineISSN
	}
	result.SourcesFailed = failedSources(evidence, available)
	result.ListsRefreshedAt = refreshed
	result.Throttled = access.throttled.Load()
	return result, nil
}

func (s *Service) findManual(ctx context.Context, name string, candidateISSNs, titles []string, issnConflict bool) (*Manual, error) {
	issns := normalizeISSNs(candidateISSNs)
	keys := []string{}
	seen := map[string]struct{}{}
	for _, candidate := range append([]string{name}, titles...) {
		key := screening.NameKey(candidate)
		if _, dup := seen[key]; key != "" && !dup {
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	rows, err := s.store.FindManual(ctx, issns, keys)
	if err != nil {
		return nil, err
	}
	if issnConflict {
		return PickWorstManual(rows, issns, keys), nil
	}
	return PickManual(rows, issns, keys), nil
}

// gatheredEvidence is what the sources said. evidence is the per-source summary
// shown to the user; found keeps every found answer, because two entered ISSNs can
// lead to two journals and each of them must be screened.
type gatheredEvidence struct {
	evidence     Evidence
	found        []SourceEvidence
	ambiguous    bool
	nameNotFound bool
	issnConflict bool
}

func (s *Service) gather(ctx context.Context, q Query, issns []string, access *externalAccess) gatheredEvidence {
	if len(issns) == 0 {
		evidence, ambiguous, notFound := s.gatherByName(ctx, q, notCheckedEvidence(), access)
		return gatheredEvidence{evidence: evidence, found: foundSources(evidence), ambiguous: ambiguous, nameNotFound: notFound}
	}
	perISSN := make([]Evidence, 0, len(issns))
	found := []SourceEvidence{}
	for _, issn := range issns {
		evidence := s.lookupAll(ctx, issn, access)
		perISSN = append(perISSN, evidence)
		found = append(found, foundSources(evidence)...)
	}
	return gatheredEvidence{evidence: mergeEvidence(perISSN), found: found, issnConflict: issnsDisagree(perISSN)}
}

func foundSources(e Evidence) []SourceEvidence {
	found := []SourceEvidence{}
	for _, source := range []SourceEvidence{e.OpenAlex, e.Crossref, e.DOAJ} {
		if source.Status == StatusFound {
			found = append(found, source)
		}
	}
	return found
}

// mergeEvidence keeps, per source, the most informative answer: an unanswered
// lookup must not read as "not found" just because the other ISSN got that reply.
func mergeEvidence(perISSN []Evidence) Evidence {
	merged := perISSN[0]
	for _, e := range perISSN[1:] {
		merged.OpenAlex = strongerSource(merged.OpenAlex, e.OpenAlex)
		merged.Crossref = strongerSource(merged.Crossref, e.Crossref)
		merged.DOAJ = strongerSource(merged.DOAJ, e.DOAJ)
	}
	return merged
}

func strongerSource(current, next SourceEvidence) SourceEvidence {
	if answerStrength(next.Status) > answerStrength(current.Status) {
		return next
	}
	return current
}

func answerStrength(status string) int {
	switch status {
	case StatusFound:
		return 3
	case StatusFailed:
		return 2
	case StatusNotChecked:
		return 1
	default:
		return 0
	}
}

// issnsDisagree reports whether two ISSNs both resolve yet no answer links them:
// sharing an ISSN or a title is enough, since sources often list only one ISSN.
func issnsDisagree(perISSN []Evidence) bool {
	for i, first := range perISSN {
		for _, second := range perISSN[i+1:] {
			a, b := foundSources(first), foundSources(second)
			if len(a) > 0 && len(b) > 0 && !anySameJournal(a, b) {
				return true
			}
		}
	}
	return false
}

func anySameJournal(a, b []SourceEvidence) bool {
	for _, x := range a {
		for _, y := range b {
			if sameJournal(x, y) {
				return true
			}
		}
	}
	return false
}

func sameJournal(a, b SourceEvidence) bool {
	for _, issn := range a.ISSNs {
		if slices.Contains(b.ISSNs, issn) {
			return true
		}
	}
	return slices.ContainsFunc(titlesOf([]SourceEvidence{a}), func(title string) bool {
		return titleMatchesExactly(b, RefKey(title))
	})
}

func (s *Service) gatherByName(ctx context.Context, q Query, evidence Evidence, access *externalAccess) (Evidence, bool, bool) {
	key := RefKey(q.Name)
	if utf8.RuneCountInString(key) < MinNameSearchRunes {
		return evidence, false, false
	}
	resolved := s.cached(ctx, SourceOpenAlexName, key, access, func(ctx context.Context) (SourceEvidence, error) {
		candidates, err := s.lookups.OpenAlexByName(ctx, q.Name)
		if err != nil {
			return SourceEvidence{}, err
		}
		return ResolveByName(candidates, q.Name), nil
	})
	switch resolved.Status {
	case statusAmbiguous:
		evidence.OpenAlex = SourceEvidence{Status: StatusNotChecked}
		return evidence, true, false
	case StatusFound:
		evidence.OpenAlex = resolved
		if len(resolved.ISSNs) > 0 {
			issn := resolved.ISSNs[0]
			var wg sync.WaitGroup
			wg.Go(func() { evidence.Crossref = s.lookupCrossref(ctx, issn, access) })
			wg.Go(func() { evidence.DOAJ = s.lookupDOAJ(ctx, issn, access) })
			wg.Wait()
		}
	default:
		evidence.OpenAlex = resolved
	}
	return evidence, false, resolved.Status == StatusNotFound
}

func (s *Service) lookupAll(ctx context.Context, issn string, access *externalAccess) Evidence {
	evidence := notCheckedEvidence()
	var wg sync.WaitGroup
	wg.Go(func() {
		evidence.OpenAlex = s.cached(ctx, SourceOpenAlex, issn, access, func(ctx context.Context) (SourceEvidence, error) {
			return s.lookups.OpenAlexByISSN(ctx, issn)
		})
	})
	wg.Go(func() { evidence.Crossref = s.lookupCrossref(ctx, issn, access) })
	wg.Go(func() { evidence.DOAJ = s.lookupDOAJ(ctx, issn, access) })
	wg.Wait()
	return evidence
}

func (s *Service) lookupCrossref(ctx context.Context, issn string, access *externalAccess) SourceEvidence {
	return s.cached(ctx, SourceCrossref, issn, access, func(ctx context.Context) (SourceEvidence, error) {
		return s.lookups.CrossrefByISSN(ctx, issn)
	})
}

func (s *Service) lookupDOAJ(ctx context.Context, issn string, access *externalAccess) SourceEvidence {
	return s.cached(ctx, SourceDOAJ, issn, access, func(ctx context.Context) (SourceEvidence, error) {
		return s.lookups.DOAJByISSN(ctx, issn)
	})
}

// cached serves one source through the cache. Failures come back as StatusFailed
// and are not stored, so the next check tries again.
func (s *Service) cached(ctx context.Context, source, key string, access *externalAccess, fetch func(context.Context) (SourceEvidence, error)) SourceEvidence {
	if payload, ok, err := s.store.CacheGet(ctx, source, key, s.now()); err != nil {
		slog.Warn("journal lookup cache read failed", "source", source, "error", err)
	} else if ok {
		var evidence SourceEvidence
		if err := json.Unmarshal(payload, &evidence); err == nil && evidence.Status != "" {
			return evidence
		}
	}
	if !access.allowed() {
		return SourceEvidence{Status: StatusNotChecked}
	}

	fetchCtx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()
	evidence, err := fetch(fetchCtx)
	switch {
	case errors.Is(err, ErrNotFound):
		evidence = SourceEvidence{Status: StatusNotFound}
	case err != nil:
		slog.Warn("journal source unavailable", "source", source, "error", err)
		return SourceEvidence{Status: StatusFailed}
	}
	s.writeCache(ctx, source, key, evidence)
	return evidence
}

func (s *Service) writeCache(ctx context.Context, source, key string, evidence SourceEvidence) {
	ttl := foundTTL
	if evidence.Status != StatusFound {
		ttl = notFoundTTL
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		slog.Warn("journal lookup cache encode failed", "source", source, "error", err)
		return
	}
	if err := s.store.CachePut(ctx, source, key, payload, s.now(), ttl); err != nil {
		slog.Warn("journal lookup cache write failed", "source", source, "error", err)
	}
}

func notCheckedEvidence() Evidence {
	notChecked := SourceEvidence{Status: StatusNotChecked}
	return Evidence{OpenAlex: notChecked, Crossref: notChecked, DOAJ: notChecked, Scopus: SourceEvidence{Status: StatusUnknown}}
}

func splitByChecksum(issns []string) ([]string, bool) {
	valid := []string{}
	bad := false
	for _, issn := range issns {
		if screening.ValidISSNChecksum(issn) {
			valid = append(valid, issn)
		} else {
			bad = true
		}
	}
	return valid, bad
}

func evidenceTitles(e Evidence) []string {
	return titlesOf(foundSources(e))
}

func titlesOf(sources []SourceEvidence) []string {
	titles := []string{}
	for _, source := range sources {
		titles = append(titles, source.Title)
		titles = append(titles, source.AlternateTitles...)
	}
	return titles
}

func refKeys(values []string) []string {
	seen := map[string]struct{}{}
	keys := []string{}
	for _, value := range values {
		key := RefKey(value)
		if _, dup := seen[key]; key != "" && !dup {
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}

func failedSources(e Evidence, listsAvailable bool) []string {
	failed := []string{}
	if e.OpenAlex.Status == StatusFailed {
		failed = append(failed, SourceOpenAlex)
	}
	if e.Crossref.Status == StatusFailed {
		failed = append(failed, SourceCrossref)
	}
	if e.DOAJ.Status == StatusFailed {
		failed = append(failed, SourceDOAJ)
	}
	if !listsAvailable {
		failed = append(failed, SourceReferenceLists)
	}
	return failed
}

// summarizeLists returns the oldest successful refresh, so the date shown never
// overstates freshness, and whether all three lists hold data.
func summarizeLists(statuses []ListStatus, now time.Time) (*time.Time, bool, bool) {
	byType := map[string]ListStatus{}
	for _, status := range statuses {
		byType[status.ListType] = status
	}
	var oldest *time.Time
	available, stale := true, false
	for _, listType := range ListTypes {
		status, ok := byType[listType]
		if !ok || status.Count == 0 {
			available = false
			continue
		}
		if status.LastSuccessAt == nil {
			stale = true
			continue
		}
		if oldest == nil || status.LastSuccessAt.Before(*oldest) {
			oldest = status.LastSuccessAt
		}
		if now.Sub(*status.LastSuccessAt) > listsStaleAfter {
			stale = true
		}
	}
	return oldest, available, stale
}
