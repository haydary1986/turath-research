package apiclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const orcidBaseURL = "https://pub.orcid.org/v3.0"

// orcidValue is ORCID's wrapper around every string it returns.
type orcidValue struct {
	Value string `json:"value"`
}

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
		GivenNames *orcidValue `json:"given-names"`
		FamilyName *orcidValue `json:"family-name"`
		CreditName *orcidValue `json:"credit-name"`
	} `json:"name"`
	OtherNames *struct {
		OtherName []struct {
			Content string `json:"content"`
		} `json:"other-name"`
	} `json:"other-names"`
	Biography *struct {
		Content string `json:"content"`
	} `json:"biography"`
}

type ORCIDRecord struct {
	Person            *ORCIDPerson     `json:"person"`
	ActivitiesSummary *ORCIDActivities `json:"activities-summary"`
}

type ORCIDActivities struct {
	Works       *ORCIDWorkGroup       `json:"works"`
	Employments *ORCIDEmploymentGroup `json:"employments"`
}

type ORCIDWorkGroup struct {
	Group []struct {
		WorkSummary []ORCIDWorkSummary `json:"work-summary"`
	} `json:"group"`
}

type ORCIDWorkSummary struct {
	PutCode int `json:"put-code"`
	Title   *struct {
		Title struct {
			Value string `json:"value"`
		} `json:"title"`
	} `json:"title"`
	Type         string `json:"type"`
	JournalTitle *struct {
		Value string `json:"value"`
	} `json:"journal-title"`
	PublicationDate *struct {
		Year *struct {
			Value string `json:"value"`
		} `json:"year"`
		Month *struct {
			Value string `json:"value"`
		} `json:"month"`
		Day *struct {
			Value string `json:"value"`
		} `json:"day"`
	} `json:"publication-date"`
	ExternalIDs *struct {
		ExternalID []struct {
			Type         string `json:"external-id-type"`
			Value        string `json:"external-id-value"`
			Relationship string `json:"external-id-relationship"`
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
		Name       string `json:"name"`
		Identifier *struct {
			Value string `json:"disambiguated-organization-identifier"`
		} `json:"disambiguated-organization"`
		Address struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"address"`
	} `json:"organization"`
	RoleTitle string `json:"role-title"`
	StartDate *struct {
		Year *struct {
			Value string `json:"value"`
		} `json:"year"`
		Month *struct {
			Value string `json:"value"`
		} `json:"month"`
	} `json:"start-date"`
}

// ── Methods ──────────────────────────────────────

// Names lists every public spelling on the record; empty when the owner hid them.
func (p *ORCIDPerson) Names() []string {
	if p == nil {
		return nil
	}
	names := []string{}
	if p.Name != nil {
		parts := []string{}
		for _, part := range []*orcidValue{p.Name.GivenNames, p.Name.FamilyName} {
			if part != nil && strings.TrimSpace(part.Value) != "" {
				parts = append(parts, strings.TrimSpace(part.Value))
			}
		}
		if len(parts) > 0 {
			names = append(names, strings.Join(parts, " "))
		}
		if p.Name.CreditName != nil && strings.TrimSpace(p.Name.CreditName.Value) != "" {
			names = append(names, strings.TrimSpace(p.Name.CreditName.Value))
		}
	}
	if p.OtherNames != nil {
		for _, other := range p.OtherNames.OtherName {
			if strings.TrimSpace(other.Content) != "" {
				names = append(names, strings.TrimSpace(other.Content))
			}
		}
	}
	return names
}

// GetPerson retrieves the name section of an ORCID record.
func (c *ORCIDClient) GetPerson(ctx context.Context, orcidID string) (*ORCIDPerson, error) {
	u := fmt.Sprintf("%s/%s/person", orcidBaseURL, url.PathEscape(orcidID))
	var result ORCIDPerson
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("orcid get person: %w", err)
	}
	return &result, nil
}

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

// GetEmployments fetches the public employment section of a record.
func (c *ORCIDClient) GetEmployments(ctx context.Context, orcidID string) (*ORCIDEmploymentGroup, error) {
	u := fmt.Sprintf("%s/%s/employments", orcidBaseURL, url.PathEscape(orcidID))
	var result ORCIDEmploymentGroup
	if err := c.client.GetJSON(ctx, u, c.headers(), &result); err != nil {
		return nil, fmt.Errorf("orcid get employments: %w", err)
	}
	return &result, nil
}

// ExtractDOI extracts the work's own DOI from ORCID external IDs. A "part-of" DOI
// belongs to the containing book or series, so it is not this work's identifier.
func ExtractDOI(work ORCIDWorkSummary) string {
	if work.ExternalIDs == nil {
		return ""
	}
	for _, eid := range work.ExternalIDs.ExternalID {
		if eid.Type == "doi" && eid.Relationship != "part-of" {
			return eid.Value
		}
	}
	return ""
}
