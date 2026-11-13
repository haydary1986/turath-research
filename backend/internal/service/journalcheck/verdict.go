package journalcheck

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

// minContainedKeyRunes stops a fragment like "j" from counting as the journal's name.
const minContainedKeyRunes = 6

// MatchInput carries everything a reference row can match against.
type MatchInput struct {
	NameKeys      []string
	PublisherKeys []string
	InputHost     string
	HomepageHosts []string
}

// MatchHits turns candidate rows into deduplicated hits, strongest evidence first
// per row: the URL the user gave, then a name, then a homepage from the indexes.
func MatchHits(rows []RefEntry, in MatchInput) []ReferenceHit {
	names := toSet(in.NameKeys)
	publishers := toSet(in.PublisherKeys)
	inputHosts := toSet(HostCandidates(in.InputHost))
	homepages := map[string]struct{}{}
	for _, host := range in.HomepageHosts {
		for _, candidate := range HostCandidates(host) {
			homepages[candidate] = struct{}{}
		}
	}

	hits := []ReferenceHit{}
	seen := map[string]struct{}{}
	for _, row := range rows {
		matchedBy := matchRow(row, names, publishers, inputHosts, homepages)
		if matchedBy == "" {
			continue
		}
		// By display name: the aliases of one composite row must read as one hit.
		id := row.ListType + "|" + row.Name + "|" + matchedBy
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		authentic := URLHost(row.AuthenticURL)
		hits = append(hits, ReferenceHit{
			List: row.ListType, MatchedBy: matchedBy, Name: row.Name, URL: row.URL,
			AuthenticName: row.AuthenticName, AuthenticURL: row.AuthenticURL,
			AuthenticMatch: authentic != "" && authentic == in.InputHost,
			Source:         row.Source,
			matchedKey:     row.NameKey,
		})
	}
	return hits
}

