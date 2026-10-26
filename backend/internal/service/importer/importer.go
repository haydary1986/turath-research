// Package importer parses bulk researcher spreadsheets (header mapping and row
// validation only — no database access, so it is unit-testable in isolation).
package importer

import (
	"regexp"
	"strings"
)

// MaxRows caps one import file; larger files must be split.
const MaxRows = 2000

// Required canonical columns.
var required = []string{"name_ar", "email"}

// headerAliases maps normalized header cells to canonical column keys.
// Normalization strips spaces, punctuation and Arabic diacritics variants.
var headerAliases = map[string]string{
	"الاسم": "name_ar", "الاسمالعربي": "name_ar", "الاسمبالعربية": "name_ar",
	"الاسمالثلاثي": "name_ar", "اسمالتدريسي": "name_ar",
	"namearabic": "name_ar", "namear": "name_ar", "arabicname": "name_ar",
	"الاسمالانكليزي": "name_en", "الاسمبالانكليزية": "name_en", "الاسمبالانجليزية": "name_en",
	"nameenglish": "name_en", "nameen": "name_en", "englishname": "name_en", "name": "name_en",
	"البريد": "email", "البريدالالكتروني": "email", "الايميل": "email",
	"الايميلالجامعي": "email", "البريدالجامعي": "email", "email": "email", "universityemail": "email",
	"اللقب": "title_ar", "اللقبالعلمي": "title_ar", "المرتبةالعلمية": "title_ar",
	"title": "title_ar", "academictitle": "title_ar", "rank": "title_ar",
	"الكلية": "college", "college": "college", "faculty": "college",
	"القسم": "department", "القسمالعلمي": "department", "الفرع": "department", "department": "department",
	"orcid": "orcid", "اوركيد": "orcid",
}

var (
	normalizePattern = regexp.MustCompile(`[\s\p{P}\p{S}]+`)
	// Pragmatic email shape check; the mail server is the real authority.
	emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// Arabic letter variants that differ between typists.
	arabicFold = strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي", "ئ", "ي", "ؤ", "و")
)

func normalizeHeader(cell string) string {
	cell = strings.ToLower(strings.TrimSpace(cell))
	cell = normalizePattern.ReplaceAllString(cell, "")
	return arabicFold.Replace(cell)
}

// normalizedAliases has the alias keys themselves normalized, so "الكلية" and "الكليه"
// (and other hamza/ta-marbuta variants) resolve identically.
var normalizedAliases = func() map[string]string {
	out := make(map[string]string, len(headerAliases))
	for alias, key := range headerAliases {
		out[normalizeHeader(alias)] = key
	}
	return out
}()

// MapHeader resolves each header cell to a canonical column and lists missing required columns.
func MapHeader(cells []string) (map[int]string, []string) {
	header := map[int]string{}
	seen := map[string]bool{}
	for i, cell := range cells {
		if key, ok := normalizedAliases[normalizeHeader(cell)]; ok && !seen[key] {
			header[i] = key
			seen[key] = true
		}
	}
	var missing []string
	for _, key := range required {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	return header, missing
}

// Row is one valid researcher line from the sheet.
type Row struct {
	Line       int    `json:"line"` // 1-based line in the sheet, for error reports
	NameAr     string `json:"name_ar"`
	NameEn     string `json:"name_en"`
	Email      string `json:"email"`
	TitleAr    string `json:"title_ar"`
	College    string `json:"college"`
	Department string `json:"department"`
	ORCID      string `json:"orcid"`
}

// Problem is one rejected line with a stable code the frontend translates.
type Problem struct {
	Line int    `json:"line"`
	Code string `json:"code"`
}

// ParseRows validates data rows. firstLine is the sheet line number of rows[0].
// Blank rows are skipped silently; duplicates (by email) keep the first occurrence.
func ParseRows(header map[int]string, rows [][]string, firstLine int) ([]Row, []Problem) {
	var valid []Row
	var problems []Problem
	seenEmail := map[string]bool{}

	for i, cells := range rows {
		line := firstLine + i
		if len(valid) >= MaxRows {
			problems = append(problems, Problem{Line: line, Code: "too_many_rows"})
			break
		}
		row := Row{Line: line}
		blank := true
		for col, key := range header {
			if col >= len(cells) {
				continue
			}
			value := strings.TrimSpace(cells[col])
			if value != "" {
				blank = false
			}
			switch key {
			case "name_ar":
				row.NameAr = value
			case "name_en":
				row.NameEn = value
			case "email":
				row.Email = strings.ToLower(value)
			case "title_ar":
				row.TitleAr = value
			case "college":
				row.College = value
			case "department":
				row.Department = value
			case "orcid":
				row.ORCID = value
			}
		}
		if blank {
			continue
		}
		switch {
		case row.NameAr == "":
			problems = append(problems, Problem{Line: line, Code: "name_required"})
		case row.Email == "" || !emailPattern.MatchString(row.Email):
			problems = append(problems, Problem{Line: line, Code: "invalid_email"})
		case seenEmail[row.Email]:
			problems = append(problems, Problem{Line: line, Code: "duplicate_email"})
		default:
			seenEmail[row.Email] = true
			valid = append(valid, row)
		}
	}
	return valid, problems
}
