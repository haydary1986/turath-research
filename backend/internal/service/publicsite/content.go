package publicsite

import (
	"net/url"
	"regexp"
	"strings"
)

// Limits on the landing-page document.
const (
	MaxHeroTitle    = 200
	MaxHeroSubtitle = 500
	MaxAbout        = 5000
	MaxServices     = 12
	MaxServiceTitle = 150
	MaxServiceDesc  = 600
	MaxLinks        = 8
	MaxLinkLabel    = 80
	MaxLinkURL      = 500
	MaxEmail        = 254
	MaxPhone        = 50
	MaxLocation     = 300
	MaxHours        = 200
)

var (
	// Shown as a mailto: link, so the shape check is strict.
	emailShape = regexp.MustCompile(`^[^@\s<>()\[\],;:"\\]+@[^@\s<>()\[\],;:"\\]+\.[^@\s<>()\[\],;:"\\.]+$`)
	// Shown as a tel: link: digits, an optional leading +, and common separators.
	phoneShape = regexp.MustCompile(`^\+?[0-9٠-٩ ()\-]{5,}$`)
)

// Service is one item of the "what we offer" list. Icon is a key from ServiceIcons
// and Link a site-relative path (see IsSafeServiceLink); both are optional.
// Every visible text has an optional English counterpart (the *En fields): Arabic
// is the primary language and the public pages fall back to it when English is empty.
type Service struct {
	Title         string `json:"title"`
	TitleEn       string `json:"title_en"`
	Description   string `json:"description"`
	DescriptionEn string `json:"description_en"`
	Icon          string `json:"icon"`
	Link          string `json:"link"`
}

// Contact holds the department's public contact details; every field is optional.
type Contact struct {
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Location   string `json:"location"`
	LocationEn string `json:"location_en"`
	Hours      string `json:"hours"`
	HoursEn    string `json:"hours_en"`
}

// Link is a quick link shown on the landing page.
type Link struct {
	Label   string `json:"label"`
	LabelEn string `json:"label_en"`
	URL     string `json:"url"`
}

// Content is the whole editable landing-page document.
type Content struct {
	HeroTitle      string    `json:"hero_title"`
	HeroTitleEn    string    `json:"hero_title_en"`
	HeroSubtitle   string    `json:"hero_subtitle"`
	HeroSubtitleEn string    `json:"hero_subtitle_en"`
	About          string    `json:"about"`
	AboutEn        string    `json:"about_en"`
	Services       []Service `json:"services"`
	Contact        Contact   `json:"contact"`
	Links          []Link    `json:"links"`
	// Blocks added with the second version of the landing page (content_blocks.go).
	Stats            []StatSetting `json:"stats"`
	Director         Director      `json:"director"`
	Colleges         []Link        `json:"colleges"`
	Socials          []Social      `json:"socials"`
	FAQ              []FAQItem     `json:"faq"`
	HidePublications bool          `json:"hide_publications"`
}

// NormalizeContent returns a cleaned copy of c; the input is not modified.
func NormalizeContent(c Content) Content {
	services := make([]Service, 0, len(c.Services))
	for _, s := range c.Services {
		services = append(services, Service{
			Title: cleanLine(s.Title), TitleEn: cleanLine(s.TitleEn),
			Description: cleanText(s.Description), DescriptionEn: cleanText(s.DescriptionEn),
			Icon: cleanLine(s.Icon), Link: cleanLine(s.Link),
		})
	}
	return Content{
		HeroTitle:      cleanLine(c.HeroTitle),
		HeroTitleEn:    cleanLine(c.HeroTitleEn),
		HeroSubtitle:   cleanText(c.HeroSubtitle),
		HeroSubtitleEn: cleanText(c.HeroSubtitleEn),
		About:          cleanText(c.About),
		AboutEn:        cleanText(c.AboutEn),
		Services:       services,
		Contact: Contact{
			Email:      strings.ToLower(cleanLine(c.Contact.Email)),
			Phone:      cleanLine(c.Contact.Phone),
			Location:   cleanLine(c.Contact.Location),
			LocationEn: cleanLine(c.Contact.LocationEn),
			Hours:      cleanLine(c.Contact.Hours),
			HoursEn:    cleanLine(c.Contact.HoursEn),
		},
		Links:            normalizeLinks(c.Links),
		Stats:            normalizeStats(c.Stats),
		Director:         normalizeDirector(c.Director),
		Colleges:         normalizeLinks(c.Colleges),
		Socials:          normalizeSocials(c.Socials),
		FAQ:              normalizeFAQ(c.FAQ),
		HidePublications: c.HidePublications,
	}
}

