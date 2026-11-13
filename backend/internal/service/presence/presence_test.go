package presence

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestThrottleOncePerInterval(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	th := NewThrottle(time.Minute, c.now)
	if !th.Allow("u1") {
		t.Fatal("first request writes")
	}
	c.t = c.t.Add(30 * time.Second)
	if th.Allow("u1") {
		t.Error("second request within the minute is skipped")
	}
	if !th.Allow("u2") {
		t.Error("users are throttled independently")
	}
	c.t = c.t.Add(31 * time.Second)
	if !th.Allow("u1") {
		t.Error("after a minute the user is due again")
	}
	th.Forget("u1")
	if !th.Allow("u1") {
		t.Error("a forgotten user is due at once")
	}
}

func TestThrottlePrunesStaleEntries(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	th := NewThrottle(time.Minute, c.now)
	for i := 0; i < pruneAbove; i++ {
		th.Allow(fmt.Sprint(i))
	}
	c.t = c.t.Add(2 * time.Minute)
	th.Allow("fresh")
	if th.Len() != 1 {
		t.Errorf("stale entries kept: %d", th.Len())
	}
}

func TestTrackerWritesInBackground(t *testing.T) {
	var mu sync.Mutex
	got := map[string]int{}
	done := make(chan struct{}, 10)
	tr := NewTracker(func(_ context.Context, id string) error {
		mu.Lock()
		got[id]++
		mu.Unlock()
		done <- struct{}{}
		return nil
	}, time.Minute)
	tr.Seen("u1")
	tr.Seen("u1") // throttled
	tr.Seen("u2")
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not write")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if got["u1"] != 1 || got["u2"] != 1 {
		t.Errorf("writes = %v", got)
	}
}

func TestTrackerNeverBlocks(t *testing.T) {
	block := make(chan struct{})
	tr := NewTracker(func(context.Context, string) error { <-block; return nil }, time.Minute)
	defer close(block)
	start := time.Now()
	for i := 0; i < queueSize*3; i++ {
		tr.Seen(fmt.Sprint(i))
	}
	if time.Since(start) > time.Second {
		t.Error("Seen blocked behind a stuck writer")
	}
	var nilTracker *Tracker
	nilTracker.Seen("u") // no panic
	NewTracker(nil, time.Minute).Seen("u")
}
