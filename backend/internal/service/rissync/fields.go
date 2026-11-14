package rissync

import (
	"regexp"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/service/contactinfo"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
)

// Field keys, also stored in ris_sync_state.field.
const (
	FieldNameAr       = "name_ar"
	FieldNameEn       = "name_en"
	FieldDepartment   = "department"
	FieldPhone        = "phone"
	FieldORCID        = "orcid"
	FieldScholar      = "google_scholar"
	FieldScopus       = "scopus"
	FieldResearchGate = "researchgate"
	FieldLinkedIn     = "linkedin"
	FieldWebOfScience = "webofscience"
)

// FieldSpec says how one field is reconciled on the first sync.
type FieldSpec struct {
	Key    string
	Policy Policy
}

// Fields lists every synced field. Official fields come from the staff sheet, so the
// portal imposes them on the first sync; contact and profile links fill each other's gaps.
var Fields = []FieldSpec{
	{FieldNameAr, PortalOfficial},
	{FieldNameEn, PortalOfficial},
	{FieldDepartment, PortalOfficial},
	{FieldPhone, FillGaps},
	{FieldORCID, FillGaps},
	{FieldScholar, FillGaps},
	{FieldScopus, FillGaps},
	{FieldResearchGate, FillGaps},
	{FieldLinkedIn, FillGaps},
	{FieldWebOfScience, FillGaps},
}

// LinkFields are the fields stored in RIS's researcher_social_profiles, by RIS platform name.
var LinkFields = []string{FieldORCID, FieldScholar, FieldScopus, FieldResearchGate, FieldLinkedIn, FieldWebOfScience}

var spaces = regexp.MustCompile(`\s+`)

// CanonicalName trims and collapses whitespace; spelling is kept as entered.
func CanonicalName(raw string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(researchlinks.Clean(raw), " "))
}

// CanonicalPhone is the 07XXXXXXXXX form, or "" when the number is not a valid Iraqi mobile.
func CanonicalPhone(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if phone, ok := contactinfo.ParsePhone(raw); ok {
		return phone
	}
	return ""
}

// CanonicalLink reduces a profile link (or ORCID) to the canonical form the portal stores:
// the same profile written two ways compares equal. ok is false for a value that is not a
// valid link of that kind, which is then never copied.
func CanonicalLink(field, raw string) (value string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", true
	}
	var in researchlinks.Input
	switch field {
	case FieldORCID:
		in.ORCID = raw
	case FieldScholar:
		in.GoogleScholarURL = raw
	case FieldScopus:
		in.ScopusURL = raw
	case FieldResearchGate:
		in.ResearchGateURL = raw
	case FieldLinkedIn:
		in.LinkedInURL = raw
	case FieldWebOfScience:
		in.WebOfScienceURL = raw
	default:
		return "", false
	}
	links, problems := researchlinks.Parse(in)
	if len(problems) > 0 {
		return "", false
	}
	switch field {
	case FieldORCID:
		return links.ORCID, true
	case FieldScholar:
		return links.GoogleScholarURL, true
	case FieldScopus:
		return links.ScopusURL, true
	case FieldResearchGate:
		return links.ResearchGateURL, true
	case FieldLinkedIn:
		return links.LinkedInURL, true
	default:
		return links.WebOfScienceURL, true
	}
}

// RISLinkURL is the address stored in RIS for a canonical portal value (ORCID becomes a URL).
func RISLinkURL(field, value string) string {
	if field == FieldORCID && value != "" {
		return researchlinks.ORCIDURL(value)
	}
	return value
}
