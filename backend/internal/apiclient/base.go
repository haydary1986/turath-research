package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// maxResponseBytes bounds one upstream answer. The largest legitimate one is a
// 200-work OpenAlex page with abstracts or a reference-list CSV, both far below this.
const maxResponseBytes = 32 << 20

// ErrResponseTooLarge means an upstream sent more than the client will read.
var ErrResponseTooLarge = errors.New("upstream response too large")

// ErrRateLimited means an upstream kept answering 429 through every retry.
var ErrRateLimited = errors.New("rate limited (429)")

// StatusError is a non-retryable 4xx answer, so callers can tell "the upstream
// has no such record" (404) from "the upstream could not answer".
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("client error %d: %s", e.Code, e.Body)
}

// IsNotFound reports whether err is an upstream 404.
func IsNotFound(err error) bool {
	var status *StatusError
	return errors.As(err, &status) && status.Code == http.StatusNotFound
}

// withoutQuery drops the query string, which may carry an API key, from a URL
// about to be logged.
func withoutQuery(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unparseable-url"
	}
	u.RawQuery = ""
	return u.String()
}

// Client is a shared HTTP client with retry, rate limiting, and circuit breaker.
type Client struct {
	http           *http.Client
	rateLimiter    *rateLimiter
	circuitBreaker *circuitBreaker
	userAgent      string
	maxBytes       int64
}

// NewClient creates a base API client.
func NewClient(timeout time.Duration, maxRPS int, userAgent string) *Client {
	return &Client{
		http:           &http.Client{Timeout: timeout},
		rateLimiter:    newRateLimiter(maxRPS),
		circuitBreaker: newCircuitBreaker(5, 30*time.Second),
		userAgent:      userAgent,
		maxBytes:       maxResponseBytes,
	}
}

// Get performs a GET request with retry, rate limiting, and circuit breaker.
func (c *Client) Get(ctx context.Context, rawURL string, headers map[string]string) ([]byte, error) {
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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
			// *url.Error embeds the full URL; keep only its cause so keys stay out of logs.
			var urlErr *url.Error
			if errors.As(err, &urlErr) {
				err = urlErr.Err
			}
			lastErr = fmt.Errorf("request failed: %w", err)
			slog.Warn("api request failed, retrying", "url", withoutQuery(rawURL), "attempt", attempt+1, "error", err)
			continue
		}

		body, err := readBounded(resp.Body, c.maxBytes)
		resp.Body.Close()
		if errors.Is(err, ErrResponseTooLarge) {
			return nil, err // the same upstream would answer the same way, so do not retry
		}
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			slog.Warn("rate limited by upstream", "url", withoutQuery(rawURL))
			lastErr = ErrRateLimited
			continue
		}

		if resp.StatusCode >= 500 {
			c.circuitBreaker.recordFailure()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, &StatusError{Code: resp.StatusCode, Body: string(body[:min(200, len(body))])}
		}

		c.circuitBreaker.recordSuccess()
		return body, nil
	}

	c.circuitBreaker.recordFailure()
	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

func readBounded(body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}

// GetJSON performs a GET and unmarshals JSON response.
func (c *Client) GetJSON(ctx context.Context, rawURL string, headers map[string]string, target any) error {
	body, err := c.Get(ctx, rawURL, headers)
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
