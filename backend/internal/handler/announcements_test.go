package handler

import "testing"

func TestActivityAnnouncementBody(t *testing.T) {
	desc, presenter, location, blank := "ندوة عن الذكاء الاصطناعي", "د. أحمد", "قاعة المؤتمرات", "  "
	cases := []struct {
		name            string
		desc, pres, loc *string
		want            string
	}{
		{"everything", &desc, &presenter, &location, desc + "\n\nالمحاضر: د. أحمد\nالمكان: قاعة المؤتمرات"},
		{"description only", &desc, nil, &blank, desc},
		{"details only", nil, &presenter, nil, "المحاضر: د. أحمد"},
		{"nothing", nil, nil, nil, ""},
	}
	for _, tc := range cases {
		if got := activityAnnouncementBody(tc.desc, tc.pres, tc.loc); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("جامعة", 3); got != "جام" {
		t.Fatalf("got %q", got)
	}
	if got := truncateRunes("abc", 5); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
