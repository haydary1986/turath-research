package loginaudit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCutoffIs180Days(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	want := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	if got := Cutoff(now); !got.Equal(want) {
		t.Errorf("cutoff = %v, want %v", got, want)
	}
}

func TestPruneOnceUsesCutoff(t *testing.T) {
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	var seen time.Time
	n, err := PruneOnce(context.Background(), func(ctx context.Context, cutoff time.Time) (int64, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("prune must run with a deadline")
		}
		seen = cutoff
		return 7, nil
	}, now)
	if err != nil || n != 7 || !seen.Equal(Cutoff(now)) {
		t.Errorf("n=%d err=%v cutoff=%v", n, err, seen)
	}
}

func TestRunRetentionRunsAtStartAndOnTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	calls := 0
	done := make(chan struct{})
	go func() {
		RunRetention(ctx, func(context.Context, time.Time) (int64, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 2 {
				cancel()
			}
			return 0, errors.New("a failure is logged, never fatal")
		}, 5*time.Millisecond, time.Now)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("retention loop did not stop with its context")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls < 2 {
		t.Errorf("calls = %d, want an immediate run and a ticked one", calls)
	}
}

func TestSanitize(t *testing.T) {
	e := Sanitize(Event{
		Email:     "  Ahmed@UOTurath.edu.iq ",
		Method:    MethodPassword,
		IP:        "203.0.113.9:51234",
		UserAgent: strings.Repeat("م", 400),
	})
	if e.Email != "ahmed@uoturath.edu.iq" {
		t.Errorf("email = %q", e.Email)
	}
	if e.IP != "203.0.113.9" {
		t.Errorf("ip = %q", e.IP)
	}
	if len([]rune(e.UserAgent)) != MaxUserAgent {
		t.Errorf("user agent kept %d runes", len([]rune(e.UserAgent)))
	}
	if e.Reason != ReasonServerError {
		t.Error("a failure always carries a reason")
	}
	if s := Sanitize(Event{Success: true, Reason: ReasonWrongPassword}); s.Reason != "" {
		t.Error("a success carries no reason")
	}
	if ClientIP("2001:db8::1") != "2001:db8::1" || ClientIP("[2001:db8::1]:443") != "2001:db8::1" {
		t.Error("IPv6 addresses")
	}
}
