// Package publicsite validates the editable public face of the scientific-affairs
// portal: the landing-page content and the announcements (activities and circulars).
// Everything is stored as plain text; the frontend renders it without raw HTML.
package publicsite

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Validation codes, surfaced to the client as "field:code".
const (
	CodeRequired      = "required"
	CodeTooLong       = "too_long"
	CodeTooMany       = "too_many"
	CodeInvalid       = "invalid"
	CodeInvalidEmail  = "invalid_email"
	CodeInvalidDate   = "invalid_date"
	CodeHTTPSRequired = "https_required"
)

// cleanLine trims a single-line value and drops every control character.
func cleanLine(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s))
}

// cleanText normalises a multi-line value: CRLF/CR become LF, other control
// characters are dropped, trailing spaces are trimmed per line, and runs of more
// than one blank line collapse to a single paragraph break.
func cleanText(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func tooLong(s string, limit int) bool { return utf8.RuneCountInString(s) > limit }

// problems collects unique "field:code" entries in order.
type problems struct {
	list []string
	seen map[string]struct{}
}

func (p *problems) add(field, code string) {
	entry := field + ":" + code
	if p.seen == nil {
		p.seen = map[string]struct{}{}
	}
	if _, dup := p.seen[entry]; dup {
		return
	}
	p.seen[entry] = struct{}{}
	p.list = append(p.list, entry)
}

// text checks a cleaned value against required/length rules.
func (p *problems) text(field, value string, required bool, limit int) {
	switch {
	case required && value == "":
		p.add(field, CodeRequired)
	case tooLong(value, limit):
		p.add(field, CodeTooLong)
	}
}
