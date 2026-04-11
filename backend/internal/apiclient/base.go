package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Client is a shared HTTP client with retry, rate limiting, and circuit breaker.
type Client struct {
	http         *http.Client
	rateLimiter  *rateLimiter
	circuitBreaker *circuitBreaker
	userAgent    string
}

// NewClient creates a base API client.
func NewClient(timeout time.Duration, maxRPS int, userAgent string) *Client {
	return &Client{
		http: &http.Client{Timeout: timeout},
		rateLimiter:    newRateLimiter(maxRPS),
		circuitBreaker: newCircuitBreaker(5, 30*time.Second),
		userAgent:      userAgent,
	}
}

// Get performs a GET request with retry, rate limiting, and circuit breaker.
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	if err := c.circuitBreaker.allow(); err != nil {
		return nil, fmt.Errorf("circuit breaker open: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		c.rateLimiter.wait(ctx)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			slog.Warn("api request failed, retrying", "url", url, "attempt", attempt+1, "error", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			slog.Warn("rate limited by upstream", "url", url)
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode >= 500 {
			c.circuitBreaker.recordFailure()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("client error %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
		}

		c.circuitBreaker.recordSuccess()
		return body, nil
	}

	c.circuitBreaker.recordFailure()
	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

// GetJSON performs a GET and unmarshals JSON response.
func (c *Client) GetJSON(ctx context.Context, url string, headers map[string]string, target any) error {
	body, err := c.Get(ctx, url, headers)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// ── Rate Limiter (token bucket) ──────────────────

type rateLimiter struct {
	mu       sync.Mutex
	tokens   int
	maxRPS   int
	lastTime time.Time
}

func newRateLimiter(maxRPS int) *rateLimiter {
	return &rateLimiter{tokens: maxRPS, maxRPS: maxRPS, lastTime: time.Now()}
}

func (rl *rateLimiter) wait(ctx context.Context) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime)
	rl.tokens += int(elapsed.Seconds()) * rl.maxRPS
	if rl.tokens > rl.maxRPS {
		rl.tokens = rl.maxRPS
	}
	rl.lastTime = now

	if rl.tokens <= 0 {
		wait := time.Second / time.Duration(rl.maxRPS)
		rl.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
		rl.mu.Lock()
		rl.tokens = 1
	}
	rl.tokens--
}

// ── Circuit Breaker ──────────────────────────────

type circuitBreaker struct {
	mu           sync.Mutex
	failures     int
	threshold    int
	resetTimeout time.Duration
	openUntil    time.Time
}

func newCircuitBreaker(threshold int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{threshold: threshold, resetTimeout: resetTimeout}
}

func (cb *circuitBreaker) allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures >= cb.threshold && time.Now().Before(cb.openUntil) {
		return fmt.Errorf("circuit breaker open, resets at %s", cb.openUntil.Format(time.RFC3339))
	}
	if cb.failures >= cb.threshold {
		cb.failures = 0 // half-open: allow one request
	}
	return nil
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.threshold {
		cb.openUntil = time.Now().Add(cb.resetTimeout)
	}
}
