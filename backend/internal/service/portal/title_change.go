package portal

import (
	"strconv"
	"unicode/utf8"
)

// KindTitleChange asks scientific affairs to change the title of a registered research.
// It is a short chain: the applicant submits, scientific affairs approves, rejects or returns.
const KindTitleChange = "title_change"

const (
	// MaxCoResearchers is the most partners a joint research plan may name.
	MaxCoResearchers     = 3
	maxCoResearcherRunes = 255
	maxReasonRunes       = 2000
	maxOldTitleRunes     = 500
	coResearcherPrefix   = "co_researcher_"
)

// FirstStage is the status a request enters when submitted.
func FirstStage(kind string) string {
	if kind == KindTitleChange {
		return StatusPending
	}
	return StatusPendingDean
}

// CoResearcherKey names the detail field holding the n-th (1-based) co-researcher.
func CoResearcherKey(n int) string {
	return coResearcherPrefix + strconv.Itoa(n)
}

// CoResearchers returns the non-blank co-researcher names in order.
func CoResearchers(d map[string]any) []string {
	names := []string{}
	for n := 1; n <= MaxCoResearchers; n++ {
		if name := str(d, CoResearcherKey(n)); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// validateCoResearchers: a joint plan names one to three partners; an individual plan names none.
func validateCoResearchers(d map[string]any) []string {
	var errs []string
	for n := MaxCoResearchers + 1; n <= MaxCoResearchers+10; n++ {
		if str(d, CoResearcherKey(n)) != "" {
			errs = append(errs, "co_researchers:too_many")
			break
		}
	}
	for n := 1; n <= MaxCoResearchers; n++ {
		if utf8.RuneCountInString(str(d, CoResearcherKey(n))) > maxCoResearcherRunes {
			errs = append(errs, CoResearcherKey(n)+":too_long")
		}
	}
	count := len(CoResearchers(d))
	switch str(d, "work_type") {
	case "joint":
		if count == 0 {
			errs = append(errs, "co_researchers:required")
		}
	case "individual":
		if count > 0 {
			errs = append(errs, "co_researchers:not_allowed")
		}
	}
	return errs
}

// validateTitleChange: the previous title and the reason are mandatory; the new title is the request title.
func validateTitleChange(d map[string]any) []string {
	var errs []string
	oldTitle := str(d, "old_title")
	if oldTitle == "" {
		errs = append(errs, "old_title:required")
	} else if utf8.RuneCountInString(oldTitle) > maxOldTitleRunes {
		errs = append(errs, "old_title:too_long")
	}
	reason := str(d, "reason")
	if reason == "" {
		errs = append(errs, "reason:required")
	} else if utf8.RuneCountInString(reason) > maxReasonRunes {
		errs = append(errs, "reason:too_long")
	}
	return errs
}
