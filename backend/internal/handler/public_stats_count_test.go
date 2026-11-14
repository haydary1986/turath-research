package handler

import (
	"context"
	"errors"
	"testing"
)

func TestPublishedCountTakesTheLargerTotal(t *testing.T) {
	cases := []struct {
		name            string
		local, external int64
		want            int64
	}{
		{"index knows more research", 990, 1754, 1754},
		{"portal knows more research", 2000, 1754, 2000},
		{"index unavailable", 990, -1, 990},
	}
	for _, c := range cases {
		if got := publishedCount(c.local, c.external); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestCountInstitutionWorksFallsBack(t *testing.T) {
	if got := (&PublicHandler{}).countInstitutionWorks(context.Background()); got != -1 {
		t.Fatalf("no counter: got %d", got)
	}
	failing := &PublicHandler{institutionWorks: func(context.Context) (int, error) { return 0, errors.New("down") }}
	if got := failing.countInstitutionWorks(context.Background()); got != -1 {
		t.Fatalf("failing counter: got %d", got)
	}
	ok := &PublicHandler{institutionWorks: func(context.Context) (int, error) { return 1754, nil }}
	if got := ok.countInstitutionWorks(context.Background()); got != 1754 {
		t.Fatalf("working counter: got %d", got)
	}
}

func TestDashboardTotalUsesInstitutionCount(t *testing.T) {
	calls := 0
	h := NewMetricsHandler(nil, func(context.Context) (int, error) { calls++; return 1754, nil })
	if got := h.institutionTotal(context.Background()); got != 1754 {
		t.Fatalf("got %d", got)
	}
	h.institutionTotal(context.Background())
	if calls != 1 {
		t.Fatalf("OpenAlex asked %d times, want 1 (cached)", calls)
	}
	if got := NewMetricsHandler(nil, nil).institutionTotal(context.Background()); got != -1 {
		t.Fatalf("no counter: got %d", got)
	}
}
