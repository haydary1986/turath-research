// Package portal holds the pure business rules of the scientific affairs portal:
// request kinds, the status state machine, field validation and numbering.
// It has no database or HTTP dependencies so it can be tested in isolation.
package portal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Request kinds.
const (
	KindResearchPlan        = "research_plan"
	KindJournalVerification = "journal_verification"
	KindAppreciationLetter  = "appreciation_letter"
)

// Request statuses.
const (
	StatusDraft              = "draft"
	StatusPendingDean        = "pending_dean"
	StatusPending            = "pending"
	StatusPendingVP          = "pending_vp"
	StatusReturned           = "returned"
	StatusApproved           = "approved"
	StatusRejected           = "rejected"
	StatusVerified           = "verified"
	StatusNotEligible        = "not_eligible"
	StatusAwaitingSignatures = "awaiting_signatures"
	StatusCompleted          = "completed"
	StatusIssued             = "issued"
)

// Attachment roles.
const (
	RoleResearchForm      = "research_form"
	RoleJournalForm       = "journal_form"
	RoleCommitteeApproval = "committee_approval"
	RolePublishedPaper    = "published_paper"
	RoleSignedForm        = "signed_form"
	RoleOther             = "other"
)

// Kinds lists every supported request kind.
var Kinds = []string{KindResearchPlan, KindJournalVerification, KindAppreciationLetter}

var kindPrefix = map[string]string{
	KindResearchPlan:        "RP",
	KindJournalVerification: "JV",
	KindAppreciationLetter:  "AR",
}

// transitions[kind][from] lists the statuses reachable from `from`.
// The chain for every kind: submit -> college dean (initial approval) ->
// scientific affairs (decision, raised as a recommendation) -> vice president
// for research (confirms or sends back) -> clerical finishing steps.
var transitions = map[string]map[string][]string{
	KindResearchPlan: {
		StatusDraft:       {StatusPendingDean},
		StatusReturned:    {StatusPendingDean},
		StatusPendingDean: {StatusPending, StatusReturned},
		StatusPending:     {StatusPendingVP, StatusReturned},
		StatusPendingVP:   {StatusApproved, StatusRejected, StatusPending},
	},
	KindJournalVerification: {
		StatusDraft:              {StatusPendingDean},
		StatusReturned:           {StatusPendingDean},
		StatusPendingDean:        {StatusPending, StatusReturned},
		StatusPending:            {StatusPendingVP, StatusReturned},
		StatusPendingVP:          {StatusVerified, StatusNotEligible, StatusPending},
		StatusVerified:           {StatusAwaitingSignatures},
		StatusAwaitingSignatures: {StatusCompleted},
	},
	KindAppreciationLetter: {
		StatusDraft:       {StatusPendingDean},
		StatusReturned:    {StatusPendingDean},
		StatusPendingDean: {StatusPending, StatusReturned},
		StatusPending:     {StatusPendingVP, StatusReturned},
		StatusPendingVP:   {StatusVerified, StatusNotEligible, StatusPending},
		StatusVerified:    {StatusIssued},
	},
}

// Role keys (mirror the users.role enum values that remain in use).
const (
	RoleAdmin             = "admin"
	RoleVPResearch        = "vp_research"
	RoleScientificAffairs = "scientific_affairs"
	RoleCollegeDean       = "college_dean"
	RoleDepartmentHead    = "department_head"
	RoleResearcher        = "researcher"
)

// RequiredRole names the role that acts on a request sitting in `from`.
// Admin may always act; the department head never decides (view only).
func RequiredRole(from string) string {
	switch from {
	case StatusPendingDean:
		return RoleCollegeDean
	case StatusPending:
		return RoleScientificAffairs
	case StatusPendingVP:
		return RoleVPResearch
	case StatusVerified, StatusAwaitingSignatures:
		return RoleScientificAffairs
	default:
		return ""
	}
}

// IsActiveStage reports whether the request sits at a stage someone still has to act on.
func IsActiveStage(status string) bool {
	return RequiredRole(status) != ""
}

// QueueStatuses lists the statuses awaiting action by the role. Admin covers every stage.
func QueueStatuses(role string) []string {
	switch role {
	case RoleAdmin:
		return []string{StatusPendingDean, StatusPending, StatusPendingVP, StatusVerified, StatusAwaitingSignatures}
	case RoleCollegeDean:
		return []string{StatusPendingDean}
	case RoleScientificAffairs:
		return []string{StatusPending, StatusVerified, StatusAwaitingSignatures}
	case RoleVPResearch:
		return []string{StatusPendingVP}
	default:
		return []string{}
	}
}

