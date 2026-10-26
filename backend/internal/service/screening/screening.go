// Package screening normalizes journal identifiers so the accredited/predatory
// watchlist can be matched reliably against free-text user input.
package screening

import (
	"regexp"
	"strings"
)

// Classifications of a watched journal.
const (
	ClassAccredited = "accredited" // معتمدة محليًا
	ClassPredatory  = "predatory"  // مفترسة
	ClassSuspended  = "suspended"  // موقوف النشر فيها
)

var classifications = map[string]struct{}{
	ClassAccredited: {}, ClassPredatory: {}, ClassSuspended: {},
}

// IsClassification reports whether c is a known classification key.
func IsClassification(c string) bool {
	_, ok := classifications[c]
	return ok
}

var issnStrip = regexp.MustCompile(`[^0-9Xx]`)

// NormalizeISSN reduces any ISSN spelling to the canonical NNNN-NNNC form,
// or returns "" when the input cannot be an ISSN.
func NormalizeISSN(raw string) string {
	s := strings.ToUpper(issnStrip.ReplaceAllString(raw, ""))
	if len(s) != 8 {
		return ""
	}
	for _, r := range s[:7] {
		if r < '0' || r > '9' {
			return ""
		}
	}
	last := s[7]
	if (last < '0' || last > '9') && last != 'X' {
		return ""
	}
	return s[:4] + "-" + s[4:]
}

var (
	spaceCollapse = regexp.MustCompile(`\s+`)
	namePunct     = regexp.MustCompile(`[\p{P}\p{S}]+`)
	arabicFold    = strings.NewReplacer(
		"أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي", "ئ", "ي", "ؤ", "و",
		"ً", "", "ٌ", "", "ٍ", "", "َ", "", "ُ", "", "ِ", "", "ّ", "", "ْ", "",
	)
)

// NameKey folds a journal name to a comparison key: lowercase, punctuation-free,
// single-spaced, with common Arabic letter variants unified.
func NameKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = arabicFold.Replace(s)
	s = namePunct.ReplaceAllString(s, " ")
	s = spaceCollapse.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
