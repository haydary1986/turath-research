package publicsite

import (
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Limits on the blocks added with the second version of the landing page.
const (
	MaxStatLabel       = 80
	MaxStatValue       = 20
	MaxDirectorTitle   = 150
	MaxDirectorName    = 150
	MaxDirectorPos     = 200
	MaxDirectorMessage = 5000
	MaxColleges        = 30
	MaxSocials         = 10
	MaxFAQ             = 20
	MaxFAQQuestion     = 300
	MaxFAQAnswer       = 3000
	MaxServiceLink     = 200
)

// Statistic keys, in their default order. The values are computed from live data
// (see handler/public_stats.go); the document only stores how each one is shown.
const (
	StatAcademics    = "academics"
	StatPublications = "publications"
	StatIndexed      = "indexed"
	StatColleges     = "colleges"
	StatActivities   = "activities"
)

// StatKeys is every statistic of the live statistics bar, in default order.
var StatKeys = []string{StatAcademics, StatPublications, StatIndexed, StatColleges, StatActivities}

// ServiceIcons are the icon keys a service card may use; the frontend maps each to a glyph.
var ServiceIcons = []string{"research", "verify", "award", "calendar", "search", "shield", "file", "users", "book", "chart"}

// SocialPlatforms are the networks the footer can link to.
var SocialPlatforms = []string{"facebook", "instagram", "x", "youtube", "linkedin", "telegram", "tiktok", "whatsapp", "website"}

// serviceLinkShape: a site-relative path (no scheme, host, query or traversal).
var serviceLinkShape = regexp.MustCompile(`^/[a-z0-9_\-/]*$`)

// Paths a service card may point to: the portal, the public journal pages and the news page.
var serviceLinkRoots = []string{"/portal", "/journals", "/news"}

// StatSetting is how one live statistic is shown. A non-empty Value replaces the
// computed number; an empty Label uses the default label of the key.
type StatSetting struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Hidden bool   `json:"hidden"`
}

// Director is the director's message block; it is hidden while Name and Message are empty.
// PhotoID is managed by the photo upload endpoint, never by the document editor.
type Director struct {
	Title    string `json:"title"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Message  string `json:"message"`
	PhotoID  string `json:"photo_id"`
}

// Social is one social-media link of the footer.
type Social struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// FAQItem is one question of the FAQ block.
type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// IsServiceIcon reports whether key names a known service icon.
func IsServiceIcon(key string) bool { return slices.Contains(ServiceIcons, key) }

// IsSocialPlatform reports whether key names a supported social network.
func IsSocialPlatform(key string) bool { return slices.Contains(SocialPlatforms, key) }

// IsStatKey reports whether key names a live statistic.
func IsStatKey(key string) bool { return slices.Contains(StatKeys, key) }

// IsSafeServiceLink accepts only site-relative paths under the portal, the journal pages or the news page.
func IsSafeServiceLink(link string) bool {
	if tooLong(link, MaxServiceLink) || !serviceLinkShape.MatchString(link) ||
		strings.Contains(link, "//") || strings.Contains(link, "..") {
		return false
	}
	for _, root := range serviceLinkRoots {
		if link == root || strings.HasPrefix(link, root+"/") {
			return true
		}
	}
	return false
}

// normalizeStats keeps one setting per known key, in the stored order, and appends
// any missing key (visible, automatic) so the editor always lists all of them.
func normalizeStats(in []StatSetting) []StatSetting {
	out := make([]StatSetting, 0, len(StatKeys))
	seen := map[string]bool{}
	for _, s := range in {
		key := cleanLine(s.Key)
		if !IsStatKey(key) || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, StatSetting{Key: key, Label: cleanLine(s.Label), Value: cleanLine(s.Value), Hidden: s.Hidden})
	}
	for _, key := range StatKeys {
		if !seen[key] {
			out = append(out, StatSetting{Key: key})
		}
	}
	return out
}

func normalizeDirector(d Director) Director {
	photo := cleanLine(d.PhotoID)
	if _, err := uuid.Parse(photo); err != nil {
		photo = ""
	}
	return Director{
		Title: cleanLine(d.Title), Name: cleanLine(d.Name), Position: cleanLine(d.Position),
		Message: cleanText(d.Message), PhotoID: photo,
	}
}

func normalizeSocials(in []Social) []Social {
	out := make([]Social, 0, len(in))
	for _, s := range in {
		out = append(out, Social{Platform: strings.ToLower(cleanLine(s.Platform)), URL: cleanLine(s.URL)})
	}
	return out
}

func normalizeFAQ(in []FAQItem) []FAQItem {
	out := make([]FAQItem, 0, len(in))
	for _, f := range in {
		out = append(out, FAQItem{Question: cleanLine(f.Question), Answer: cleanText(f.Answer)})
	}
	return out
}

// validateBlocks checks the second-version blocks of an already normalised document.
func validateBlocks(p *problems, c Content) {
	for _, s := range c.Stats {
		p.text("stat_label", s.Label, false, MaxStatLabel)
		p.text("stat_value", s.Value, false, MaxStatValue)
	}

	p.text("director_title", c.Director.Title, false, MaxDirectorTitle)
	p.text("director_name", c.Director.Name, false, MaxDirectorName)
	p.text("director_position", c.Director.Position, false, MaxDirectorPos)
	p.text("director_message", c.Director.Message, false, MaxDirectorMessage)

	validateLinks(p, "colleges", "college", c.Colleges, MaxColleges)

	if len(c.Socials) > MaxSocials {
		p.add("socials", CodeTooMany)
	}
	for _, s := range c.Socials {
		if !IsSocialPlatform(s.Platform) {
			p.add("social_platform", CodeInvalid)
		}
		validateHTTPS(p, "social_url", s.URL)
	}

	if len(c.FAQ) > MaxFAQ {
		p.add("faq", CodeTooMany)
	}
	for _, f := range c.FAQ {
		p.text("faq_question", f.Question, true, MaxFAQQuestion)
		p.text("faq_answer", f.Answer, true, MaxFAQAnswer)
	}
}
