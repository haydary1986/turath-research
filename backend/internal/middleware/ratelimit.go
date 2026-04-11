package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	lastSeen time.Time
	tokens   int
}

// RateLimit returns a simple in-memory token-bucket rate limiter.
// maxRequests per window per IP.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// Cleanup stale entries every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 2*window {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				v = &visitor{tokens: maxRequests}
				visitors[ip] = v
			}

			// Refill tokens based on elapsed time.
			elapsed := time.Since(v.lastSeen)
			refill := int(elapsed / window) * maxRequests
			v.tokens += refill
			if v.tokens > maxRequests {
				v.tokens = maxRequests
			}
			v.lastSeen = time.Now()

			if v.tokens <= 0 {
				mu.Unlock()
				http.Error(w, `{"success":false,"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			v.tokens--
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
