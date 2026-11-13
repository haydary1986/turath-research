package researchlinks

import (
	"slices"
	"testing"
)

func TestParseORCID(t *testing.T) {
	valid := map[string]string{
		"0000-0002-1825-0097":                       "0000-0002-1825-0097",
		"  0000-0002-1825-0097 ":                    "0000-0002-1825-0097",
		"https://orcid.org/0000-0002-1825-0097":     "0000-0002-1825-0097",
		"https://orcid.org/0000-0002-1825-0097/":    "0000-0002-1825-0097",
		"https://www.orcid.org/0000-0002-1825-0097": "0000-0002-1825-0097",
		"orcid.org/0000-0002-1825-0097":             "0000-0002-1825-0097",
		"\u200f0000-0002-1825-0097\u200e":           "0000-0002-1825-0097",
		"0000-0002-9079-593x":                       "0000-0002-9079-593X",
	}
	for in, want := range valid {
		if got, ok := ParseORCID(in); !ok || got != want {
			t.Errorf("ParseORCID(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}

	invalid := []string{
		"", "0000-0002-1825-0098", "0000000218250097", "0000-0002-1825-009",
		"http://orcid.org/0000-0002-1825-0097",
		"https://sandbox.orcid.org/0000-0002-1825-0097",
		"https://orcid.org.evil.com/0000-0002-1825-0097",
		"https://evil.com/orcid.org/0000-0002-1825-0097",
		"https://orcid.org/0000-0002-1825-0097/works",
		"٠٠٠٠-٠٠٠٢-١٨٢٥-٠٠٩٧",
	}
	for _, in := range invalid {
		if got, ok := ParseORCID(in); ok {
			t.Errorf("ParseORCID(%q) accepted as %q", in, got)
		}
	}
}

func TestParseCanonicalizesAndExtracts(t *testing.T) {
	got, problems := Parse(Input{
		GoogleScholarURL: "https://scholar.google.co.uk/citations?hl=ar&user=AbC-12_xyzAAJ&view_op=list_works",
		ScopusURL:        "www.scopus.com/authid/detail.uri?origin=x&authorID=57193000000",
		ResearchGateURL:  "https://researchgate.net/profile/Ali-Hasan/research?x=1",
		LinkedInURL:      "https://iq.linkedin.com/in/ali-hasan-123/?trk=public",
		ORCID:            "https://orcid.org/0000-0002-1825-0097",
	})
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	want := Links{
		GoogleScholarURL: "https://scholar.google.com/citations?user=AbC-12_xyzAAJ",
		GoogleScholarID:  "AbC-12_xyzAAJ",
		ScopusURL:        "https://www.scopus.com/authid/detail.uri?authorId=57193000000",
		ScopusAuthorID:   "57193000000",
		ResearchGateURL:  "https://www.researchgate.net/profile/Ali-Hasan",
		LinkedInURL:      "https://www.linkedin.com/in/ali-hasan-123",
		ORCID:            "0000-0002-1825-0097",
	}
	if got != want {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestParseEmptyIsAllowed(t *testing.T) {
	got, problems := Parse(Input{GoogleScholarURL: "  ", ORCID: "\u200f"})
	if len(problems) != 0 || got != (Links{}) {
		t.Errorf("got %+v, %v", got, problems)
	}
}

func TestParseRejections(t *testing.T) {
	cases := []struct {
		name string
		in   Input
		want string
	}{
		{"http", Input{GoogleScholarURL: "http://scholar.google.com/citations?user=AbC-12_xyzAAJ"}, "google_scholar_url:https_required"},
		{"javascript", Input{LinkedInURL: "javascript:alert(1)//linkedin.com/in/x"}, "linkedin_url:invalid_profile_url"},
		{"javascript scheme", Input{LinkedInURL: "javascript://linkedin.com/in/x"}, "linkedin_url:https_required"},
		{"scholar lookalike", Input{GoogleScholarURL: "https://scholar.google.evil.com/citations?user=AbC-12_xyzAAJ"}, "google_scholar_url:wrong_host"},
		{"scholar other tld", Input{GoogleScholarURL: "https://scholar.google.evil/citations?user=AbC-12_xyzAAJ"}, "google_scholar_url:wrong_host"},
		{"scholar prefix", Input{GoogleScholarURL: "https://notscholar.google.com/citations?user=AbC-12_xyzAAJ"}, "google_scholar_url:wrong_host"},
		{"host suffix", Input{ScopusURL: "https://scopus.com.evil.io/authid/detail.uri?authorId=57193000000"}, "scopus_url:wrong_host"},
		{"host suffix glued", Input{LinkedInURL: "https://evillinkedin.com/in/x"}, "linkedin_url:wrong_host"},
		{"userinfo", Input{ResearchGateURL: "https://researchgate.net@evil.com/profile/x"}, "researchgate_url:invalid_profile_url"},
		{"userinfo on allowed host", Input{ResearchGateURL: "https://me@researchgate.net/profile/x"}, "researchgate_url:invalid_profile_url"},
		{"port", Input{ResearchGateURL: "https://researchgate.net:8443/profile/x"}, "researchgate_url:invalid_profile_url"},
		{"scholar search", Input{GoogleScholarURL: "https://scholar.google.com/scholar?q=ali"}, "google_scholar_url:not_a_profile_link"},
		{"scholar bad id", Input{GoogleScholarURL: "https://scholar.google.com/citations?user=a<b"}, "google_scholar_url:not_a_profile_link"},
		{"scopus no author", Input{ScopusURL: "https://www.scopus.com/search/form.uri"}, "scopus_url:not_a_profile_link"},
		{"scopus non numeric", Input{ScopusURL: "https://www.scopus.com/authid/detail.uri?authorId=abc"}, "scopus_url:not_a_profile_link"},
		{"linkedin company", Input{LinkedInURL: "https://www.linkedin.com/company/x"}, "linkedin_url:not_a_profile_link"},
		{"researchgate home", Input{ResearchGateURL: "https://www.researchgate.net/"}, "researchgate_url:not_a_profile_link"},
		{"orcid checksum", Input{ORCID: "0000-0002-1825-0098"}, "orcid:invalid_orcid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, problems := Parse(tc.in)
			if !slices.Equal(problems, []string{tc.want}) {
				t.Errorf("got %v, want [%s]", problems, tc.want)
			}
		})
	}
}

func TestParseReportsEveryProblem(t *testing.T) {
	_, problems := Parse(Input{GoogleScholarURL: "https://evil.com", ORCID: "nope"})
	want := []string{"google_scholar_url:wrong_host", "orcid:invalid_orcid"}
	if !slices.Equal(problems, want) {
		t.Errorf("got %v, want %v", problems, want)
	}
}

func TestParseWebOfScience(t *testing.T) {
	cases := []struct{ raw, want, code string }{
		{"https://www.webofscience.com/wos/author/record/123456", "https://www.webofscience.com/wos/author/record/123456", ""},
		{"webofscience.com/wos/woscc/author/rid/AAA-1111-2222/?x=1", "https://www.webofscience.com/wos/author/rid/AAA-1111-2222", ""},
		{"http://www.webofscience.com/wos/author/record/123456", "", CodeHTTPSRequired},
		{"https://webofscience.com.evil.org/wos/author/record/123456", "", CodeWrongHost},
		{"https://evilwebofscience.com/wos/author/record/123456", "", CodeWrongHost},
		{"https://www.webofscience.com/wos/woscc/full-record/WOS:1", "", CodeNotProfile},
		{"https://www.webofscience.com/wos/author/record/..", "", CodeNotProfile},
	}
	for _, c := range cases {
		links, problems := Parse(Input{WebOfScienceURL: c.raw})
		if links.WebOfScienceURL != c.want {
			t.Errorf("%q: got %q", c.raw, links.WebOfScienceURL)
		}
		wantProblems := 0
		if c.code != "" {
			wantProblems = 1
		}
		if len(problems) != wantProblems || (c.code != "" && problems[0] != FieldWebOfScience+":"+c.code) {
			t.Errorf("%q: problems = %v, want %q", c.raw, problems, c.code)
		}
	}
}
