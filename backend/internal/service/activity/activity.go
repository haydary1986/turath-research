// Package activity validates scientific-activity submissions — every record of the
// ministry form FS 003 (workshops, courses, books, agreements…) — and decides who may
// enter, edit and export them.
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
	maxShortText = 255
	maxPresenter = 500
	maxAttendees = 1_000_000
	maxDetail    = 500
	maxNotes     = 2000
	maxItemCount = 1_000_000
)

// Attendance modes the ministry report distinguishes.
var attendanceModes = map[string]struct{}{"in_person": {}, "online": {}, "hybrid": {}}

// Input is one activity as submitted by the organizer.
type Input struct {
	Type           string `json:"activity_type"`
	TitleAr        string `json:"title_ar"`
	Description    string `json:"description"`
	Presenter      string `json:"presenter"`
	ActivityDate   string `json:"activity_date"`
	Location       string `json:"location"`
	Attendees      int    `json:"attendees_count"`
	TargetAudience string `json:"target_audience"`
	Organizer      string `json:"organizer"`
	AttendanceMode string `json:"attendance_mode"`
	Category       string `json:"category"`
	CategoryDetail string `json:"category_detail"`
	ItemCount      *int   `json:"item_count"`
	Notes          string `json:"notes"`
}

// Normalize resolves the category and the legacy activity type from each other and
// fills the form column of venue categories from the location (and back). Clients that
// still send only activity_type get the category of that type.
func Normalize(in Input) Input {
	out := in
	out.Category = strings.TrimSpace(in.Category)
	out.CategoryDetail = strings.TrimSpace(in.CategoryDetail)
	out.Location = strings.TrimSpace(in.Location)
	switch {
	case out.Category != "":
		if IsCategory(out.Category) {
			out.Type = TypeForCategory(out.Category, in.Type)
		}
	case IsType(in.Type):
		out.Category = CategoryForType(in.Type)
	}
	if IsPlaceCategory(out.Category) {
		if out.CategoryDetail == "" {
			out.CategoryDetail = out.Location
		}
		if out.Location == "" {
			out.Location = out.CategoryDetail
		}
	}
	if out.Category != CategoryInfrastructure {
		out.ItemCount = nil
	}
	return out
}

// Validate returns one "field:code" entry per problem for a normalized input. hasCollege
// tells whether the submission resolves to a college: required for members' own work,
// optional for institutional records (no college means the whole university).
func Validate(in Input, hasCollege bool) []string {
	errs := validateCategory(in)
	if strings.TrimSpace(in.TitleAr) == "" {
		errs = append(errs, "title_ar:required")
	} else if utf8.RuneCountInString(in.TitleAr) > maxTitle {
		errs = append(errs, "title_ar:too_long")
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
	if !hasCollege && !IsInstitutional(in.Category) {
		errs = append(errs, "college:required")
	}
	if in.AttendanceMode != "" {
		if _, ok := attendanceModes[in.AttendanceMode]; !ok {
			errs = append(errs, "attendance_mode:invalid")
		}
	}
	for field, value := range map[string]string{
		"target_audience": in.TargetAudience,
		"organizer":       in.Organizer,
		"location":        in.Location,
	} {
		if utf8.RuneCountInString(value) > maxShortText {
			errs = append(errs, field+":too_long")
		}
	}
	if utf8.RuneCountInString(in.Presenter) > maxPresenter {
		errs = append(errs, "presenter:too_long")
	}
	if utf8.RuneCountInString(in.Notes) > maxNotes {
		errs = append(errs, "notes:too_long")
	}
	return errs
}

// validateCategory checks the category and its form column.
func validateCategory(in Input) []string {
	if in.Category == "" {
		// Normalize derives the category from any known type; an input that skipped it is still legal.
		if IsType(in.Type) {
			return nil
		}
		if strings.TrimSpace(in.Type) != "" {
			return []string{"activity_type:invalid"}
		}
		return []string{"category:required"}
	}
	if !IsCategory(in.Category) {
		return []string{"category:invalid"}
	}
	var errs []string
	if _, required := detailRequired[in.Category]; required && in.CategoryDetail == "" {
		errs = append(errs, "category_detail:required")
	}
	if utf8.RuneCountInString(in.CategoryDetail) > maxDetail {
		errs = append(errs, "category_detail:too_long")
	}
	if in.Category == CategoryInfrastructure {
		switch {
		case in.ItemCount == nil:
			errs = append(errs, "item_count:required")
		case *in.ItemCount < 1 || *in.ItemCount > maxItemCount:
			errs = append(errs, "item_count:invalid")
		}
	}
	return errs
}
