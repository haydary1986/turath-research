// Package googleauth verifies Google Sign-In ID tokens and restricts access to one institutional domain.
package googleauth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken means the token is malformed, badly signed, expired or not meant for this app.
	ErrInvalidToken = errors.New("invalid google token")
	// ErrDomainNotAllowed means the token is genuine but belongs to an account outside the university domain.
	ErrDomainNotAllowed = errors.New("account is not in the allowed domain")
)

const clockLeeway = 30 * time.Second

var allowedIssuers = map[string]struct{}{
	"https://accounts.google.com": {},
	"accounts.google.com":         {},
}

// KeyFunc returns Google's public signing key for a key id.
type KeyFunc func(ctx context.Context, kid string) (*rsa.PublicKey, error)

// Identity is the verified Google account of the signed-in person.
type Identity struct {
	Subject string
	Email   string
	Name    string
	Picture string
}

// Verifier validates ID tokens issued for one OAuth client and one hosted domain.
type Verifier struct {
	clientID string
	domain   string
	keys     KeyFunc
	now      func() time.Time
}

// NewVerifier builds a verifier. domain is matched case-insensitively and exactly (no subdomains).
func NewVerifier(clientID, domain string, keys KeyFunc) *Verifier {
	return &Verifier{
		clientID: clientID,
		domain:   strings.ToLower(strings.TrimSpace(domain)),
		keys:     keys,
		now:      time.Now,
	}
}

type tokenClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HostedDomain  string `json:"hd"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Verify checks signature, audience, issuer, expiry, verified email and the university domain.
// The email domain and Google's hosted-domain (hd) claim must both match, so a personal
// Google account can never pass by merely using a look-alike address.
func (v *Verifier) Verify(ctx context.Context, raw string) (Identity, error) {
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing key id")
		}
		return v.keys(ctx, kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(v.clientID),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockLeeway),
		jwt.WithTimeFunc(v.now),
	)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if _, ok := allowedIssuers[claims.Issuer]; !ok {
		return Identity{}, fmt.Errorf("%w: unexpected issuer", ErrInvalidToken)
	}
	if claims.Subject == "" || claims.Email == "" || !claims.EmailVerified {
		return Identity{}, fmt.Errorf("%w: incomplete or unverified account", ErrInvalidToken)
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if emailDomain(email) != v.domain || strings.ToLower(claims.HostedDomain) != v.domain {
		return Identity{}, ErrDomainNotAllowed
	}

	return Identity{
		Subject: claims.Subject,
		Email:   email,
		Name:    strings.TrimSpace(claims.Name),
		Picture: claims.Picture,
	}, nil
}

func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return ""
	}
	return email[at+1:]
}
