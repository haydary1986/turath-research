// Package researchlinks validates the profile links a researcher enters and
// extracts the identifiers they carry. It reduces each link to a canonical URL
// built from its identifier, so nothing else from the pasted text is ever stored.
package researchlinks

import (
	"net/url"
	"regexp"
	"strings"
)

const maxLinkLength = 500

// Validation codes, surfaced to the client as "field:code".
const (
	CodeInvalidURL     = "invalid_profile_url"
	CodeHTTPSRequired  = "https_required"
	CodeWrongHost      = "wrong_host"
	CodeNotProfile     = "not_a_profile_link"
	CodeInvalidORCID   = "invalid_orcid"
	FieldGoogleScholar = "google_scholar_url"
	FieldScopus        = "scopus_url"
	FieldResearchGate  = "researchgate_url"
	FieldLinkedIn      = "linkedin_url"
	FieldWebOfScience  = "web_of_science_url"
	FieldORCID         = "orcid"
)

var (
	scholarHost      = regexp.MustCompile(`^scholar\.google\.(com|[a-z]{2}|com?\.[a-z]{2})$`)
	scopusHost       = regexp.MustCompile(`^(www\.)?scopus\.com$`)
	researchGateHost = regexp.MustCompile(`^(www\.)?researchgate\.net$`)
	linkedInHost     = regexp.MustCompile(`^([a-z]{2,3}\.)?linkedin\.com$`)
	orcidHost        = regexp.MustCompile(`^(www\.)?orcid\.org$`)
	webOfScienceHost = regexp.MustCompile(`^(www\.)?webofscience\.com$`)

	scholarID = regexp.MustCompile(`^[A-Za-z0-9_-]{6,50}$`)
	scopusID  = regexp.MustCompile(`^[0-9]{5,20}$`)
	orcidID   = regexp.MustCompile(`^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{3}[0-9X]$`)
	wosID     = regexp.MustCompile(`^[A-Za-z0-9-]{3,40}$`)

	// Copying from an RTL page often carries invisible direction marks with the text.
	invisibleMarks = strings.NewReplacer(
		"\u200e", "", "\u200f", "", "\u202a", "", "\u202b", "", "\u202c", "", "\u202d", "", "\u202e", "",
		"\u2066", "", "\u2067", "", "\u2068", "", "\u2069", "", "\ufeff", "", "\u200b", "",
	)
)

// Input is the raw form as the researcher typed it. Empty fields are allowed.
type Input struct {
	GoogleScholarURL string `json:"google_scholar_url"`
	ScopusURL        string `json:"scopus_url"`
	ResearchGateURL  string `json:"researchgate_url"`
	LinkedInURL      string `json:"linkedin_url"`
	WebOfScienceURL  string `json:"web_of_science_url"`
	ORCID            string `json:"orcid"`
}

// Links is the validated, canonical form. Empty strings mean "not provided".
type Links struct {
	GoogleScholarURL string
	GoogleScholarID  string
	ScopusURL        string
	ScopusAuthorID   string
	ResearchGateURL  string
	LinkedInURL      string
	WebOfScienceURL  string
	ORCID            string
}

// Parse validates every field and reports all problems as "field:code".
func Parse(in Input) (Links, []string) {
	var links Links
	var problems []string
	collect := func(field, code string) {
		if code != "" {
			problems = append(problems, field+":"+code)
		}
	}

	var code string
	links.GoogleScholarURL, links.GoogleScholarID, code = parseScholar(in.GoogleScholarURL)
	collect(FieldGoogleScholar, code)
	links.ScopusURL, links.ScopusAuthorID, code = parseScopus(in.ScopusURL)
	collect(FieldScopus, code)
	links.ResearchGateURL, code = parseSlugProfile(in.ResearchGateURL, researchGateHost, "profile", "https://www.researchgate.net")
	collect(FieldResearchGate, code)
	links.LinkedInURL, code = parseSlugProfile(in.LinkedInURL, linkedInHost, "in", "https://www.linkedin.com")
	collect(FieldLinkedIn, code)
	links.WebOfScienceURL, code = parseWebOfScience(in.WebOfScienceURL)
	collect(FieldWebOfScience, code)
	links.ORCID, code = parseORCIDField(in.ORCID)
	collect(FieldORCID, code)

	return links, problems
}

// ScholarURL is the canonical Google Scholar profile address for an id.
func ScholarURL(id string) string {
	return "https://scholar.google.com/citations?user=" + url.QueryEscape(id)
}

// ScopusURL is the canonical Scopus author address for an id.
func ScopusURL(id string) string {
	return "https://www.scopus.com/authid/detail.uri?authorId=" + url.QueryEscape(id)
}

