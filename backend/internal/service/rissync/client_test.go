package rissync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendsServiceKeyAndWritesProfiles(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "k" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("missing service key on %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery+" "+r.Header.Get("Prefer")+" "+string(body))
		switch {
		case r.URL.Path == "/rest/v1/researchers" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]RISResearcher{{ID: "r1", FullNameAr: "أ", Profiles: []RISProfile{{Platform: "orcid", URL: "u"}}}})
		case r.URL.Path == "/auth/v1/admin/users":
			_, _ = w.Write([]byte(`{"users":[{"id":"u1","email":"A@UOTURATH.EDU.IQ"}]}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "k")
	ctx := context.Background()
	rs, err := c.Researchers(ctx)
	if err != nil || len(rs) != 1 || rs[0].Profiles[0].Platform != "orcid" {
		t.Fatalf("researchers = %+v, %v", rs, err)
	}
	emails, err := c.UserEmails(ctx)
	if err != nil || emails["u1"] != "a@uoturath.edu.iq" {
		t.Fatalf("emails = %v, %v", emails, err)
	}
	if err := c.SetProfile(ctx, "r1", "scopus", "https://www.scopus.com/authid/detail.uri?authorId=1"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProfile(ctx, "r1", "linkedin", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateResearcher(ctx, "r1", map[string]any{"full_name_ar": "ب"}); err != nil {
		t.Fatal(err)
	}
	log := strings.Join(calls, "\n")
	for _, want := range []string{
		"POST /rest/v1/researcher_social_profiles?on_conflict=researcher_id,platform resolution=merge-duplicates",
		"DELETE /rest/v1/researcher_social_profiles?researcher_id=eq.r1&platform=eq.linkedin",
		`PATCH /rest/v1/researchers?id=eq.r1 return=minimal {"full_name_ar":"ب"}`,
	} {
		if !strings.Contains(log, want) {
			t.Errorf("missing call %q in\n%s", want, log)
		}
	}
}

func TestClientErrorsDoNotLeakBodies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"secret value 07701234567"}`))
	}))
	defer srv.Close()
	err := NewClient(srv.URL, "k").UpdateResearcher(context.Background(), "r1", map[string]any{"x": 1})
	if err == nil || strings.Contains(err.Error(), "0770") || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("err = %v", err)
	}
	if NewClient("", "k") != nil || NewClient("https://x", " ") != nil {
		t.Fatal("unconfigured RIS must yield a nil client")
	}
}
