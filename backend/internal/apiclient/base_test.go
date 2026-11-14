package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGetRefusesOversizedResponses(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Write([]byte(strings.Repeat("x", 1024))) //nolint:errcheck // test server
	}))
	defer srv.Close()

	client := NewClient(time.Second, 100, "test")
	client.maxBytes = 1023
	if _, err := client.Get(context.Background(), srv.URL, nil); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if requests != 1 {
		t.Errorf("an oversized answer must not be retried, got %d requests", requests)
	}
	client.maxBytes = 1024
	body, err := client.Get(context.Background(), srv.URL, nil)
	if err != nil || len(body) != 1024 {
		t.Fatalf("a response at the limit must pass: %d bytes, %v", len(body), err)
	}
}

func worksServer(t *testing.T, total, pageSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		count := min(pageSize, max(0, total-(page-1)*200))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OpenAlexWorkResult{
			Meta: OpenAlexMeta{Count: total}, Results: make([]OpenAlexWork, count),
		})
	}))
}

func TestListWorksReportsTruncation(t *testing.T) {
	cases := []struct {
		name            string
		total, pageSize int
		maxPages, want  int
		truncated       bool
	}{
		{"fits", 150, 200, 5, 150, false},
		{"exactly full", 400, 200, 2, 400, false},
		{"page limit hit", 1001, 200, 5, 1000, true},
		{"upstream returns less than it counted", 300, 100, 1, 100, true},
		{"none", 0, 200, 5, 0, false},
	}
	for _, tc := range cases {
		srv := worksServer(t, tc.total, tc.pageSize)
		client := NewOpenAlexClient("", "")
		client.baseURL = srv.URL
		works, truncated, err := client.ListWorksByAuthor(context.Background(), "A1", tc.maxPages)
		srv.Close()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(works) != tc.want || truncated != tc.truncated {
			t.Errorf("%s: got %d works, truncated=%v", tc.name, len(works), truncated)
		}
	}
}

func TestORCIDPersonNames(t *testing.T) {
	var person ORCIDPerson
	raw := `{"name":{"given-names":{"value":"Josiah"},"family-name":{"value":"Carberry"},"credit-name":null},
		"other-names":{"other-name":[{"content":"Josiah Stinkney Carberry"}]}}`
	if err := json.Unmarshal([]byte(raw), &person); err != nil {
		t.Fatal(err)
	}
	names := person.Names()
	if len(names) != 2 || names[0] != "Josiah Carberry" || names[1] != "Josiah Stinkney Carberry" {
		t.Errorf("names = %v", names)
	}
	var hidden ORCIDPerson
	if err := json.Unmarshal([]byte(`{"name":null}`), &hidden); err != nil {
		t.Fatal(err)
	}
	if len(hidden.Names()) != 0 {
		t.Error("a hidden name must yield no names")
	}
}

func TestCountWorksByInstitution(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OpenAlexWorkResult{Meta: OpenAlexMeta{Count: 1754}})
	}))
	defer srv.Close()
	client := NewOpenAlexClient("", "")
	client.baseURL = srv.URL
	n, err := client.CountWorksByInstitution(context.Background(), "019vd4365")
	if err != nil || n != 1754 {
		t.Fatalf("count = %d, %v", n, err)
	}
	if gotFilter != "institutions.ror:019vd4365" {
		t.Fatalf("filter = %q", gotFilter)
	}
}
