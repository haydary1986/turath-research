// Package journalcheck gathers advisory evidence about a journal from public
// predatory/hijacked lists, OpenAlex, Crossref and DOAJ. It never decides: the
// result supports the scientific affairs review.
package journalcheck

import "time"

// Reference list types.
const (
	ListPredatoryJournal   = "predatory_journal"
	ListPredatoryPublisher = "predatory_publisher"
	ListHijacked           = "hijacked"
)

// ListTypes is every reference list kept by the refresher.
var ListTypes = []string{ListPredatoryJournal, ListPredatoryPublisher, ListHijacked}

// Overall verdicts. VerdictFound only means the journal exists in the checked
// sources; it is not a statement of quality.
const (
	VerdictDanger  = "danger"
	VerdictCaution = "caution"
	VerdictFound   = "found"
	VerdictUnknown = "unknown"
)

// Per-source statuses.
const (
	StatusFound      = "found"
	StatusNotFound   = "not_found"
	StatusFailed     = "failed"
	StatusNotChecked = "not_checked"
	StatusUnknown    = "unknown"
	statusAmbiguous  = "ambiguous"
)

// Source names, as reported in Result.SourcesFailed and used as cache keys.
const (
	SourceOpenAlex       = "openalex"
	SourceOpenAlexName   = "openalex_name"
	SourceCrossref       = "crossref"
	SourceDOAJ           = "doaj"
	SourceReferenceLists = "reference_lists"
)

// How a reference-list row matched.
const (
	MatchName      = "name"
	MatchPublisher = "publisher"
	MatchURL       = "url"
	MatchHomepage  = "homepage"
)

// Reason codes explaining a verdict; the frontend translates them.
const (
	ReasonManualPredatory   = "manual_predatory"
	ReasonManualSuspended   = "manual_suspended"
	ReasonManualAccredited  = "manual_accredited"
	ReasonManualOverride    = "manual_override"
	ReasonHijackedURL       = "hijacked_url"
	ReasonHijackedName      = "hijacked_name"
	ReasonPredatoryURL      = "predatory_url"
	ReasonPredatoryName     = "predatory_name"
	ReasonPredatoryPub      = "predatory_publisher"
	ReasonListConflictDOAJ  = "list_conflict_doaj"
	ReasonNameMismatch      = "name_mismatch"
	ReasonISSNChecksum      = "issn_checksum_invalid"
	ReasonISSNMalformed     = "issn_malformed"
	ReasonNameAmbiguous     = "name_ambiguous"
	ReasonIndexed           = "found_in_sources"
	ReasonNoEvidence        = "no_evidence"
	ReasonSourcesFailed     = "sources_unavailable"
	ReasonListsUnavailable  = "lists_unavailable"
	ReasonListsStale        = "lists_stale"
	ReasonNotChecked        = "not_checked"
	ReasonURLRejected       = "url_rejected"
	ReasonISSNConflict      = "issn_conflict"
	ReasonManualUnconfirmed = "manual_name_unconfirmed"
)

// RefEntry is one row of a reference list.
type RefEntry struct {
	ListType      string
	NameKey       string
	Name          string
	URL           string
	URLHost       string
	AuthenticName string
	AuthenticURL  string
	Source        string
	FetchedAt     time.Time
}

// SourceEvidence is what one external source says about a journal.
type SourceEvidence struct {
	Status          string   `json:"status"`
	Title           string   `json:"title,omitempty"`
	AlternateTitles []string `json:"alternate_titles,omitempty"`
	Publisher       string   `json:"publisher,omitempty"`
	ISSNs           []string `json:"issns,omitempty"`
	PrintISSN       string   `json:"print_issn,omitempty"`
	OnlineISSN      string   `json:"online_issn,omitempty"`
	Type            string   `json:"type,omitempty"`
	InDOAJ          *bool    `json:"in_doaj,omitempty"`
	HIndex          *int     `json:"h_index,omitempty"`
	MeanCitedness   *float64 `json:"two_year_mean_citedness,omitempty"`
	WorksCount      *int     `json:"works_count,omitempty"`
	TotalDOIs       *int     `json:"total_dois,omitempty"`
	URL             string   `json:"url,omitempty"`
}

// Evidence groups the per-source answers. Scopus has no keyless API, so its
// status stays "unknown" and must never read as "not indexed".
type Evidence struct {
	OpenAlex SourceEvidence `json:"openalex"`
	Crossref SourceEvidence `json:"crossref"`
	DOAJ     SourceEvidence `json:"doaj"`
	Scopus   SourceEvidence `json:"scopus"`
}

// ReferenceHit is a reference-list row that matched the checked journal.
type ReferenceHit struct {
	List           string `json:"list"`
	MatchedBy      string `json:"matched_by"`
	Name           string `json:"name"`
	URL            string `json:"url,omitempty"`
	AuthenticName  string `json:"authentic_name,omitempty"`
	AuthenticURL   string `json:"authentic_url,omitempty"`
	AuthenticMatch bool   `json:"authentic_match,omitempty"`
	Source         string `json:"source"`
	// matchedKey is the row's folded name or alias, which may be only part of Name.
	matchedKey string
}

// Manual is the staff-maintained watchlist entry that applies, if any.
type Manual struct {
	Classification string
	Name           string
	Note           *string
	MatchedBy      string
}

// Result is the advisory outcome. Matched, Classification, Name, Note and
// MatchedBy describe the manual watchlist and keep the original journal_flag shape.
type Result struct {
	Matched          bool           `json:"matched"`
	Classification   string         `json:"classification,omitempty"`
	Name             string         `json:"name,omitempty"`
	Note             *string        `json:"note,omitempty"`
	MatchedBy        string         `json:"matched_by,omitempty"`
	Advisory         bool           `json:"advisory"`
	Verdict          string         `json:"verdict"`
	Reasons          []string       `json:"reasons"`
	ReferenceHits    []ReferenceHit `json:"reference_hits"`
	Evidence         Evidence       `json:"evidence"`
	CanonicalName    string         `json:"canonical_name,omitempty"`
	Publisher        string         `json:"publisher,omitempty"`
	ISSNs            []string       `json:"issns"`
	OnlineISSN       string         `json:"issn_online,omitempty"`
	SourcesFailed    []string       `json:"sources_failed"`
	ListsRefreshedAt *time.Time     `json:"lists_refreshed_at,omitempty"`
	// Throttled means the caller's uncached-lookup allowance ran out, so external
	// sources were skipped this time. Local lists were still consulted.
	Throttled bool `json:"throttled,omitempty"`
}

// ListStatus reports the state of one reference list.
type ListStatus struct {
	ListType      string     `json:"list_type"`
	Count         int        `json:"count"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	LastFailed    bool       `json:"last_failed"`
}
