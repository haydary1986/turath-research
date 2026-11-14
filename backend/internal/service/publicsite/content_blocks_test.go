package publicsite

import (
	"slices"
	"strings"
	"testing"
)

func TestBlockProblems(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Content)
		want   string
	}{
		{"unknown icon", func(c *Content) { c.Services = []Service{{Title: "x", Icon: "rocket"}} }, "service_icon:invalid"},
		{"external service link", func(c *Content) { c.Services = []Service{{Title: "x", Link: "https://a.iq"}} }, "service_link:invalid"},
		{"protocol-relative link", func(c *Content) { c.Services = []Service{{Title: "x", Link: "//evil.iq/portal"}} }, "service_link:invalid"},
		{"link outside allowed roots", func(c *Content) { c.Services = []Service{{Title: "x", Link: "/admin"}} }, "service_link:invalid"},
		{"traversal link", func(c *Content) { c.Services = []Service{{Title: "x", Link: "/portal/../admin"}} }, "service_link:invalid"},
		{"long stat value", func(c *Content) {
			c.Stats = []StatSetting{{Key: StatAcademics, Value: strings.Repeat("9", MaxStatValue+1)}}
		}, "stat_value:too_long"},
		{"long director message", func(c *Content) { c.Director.Message = strings.Repeat("x", MaxDirectorMessage+1) }, "director_message:too_long"},
		{"http college site", func(c *Content) { c.Colleges = []Link{{Label: "x", URL: "http://a.iq"}} }, "college_url:https_required"},
		{"college without name", func(c *Content) { c.Colleges = []Link{{URL: "https://a.iq"}} }, "college_label:required"},
		{"unknown social", func(c *Content) { c.Socials = []Social{{Platform: "myspace", URL: "https://a.iq"}} }, "social_platform:invalid"},
		{"javascript social", func(c *Content) { c.Socials = []Social{{Platform: "x", URL: "javascript:alert(1)"}} }, "social_url:https_required"},
		{"too many socials", func(c *Content) {
			c.Socials = slices.Repeat([]Social{{Platform: "x", URL: "https://x.com/a"}}, MaxSocials+1)
		}, "socials:too_many"},
		{"faq without answer", func(c *Content) { c.FAQ = []FAQItem{{Question: "q"}} }, "faq_answer:required"},
		{"too many faq", func(c *Content) { c.FAQ = slices.Repeat([]FAQItem{{Question: "q", Answer: "a"}}, MaxFAQ+1) }, "faq:too_many"},
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

func TestValidBlocksPass(t *testing.T) {
	c := validContent()
	c.Services = []Service{
		{Title: "a", Icon: "research", Link: "/portal/new/research_plan"},
		{Title: "b", Icon: "shield", Link: "/journals/predatory"},
		{Title: "c", Link: "/news"},
	}
	c.Stats = []StatSetting{{Key: StatIndexed, Label: "بحوث مفهرسة", Value: "+120"}}
	c.Director = Director{Title: "كلمة", Name: "د. فلان", Message: "نص"}
	c.Colleges = []Link{{Label: "كلية الهندسة", URL: "https://tucoe.uoturath.edu.iq"}}
	c.Socials = []Social{{Platform: "Facebook", URL: "https://facebook.com/uoturath"}}
	c.FAQ = []FAQItem{{Question: "س", Answer: "ج"}}
	if got := ValidateContent(NormalizeContent(c)); len(got) != 0 {
		t.Fatalf("unexpected problems: %v", got)
	}
}

func TestNormalizeStatsKeepsEveryKeyOnce(t *testing.T) {
	in := []StatSetting{
		{Key: StatColleges, Hidden: true},
		{Key: "unknown", Value: "1"},
		{Key: StatColleges, Value: "dup"},
		{Key: StatAcademics, Value: " 250+ "},
	}
	out := normalizeStats(in)
	keys := make([]string, 0, len(out))
	for _, s := range out {
		keys = append(keys, s.Key)
	}
	want := []string{StatColleges, StatAcademics, StatPublications, StatIndexed, StatActivities}
	if !slices.Equal(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if !out[0].Hidden || out[0].Value != "" || out[1].Value != "250+" {
		t.Fatalf("settings = %+v", out)
	}
}

func TestNormalizeDropsMalformedPhotoID(t *testing.T) {
	out := NormalizeContent(Content{Director: Director{PhotoID: "../../etc/passwd"}})
	if out.Director.PhotoID != "" {
		t.Fatalf("photo id = %q", out.Director.PhotoID)
	}
	ok := NormalizeContent(Content{Director: Director{PhotoID: "6f1c1c1e-1111-4a4a-8b8b-123456789abc"}})
	if ok.Director.PhotoID == "" {
		t.Fatal("a valid photo id was dropped")
	}
}

func TestNormalizeNeverReturnsNilBlockLists(t *testing.T) {
	out := NormalizeContent(Content{})
	if out.Colleges == nil || out.Socials == nil || out.FAQ == nil || len(out.Stats) != len(StatKeys) {
		t.Fatalf("normalised = %+v", out)
	}
}

func TestEnglishFieldsAreOptionalButBounded(t *testing.T) {
	c := validContent()
	c.HeroTitleEn = "Scientific Affairs Department"
	c.Services = []Service{{Title: "خدمة", TitleEn: "Service", DescriptionEn: "  An English description  "}}
	c.Director = Director{Name: "د. فلان", NameEn: "Dr. Example", MessageEn: "Hello"}
	c.FAQ = []FAQItem{{Question: "س", Answer: "ج", QuestionEn: "Q?"}}
	out := NormalizeContent(c)
	if got := ValidateContent(out); len(got) != 0 {
		t.Fatalf("unexpected problems: %v", got)
	}
	if out.Services[0].DescriptionEn != "An English description" {
		t.Fatalf("english description not cleaned: %q", out.Services[0].DescriptionEn)
	}

	cases := []struct {
		name   string
		mutate func(*Content)
		want   string
	}{
		{"long english hero", func(c *Content) { c.HeroTitleEn = strings.Repeat("x", MaxHeroTitle+1) }, "hero_title_en:too_long"},
		{"long english service", func(c *Content) { c.Services = []Service{{Title: "x", TitleEn: strings.Repeat("x", MaxServiceTitle+1)}} }, "service_title_en:too_long"},
		{"long english link label", func(c *Content) {
			c.Links = []Link{{Label: "x", LabelEn: strings.Repeat("x", MaxLinkLabel+1), URL: "https://a.iq"}}
		}, "link_label_en:too_long"},
		{"long english college label", func(c *Content) {
			c.Colleges = []Link{{Label: "x", LabelEn: strings.Repeat("x", MaxLinkLabel+1), URL: "https://a.iq"}}
		}, "college_label_en:too_long"},
		{"long english stat label", func(c *Content) {
			c.Stats = []StatSetting{{Key: StatAcademics, LabelEn: strings.Repeat("x", MaxStatLabel+1)}}
		}, "stat_label_en:too_long"},
		{"long english director message", func(c *Content) { c.Director.MessageEn = strings.Repeat("x", MaxDirectorMessage+1) }, "director_message_en:too_long"},
		{"long english answer", func(c *Content) {
			c.FAQ = []FAQItem{{Question: "q", Answer: "a", AnswerEn: strings.Repeat("x", MaxFAQAnswer+1)}}
		}, "faq_answer_en:too_long"},
		{"long english hours", func(c *Content) { c.Contact.HoursEn = strings.Repeat("x", MaxHours+1) }, "contact_hours_en:too_long"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validContent()
			tc.mutate(&c)
			if got := ValidateContent(NormalizeContent(c)); !slices.Contains(got, tc.want) {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}

func TestEnglishFieldsSurviveNormalisation(t *testing.T) {
	in := Content{
		HeroTitle: "ع", HeroTitleEn: "E", HeroSubtitleEn: "S", AboutEn: "A",
		Contact:  Contact{LocationEn: "Baghdad", HoursEn: "8-2"},
		Links:    []Link{{Label: "ع", LabelEn: "L", URL: "https://a.iq"}},
		Colleges: []Link{{Label: "ع", LabelEn: "C", URL: "https://a.iq"}},
		Stats:    []StatSetting{{Key: StatColleges, LabelEn: "Colleges"}},
		Director: Director{TitleEn: "T", NameEn: "N", PositionEn: "P", MessageEn: "M"},
	}
	out := NormalizeContent(in)
	if out.HeroTitleEn != "E" || out.HeroSubtitleEn != "S" || out.AboutEn != "A" ||
		out.Contact.LocationEn != "Baghdad" || out.Contact.HoursEn != "8-2" ||
		out.Links[0].LabelEn != "L" || out.Colleges[0].LabelEn != "C" || out.Stats[0].LabelEn != "Colleges" ||
		out.Director.TitleEn != "T" || out.Director.NameEn != "N" || out.Director.PositionEn != "P" || out.Director.MessageEn != "M" {
		t.Fatalf("english fields lost: %+v", out)
	}
}
