package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

const orcidBaseURL = "https://pub.orcid.org/v3.0"

// ORCIDClient interacts with the ORCID Public API v3.0 (free, no key for public API).
type ORCIDClient struct {
	client *Client
}

func NewORCIDClient() *ORCIDClient {
	return &ORCIDClient{
		client: NewClient(30*time.Second, 20, "turath-research/1.0"),
	}
}

func (c *ORCIDClient) headers() map[string]string {
	return map[string]string{
		"Accept": "application/json",
	}
}

// ── Response Types ───────────────────────────────

type ORCIDPerson struct {
	Name *struct {
		GivenNames string `json:"given-names"`
		FamilyName string `json:"family-name"`
	} `json:"name"`
	Biography *struct {
		Content string `json:"content"`
	} `json:"biography"`
}

type ORCIDRecord struct {
	Person         *ORCIDPerson        `json:"person"`
	ActivitiesSummary *ORCIDActivities `json:"activities-summary"`
}

type ORCIDActivities struct {
	Works       *ORCIDWorkGroup      `json:"works"`
	Employments *ORCIDEmploymentGroup `json:"employments"`
}

type ORCIDWorkGroup struct {
	Group []struct {
		WorkSummary []ORCIDWorkSummary `json:"work-summary"`
	} `json:"group"`
}

type ORCIDWorkSummary struct {
	PutCode   int    `json:"put-code"`
	Title     *struct {
		Title struct {
			Value string `json:"value"`
		} `json:"title"`
	} `json:"title"`
	Type     string `json:"type"`
	JournalTitle *struct {
		Value string `json:"value"`
	} `json:"journal-title"`
	PublicationDate *struct {
		Year  *struct{ Value string `json:"value"` } `json:"year"`
		Month *struct{ Value string `json:"value"` } `json:"month"`
		Day   *struct{ Value string `json:"value"` } `json:"day"`
	} `json:"publication-date"`
	ExternalIDs *struct {
		ExternalID []struct {
			Type  string `json:"external-id-type"`
			Value string `json:"external-id-value"`
		} `json:"external-id"`
	} `json:"external-ids"`
}

type ORCIDEmploymentGroup struct {
	AffiliationGroup []struct {
		Summaries []struct {
			Employment ORCIDEmployment `json:"employment-summary"`
		} `json:"summaries"`
	} `json:"affiliation-group"`
}

type ORCIDEmployment struct {
	Organization struct {
		Name    string `json:"name"`
		Address struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"address"`
	} `json:"organization"`
	RoleTitle string `json:"role-title"`
	StartDate *struct {
		Year  *struct{ Value string `json:"value"` } `json:"year"`
		Month *struct{ Value string `json:"value"` } `json:"month"`
	} `json:"start-date"`
}

// ── Methods ──────────────────────────────────────

// GetRecord retrieves the full ORCID record for a researcher.
func (c *ORCIDClient) GetRecord(ctx context.Context, orcidID string) (*ORCIDRecord, error) {
	u := fmt.Sprintf("%s/%s/record", orcidBaseURL, url.PathEscape(orcidID))
	var result ORCIDRecord
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("orcid get record: %w", err)
	}
	return &result, nil
}

// GetWorks retrieves works for an ORCID ID.
func (c *ORCIDClient) GetWorks(ctx context.Context, orcidID string) (*ORCIDWorkGroup, error) {
	u := fmt.Sprintf("%s/%s/works", orcidBaseURL, url.PathEscape(orcidID))
	var result ORCIDWorkGroup
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("orcid get works: %w", err)
	}
	return &result, nil
}

// ExtractDOI extracts DOI from ORCID work external IDs.
func ExtractDOI(work ORCIDWorkSummary) string {
	if work.ExternalIDs == nil {
		return ""
	}
	for _, eid := range work.ExternalIDs.ExternalID {
		if eid.Type == "doi" {
			return eid.Value
		}
	}
	return ""
}
