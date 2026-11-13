package identity

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type fakeDB struct {
	rows  map[string]Current
	calls int
	err   error
}

func (f *fakeDB) load(_ context.Context, id string) (Current, bool, error) {
	f.calls++
	if f.err != nil {
		return Current{}, false, f.err
	}
	cur, ok := f.rows[id]
	return cur, ok, nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestCacheServesFromMemoryUntilTTL(t *testing.T) {
	db := &fakeDB{rows: map[string]Current{"u1": {Role: "researcher"}}}
	c := &clock{t: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := NewCache(db.load, time.Minute, c.now)

	for i := 0; i < 3; i++ {
		if cur, found, err := cache.Get(context.Background(), "u1"); err != nil || !found || cur.Role != "researcher" {
			t.Fatalf("get: %+v %v %v", cur, found, err)
		}
	}
	if db.calls != 1 {
		t.Errorf("one query per TTL, got %d", db.calls)
	}

	db.rows["u1"] = Current{Role: "college_dean", CollegeID: "c1"}
	c.t = c.t.Add(61 * time.Second)
	if cur, _, _ := cache.Get(context.Background(), "u1"); cur.Role != "college_dean" || cur.CollegeID != "c1" {
		t.Errorf("a role change applies after the TTL: %+v", cur)
	}
}

func TestInvalidateAppliesAtOnce(t *testing.T) {
	db := &fakeDB{rows: map[string]Current{"u1": {Role: "researcher"}}}
	cache := NewCache(db.load, time.Minute, time.Now)
	_, _, _ = cache.Get(context.Background(), "u1")
	db.rows["u1"] = Current{Role: "scientific_affairs"}
	cache.Invalidate("u1")
	if cur, _, _ := cache.Get(context.Background(), "u1"); cur.Role != "scientific_affairs" {
		t.Errorf("invalidated entry reloads: %+v", cur)
	}
	var none *Cache
	none.Invalidate("u1") // no panic
	if NewCache(nil, time.Minute, time.Now) != nil {
		t.Error("no loader, no cache")
	}
}

func TestDeletedUserIsReportedAndErrorsAreNotCached(t *testing.T) {
	db := &fakeDB{rows: map[string]Current{}}
	cache := NewCache(db.load, time.Minute, time.Now)
	if _, found, err := cache.Get(context.Background(), "gone"); err != nil || found {
		t.Errorf("deleted user: found=%v err=%v", found, err)
	}

	db.err = errors.New("database down")
	if _, _, err := cache.Get(context.Background(), "u2"); err == nil {
		t.Fatal("a load error surfaces")
	}
	db.err = nil
	db.rows["u2"] = Current{Role: "researcher"}
	if _, found, err := cache.Get(context.Background(), "u2"); err != nil || !found {
		t.Error("a failed load is retried, not cached")
	}
}

func TestCachePrunesExpiredEntries(t *testing.T) {
	db := &fakeDB{rows: map[string]Current{}}
	c := &clock{t: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := NewCache(db.load, time.Minute, c.now)
	for i := 0; i < pruneAbove; i++ {
		_, _, _ = cache.Get(context.Background(), fmt.Sprint(i))
	}
	c.t = c.t.Add(2 * time.Minute)
	_, _, _ = cache.Get(context.Background(), "fresh")
	if cache.Len() != 1 {
		t.Errorf("expired entries kept: %d", cache.Len())
	}
}
