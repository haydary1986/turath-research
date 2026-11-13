package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	lastSeen time.Time
	tokens   float64
}

// bucket is a per-key token bucket refilled continuously at limit tokens per window.
type bucket struct {
	mu       sync.Mutex
	limit    float64
	window   time.Duration
	now      func() time.Time
	visitors map[string]*visitor
}

func newBucket(limit int, window time.Duration) *bucket {
	return &bucket{limit: float64(limit), window: window, now: time.Now, visitors: map[string]*visitor{}}
}

// allow spends one token of key and reports whether one was available. Tokens refill
// in proportion to the time since the key's last request, so steady traffic below the
// rate is never refused (the previous version only refilled after a full idle window).
func (b *bucket) allow(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	v, exists := b.visitors[key]
	if !exists {
		v = &visitor{tokens: b.limit, lastSeen: now}
		b.visitors[key] = v
	}
	elapsed := now.Sub(v.lastSeen)
	v.tokens = min(b.limit, v.tokens+b.limit*elapsed.Seconds()/b.window.Seconds())
	v.lastSeen = now
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

// sweep forgets keys idle for more than two windows (their bucket would be full anyway).
func (b *bucket) sweep() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for key, v := range b.visitors {
		if b.now().Sub(v.lastSeen) > 2*b.window {
			delete(b.visitors, key)
		}
	}
}

// RateLimit returns an in-memory token-bucket rate limiter:
// maxRequests per window per IP, refilled continuously.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	b := newBucket(maxRequests, window)

	// Cleanup stale entries every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			b.sweep()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}
			if !b.allow(ip) {
				http.Error(w, `{"success":false,"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
