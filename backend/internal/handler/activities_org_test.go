package handler

import (
	"reflect"
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

func TestResolveActivityDepartment(t *testing.T) {
	const own, other, ownDept, chosenDept = "c-own", "c-other", "d-own", "d-chosen"
	cases := []struct {
		name                string
		college, dept, want string
	}{
		{"no college given inherits", "", "", ownDept},
		{"own college inherits", own, "", ownDept},
		{"own college in another case inherits", "C-OWN", "", ownDept},
		{"another college does not inherit", other, "", ""},
		{"explicit department wins", other, chosenDept, chosenDept},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveActivityDepartment(tc.college, tc.dept, own, ownDept); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
	if got := resolveActivityDepartment("", "", "", ownDept); got != "" {
		t.Errorf("a user without a college has nothing to inherit into: %q", got)
	}
}

func TestActivityReviewers(t *testing.T) {
	if exclude, admins := activityReviewers(portal.RoleScientificAffairs, "u1"); exclude != "u1" || !admins {
		t.Errorf("scientific-affairs author: exclude %q, admins %v", exclude, admins)
	}
	if exclude, admins := activityReviewers("researcher", "u1"); exclude != "" || admins {
		t.Errorf("researcher author: exclude %q, admins %v", exclude, admins)
	}
}

func TestRoleAudienceNumbersPlaceholders(t *testing.T) {
	college := "c1"
	where, args := roleAudience([]any{"a", "b"}, "admin", &college, "u1")
	if want := `role = $3::user_role AND college_id = $4 AND id <> $5`; where != want {
		t.Errorf("where = %q", where)
	}
	if !reflect.DeepEqual(args, []any{"a", "b", "admin", "c1", "u1"}) {
		t.Errorf("args = %v", args)
	}
	if where, _ := roleAudience(nil, "admin", nil, ""); where != `role = $1::user_role` {
		t.Errorf("bare where = %q", where)
	}
}

func TestNormalizeDetailDigits(t *testing.T) {
	in := map[string]any{"issn": "٠٣٧٨-٧٧٨٨", "issn_online": "۱۸۷۲-۶۱۷۸", "volume": "٥"}
	got := normalizeDetailDigits(in)
	if got["issn"] != "0378-7788" || got["issn_online"] != "1872-6178" {
		t.Errorf("ISSNs: %v", got)
	}
	if got["volume"] != "٥" || in["issn"] != "٠٣٧٨-٧٧٨٨" {
		t.Errorf("other fields and the input must stay untouched: %v %v", got, in)
	}
	if normalizeDetailDigits(nil) == nil {
		t.Error("nil details must become an empty object")
	}
}