// FinalDecisions lists the outcomes scientific affairs may recommend (and the VP confirm).
func FinalDecisions(kind string) []string {
	if kind == KindResearchPlan {
		return []string{StatusApproved, StatusRejected}
	}
	return []string{StatusVerified, StatusNotEligible}
}

func IsFinalDecision(kind, status string) bool {
	for _, s := range FinalDecisions(kind) {
		if s == status {
			return true
		}
	}
	return false
}

var staffRoles = map[string]struct{}{
	RoleAdmin:             {},
	RoleScientificAffairs: {},
	RoleVPResearch:        {},
}

// IsStaff reports whether the role may review requests and issue documents.
func IsStaff(role string) bool {
	_, ok := staffRoles[role]
	return ok
}

// IsKind reports whether kind is a supported request kind.
func IsKind(kind string) bool {
	_, ok := kindPrefix[kind]
	return ok
}

// CanTransition reports whether a request of the given kind may move from one status to another.
func CanTransition(kind, from, to string) bool {
	for _, next := range transitions[kind][from] {
		if next == to {
			return true
		}
	}
	return false
}

// RequiresNote reports whether moving to the status needs an explanation for the applicant.
func RequiresNote(to string) bool {
	return to == StatusRejected || to == StatusNotEligible || to == StatusReturned
}

// NextStatuses lists the statuses reachable from `from` for the kind (a copy, safe to modify).
func NextStatuses(kind, from string) []string {
	return append([]string(nil), transitions[kind][from]...)
}

// FormatRequestNo builds the public request number, e.g. RP-2026-00124.
func FormatRequestNo(kind string, year, seq int) string {
	return fmt.Sprintf("%s-%d-%05d", kindPrefix[kind], year, seq)
}

// NormalizeDOI strips URL/prefix noise and lowercases the DOI so duplicates compare equal.
func NormalizeDOI(raw string) string {
	doi := strings.ToLower(strings.TrimSpace(raw))
	for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "https://dx.doi.org/", "http://dx.doi.org/", "doi:"} {
		doi = strings.TrimPrefix(doi, prefix)
	}
	return strings.TrimSpace(doi)
}

var attachmentRoles = map[string][]string{
	KindResearchPlan:        {RoleResearchForm, RoleSignedForm, RoleOther},
	KindJournalVerification: {RoleJournalForm, RoleCommitteeApproval, RolePublishedPaper, RoleSignedForm, RoleOther},
	KindAppreciationLetter:  {RoleJournalForm, RoleCommitteeApproval, RolePublishedPaper, RoleOther},
}

// AllowedAttachmentRole reports whether a request kind accepts files of this role.
func AllowedAttachmentRole(kind, role string) bool {
	for _, r := range attachmentRoles[kind] {
		if r == role {
			return true
		}
	}
	return false
}

// RequiredAttachments lists the roles that must be present before a request can be submitted.
// An appreciation letter linked to a completed journal verification reuses its signed form.
func RequiredAttachments(kind string, hasVerifiedJournal bool) []string {
	switch kind {
	case KindResearchPlan:
		// The registration form must already carry the department scientific committee's signature.
		return []string{RoleResearchForm}
	case KindAppreciationLetter:
	default:
		return nil
	}
	if hasVerifiedJournal {
		return []string{RoleCommitteeApproval}
	}
	return []string{RoleJournalForm, RoleCommitteeApproval}
}

var (
	issnPattern     = regexp.MustCompile(`^\d{4}-?\d{3}[\dXx]$`)
	validWorkTypes  = map[string]struct{}{"individual": {}, "joint": {}}
	validIndexDBs   = map[string]struct{}{"scopus": {}, "wos": {}, "local": {}, "international_unindexed": {}}
	validExtracted  = map[string]struct{}{"not_extracted": {}, "extracted_master": {}, "extracted_phd": {}}
	validQuartiles  = map[string]struct{}{"Q1": {}, "Q2": {}, "Q3": {}, "Q4": {}}
	journalURLKeys  = []string{"article_url", "journal_url", "index_url", "author_url", "local_journal_url"}
	journalRequired = []string{"journal_name", "publisher", "index_db", "publication_date"}
)

