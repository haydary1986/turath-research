package publicsite

import (
	"strings"
	"time"
)

// Announcement types.
const (
	TypeActivity = "activity"
	TypeCircular = "circular"
)

// Limits on an announcement.
const (
	MaxTitle       = 300
	MaxBody        = 20000
	MaxReferenceNo = 100
)

// Languages an announcement can be written in; Arabic is the default.
const (
	LangArabic  = "ar"
	LangEnglish = "en"
)

// IsLang reports whether l is a supported announcement language.
func IsLang(l string) bool { return l == LangArabic || l == LangEnglish }

// IsType reports whether t is a known announcement type.
func IsType(t string) bool { return t == TypeActivity || t == TypeCircular }

// AnnouncementInput is an announcement as edited by staff.
type AnnouncementInput struct {
	Type        string `json:"type"`
	Lang        string `json:"lang"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	EventDate   string `json:"event_date"`
	ReferenceNo string `json:"reference_no"`
}

// NormalizeAnnouncement returns a cleaned copy of in. An event date belongs to
// activities and a circular number to circulars, so the other one is dropped.
func NormalizeAnnouncement(in AnnouncementInput) AnnouncementInput {
	out := AnnouncementInput{
		Type:        cleanLine(in.Type),
		Lang:        strings.ToLower(cleanLine(in.Lang)),
		Title:       cleanLine(in.Title),
		Body:        cleanText(in.Body),
		EventDate:   cleanLine(in.EventDate),
		ReferenceNo: cleanLine(in.ReferenceNo),
	}
	if out.Lang == "" {
		out.Lang = LangArabic
	}
	switch out.Type {
	case TypeActivity:
		out.ReferenceNo = ""
	case TypeCircular:
		out.EventDate = ""
	}
	return out
}

// ValidateAnnouncement returns one "field:code" entry per problem in a normalised input.
func ValidateAnnouncement(in AnnouncementInput) []string {
	var p problems
	if !IsType(in.Type) {
		p.add("type", CodeInvalid)
	}
	if !IsLang(in.Lang) {
		p.add("lang", CodeInvalid)
	}
	p.text("title", in.Title, true, MaxTitle)
	p.text("body", in.Body, false, MaxBody)
	p.text("reference_no", in.ReferenceNo, false, MaxReferenceNo)
	if in.EventDate != "" {
		if _, err := time.Parse("2006-01-02", in.EventDate); err != nil {
			p.add("event_date", CodeInvalidDate)
		}
	}
	return p.list
}
