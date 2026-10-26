// Package activity validates scientific-activity submissions (seminars,
// workshops, panels…) before they reach the database.
package activity

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Types of scientific activities a college reports.
var Types = []string{"seminar", "workshop", "panel", "lecture", "course", "conference", "other"}

var typeSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(Types))
	for _, t := range Types {
		m[t] = struct{}{}
	}
	return m
}()

// IsType reports whether t is a known activity type.
func IsType(t string) bool {
	_, ok := typeSet[t]
	return ok
}

// Statuses of an activity record.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusReturned = "returned"
	StatusRejected = "rejected"
)

const (
	maxTitle     = 500
	maxAttendees = 1_000_000
)

// Input is one activity as submitted by the organizer.
type Input struct {
	Type         string `json:"activity_type"`
	TitleAr      string `json:"title_ar"`
	Description  string `json:"description"`
	Presenter    string `json:"presenter"`
	ActivityDate string `json:"activity_date"`
	Location     string `json:"location"`
	Attendees    int    `json:"attendees_count"`
}

// Validate returns one "field:code" entry per problem. hasCollege tells whether
// the submission resolves to a college (required for ministry reporting).
func Validate(in Input, hasCollege bool) []string {
	var errs []string
	if strings.TrimSpace(in.TitleAr) == "" {
		errs = append(errs, "title_ar:required")
	} else if utf8.RuneCountInString(in.TitleAr) > maxTitle {
		errs = append(errs, "title_ar:too_long")
	}
	if !IsType(in.Type) {
		errs = append(errs, "activity_type:invalid")
	}
	switch {
	case strings.TrimSpace(in.ActivityDate) == "":
		errs = append(errs, "activity_date:required")
	default:
		if _, err := time.Parse("2006-01-02", in.ActivityDate); err != nil {
			errs = append(errs, "activity_date:invalid_date")
		}
	}
	if in.Attendees < 0 || in.Attendees > maxAttendees {
		errs = append(errs, "attendees_count:invalid")
	}
	if !hasCollege {
		errs = append(errs, "college:required")
	}
	return errs
}
