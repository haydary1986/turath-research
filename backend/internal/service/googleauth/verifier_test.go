package googleauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testClientID = "client-123.apps.googleusercontent.com"
	testDomain   = "uoturath.edu.iq"
)

var fixedNow = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

type fixture struct {
	key      *rsa.PrivateKey
	verifier *Verifier
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := func(_ context.Context, kid string) (*rsa.PublicKey, error) {
		if kid != "kid-1" {
			return nil, errors.New("unknown kid")
		}
		return &key.PublicKey, nil
	}
	v := NewVerifier(testClientID, testDomain, keys)
	v.now = func() time.Time { return fixedNow }
	return fixture{key: key, verifier: v}
}

type claims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HD            string `json:"hd,omitempty"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func validClaims() claims {
	return claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://accounts.google.com",
			Subject:   "1098765",
			Audience:  jwt.ClaimStrings{testClientID},
			IssuedAt:  jwt.NewNumericDate(fixedNow.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(fixedNow.Add(time.Hour)),
		},
		Email:         "Ahmed@UOTurath.edu.iq",
		EmailVerified: true,
		HD:            testDomain,
		Name:          "أحمد محمد",
		Picture:       "https://example.org/p.png",
	}
}

func (f fixture) sign(t *testing.T, c claims, method jwt.SigningMethod, key any) string {
	t.Helper()
	token := jwt.NewWithClaims(method, c)
	token.Header["kid"] = "kid-1"
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestVerifyAcceptsValidUniversityToken(t *testing.T) {
	f := newFixture(t)
	id, err := f.verifier.Verify(context.Background(), f.sign(t, validClaims(), jwt.SigningMethodRS256, f.key))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Email != "ahmed@uoturath.edu.iq" || id.Subject != "1098765" || id.Name != "أحمد محمد" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestVerifyAcceptsLegacyIssuer(t *testing.T) {
	f := newFixture(t)
	c := validClaims()
	c.Issuer = "accounts.google.com"
	if _, err := f.verifier.Verify(context.Background(), f.sign(t, c, jwt.SigningMethodRS256, f.key)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyRejects(t *testing.T) {
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		mutate  func(*claims)
		key     func(fixture) any
		method  jwt.SigningMethod
		wantErr error
	}{
		{"wrong audience", func(c *claims) { c.Audience = jwt.ClaimStrings{"other-client"} }, nil, nil, ErrInvalidToken},
		{"expired", func(c *claims) { c.ExpiresAt = jwt.NewNumericDate(fixedNow.Add(-time.Hour)) }, nil, nil, ErrInvalidToken},
		{"missing expiry", func(c *claims) { c.ExpiresAt = nil }, nil, nil, ErrInvalidToken},
		{"wrong issuer", func(c *claims) { c.Issuer = "https://evil.example" }, nil, nil, ErrInvalidToken},
		{"email not verified", func(c *claims) { c.EmailVerified = false }, nil, nil, ErrInvalidToken},
		{"missing subject", func(c *claims) { c.Subject = "" }, nil, nil, ErrInvalidToken},
		{"gmail address", func(c *claims) { c.Email = "someone@gmail.com"; c.HD = "" }, nil, nil, ErrDomainNotAllowed},
		{"lookalike domain", func(c *claims) { c.Email = "a@evil-uoturath.edu.iq"; c.HD = "evil-uoturath.edu.iq" }, nil, nil, ErrDomainNotAllowed},
		{"subdomain", func(c *claims) { c.Email = "a@x.uoturath.edu.iq"; c.HD = "x.uoturath.edu.iq" }, nil, nil, ErrDomainNotAllowed},
		{"email domain ok but hd missing", func(c *claims) { c.HD = "" }, nil, nil, ErrDomainNotAllowed},
		{"email domain ok but hd differs", func(c *claims) { c.HD = "other.edu" }, nil, nil, ErrDomainNotAllowed},
		{"signed by another key", nil, func(fixture) any { return other }, nil, ErrInvalidToken},
		{"wrong algorithm", nil, func(fixture) any { return []byte("secret") }, jwt.SigningMethodHS256, ErrInvalidToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			c := validClaims()
			if tc.mutate != nil {
				tc.mutate(&c)
			}
			var key any = f.key
			if tc.key != nil {
				key = tc.key(f)
			}
			method := jwt.SigningMethod(jwt.SigningMethodRS256)
			if tc.method != nil {
				method = tc.method
			}
			_, err := f.verifier.Verify(context.Background(), f.sign(t, c, method, key))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	f := newFixture(t)
	for _, raw := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := f.verifier.Verify(context.Background(), raw); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("Verify(%q) = %v, want ErrInvalidToken", raw, err)
		}
	}
}

func TestVerifyRejectsUnknownKeyID(t *testing.T) {
	f := newFixture(t)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims())
	token.Header["kid"] = "other-kid"
	signed, err := token.SignedString(f.key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.verifier.Verify(context.Background(), signed); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("got %v, want ErrInvalidToken", err)
	}
}
