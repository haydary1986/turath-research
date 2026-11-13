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

var (
	issnStrip   = regexp.MustCompile(`[^0-9Xx]`)
	localDigits = strings.NewReplacer(
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	)
)

// LatinDigits rewrites Arabic-Indic and Persian digits as ASCII, the form Arabic
// keyboards do not type but every identifier comparison expects.
func LatinDigits(s string) string {
	return localDigits.Replace(s)
}

// NormalizeISSN reduces any ISSN spelling to the canonical NNNN-NNNC form,
// or returns "" when the input cannot be an ISSN.
func NormalizeISSN(raw string) string {
	s := strings.ToUpper(issnStrip.ReplaceAllString(LatinDigits(raw), ""))
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

// ValidISSNChecksum verifies the check digit of a NormalizeISSN result.
func ValidISSNChecksum(issn string) bool {
	if len(issn) != 9 || issn[4] != '-' {
		return false
	}
	digits := issn[:4] + issn[5:]
	sum := 0
	for i := 0; i < 7; i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
		sum += int(digits[i]-'0') * (8 - i)
	}
	check := (11 - sum%11) % 11
	if check == 10 {
		return digits[7] == 'X'
	}
	return digits[7] == byte('0'+check)
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
