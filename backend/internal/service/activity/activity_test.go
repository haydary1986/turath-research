package activity

import (
	"strings"
	"testing"
)

func valid() Input {
	return Input{
		Type:         "workshop",
		TitleAr:      "ورشة الذكاء الاصطناعي في التعليم",
		Presenter:    "د. أحمد محمد",
		ActivityDate: "2026-11-02",
		Location:     "قاعة المؤتمرات",
		Attendees:    40,
	}
}

func TestValidateAccepts(t *testing.T) {
	if errs := Validate(valid(), true); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Input)
		college bool
		want    string
	}{
		{"missing title", func(i *Input) { i.TitleAr = " " }, true, "title_ar:required"},
		{"unknown type", func(i *Input) { i.Type = "party" }, true, "activity_type:invalid"},
		{"bad date", func(i *Input) { i.ActivityDate = "02/11/2026" }, true, "activity_date:invalid_date"},
		{"missing date", func(i *Input) { i.ActivityDate = "" }, true, "activity_date:required"},
		{"negative attendees", func(i *Input) { i.Attendees = -3 }, true, "attendees_count:invalid"},
		{"huge attendees", func(i *Input) { i.Attendees = 2000000 }, true, "attendees_count:invalid"},
		{"no college", func(*Input) {}, false, "college:required"},
		{"long title", func(i *Input) { i.TitleAr = strings.Repeat("ع", 501) }, true, "title_ar:too_long"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := valid()
			c.mutate(&in)
			errs := Validate(in, c.college)
			found := false
			for _, e := range errs {
				if e == c.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("want %q in %v", c.want, errs)
			}
		})
	}
}

func TestTypes(t *testing.T) {
	if len(Types) == 0 || !IsType("seminar") || IsType("x") {
		t.Fatal("type registry broken")
	}
}
