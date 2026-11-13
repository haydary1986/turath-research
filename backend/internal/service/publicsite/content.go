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

// Service is one item of the "what we offer" list.
type Service struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Contact holds the department's public contact details; every field is optional.
type Contact struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Hours    string `json:"hours"`
}

// Link is a quick link shown on the landing page.
type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Content is the whole editable landing-page document.
type Content struct {
	HeroTitle    string    `json:"hero_title"`
	HeroSubtitle string    `json:"hero_subtitle"`
	About        string    `json:"about"`
	Services     []Service `json:"services"`
	Contact      Contact   `json:"contact"`
	Links        []Link    `json:"links"`
}

// NormalizeContent returns a cleaned copy of c; the input is not modified.
func NormalizeContent(c Content) Content {
	services := make([]Service, 0, len(c.Services))
	for _, s := range c.Services {
		services = append(services, Service{Title: cleanLine(s.Title), Description: cleanText(s.Description)})
	}
	links := make([]Link, 0, len(c.Links))
	for _, l := range c.Links {
		links = append(links, Link{Label: cleanLine(l.Label), URL: cleanLine(l.URL)})
	}
	return Content{
		HeroTitle:    cleanLine(c.HeroTitle),
		HeroSubtitle: cleanText(c.HeroSubtitle),
		About:        cleanText(c.About),
		Services:     services,
		Contact: Contact{
			Email:    strings.ToLower(cleanLine(c.Contact.Email)),
			Phone:    cleanLine(c.Contact.Phone),
			Location: cleanLine(c.Contact.Location),
			Hours:    cleanLine(c.Contact.Hours),
		},
		Links: links,
	}
}

// ValidateContent returns one "field:code" entry per problem in an already normalised document.
func ValidateContent(c Content) []string {
	var p problems
	p.text("hero_title", c.HeroTitle, true, MaxHeroTitle)
	p.text("hero_subtitle", c.HeroSubtitle, false, MaxHeroSubtitle)
	p.text("about", c.About, false, MaxAbout)

	if len(c.Services) > MaxServices {
		p.add("services", CodeTooMany)
	}
	for _, s := range c.Services {
		p.text("service_title", s.Title, true, MaxServiceTitle)
		p.text("service_description", s.Description, false, MaxServiceDesc)
	}

	validateContact(&p, c.Contact)

	if len(c.Links) > MaxLinks {
		p.add("links", CodeTooMany)
	}
	for _, l := range c.Links {
		p.text("link_label", l.Label, true, MaxLinkLabel)
		switch {
		case l.URL == "":
			p.add("link_url", CodeRequired)
		case tooLong(l.URL, MaxLinkURL):
			p.add("link_url", CodeTooLong)
		case !IsSafeHTTPSURL(l.URL):
			p.add("link_url", CodeHTTPSRequired)
		}
	}
	return p.list
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
	p.text("contact_hours", c.Hours, false, MaxHours)
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
