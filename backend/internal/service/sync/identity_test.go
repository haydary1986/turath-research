package sync

import (
	"encoding/json"
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
)

func employments(t *testing.T, raw string) *apiclient.ORCIDEmploymentGroup {
	t.Helper()
	var group apiclient.ORCIDEmploymentGroup
	if err := json.Unmarshal([]byte(raw), &group); err != nil {
		t.Fatal(err)
	}
	return &group
}

func TestListsUniversity(t *testing.T) {
	wrap := func(org string) string {
		return `{"affiliation-group":[{"summaries":[{"employment-summary":{"organization":` + org + `}}]}]}`
	}
	cases := map[string]struct {
		org  string
		want bool
	}{
		"latin name":  {`{"name":"Al-Turath University"}`, true},
		"arabic name": {`{"name":"جامعة التراث"}`, true},
		"ror id":      {`{"name":"X","disambiguated-organization":{"disambiguated-organization-identifier":"https://ror.org/019vd4365"}}`, true},
		"other":       {`{"name":"University of Baghdad"}`, false},
	}
	for name, tc := range cases {
		if got := listsUniversity(employments(t, wrap(tc.org))); got != tc.want {
			t.Errorf("%s: got %v", name, got)
		}
	}
	if listsUniversity(nil) {
		t.Error("nil must be false")
	}
}
