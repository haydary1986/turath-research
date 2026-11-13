package portal

import (
	"fmt"
	"regexp"
	"strings"
)

// LetterPlaceholders are the variables an appreciation-letter template may use.
var LetterPlaceholders = []string{
	"researcher_name", "college", "department",
	"research_title", "journal_name", "index_db", "quartile", "publication_date",
}

var placeholderPattern = regexp.MustCompile(`\{\{\s*([a-z_]+)\s*\}\}`)

// RenderBody substitutes {{name}} placeholders in one pass.
// Unknown placeholders are kept as-is and substituted values are never re-expanded.
func RenderBody(template string, vars map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return match
	})
}

// UnknownPlaceholders returns the distinct placeholder names in the text that are not supported.
func UnknownPlaceholders(text string) []string {
	known := make(map[string]struct{}, len(LetterPlaceholders))
	for _, p := range LetterPlaceholders {
		known[p] = struct{}{}
	}
	seen := map[string]struct{}{}
	var unknown []string
	for _, m := range placeholderPattern.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if _, ok := known[name]; ok {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		unknown = append(unknown, name)
	}
	return unknown
}

// FormatLetterNo builds the official letter number, e.g. 1245/2026.
func FormatLetterNo(seq, year int) string {
	return fmt.Sprintf("%d/%d", seq, year)
}

// IndexDBLabel returns the Arabic label of an indexing database key.
func IndexDBLabel(key string) string {
	switch strings.ToLower(key) {
	case "scopus":
		return "Scopus"
	case "wos":
		return "Web of Science"
	case "local":
		return "محلي"
	case "international_unindexed":
		return "عالمية غير مستوعبة"
	default:
		return key
	}
}
