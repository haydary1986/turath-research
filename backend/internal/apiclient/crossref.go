package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

const crossRefBaseURL = "https://api.crossref.org"

// CrossRefClient interacts with the CrossRef API (free, polite pool with mailto).
type CrossRefClient struct {
	client *Client
	mailto string
}

func NewCrossRefClient(mailto string) *CrossRefClient {
	return &CrossRefClient{
		client: NewClient(30*time.Second, 20, "turath-research/1.0 (mailto:"+mailto+")"),
		mailto: mailto,
	}
}

// ── Response Types ───────────────────────────────

type CrossRefResponse struct {
	Status  string          `json:"status"`
	Message CrossRefMessage `json:"message"`
}

type CrossRefMessage struct {
	TotalResults int             `json:"total-results"`
	Items        []CrossRefWork  `json:"items"`
}

type CrossRefWork struct {
	DOI            string              `json:"DOI"`
	Title          []string            `json:"title"`
	ContainerTitle []string            `json:"container-title"`
	Type           string              `json:"type"`
	Published      *CrossRefDateParts  `json:"published"`
	IssuedDate     *CrossRefDateParts  `json:"issued"`
	Author         []CrossRefAuthor    `json:"author"`
	IsReferencedBy int                 `json:"is-referenced-by-count"`
	ISSN           []string            `json:"ISSN"`
	Volume         string              `json:"volume"`
	Issue          string              `json:"issue"`
	Page           string              `json:"page"`
	Publisher      string              `json:"publisher"`
	Language       string              `json:"language"`
	URL            string              `json:"URL"`
	Abstract       string              `json:"abstract"`
}

type CrossRefDateParts struct {
	DateParts [][]int `json:"date-parts"`
}

type CrossRefAuthor struct {
	Given       string `json:"given"`
	Family      string `json:"family"`
	ORCID       string `json:"ORCID"`
	Sequence    string `json:"sequence"` // "first" or "additional"
	Affiliation []struct {
		Name string `json:"name"`
	} `json:"affiliation"`
}

// ── Methods ──────────────────────────────────────

// GetWorkByDOI retrieves a single work by DOI.
func (c *CrossRefClient) GetWorkByDOI(ctx context.Context, doi string) (*CrossRefWork, error) {
	u := fmt.Sprintf("%s/works/%s", crossRefBaseURL, url.PathEscape(doi))
	var resp struct {
		Status  string       `json:"status"`
		Message CrossRefWork `json:"message"`
	}
	if err := c.client.GetJSON(ctx, u, nil, &resp); err != nil {
		return nil, fmt.Errorf("crossref get work: %w", err)
	}
	return &resp.Message, nil
}

// SearchWorks searches for works by query string.
func (c *CrossRefClient) SearchWorks(ctx context.Context, query string, rows, offset int) (*CrossRefResponse, error) {
	params := url.Values{
		"query":  {query},
		"rows":   {fmt.Sprintf("%d", rows)},
		"offset": {fmt.Sprintf("%d", offset)},
		"sort":   {"published"},
		"order":  {"desc"},
	}
	if c.mailto != "" {
		params.Set("mailto", c.mailto)
	}

	u := fmt.Sprintf("%s/works?%s", crossRefBaseURL, params.Encode())
	var resp CrossRefResponse
	if err := c.client.GetJSON(ctx, u, nil, &resp); err != nil {
		return nil, fmt.Errorf("crossref search works: %w", err)
	}
	return &resp, nil
}

// GetPublicationYear extracts the year from CrossRef date parts.
func (w *CrossRefWork) GetPublicationYear() int {
	if w.Published != nil && len(w.Published.DateParts) > 0 && len(w.Published.DateParts[0]) > 0 {
		return w.Published.DateParts[0][0]
	}
	if w.IssuedDate != nil && len(w.IssuedDate.DateParts) > 0 && len(w.IssuedDate.DateParts[0]) > 0 {
		return w.IssuedDate.DateParts[0][0]
	}
	return 0
}

// GetPublicationDate returns a formatted date string.
func (w *CrossRefWork) GetPublicationDate() string {
	dp := w.Published
	if dp == nil {
		dp = w.IssuedDate
	}
	if dp == nil || len(dp.DateParts) == 0 || len(dp.DateParts[0]) == 0 {
		return ""
	}
	parts := dp.DateParts[0]
	switch len(parts) {
	case 1:
		return fmt.Sprintf("%d-01-01", parts[0])
	case 2:
		return fmt.Sprintf("%d-%02d-01", parts[0], parts[1])
	default:
		return fmt.Sprintf("%d-%02d-%02d", parts[0], parts[1], parts[2])
	}
}

// GetFirstTitle returns the first title or empty string.
func (w *CrossRefWork) GetFirstTitle() string {
	if len(w.Title) > 0 {
		return w.Title[0]
	}
	return ""
}
