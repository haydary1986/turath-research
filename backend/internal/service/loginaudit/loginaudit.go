// Package loginaudit describes sign-in attempts for the admin's login log and keeps
// that log to its retention period. It never sees passwords or tokens: callers pass
// only the attempted email, the method, the outcome and the request's origin.
package loginaudit

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"
	"unicode/utf8"
)

// Sign-in methods.
const (
	MethodPassword = "password"
	MethodGoogle   = "google"
)

// Failure reasons, stored as codes and translated by the admin page.
const (
	ReasonInvalidRequest    = "invalid_request"
	ReasonUnknownUser       = "unknown_user"
	ReasonWrongPassword     = "wrong_password"
	ReasonGoogleOnlyAccount = "google_only_account"
	ReasonDomainNotAllowed  = "domain_not_allowed"
	ReasonAccountConflict   = "account_conflict"
	ReasonInvalidToken      = "invalid_token"
	ReasonNotConfigured     = "not_configured"
	ReasonServerError       = "server_error"
)

// Limits of the stored columns.
const (
	MaxEmail     = 255
	MaxUserAgent = 300
	MaxIP        = 64
)

// RetentionDays is how long login events are kept.
const RetentionDays = 180

// Event is one sign-in attempt. Reason is empty exactly when Success is true.
type Event struct {
	UserID    *string
	Email     string
	Method    string
	Success   bool
	Reason    string
	IP        string
	UserAgent string
}

// Sanitize trims and truncates every free-text field to its column, so a hostile
// client cannot make the insert fail or bloat the table.
func Sanitize(e Event) Event {
	out := e
	out.Email = truncate(strings.ToLower(strings.TrimSpace(e.Email)), MaxEmail)
	out.UserAgent = truncate(strings.TrimSpace(e.UserAgent), MaxUserAgent)
	out.IP = truncate(ClientIP(e.IP), MaxIP)
	if out.Success {
		out.Reason = ""
	} else if out.Reason == "" {
		out.Reason = ReasonServerError
	}
	return out
}

// ClientIP strips the port from a remote address. Behind the RealIP middleware the
// address is already the client's (from X-Real-IP / X-Forwarded-For) and has no port.
func ClientIP(remoteAddr string) string {
	addr := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

// Cutoff is the oldest moment still kept at now.
func Cutoff(now time.Time) time.Time {
	return now.AddDate(0, 0, -RetentionDays)
}

// Pruner deletes events created before the cutoff and reports how many went.
type Pruner func(ctx context.Context, cutoff time.Time) (int64, error)

// PruneOnce applies the retention period once.
func PruneOnce(ctx context.Context, prune Pruner, now time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	return prune(ctx, Cutoff(now))
}

// RunRetention prunes at start and then every interval until ctx ends.
func RunRetention(ctx context.Context, prune Pruner, interval time.Duration, now func() time.Time) {
	run := func() {
		n, err := PruneOnce(ctx, prune, now())
		if err != nil {
			slog.Error("loginaudit: retention failed", "error", err)
			return
		}
		if n > 0 {
			slog.Info("loginaudit: old login events deleted", "count", n)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
