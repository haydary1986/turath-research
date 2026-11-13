package publicsite

import (
	"slices"
	"strings"
	"testing"
)

func validContent() Content {
	return Content{
		HeroTitle:    "قسم الشؤون العلمية",
		HeroSubtitle: "جامعة التراث",
		About:        "نبذة",
		Services:     []Service{{Title: "تدقيق المجلات", Description: "وصف"}},
		Contact:      Contact{Email: "sa@uoturath.edu.iq", Phone: "+964 770 000 0000"},
		Links:        []Link{{Label: "الموقع الرسمي", URL: "https://uoturath.edu.iq"}},
	}
}

func TestValidContentPasses(t *testing.T) {
	if got := ValidateContent(NormalizeContent(validContent())); len(got) != 0 {
		t.Fatalf("unexpected problems: %v", got)
	}
}

func TestEmptyContactIsAllowed(t *testing.T) {
	c := validContent()
	c.Contact = Contact{}
	if got := ValidateContent(NormalizeContent(c)); len(got) != 0 {
		t.Fatalf("unexpected problems: %v", got)
	}
}

func TestContentProblems(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Content)
		want   string
	}{
		{"missing hero title", func(c *Content) { c.HeroTitle = "   " }, "hero_title:required"},
		{"long hero title", func(c *Content) { c.HeroTitle = strings.Repeat("ع", MaxHeroTitle+1) }, "hero_title:too_long"},
		{"long about", func(c *Content) { c.About = strings.Repeat("a", MaxAbout+1) }, "about:too_long"},
		{"service without title", func(c *Content) { c.Services = []Service{{Description: "x"}} }, "service_title:required"},
		{"too many services", func(c *Content) {
			c.Services = slices.Repeat([]Service{{Title: "x"}}, MaxServices+1)
		}, "services:too_many"},
		{"too many links", func(c *Content) {
			c.Links = slices.Repeat([]Link{{Label: "x", URL: "https://a.iq"}}, MaxLinks+1)
		}, "links:too_many"},
		{"http link", func(c *Content) { c.Links = []Link{{Label: "x", URL: "http://a.iq"}} }, "link_url:https_required"},
		{"javascript link", func(c *Content) { c.Links = []Link{{Label: "x", URL: "javascript:alert(1)"}} }, "link_url:https_required"},
		{"credentials in link", func(c *Content) { c.Links = []Link{{Label: "x", URL: "https://u:p@a.iq"}} }, "link_url:https_required"},
		{"link without host", func(c *Content) { c.Links = []Link{{Label: "x", URL: "https:///path"}} }, "link_url:https_required"},
		{"link without label", func(c *Content) { c.Links = []Link{{URL: "https://a.iq"}} }, "link_label:required"},
		{"link without url", func(c *Content) { c.Links = []Link{{Label: "x"}} }, "link_url:required"},
		{"bad email", func(c *Content) { c.Contact.Email = "not-an-email" }, "contact_email:invalid_email"},
		{"email with markup", func(c *Content) { c.Contact.Email = "a<b>@x.iq" }, "contact_email:invalid_email"},
		{"bad phone", func(c *Content) { c.Contact.Phone = "call me" }, "contact_phone:invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validContent()
			tc.mutate(&c)
			got := ValidateContent(NormalizeContent(c))
			if !slices.Contains(got, tc.want) {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}

func TestProblemsAreReportedOnce(t *testing.T) {
	c := validContent()
	c.Services = []Service{{}, {}, {}}
	got := ValidateContent(NormalizeContent(c))
	if len(got) != 1 || got[0] != "service_title:required" {
		t.Fatalf("got %v", got)
	}
}

func TestNormalizeContentDoesNotMutate(t *testing.T) {
	c := validContent()
	c.HeroTitle = "  عنوان\x00  "
	c.Contact.Email = " SA@UOTURATH.EDU.IQ "
	out := NormalizeContent(c)
	if out.HeroTitle != "عنوان" || out.Contact.Email != "sa@uoturath.edu.iq" {
		t.Fatalf("normalised = %+v", out)
	}
	if c.HeroTitle != "  عنوان\x00  " {
		t.Fatal("input was modified")
	}
}

func TestCleanTextKeepsParagraphs(t *testing.T) {
	in := "سطر أول  \r\nسطر ثان\r\n\r\n\r\n\r\nفقرة\x07 ثانية\n"
	want := "سطر أول\nسطر ثان\n\nفقرة ثانية"
	if got := cleanText(in); got != want {
		t.Fatalf("cleanText = %q, want %q", got, want)
	}
}

func TestNormalizeContentNeverReturnsNilLists(t *testing.T) {
	out := NormalizeContent(Content{HeroTitle: "x"})
	if out.Services == nil || out.Links == nil {
		t.Fatal("lists must encode as [] not null")
	}
}

func TestAnnouncementValidation(t *testing.T) {
	cases := []struct {
		name string
		in   AnnouncementInput
		want []string
	}{
		{"valid circular", AnnouncementInput{Type: TypeCircular, Title: "تعميم", ReferenceNo: "12/2026"}, nil},
		{"valid activity", AnnouncementInput{Type: TypeActivity, Title: "ندوة", EventDate: "2026-11-20"}, nil},
		{"unknown type", AnnouncementInput{Type: "news", Title: "x"}, []string{"type:invalid"}},
		{"missing title", AnnouncementInput{Type: TypeCircular}, []string{"title:required"}},
		{"long title", AnnouncementInput{Type: TypeCircular, Title: strings.Repeat("x", MaxTitle+1)}, []string{"title:too_long"}},
		{"long body", AnnouncementInput{Type: TypeCircular, Title: "x", Body: strings.Repeat("x", MaxBody+1)}, []string{"body:too_long"}},
		{"bad date", AnnouncementInput{Type: TypeActivity, Title: "x", EventDate: "20/11/2026"}, []string{"event_date:invalid_date"}},
		{"long reference", AnnouncementInput{Type: TypeCircular, Title: "x", ReferenceNo: strings.Repeat("1", MaxReferenceNo+1)}, []string{"reference_no:too_long"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateAnnouncement(NormalizeAnnouncement(tc.in))
			if !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeAnnouncementDropsFieldsOfTheOtherType(t *testing.T) {
	circular := NormalizeAnnouncement(AnnouncementInput{Type: TypeCircular, Title: "x", EventDate: "bad", ReferenceNo: " 5 "})
	if circular.EventDate != "" || circular.ReferenceNo != "5" {
		t.Fatalf("circular = %+v", circular)
	}
	activity := NormalizeAnnouncement(AnnouncementInput{Type: TypeActivity, Title: "x", EventDate: "2026-01-02", ReferenceNo: "5"})
	if activity.ReferenceNo != "" || activity.EventDate != "2026-01-02" {
		t.Fatalf("activity = %+v", activity)
	}
}

func TestIsSafeHTTPSURL(t *testing.T) {
	good := []string{"https://uoturath.edu.iq", "https://scholar.google.com/x?y=1"}
	bad := []string{"", "http://a.iq", "//a.iq", "https://", "https://a.iq/\"onmouseover", "data:text/html,x", "ftp://a.iq"}
	for _, u := range good {
		if !IsSafeHTTPSURL(u) {
			t.Errorf("%q rejected", u)
		}
	}
	for _, u := range bad {
		if IsSafeHTTPSURL(u) {
			t.Errorf("%q accepted", u)
		}
	}
}
