package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

const openAlexBaseURL = "https://api.openalex.org"

// OpenAlexClient interacts with the OpenAlex API (free, no key needed).
type OpenAlexClient struct {
	client *Client
	mailto string
}

// NewOpenAlexClient creates an OpenAlex API client.
// mailto is used for the polite pool (higher rate limits).
func NewOpenAlexClient(mailto string) *OpenAlexClient {
	return &OpenAlexClient{
		client: NewClient(30*time.Second, 10, "turath-research/1.0"),
		mailto: mailto,
	}
}

func (c *OpenAlexClient) headers() map[string]string {
	h := map[string]string{}
	if c.mailto != "" {
		h["mailto"] = c.mailto
	}
	return h
}

// ── Response Types ───────────────────────────────

type OpenAlexWorkResult struct {
	Meta    OpenAlexMeta   `json:"meta"`
	Results []OpenAlexWork `json:"results"`
}

type OpenAlexMeta struct {
	Count   int `json:"count"`
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
}

type OpenAlexWork struct {
	ID              string            `json:"id"`
	DOI             string            `json:"doi"`
	Title           string            `json:"title"`
	DisplayName     string            `json:"display_name"`
	PublicationDate string            `json:"publication_date"`
	PublicationYear int               `json:"publication_year"`
	Type            string            `json:"type"`
	CitedByCount    int               `json:"cited_by_count"`
	IsOA            bool              `json:"is_oa"`
	Language        string            `json:"language"`
	Authorships     []OpenAlexAuthor  `json:"authorships"`
	PrimaryLocation *OpenAlexLocation `json:"primary_location"`
	AbstractInvIdx  map[string][]int  `json:"abstract_inverted_index"`
}

type OpenAlexAuthor struct {
	AuthorPosition string              `json:"author_position"`
	Author         OpenAlexAuthorInfo  `json:"author"`
	Institutions   []OpenAlexInstitution `json:"institutions"`
}

type OpenAlexAuthorInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	ORCID       string `json:"orcid"`
}

type OpenAlexInstitution struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CountryCode string `json:"country_code"`
}

type OpenAlexLocation struct {
	Source *OpenAlexSource `json:"source"`
}

type OpenAlexSource struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	ISSN        string `json:"issn_l"`
	IsOA        bool   `json:"is_oa"`
	Type        string `json:"type"`
}

type OpenAlexAuthorResult struct {
	Meta    OpenAlexMeta     `json:"meta"`
	Results []OpenAlexPerson `json:"results"`
}

type OpenAlexPerson struct {
	ID              string `json:"id"`
	DisplayName     string `json:"display_name"`
	ORCID           string `json:"orcid"`
	WorksCount      int    `json:"works_count"`
	CitedByCount    int    `json:"cited_by_count"`
	HIndex          int    `json:"summary_stats.h_index"`
	SummaryStats    struct {
		HIndex   int `json:"h_index"`
		I10Index int `json:"i10_index"`
	} `json:"summary_stats"`
}

// ── Methods ──────────────────────────────────────

// SearchWorksByAuthorName searches for publications by author name.
func (c *OpenAlexClient) SearchWorksByAuthorName(ctx context.Context, authorName string, page, perPage int) (*OpenAlexWorkResult, error) {
	params := url.Values{
		"filter":   {fmt.Sprintf("authorships.author.display_name.search:%s", authorName)},
		"page":     {fmt.Sprintf("%d", page)},
		"per_page": {fmt.Sprintf("%d", perPage)},
		"sort":     {"publication_date:desc"},
	}

	u := fmt.Sprintf("%s/works?%s", openAlexBaseURL, params.Encode())
	var result OpenAlexWorkResult
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("openalex search works: %w", err)
	}
	return &result, nil
}

// GetWorkByDOI retrieves a single work by DOI.
func (c *OpenAlexClient) GetWorkByDOI(ctx context.Context, doi string) (*OpenAlexWork, error) {
	u := fmt.Sprintf("%s/works/doi:%s", openAlexBaseURL, url.PathEscape(doi))
	var result OpenAlexWork
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("openalex get work by DOI: %w", err)
	}
	return &result, nil
}

// SearchAuthors searches for authors by name.
func (c *OpenAlexClient) SearchAuthors(ctx context.Context, name string) (*OpenAlexAuthorResult, error) {
	params := url.Values{
		"search": {name},
	}
	u := fmt.Sprintf("%s/authors?%s", openAlexBaseURL, params.Encode())
	var result OpenAlexAuthorResult
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("openalex search authors: %w", err)
	}
	return &result, nil
}

// GetAuthorByORCID retrieves an author by ORCID.
func (c *OpenAlexClient) GetAuthorByORCID(ctx context.Context, orcid string) (*OpenAlexPerson, error) {
	u := fmt.Sprintf("%s/authors/orcid:%s", openAlexBaseURL, url.PathEscape(orcid))
	var result OpenAlexPerson
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("openalex get author by ORCID: %w", err)
	}
	return &result, nil
}

// ReconstructAbstract rebuilds abstract from inverted index.
func ReconstructAbstract(invertedIndex map[string][]int) string {
	if len(invertedIndex) == 0 {
		return ""
	}

	maxPos := 0
	for _, positions := range invertedIndex {
		for _, pos := range positions {
			if pos > maxPos {
				maxPos = pos
			}
		}
	}

	words := make([]string, maxPos+1)
	for word, positions := range invertedIndex {
		for _, pos := range positions {
			if pos < len(words) {
				words[pos] = word
			}
		}
	}

	result := ""
	for i, w := range words {
		if w == "" {
			continue
		}
		if i > 0 && result != "" {
			result += " "
		}
		result += w
	}
	return result
}