func matchRow(row RefEntry, names, publishers, inputHosts, homepages map[string]struct{}) string {
	if _, ok := inputHosts[row.URLHost]; ok && row.URLHost != "" {
		return MatchURL
	}
	if row.ListType == ListPredatoryPublisher {
		if _, ok := publishers[row.NameKey]; ok {
			return MatchPublisher
		}
	} else if _, ok := names[row.NameKey]; ok {
		return MatchName
	}
	if _, ok := homepages[row.URLHost]; ok && row.URLHost != "" {
		return MatchHomepage
	}
	return ""
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

// DecideInput is everything the overall verdict depends on.
type DecideInput struct {
	Manual        *Manual
	Hits          []ReferenceHit
	Evidence      Evidence
	InputName     string
	ISSNMalformed bool
	ISSNBadCheck  bool
	NameAmbiguous bool
	// NameNotFound: the only search a bare name allows ran and found nothing.
	NameNotFound bool
	// ISSNConflict: the entered ISSNs resolve to different journals.
	ISSNConflict   bool
	URLRejected    bool
	ListsAvailable bool
	ListsStale     bool
}

const (
	rankUnknown = iota
	rankFound
	rankCaution
	rankDanger
)

// Decide combines the evidence. A failed source can only add a reason or lower
// "found" to "unknown": missing data is never treated as a clean result.
func Decide(in DecideInput) (string, []string) {
	reasons := newReasons()
	accredited := accreditationApplies(in.Manual, in.Evidence, in.ISSNConflict)

	rank := rankUnknown
	if in.Manual != nil {
		switch in.Manual.Classification {
		case screening.ClassPredatory:
			rank = rankDanger
			reasons.add(ReasonManualPredatory)
		case screening.ClassSuspended:
			rank = rankCaution
			reasons.add(ReasonManualSuspended)
		case screening.ClassAccredited:
			if accredited {
				rank = rankFound
				reasons.add(ReasonManualAccredited)
			} else {
				reasons.add(ReasonManualUnconfirmed)
			}
		}
	}

	for _, hit := range in.Hits {
		hitRank, reason := rankHit(hit, in.Evidence)
		// A ministry exception covers the authentic journal, never a clone's website.
		if accredited && !(hit.List == ListHijacked && hit.MatchedBy == MatchURL) {
			if hitRank > rankFound {
				reasons.add(ReasonManualOverride)
			}
			continue
		}
		if hitRank > rank {
			rank = hitRank
		}
		reasons.add(reason)
	}

	// A name-only exception says nothing about a mistyped ISSN.
	if in.ISSNBadCheck && (!accredited || in.Manual.MatchedBy == MatchName) {
		rank = max(rank, rankCaution)
		reasons.add(ReasonISSNChecksum)
	}
	if !accredited && nameConflicts(in.InputName, in.Evidence) {
		rank = max(rank, rankCaution)
		reasons.add(ReasonNameMismatch)
	}
	if in.ISSNConflict {
		rank = max(rank, rankCaution)
		reasons.add(ReasonISSNConflict)
	}
	if in.URLRejected {
		reasons.add(ReasonURLRejected)
	}
	if in.ISSNMalformed {
		reasons.add(ReasonISSNMalformed)
	}
	if in.NameAmbiguous {
		reasons.add(ReasonNameAmbiguous)
	}

	sources := []SourceEvidence{in.Evidence.OpenAlex, in.Evidence.Crossref, in.Evidence.DOAJ}
	found, failed, notChecked := false, false, false
	for _, source := range sources {
		found = found || source.Status == StatusFound
		failed = failed || source.Status == StatusFailed
		notChecked = notChecked || source.Status == StatusNotChecked
	}
	if failed {
		reasons.add(ReasonSourcesFailed)
	}
	if !in.ListsAvailable {
		reasons.add(ReasonListsUnavailable)
	} else if in.ListsStale {
		reasons.add(ReasonListsStale)
	}
	if found {
		reasons.add(ReasonIndexed)
		// Without the lists no predatory check happened, so existence alone must not read as fine.
		if rank < rankFound && in.ListsAvailable {
			rank = rankFound
		}
	} else if in.NameNotFound {
		// Crossref and DOAJ have no name search, so their "not checked" is not missing input.
		reasons.add(ReasonNoEvidence)
	} else if notChecked {
		reasons.add(ReasonNotChecked)
	} else if !failed {
		reasons.add(ReasonNoEvidence)
	}
	return verdictFor(rank), reasons.values
}

// accreditationApplies reports whether a manual accredited entry speaks for the
// journal under check. One matched only by name cannot vouch for an ISSN that
// resolves elsewhere: clones copy names, and an Arabic name cannot be compared
// with an English title, so agreement must be shown rather than assumed.
func accreditationApplies(manual *Manual, e Evidence, issnConflict bool) bool {
	if manual == nil || manual.Classification != screening.ClassAccredited || issnConflict {
		return false
	}
	if manual.MatchedBy != MatchName {
		return true
	}
	titles := evidenceTitles(e)
	return len(titles) == 0 || titleAgrees(RefKey(manual.Name), titles)
}

func rankHit(hit ReferenceHit, e Evidence) (int, string) {
	if hit.List == ListHijacked {
		switch {
		case hit.MatchedBy == MatchURL:
			return rankDanger, ReasonHijackedURL
		case hit.AuthenticMatch:
			return rankUnknown, ReasonHijackedName
		default:
			return rankCaution, ReasonHijackedName
		}
	}
	if hit.MatchedBy == MatchURL {
		return rankDanger, ReasonPredatoryURL
	}
	if doajConfirmsSameJournal(hit, e) {
		return rankCaution, ReasonListConflictDOAJ
	}
	if hit.List == ListPredatoryPublisher {
		return rankDanger, ReasonPredatoryPub
	}
	return rankDanger, ReasonPredatoryName
}

func verdictFor(rank int) string {
	switch rank {
	case rankDanger:
		return VerdictDanger
	case rankCaution:
		return VerdictCaution
	case rankFound:
		return VerdictFound
	default:
		return VerdictUnknown
	}
}

// doajConfirmsSameJournal reports whether a DOAJ listing concerns the journal the
// hit names. Otherwise a typed predatory name could borrow the DOAJ standing of
// whatever journal the ISSN resolves to.
func doajConfirmsSameJournal(hit ReferenceHit, e Evidence) bool {
	if e.DOAJ.Status == StatusFound && hitDescribes(hit, e.DOAJ) {
		return true
	}
	return e.OpenAlex.Status == StatusFound && e.OpenAlex.InDOAJ != nil && *e.OpenAlex.InDOAJ && hitDescribes(hit, e.OpenAlex)
}

func hitDescribes(hit ReferenceHit, source SourceEvidence) bool {
	switch hit.MatchedBy {
	case MatchHomepage:
		return slices.Contains(HostCandidates(URLHost(source.URL)), URLHost(hit.URL))
	case MatchPublisher:
		return hit.matchedKey == "" || hit.matchedKey == RefKey(source.Publisher)
	default:
		return hit.matchedKey == "" || titleMatchesExactly(source, hit.matchedKey)
	}
}

// nameConflicts compares the typed name with the titles the ISSN resolved to.
// Non-Latin names are skipped: an Arabic name differing from an English index
// title proves nothing.
func nameConflicts(input string, e Evidence) bool {
	key := RefKey(input)
	if key == "" || !isLatin(key) {
		return false
	}
	titles := evidenceTitles(e)
	usable := false
	for _, title := range titles {
		usable = usable || RefKey(title) != ""
	}
	return usable && !titleAgrees(key, titles)
}

// titleAgrees reports whether key equals a title or, when both are long enough
// to mean something, contains or is contained by one. An empty or short title
// is inside almost any name, so it never agrees by containment.
func titleAgrees(key string, titles []string) bool {
	if key == "" {
		return false
	}
	long := utf8.RuneCountInString(key) >= minContainedKeyRunes
	for _, title := range titles {
		titleKey := RefKey(title)
		if titleKey == key {
			return true
		}
		if long && utf8.RuneCountInString(titleKey) >= minContainedKeyRunes &&
			(strings.Contains(titleKey, key) || strings.Contains(key, titleKey)) {
			return true
		}
	}
	return false
}

func isLatin(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.Is(unicode.Latin, r) {
			return false
		}
	}
	return true
}

