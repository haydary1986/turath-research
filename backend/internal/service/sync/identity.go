package sync

import (
	"regexp"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
	"github.com/haydary1986/turath-research/backend/internal/service/namematch"
)

// fetchResult holds only what the identity checks allowed through.
type fetchResult struct {
	works       []importedWork
	author      *apiclient.OpenAlexPerson
	orcidErr    error
	openalexErr error
	// truncated: OpenAlex holds more works than were read.
	truncated bool
	// identityCode names a failed name check, for staff to review.
	identityCode string
	// identityUnknown: the ORCID record could not be read, so nothing could be checked.
	identityUnknown bool
}

// completeAnswer reports whether works is everything the trusted sources hold, the
// only case in which absence from the list means a link should be withdrawn.
func (f fetchResult) completeAnswer() bool {
	return f.orcidErr == nil && f.openalexErr == nil && !f.truncated && !f.identityUnknown &&
		f.identityCode != CodeORCIDNameMismatch && f.identityCode != CodeORCIDNameUnverified
}

// orcidIdentityCode returns "" when the record carries the researcher's name.
func orcidIdentityCode(researcherNames, orcidNames []string) string {
	if len(orcidNames) == 0 {
		return CodeORCIDNameUnverified // the owner hid the name, so nothing ties it to the researcher
	}
	if !namematch.MatchAny(researcherNames, orcidNames) {
		return CodeORCIDNameMismatch
	}
	return ""
}

// universityPattern recognises the university on an ORCID employment, by name or ROR id.
var universityPattern = regexp.MustCompile(`(?i)turath|التراث|019vd4365`)

// listsUniversity reports whether the ORCID record names the university as an employer.
func listsUniversity(employments *apiclient.ORCIDEmploymentGroup) bool {
	if employments == nil {
		return false
	}
	for _, group := range employments.AffiliationGroup {
		for _, summary := range group.Summaries {
			org := summary.Employment.Organization
			if universityPattern.MatchString(org.Name) {
				return true
			}
			if org.Identifier != nil && universityPattern.MatchString(org.Identifier.Value) {
				return true
			}
		}
	}
	return false
}

// openAlexTrusted compares only the display name. OpenAlex's alternatives are
// unusable as evidence: a merged author lists hundreds of other people's names.
func openAlexTrusted(orcidNames []string, author *apiclient.OpenAlexPerson) bool {
	return namematch.MatchAny(orcidNames, []string{author.DisplayName})
}

// outcomeFor summarizes a run. A refused or unreadable identity outranks every other code.
func outcomeFor(f fetchResult, failedRows int) syncOutcome {
	switch {
	case f.identityUnknown:
		if apiclient.IsNotFound(f.orcidErr) {
			return syncOutcome{Code: codeORCIDNotFound}
		}
		return syncOutcome{Code: codeORCIDUnavailable}
	case f.identityCode == CodeOpenAlexNameMismatch:
		return syncOutcome{Code: f.identityCode, Succeeded: f.orcidErr == nil}
	case f.identityCode != "":
		return syncOutcome{Code: f.identityCode}
	}
	outcome := outcomeOf(f.orcidErr, f.openalexErr, failedRows)
	if outcome.Code == "" && f.truncated {
		outcome.Code = codeOpenAlexTruncated
	}
	return outcome
}

// IsIdentityCode reports whether a stored sync code asks for staff review.
func IsIdentityCode(code string) bool {
	return code == CodeORCIDNameMismatch || code == CodeORCIDNameUnverified || code == CodeOpenAlexNameMismatch
}