func normalizeLinks(in []Link) []Link {
	out := make([]Link, 0, len(in))
	for _, l := range in {
		out = append(out, Link{Label: cleanLine(l.Label), LabelEn: cleanLine(l.LabelEn), URL: cleanLine(l.URL)})
	}
	return out
}

// ValidateContent returns one "field:code" entry per problem in an already normalised document.
func ValidateContent(c Content) []string {
	var p problems
	p.text("hero_title", c.HeroTitle, true, MaxHeroTitle)
	p.text("hero_title_en", c.HeroTitleEn, false, MaxHeroTitle)
	p.text("hero_subtitle", c.HeroSubtitle, false, MaxHeroSubtitle)
	p.text("hero_subtitle_en", c.HeroSubtitleEn, false, MaxHeroSubtitle)
	p.text("about", c.About, false, MaxAbout)
	p.text("about_en", c.AboutEn, false, MaxAbout)

	if len(c.Services) > MaxServices {
		p.add("services", CodeTooMany)
	}
	for _, s := range c.Services {
		p.text("service_title", s.Title, true, MaxServiceTitle)
		p.text("service_title_en", s.TitleEn, false, MaxServiceTitle)
		p.text("service_description", s.Description, false, MaxServiceDesc)
		p.text("service_description_en", s.DescriptionEn, false, MaxServiceDesc)
		if s.Icon != "" && !IsServiceIcon(s.Icon) {
			p.add("service_icon", CodeInvalid)
		}
		if s.Link != "" && !IsSafeServiceLink(s.Link) {
			p.add("service_link", CodeInvalid)
		}
	}

	validateContact(&p, c.Contact)
	validateLinks(&p, "links", "link", c.Links, MaxLinks)
	validateBlocks(&p, c)
	return p.list
}

// validateLinks checks a list of labelled https links; fields are reported as
// "<list>" for the count and "<item>_label" / "<item>_url" for each entry.
func validateLinks(p *problems, list, item string, links []Link, limit int) {
	if len(links) > limit {
		p.add(list, CodeTooMany)
	}
	for _, l := range links {
		p.text(item+"_label", l.Label, true, MaxLinkLabel)
		p.text(item+"_label_en", l.LabelEn, false, MaxLinkLabel)
		validateHTTPS(p, item+"_url", l.URL)
	}
}

func validateHTTPS(p *problems, field, value string) {
	switch {
	case value == "":
		p.add(field, CodeRequired)
	case tooLong(value, MaxLinkURL):
		p.add(field, CodeTooLong)
	case !IsSafeHTTPSURL(value):
		p.add(field, CodeHTTPSRequired)
	}
}

func validateContact(p *problems, c Contact) {
	switch {
	case tooLong(c.Email, MaxEmail):
		p.add("contact_email", CodeTooLong)
	case c.Email != "" && !emailShape.MatchString(c.Email):
		p.add("contact_email", CodeInvalidEmail)
	}
	switch {
	case tooLong(c.Phone, MaxPhone):
		p.add("contact_phone", CodeTooLong)
	case c.Phone != "" && !phoneShape.MatchString(c.Phone):
		p.add("contact_phone", CodeInvalid)
	}
	p.text("contact_location", c.Location, false, MaxLocation)
	p.text("contact_location_en", c.LocationEn, false, MaxLocation)
	p.text("contact_hours", c.Hours, false, MaxHours)
	p.text("contact_hours_en", c.HoursEn, false, MaxHours)
}

// IsSafeHTTPSURL accepts only absolute https URLs with a host and no embedded credentials.
func IsSafeHTTPSURL(raw string) bool {
	if strings.ContainsAny(raw, " \t\n<>\"'`") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Host != "" && u.User == nil && u.Hostname() != ""
}