// ORCIDURL is the canonical ORCID record address for an id.
func ORCIDURL(id string) string {
	return "https://orcid.org/" + id
}

// Clean strips surrounding space and the invisible marks an RTL page leaves in copied text.
func Clean(raw string) string { return clean(raw) }

func clean(raw string) string {
	return strings.TrimSpace(invisibleMarks.Replace(raw))
}

// parseHTTPS accepts a pasted address on an allowed host. A missing scheme is taken
// as https because people often copy from the address bar; any other scheme is refused.
func parseHTTPS(raw string, host *regexp.Regexp) (*url.URL, string) {
	if len(raw) > maxLinkLength {
		return nil, CodeInvalidURL
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, CodeInvalidURL
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return nil, CodeHTTPSRequired
	}
	if u.User != nil || u.Port() != "" {
		return nil, CodeInvalidURL
	}
	if !host.MatchString(strings.ToLower(u.Hostname())) {
		return nil, CodeWrongHost
	}
	return u, ""
}

func parseScholar(raw string) (canonical, id, code string) {
	raw = clean(raw)
	if raw == "" {
		return "", "", ""
	}
	u, code := parseHTTPS(raw, scholarHost)
	if code != "" {
		return "", "", code
	}
	id = u.Query().Get("user")
	if strings.TrimRight(u.Path, "/") != "/citations" || !scholarID.MatchString(id) {
		return "", "", CodeNotProfile
	}
	return ScholarURL(id), id, ""
}

func parseScopus(raw string) (canonical, id, code string) {
	raw = clean(raw)
	if raw == "" {
		return "", "", ""
	}
	u, code := parseHTTPS(raw, scopusHost)
	if code != "" {
		return "", "", code
	}
	for key, values := range u.Query() {
		if strings.EqualFold(key, "authorId") && len(values) > 0 {
			id = values[0]
		}
	}
	if !scopusID.MatchString(id) {
		return "", "", CodeNotProfile
	}
	return ScopusURL(id), id, ""
}

// parseSlugProfile handles sites whose profile address is /<section>/<slug>.
func parseSlugProfile(raw string, host *regexp.Regexp, section, base string) (canonical, code string) {
	raw = clean(raw)
	if raw == "" {
		return "", ""
	}
	u, code := parseHTTPS(raw, host)
	if code != "" {
		return "", code
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != section || parts[1] == "" || parts[1] == "." || parts[1] == ".." {
		return "", CodeNotProfile
	}
	profile := url.URL{Path: "/" + section + "/" + parts[1]}
	return base + profile.EscapedPath(), ""
}

// parseWebOfScience accepts an author record address such as /wos/author/record/123 or
// /wos/woscc/author/rid/AAA-1111-2222, and keeps only the kind and the id.
func parseWebOfScience(raw string) (canonical, code string) {
	raw = clean(raw)
	if raw == "" {
		return "", ""
	}
	u, code := parseHTTPS(raw, webOfScienceHost)
	if code != "" {
		return "", code
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, part := range parts {
		if part != "author" || i+2 >= len(parts) {
			continue
		}
		kind, id := parts[i+1], parts[i+2]
		if (kind == "record" || kind == "rid") && wosID.MatchString(id) {
			return "https://www.webofscience.com/wos/author/" + kind + "/" + id, ""
		}
	}
	return "", CodeNotProfile
}

func parseORCIDField(raw string) (id, code string) {
	raw = clean(raw)
	if raw == "" {
		return "", ""
	}
	id, ok := ParseORCID(raw)
	if !ok {
		return "", CodeInvalidORCID
	}
	return id, ""
}

// ParseORCID accepts a bare id or an https://orcid.org address and returns the
// canonical id. Case and the ISO 7064 check digit are both verified.
func ParseORCID(raw string) (string, bool) {
	raw = clean(raw)
	if strings.Contains(raw, "orcid.org") {
		u, code := parseHTTPS(raw, orcidHost)
		if code != "" {
			return "", false
		}
		raw = strings.Trim(u.Path, "/")
	}
	id := strings.ToUpper(raw)
	if !orcidID.MatchString(id) || !validORCIDChecksum(id) {
		return "", false
	}
	return id, true
}

func validORCIDChecksum(id string) bool {
	digits := strings.ReplaceAll(id, "-", "")
	total := 0
	for _, c := range digits[:len(digits)-1] {
		total = (total + int(c-'0')) * 2
	}
	check := (12 - total%11) % 11
	want := byte('0' + check)
	if check == 10 {
		want = 'X'
	}
	return digits[len(digits)-1] == want
}
