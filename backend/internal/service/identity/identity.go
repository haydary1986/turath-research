// Package identity keeps a short-lived per-user copy of the authorization fields
// (role, institution, college, department) read from the database, so a role change
// or a deleted account takes effect within the cache TTL instead of at token expiry,
// without a database query on every request.
package identity

import (
	"context"
	"sync"
	"time"
)

// DefaultTTL bounds how long a change can take to apply when nobody invalidates it.
const DefaultTTL = 60 * time.Second

// pruneAbove bounds memory: past this many cached users, expired entries are dropped.
const pruneAbove = 5000

// Current is what authorization uses, as stored now.
type Current struct {
	Role          string
	InstitutionID string
	CollegeID     string
	DepartmentID  string
}

// Loader reads a user's current fields; found is false for a deleted account.
type Loader func(ctx context.Context, userID string) (cur Current, found bool, err error)

type entry struct {
	cur   Current
	found bool
	at    time.Time
}

// Cache answers from memory for TTL, then reloads. Errors are never cached.
type Cache struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	load    Loader
	entries map[string]entry
}

// NewCache returns a cache over load. A nil load gives a nil cache (no refreshing).
func NewCache(load Loader, ttl time.Duration, now func() time.Time) *Cache {
	if load == nil {
		return nil
	}
	return &Cache{ttl: ttl, now: now, load: load, entries: map[string]entry{}}
}

// Get returns the user's current fields, from memory when fresh.
func (c *Cache) Get(ctx context.Context, userID string) (Current, bool, error) {
	c.mu.Lock()
	e, ok := c.entries[userID]
	now := c.now()
	c.mu.Unlock()
	if ok && now.Sub(e.at) < c.ttl {
		return e.cur, e.found, nil
	}

	cur, found, err := c.load(ctx, userID)
	if err != nil {
		return Current{}, false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= pruneAbove {
		for id, old := range c.entries {
			if now.Sub(old.at) >= c.ttl {
				delete(c.entries, id)
			}
		}
	}
	c.entries[userID] = entry{cur: cur, found: found, at: now}
	return cur, found, nil
}

// Invalidate drops a user's entry so the next request reads the database.
// Safe on a nil cache.
func (c *Cache) Invalidate(userID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}

// Len is the number of cached users.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
