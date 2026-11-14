package rissync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	clientTimeout = 30 * time.Second
	pageSize      = 500
	maxBody       = 32 << 20
)

// Client talks to the RIS Supabase (PostgREST and the GoTrue admin API) with the
// service-role key. It never logs the key or the response bodies.
type Client struct {
	base string
	key  string
	http *http.Client
}

// NewClient returns nil when the RIS is not configured.
func NewClient(baseURL, serviceKey string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(serviceKey) == "" {
		return nil
	}
	return &Client{base: baseURL, key: serviceKey, http: &http.Client{Timeout: clientTimeout}}
}

// RISProfile is one researcher_social_profiles row.
type RISProfile struct {
	Platform  string    `json:"platform"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RISResearcher is the part of a RIS researcher the sync reads.
type RISResearcher struct {
	ID           string       `json:"id"`
	UserID       *string      `json:"user_id"`
	FullNameAr   string       `json:"full_name_ar"`
	FullNameEn   string       `json:"full_name_en"`
	PrivateEmail *string      `json:"private_email"`
	PrivatePhone *string      `json:"private_phone"`
	DepartmentID *string      `json:"department_id"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Profiles     []RISProfile `json:"researcher_social_profiles"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, headers map[string]string, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.key)
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ris %s %s: %w", method, strings.SplitN(path, "?", 2)[0], err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		// Only the status reaches the logs: RIS error bodies may echo submitted values.
		return fmt.Errorf("ris %s %s: HTTP %d", method, strings.SplitN(path, "?", 2)[0], resp.StatusCode)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// Researchers reads every RIS researcher with its profile links.
func (c *Client) Researchers(ctx context.Context) ([]RISResearcher, error) {
	var all []RISResearcher
	for offset := 0; ; offset += pageSize {
		q := url.Values{
			"select": {"id,user_id,full_name_ar,full_name_en,private_email,private_phone,department_id,updated_at," +
				"researcher_social_profiles(platform,url,updated_at)"},
			"order":  {"id"},
			"limit":  {strconv.Itoa(pageSize)},
			"offset": {strconv.Itoa(offset)},
		}
		var page []RISResearcher
		if err := c.do(ctx, http.MethodGet, "/rest/v1/researchers?"+q.Encode(), nil, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			return all, nil
		}
	}
}

// Orgs reads the RIS colleges and departments.
func (c *Client) Orgs(ctx context.Context) (colleges, departments []Org, err error) {
	var ds []struct {
		ID        string `json:"id"`
		NameAr    string `json:"name_ar"`
		CollegeID string `json:"college_id"`
	}
	type college struct {
		ID     string `json:"id"`
		NameAr string `json:"name_ar"`
	}
	var cols []college
	if err := c.do(ctx, http.MethodGet, "/rest/v1/colleges?select=id,name_ar", nil, nil, &cols); err != nil {
		return nil, nil, err
	}
	if err := c.do(ctx, http.MethodGet, "/rest/v1/departments?select=id,name_ar,college_id", nil, nil, &ds); err != nil {
		return nil, nil, err
	}
	for _, x := range cols {
		colleges = append(colleges, Org{ID: x.ID, NameAr: x.NameAr})
	}
	for _, x := range ds {
		departments = append(departments, Org{ID: x.ID, NameAr: x.NameAr, CollegeID: x.CollegeID})
	}
	return colleges, departments, nil
}

// UserEmails maps RIS auth user ids to their (lower-case) email addresses.
func (c *Client) UserEmails(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	for page := 1; ; page++ {
		var resp struct {
			Users []struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"users"`
		}
		path := fmt.Sprintf("/auth/v1/admin/users?page=%d&per_page=%d", page, pageSize)
		if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
			return nil, err
		}
		for _, u := range resp.Users {
			out[u.ID] = strings.ToLower(strings.TrimSpace(u.Email))
		}
		if len(resp.Users) < pageSize {
			return out, nil
		}
	}
}

// UpdateResearcher patches columns of one RIS researcher.
func (c *Client) UpdateResearcher(ctx context.Context, id string, fields map[string]any) error {
	return c.do(ctx, http.MethodPatch, "/rest/v1/researchers?id=eq."+url.QueryEscape(id), fields,
		map[string]string{"Prefer": "return=minimal"}, nil)
}

// SetProfile creates or replaces one profile link; an empty url removes it.
func (c *Client) SetProfile(ctx context.Context, researcherID, platform, link string) error {
	if link == "" {
		q := "researcher_id=eq." + url.QueryEscape(researcherID) + "&platform=eq." + url.QueryEscape(platform)
		return c.do(ctx, http.MethodDelete, "/rest/v1/researcher_social_profiles?"+q, nil,
			map[string]string{"Prefer": "return=minimal"}, nil)
	}
	row := map[string]any{"researcher_id": researcherID, "platform": platform, "url": link}
	return c.do(ctx, http.MethodPost, "/rest/v1/researcher_social_profiles?on_conflict=researcher_id,platform", row,
		map[string]string{"Prefer": "resolution=merge-duplicates,return=minimal"}, nil)
}
