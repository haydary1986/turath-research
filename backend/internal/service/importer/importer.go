// Package importer parses bulk researcher spreadsheets (header mapping and row
// validation only — no database access, so it is unit-testable in isolation).
package importer

import (
	"regexp"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/service/contactinfo"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
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
	"titlear": "title_ar",
	"phone":   "phone", "mobile": "phone", "الهاتف": "phone", "رقمالهاتف": "phone", "الموبايل": "phone",
	"رقمالموبايل": "phone", "الجوال": "phone",
	"phonesecondary": "phone_secondary", "secondaryphone": "phone_secondary", "الهاتفالثانوي": "phone_secondary",
	"الهاتفالثاني": "phone_secondary", "رقمالهاتفالثاني": "phone_secondary",
	"recoveryemail": "recovery_email", "البريدالاحتياطي": "recovery_email", "بريدالاسترداد": "recovery_email",
	"googlescholarurl": "google_scholar_url", "googlescholar": "google_scholar_url", "كوكلسكولار": "google_scholar_url",
	"scopusurl": "scopus_url", "scopus": "scopus_url", "سكوبس": "scopus_url",
	"researchgateurl": "researchgate_url", "researchgate": "researchgate_url",
	"linkedinurl": "linkedin_url", "linkedin": "linkedin_url",
	"webofscienceurl": "web_of_science_url", "webofscience": "web_of_science_url",
	"employeeno": "employee_no", "employeenumber": "employee_no", "الرقمالوظيفي": "employee_no",
	"accountstatus": "account_status", "حالةالحساب": "account_status",
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

	// Validated, canonical links. Empty never erases a stored value.
	Links researchlinks.Links `json:"-"`
	// Contact data stays out of the preview, which needs none of it.
	Phone          string `json:"-"`
	PhoneSecondary string `json:"-"`
	RecoveryEmail  string `json:"-"`
	EmployeeNo     string `json:"-"`
	AccountStatus  string `json:"-"`

	linkInput researchlinks.Input
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
			row.set(key, value)
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
			normalized, codes := row.normalizeOptional()
			if len(codes) > 0 {
				for _, code := range codes {
					problems = append(problems, Problem{Line: line, Code: code})
				}
				continue
			}
			row = normalized
			seenEmail[row.Email] = true
			valid = append(valid, row)
		}
	}
	return valid, problems
}

func (r *Row) set(key, value string) {
	switch key {
	case "name_ar":
		r.NameAr = value
	case "name_en":
		r.NameEn = value
	case "email":
		r.Email = strings.ToLower(value)
	case "title_ar":
		r.TitleAr = value
	case "college":
		r.College = value
	case "department":
		r.Department = value
	case "orcid":
		r.linkInput.ORCID = value
	case "google_scholar_url":
		r.linkInput.GoogleScholarURL = value
	case "scopus_url":
		r.linkInput.ScopusURL = value
	case "researchgate_url":
		r.linkInput.ResearchGateURL = value
	case "linkedin_url":
		r.linkInput.LinkedInURL = value
	case "web_of_science_url":
		r.linkInput.WebOfScienceURL = value
	case "phone":
		r.Phone = value
	case "phone_secondary":
		r.PhoneSecondary = value
	case "recovery_email":
		r.RecoveryEmail = value
	case "employee_no":
		r.EmployeeNo = value
	case "account_status":
		r.AccountStatus = value
	}
}

// normalizeOptional returns the row with its optional cells in canonical form, or one
// code per bad cell. A bad optional cell rejects the row, so the admin sees it.
func (r Row) normalizeOptional() (Row, []string) {
	var codes []string
	links, problems := researchlinks.Parse(r.linkInput)
	for _, problem := range problems {
		field, _, _ := strings.Cut(problem, ":")
		codes = append(codes, "invalid_"+field)
	}
	r.Links, r.ORCID = links, links.ORCID

	var ok bool
	if r.Phone, ok = contactinfo.ParseImportedPhone(r.Phone); !ok {
		codes = append(codes, "invalid_phone")
	}
	if r.PhoneSecondary, ok = contactinfo.ParseImportedPhone(r.PhoneSecondary); !ok {
		codes = append(codes, "invalid_phone_secondary")
	}
	if r.RecoveryEmail, ok = contactinfo.ParseEmail(r.RecoveryEmail); !ok {
		codes = append(codes, "invalid_recovery_email")
	}
	if r.EmployeeNo, ok = contactinfo.ParseEmployeeNo(r.EmployeeNo); !ok {
		codes = append(codes, "invalid_employee_no")
	}
	if r.AccountStatus, ok = contactinfo.ParseAccountStatus(r.AccountStatus); !ok {
		codes = append(codes, "invalid_account_status")
	}
	return r, codes
}