// Canonical holds the identity offered back to the form.
type Canonical struct {
	Name       string
	Publisher  string
	ISSNs      []string
	OnlineISSN string
}

// CanonicalFrom merges sources, OpenAlex first. OpenAlex does not type its ISSNs,
// so the online one comes from a typed source, or is inferred only when OpenAlex
// lists exactly two ISSNs and the other is known to be the print one.
func CanonicalFrom(e Evidence, enteredPrintISSN string) Canonical {
	sources := []SourceEvidence{e.OpenAlex, e.Crossref, e.DOAJ}
	out := Canonical{}
	all := []string{}
	print := ""
	for _, source := range sources {
		if source.Status != StatusFound {
			continue
		}
		if out.Name == "" {
			out.Name = source.Title
		}
		if out.Publisher == "" {
			out.Publisher = source.Publisher
		}
		if out.OnlineISSN == "" {
			out.OnlineISSN = source.OnlineISSN
		}
		if print == "" {
			print = source.PrintISSN
		}
		all = append(all, source.ISSNs...)
	}
	out.ISSNs = normalizeISSNs(all)
	if out.ISSNs == nil {
		out.ISSNs = []string{}
	}
	if print == "" {
		print = enteredPrintISSN
	}
	if issns := e.OpenAlex.ISSNs; out.OnlineISSN == "" && print != "" && len(issns) == 2 {
		switch print {
		case issns[0]:
			out.OnlineISSN = issns[1]
		case issns[1]:
			out.OnlineISSN = issns[0]
		}
	}
	return out
}

// manualSeverity orders classifications when the entered ISSNs disagree, so the
// entry of one journal cannot hide a warning about the other.
var manualSeverity = []string{screening.ClassPredatory, screening.ClassSuspended, screening.ClassAccredited}

// PickWorstManual is PickManual restricted to the most severe classification present.
func PickWorstManual(rows []ManualRow, issns, nameKeys []string) *Manual {
	for _, class := range manualSeverity {
		matching := slices.DeleteFunc(slices.Clone(rows), func(row ManualRow) bool { return row.Classification != class })
		if picked := PickManual(matching, issns, nameKeys); picked != nil {
			return picked
		}
	}
	return nil
}

// PickManual applies the watchlist precedence: ISSNs in the given order, then names.
func PickManual(rows []ManualRow, issns, nameKeys []string) *Manual {
	for _, issn := range issns {
		for _, row := range rows {
			if row.ISSN == issn {
				return &Manual{row.Classification, row.Name, row.Note, MatchISSN}
			}
		}
	}
	for _, key := range nameKeys {
		for _, row := range rows {
			if row.NameKey == key {
				return &Manual{row.Classification, row.Name, row.Note, MatchName}
			}
		}
	}
	return nil
}

type reasonList struct {
	values []string
	seen   map[string]struct{}
}

func newReasons() *reasonList {
	return &reasonList{values: []string{}, seen: map[string]struct{}{}}
}

func (r *reasonList) add(reason string) {
	if _, dup := r.seen[reason]; !dup {
		r.seen[reason] = struct{}{}
		r.values = append(r.values, reason)
	}
}
