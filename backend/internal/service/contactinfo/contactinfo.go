// Package contactinfo validates the personal contact data held on a researcher. Every
// function returns "" and true for an empty value, which means "not provided".
package contactinfo

import (
	"regexp"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
)

const (
	maxEmailLength = 254

	// Validation codes, surfaced to the client as "field:code".
	CodeInvalidPhone = "invalid_phone"
	CodeInvalidEmail = "invalid_email"
	FieldPhone       = "phone"
	FieldSecondary   = "phone_secondary"
	FieldRecovery    = "recovery_email"

	StatusLinked      = "linked"
	StatusNeedsReview = "needs_review"
)

var (
	iraqiMobile = regexp.MustCompile(`^07[0-9]{9}$`)
	bareMobile  = regexp.MustCompile(`^7[0-9]{9}$`)
	// Stricter than a shape check: this address is shown as a mailto link.
	emailShape = regexp.MustCompile(`^[^@\s<>()\[\],;:"\\]+@[^@\s<>()\[\],;:"\\]+\.[^@\s<>()\[\],;:"\\.]+$`)
	employeeNo = regexp.MustCompile(`^[\p{L}\p{N}/_-]{1,50}$`)

	phoneSeparators = strings.NewReplacer(" ", "", "\u00a0", "", "-", "", "(", "", ")", "", ".", "")
	easternDigits   = strings.NewReplacer(
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	)
)

func phoneDigits(raw string) string {
	value := easternDigits.Replace(phoneSeparators.Replace(researchlinks.Clean(raw)))
	for _, prefix := range []string{"+964", "00964", "964"} {
		if rest, found := strings.CutPrefix(value, prefix); found {
			if strings.HasPrefix(rest, "0") {
				return rest
			}
			return "0" + rest
		}
	}
	return value
}

// ParsePhone returns the canonical 07XXXXXXXXX form of an Iraqi mobile number.
func ParsePhone(raw string) (string, bool) {
	value := phoneDigits(raw)
	if value == "" {
		return "", true
	}
	if !iraqiMobile.MatchString(value) {
		return "", false
	}
	return value, true
}

// ParseImportedPhone also accepts a number whose leading zero Excel dropped, which
// happens to every phone typed into a numeric cell.
func ParseImportedPhone(raw string) (string, bool) {
	if value := phoneDigits(raw); bareMobile.MatchString(value) {
		return "0" + value, true
	}
	return ParsePhone(raw)
}

// ParseEmail returns the address in lower case.
func ParseEmail(raw string) (string, bool) {
	value := strings.ToLower(researchlinks.Clean(raw))
	if value == "" {
		return "", true
	}
	if len(value) > maxEmailLength || !emailShape.MatchString(value) {
		return "", false
	}
	return value, true
}

// ParseEmployeeNo accepts letters, digits and the separators used in staff numbers.
func ParseEmployeeNo(raw string) (string, bool) {
	value := easternDigits.Replace(researchlinks.Clean(raw))
	if value == "" {
		return "", true
	}
	if !employeeNo.MatchString(value) {
		return "", false
	}
	return value, true
}

// ParseAccountStatus accepts the two states an account can be in.
func ParseAccountStatus(raw string) (string, bool) {
	value := strings.ToLower(researchlinks.Clean(raw))
	switch value {
	case "", StatusLinked, StatusNeedsReview:
		return value, true
	}
	return "", false
}
