package middleware

import (
	"testing"
	"time"
)

func TestBucketRefusesBurstAboveLimit(t *testing.T) {
	now := time.Unix(0, 0)
	b := newBucket(3, time.Minute)
	b.now = func() time.Time { return now }
	for i := range 3 {
		if !b.allow("ip") {
			t.Fatalf("request %d refused", i+1)
		}
	}
	if b.allow("ip") {
		t.Fatal("fourth request in the same instant was allowed")
	}
	if !b.allow("other") {
		t.Fatal("keys must not share a bucket")
	}
}

// Steady traffic below the rate must keep flowing even though the key is never idle
// for a whole window (the old limiter starved such callers forever).
func TestBucketRefillsContinuously(t *testing.T) {
	now := time.Unix(0, 0)
	b := newBucket(60, time.Minute)
	b.now = func() time.Time { return now }
	for i := range 60 {
		if !b.allow("ip") {
			t.Fatalf("burst request %d refused", i+1)
		}
	}
	for i := range 120 {
		now = now.Add(time.Second)
		if !b.allow("ip") {
			t.Fatalf("steady request %d refused", i+1)
		}
	}
}

func TestBucketSweepForgetsIdleKeys(t *testing.T) {
	now := time.Unix(0, 0)
	b := newBucket(1, time.Minute)
	b.now = func() time.Time { return now }
	b.allow("ip")
	now = now.Add(3 * time.Minute)
	b.sweep()
	if len(b.visitors) != 0 {
		t.Fatalf("visitors = %d, want 0", len(b.visitors))
	}
}
