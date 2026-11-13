package journalcheck

import (
	"sync"
	"sync/atomic"
	"time"
)

// UserThrottle counts uncached checks per user in a fixed window. It is in memory:
// the backend runs as one process, and a restart merely forgives a window.
type UserThrottle struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	users  map[string]*throttleWindow
}

type throttleWindow struct {
	startedAt time.Time
	used      int
}

// NewUserThrottle allows limit uses per window for each user.
func NewUserThrottle(limit int, window time.Duration) *UserThrottle {
	return &UserThrottle{limit: limit, window: window, now: time.Now, users: map[string]*throttleWindow{}}
}

// Allow spends one use and reports whether the user still had any.
func (t *UserThrottle) Allow(user string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.forgetExpired(now)
	current, ok := t.users[user]
	if !ok {
		current = &throttleWindow{startedAt: now}
		t.users[user] = current
	}
	if current.used >= t.limit {
		return false
	}
	current.used++
	return true
}

func (t *UserThrottle) forgetExpired(now time.Time) {
	for user, current := range t.users {
		if now.Sub(current.startedAt) >= t.window {
			delete(t.users, user)
		}
	}
}

// externalAccess decides once per check whether uncached lookups may go out. The
// sources run concurrently, so the first miss spends the unit and the rest reuse it.
type externalAccess struct {
	once       sync.Once
	cachedOnly bool
	allow      func() bool
	permitted  bool
	throttled  atomic.Bool
}

func newExternalAccess(q Query) *externalAccess {
	return &externalAccess{cachedOnly: q.CachedOnly, allow: q.AllowExternal}
}

func (a *externalAccess) allowed() bool {
	if a.cachedOnly {
		return false
	}
	if a.allow == nil {
		return true
	}
	a.once.Do(func() {
		a.permitted = a.allow()
		a.throttled.Store(!a.permitted)
	})
	return a.permitted
}
