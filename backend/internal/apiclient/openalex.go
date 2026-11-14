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
	client  *Client
	baseURL string
	mailto  string
	apiKey  string
}

// NewOpenAlexClient creates an OpenAlex API client.
// mailto is used for the polite pool (higher rate limits); apiKey is optional.
func NewOpenAlexClient(mailto, apiKey string) *OpenAlexClient {
	return &OpenAlexClient{
		client:  NewClient(30*time.Second, 10, "turath-research/1.0"),
		baseURL: openAlexBaseURL,
		mailto:  mailto,
		apiKey:  apiKey,
	}
}

func (c *OpenAlexClient) withIdentity(params url.Values) url.Values {
	if c.mailto != "" {
		params.Set("mailto", c.mailto)
	}
	if c.apiKey != "" {
		params.Set("api_key", c.apiKey)
	}
	return params
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
	AuthorPosition string                `json:"author_position"`
	Author         OpenAlexAuthorInfo    `json:"author"`
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
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	ORCID        string `json:"orcid"`
	WorksCount   int    `json:"works_count"`
	CitedByCount int    `json:"cited_by_count"`
	HIndex       int    `json:"summary_stats.h_index"`
	SummaryStats struct {
		HIndex   int `json:"h_index"`
		I10Index int `json:"i10_index"`
	} `json:"summary_stats"`
}

// ── Methods ──────────────────────────────────────

// FindAuthorByORCID returns the OpenAlex author carrying this ORCID, or nil when
// OpenAlex has none. When several records share it, the one with the most works wins.
func (c *OpenAlexClient) FindAuthorByORCID(ctx context.Context, orcid string) (*OpenAlexPerson, error) {
	params := c.withIdentity(url.Values{
		"filter": {"orcid:" + orcid},
		"select": {"id,display_name,orcid,works_count,cited_by_count,summary_stats"},
	})
	var result OpenAlexAuthorResult
	u := fmt.Sprintf("%s/authors?%s", c.baseURL, params.Encode())
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("openalex find author by orcid: %w", err)
	}
	var best *OpenAlexPerson
	for i := range result.Results {
		if best == nil || result.Results[i].WorksCount > best.WorksCount {
			best = &result.Results[i]
		}
	}
	return best, nil
}

// ListWorksByAuthor pages through an author's works by OpenAlex id, up to maxPages.
// It uses page numbers because cursor paging past page one needs an API key.
// truncated reports that OpenAlex holds more works than were returned, so the list
// must not be read as the author's complete output.
func (c *OpenAlexClient) ListWorksByAuthor(ctx context.Context, authorID string, maxPages int) (works []OpenAlexWork, truncated bool, err error) {
	const perPage = 200
	total := 0
	for page := 1; page <= maxPages; page++ {
		params := c.withIdentity(url.Values{
			"filter":   {"author.id:" + authorID},
			"per_page": {fmt.Sprintf("%d", perPage)},
			"page":     {fmt.Sprintf("%d", page)},
			"sort":     {"publication_date:desc"},
			"select":   {"id,doi,display_name,publication_date,type,cited_by_count,language,primary_location,abstract_inverted_index"},
		})
		var result OpenAlexWorkResult
		u := fmt.Sprintf("%s/works?%s", c.baseURL, params.Encode())
		if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
			return nil, false, fmt.Errorf("openalex list works: %w", err)
		}
		works = append(works, result.Results...)
		total = result.Meta.Count
		if len(result.Results) == 0 || page*perPage >= total {
			break
		}
	}
	return works, len(works) < total, nil
}

// CountWorksByInstitution returns how many works OpenAlex attributes to the institution
// with this ROR id (e.g. "019vd4365"). It asks for a single result and reads the total.
func (c *OpenAlexClient) CountWorksByInstitution(ctx context.Context, ror string) (int, error) {
	params := c.withIdentity(url.Values{
		"filter":   {"institutions.ror:" + ror},
		"per_page": {"1"},
		"select":   {"id"},
	})
	var result OpenAlexWorkResult
	u := fmt.Sprintf("%s/works?%s", c.baseURL, params.Encode())
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return 0, fmt.Errorf("openalex count institution works: %w", err)
	}
	return result.Meta.Count, nil
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
