package handler

import (
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
)

func TestIsResearchStaff(t *testing.T) {
	staff := map[string]bool{
		middleware.RoleAdmin: true, middleware.RoleScientificAffairs: true, middleware.RoleVPResearch: true,
		middleware.RoleDeanResearch: false, middleware.RoleCollegeDean: false, middleware.RoleDepartmentHead: false,
		middleware.RoleResearcher: false, middleware.RoleViewer: false, "": false,
	}
	for role, want := range staff {
		if isResearchStaff(role) != want {
			t.Errorf("isResearchStaff(%q) = %v", role, !want)
		}
	}
}

func TestCanonicalStoredORCID(t *testing.T) {
	const canonical = "0000-0002-1825-0097"
	stored := func(s string) *string { return &s }
	cases := []struct {
		in   *string
		want string
	}{
		{nil, ""},
		{stored(canonical), canonical},
		{stored("https://orcid.org/" + canonical), canonical},
		{stored("0000-0002-1694-233x"), "0000-0002-1694-233X"},
		{stored("legacy junk"), "legacy junk"},
	}
	for _, tc := range cases {
		if got := canonicalStoredORCID(tc.in); got != tc.want {
			t.Errorf("canonicalStoredORCID = %q, want %q", got, tc.want)
		}
	}
}