const (
	minDateYear = 1900
	maxDateYear = 2100
	maxAuthors  = 10000
)

// QuartileApplies reports whether the indexing database ranks journals in quartiles.
func QuartileApplies(indexDB string) bool {
	return indexDB == "scopus" || indexDB == "wos"
}

// ValidateDetails checks the kind-specific fields and returns one "field:code" entry per problem.
func ValidateDetails(kind string, details map[string]any) []string {
	switch kind {
	case KindResearchPlan:
		return validatePlan(details)
	case KindJournalVerification, KindAppreciationLetter:
		return validateJournal(details)
	default:
		return []string{"kind:invalid"}
	}
}

func validatePlan(d map[string]any) []string {
	var errs []string
	if _, ok := validWorkTypes[str(d, "work_type")]; !ok {
		errs = append(errs, "work_type:invalid")
	}
	errs = append(errs, requireDate(d, "expected_completion")...)
	return errs
}

func validateJournal(d map[string]any) []string {
	var errs []string
	for _, key := range journalRequired {
		if str(d, key) == "" {
			errs = append(errs, key+":required")
		}
	}
	if db := str(d, "index_db"); db != "" {
		if _, ok := validIndexDBs[db]; !ok {
			errs = append(errs, "index_db:invalid")
		}
	}
	if date := str(d, "publication_date"); date != "" {
		errs = append(errs, requireDate(d, "publication_date")...)
	}
	// Ministry-form dates (submission to the journal, acceptance): optional but must parse.
	for _, key := range []string{"submission_date", "acceptance_date"} {
		if str(d, key) != "" {
			errs = append(errs, requireDate(d, key)...)
		}
	}
	if issn := str(d, "issn"); issn != "" && !issnPattern.MatchString(issn) {
		errs = append(errs, "issn:malformed")
	}
	if issn := str(d, "issn_online"); issn != "" && !issnPattern.MatchString(issn) {
		errs = append(errs, "issn_online:malformed")
	}
	if ex := str(d, "extracted_thesis"); ex != "" {
		if _, ok := validExtracted[ex]; !ok {
			errs = append(errs, "extracted_thesis:invalid")
		}
	}
	if q := str(d, "quartile"); q != "" {
		if _, ok := validQuartiles[q]; !ok || !QuartileApplies(str(d, "index_db")) {
			errs = append(errs, "quartile:invalid")
		}
	}
	errs = append(errs, validateNumbers(d)...)
	for _, key := range journalURLKeys {
		if u := str(d, key); u != "" && !isHTTPURL(u) {
			errs = append(errs, key+":invalid_url")
		}
	}
	return errs
}

// validateNumbers checks the optional numeric ministry-form fields.
func validateNumbers(d map[string]any) []string {
	var errs []string
	if v := numeric(d, "plagiarism_pct"); v != "" {
		// The inverted comparison also rejects NaN, which ParseFloat accepts.
		if f, err := strconv.ParseFloat(v, 64); err != nil || !(f >= 0 && f <= 100) {
			errs = append(errs, "plagiarism_pct:invalid")
		}
	}
	count, countOK := 0, false
	if v := numeric(d, "authors_count"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxAuthors {
			errs = append(errs, "authors_count:invalid")
		} else {
			count, countOK = n, true
		}
	}
	if v := numeric(d, "author_order"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxAuthors || (countOK && n > count) {
			errs = append(errs, "author_order:invalid")
		}
	}
	return errs
}

// numeric returns the field as text whether the JSON held a string or a number.
// Any other JSON type yields text that fails numeric parsing.
func numeric(d map[string]any, key string) string {
	switch v := d[key].(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return "invalid"
	}
}

func requireDate(d map[string]any, key string) []string {
	v := str(d, key)
	if v == "" {
		return []string{key + ":required"}
	}
	parsed, err := time.Parse("2006-01-02", v)
	if err != nil || parsed.Year() < minDateYear || parsed.Year() > maxDateYear {
		return []string{key + ":invalid_date"}
	}
	return nil
}

func isHTTPURL(u string) bool {
	return strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")
}

func str(d map[string]any, key string) string {
	s, _ := d[key].(string)
	return strings.TrimSpace(s)
}
