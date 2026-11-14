package rissync

import "testing"

func TestDecideFirstSync(t *testing.T) {
	cases := []struct {
		name   string
		v      Values
		policy Policy
		want   Action
	}{
		{"same value", Values{Portal: "a", RIS: "a"}, FillGaps, Agree},
		{"both empty", Values{}, FillGaps, Agree},
		{"fill RIS gap", Values{Portal: "a"}, FillGaps, ToRIS},
		{"fill portal gap", Values{RIS: "b"}, FillGaps, ToPortal},
		{"real conflict goes to review", Values{Portal: "a", RIS: "b"}, FillGaps, Review},
		{"official portal value wins", Values{Portal: "أحمد علي حسين", RIS: "م.م. أحمد علي"}, PortalOfficial, ToRIS},
		{"no official value keeps RIS", Values{RIS: "b"}, PortalOfficial, ToPortal},
	}
	for _, c := range cases {
		if got := Decide(c.v, c.policy); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestDecideAfterFirstSync(t *testing.T) {
	last := func(p, r string, portalNewer bool) Values {
		return Values{Portal: p, RIS: r, Last: "x", HasLast: true, PortalNewer: portalNewer}
	}
	cases := []struct {
		name string
		v    Values
		want Action
	}{
		{"unchanged", last("x", "x", false), Nothing},
		{"edited in portal", last("p", "x", false), ToRIS},
		{"edited in RIS", last("x", "r", true), ToPortal},
		{"cleared in portal", last("", "x", false), ToRIS},
		{"same edit on both", last("n", "n", false), Agree},
		{"both edited, portal newer", last("p", "r", true), ToRIS},
		{"both edited, RIS newer", last("p", "r", false), ToPortal},
	}
	for _, c := range cases {
		// The first-sync policy must not matter once a value was agreed.
		for _, policy := range []Policy{PortalOfficial, FillGaps} {
			if got := Decide(c.v, policy); got != c.want {
				t.Errorf("%s (policy %d): got %s, want %s", c.name, policy, got, c.want)
			}
		}
	}
}
