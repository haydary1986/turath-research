// Package presence records when a user was last seen without slowing requests:
// writes are throttled per user and handed to one background worker, and are
// dropped (to be retried on a later request) rather than ever blocking a request.
package presence

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// DefaultInterval is the minimum gap between two writes for one user.
	DefaultInterval = time.Minute
	queueSize       = 256
	writeTimeout    = 2 * time.Second
	// pruneAbove bounds the throttle's memory: past this many users, stale entries go.
	pruneAbove = 5000
)

// Throttle decides whether a user's presence is due for a write.
type Throttle struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	last     map[string]time.Time
}

// NewThrottle allows one write per user per interval.
func NewThrottle(interval time.Duration, now func() time.Time) *Throttle {
	return &Throttle{interval: interval, now: now, last: map[string]time.Time{}}
}

// Allow reports whether userID is due and, if so, marks it written now.
func (t *Throttle) Allow(userID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if last, ok := t.last[userID]; ok && now.Sub(last) < t.interval {
		return false
	}
	if len(t.last) >= pruneAbove {
		t.pruneLocked(now)
	}
	t.last[userID] = now
	return true
}

// Forget clears a user's mark so the next request tries again (used when a write was dropped).
func (t *Throttle) Forget(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.last, userID)
}

// Len is the number of users currently remembered.
func (t *Throttle) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.last)
}

func (t *Throttle) pruneLocked(now time.Time) {
	for id, at := range t.last {
		if now.Sub(at) >= t.interval {
			delete(t.last, id)
		}
	}
}

// Toucher stores "seen now" for one user.
type Toucher func(ctx context.Context, userID string) error

// Tracker queues throttled presence writes for a single background worker.
type Tracker struct {
	throttle *Throttle
	queue    chan string
	touch    Toucher
}

// NewTracker starts the worker. A nil toucher gives a tracker that records nothing.
func NewTracker(touch Toucher, interval time.Duration) *Tracker {
	t := &Tracker{throttle: NewThrottle(interval, time.Now), queue: make(chan string, queueSize), touch: touch}
	if touch != nil {
		go t.work()
	}
	return t
}

// Seen notes a request by userID. It never blocks: a full queue drops the write.
func (t *Tracker) Seen(userID string) {
	if t == nil || t.touch == nil || userID == "" || !t.throttle.Allow(userID) {
		return
	}
	select {
	case t.queue <- userID:
	default:
		t.throttle.Forget(userID)
	}
}

func (t *Tracker) work() {
	for id := range t.queue {
		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		if err := t.touch(ctx, id); err != nil {
			slog.Warn("presence: last-seen update failed", "error", err)
			t.throttle.Forget(id)
		}
		cancel()
	}
}
