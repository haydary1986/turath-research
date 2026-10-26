package googleauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

const (
	googleCertsURL    = "https://www.googleapis.com/oauth2/v3/certs"
	keyCacheTTL       = time.Hour
	minRefreshSpacing = time.Minute
	maxCertsBodyBytes = 1 << 20
)

// KeySource fetches and caches Google's signing keys. An unknown key id triggers at most one refresh per minute.
type KeySource struct {
	url    string
	client *http.Client

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	fetchedAt   time.Time
	lastAttempt time.Time
}

// NewKeySource returns a source backed by Google's public certificate endpoint.
func NewKeySource() *KeySource {
	return &KeySource{url: googleCertsURL, client: &http.Client{Timeout: 5 * time.Second}}
}

// Key implements KeyFunc.
func (s *KeySource) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if key, ok := s.fresh(kid); ok {
		return key, nil
	}
	if time.Since(s.lastAttempt) >= minRefreshSpacing || len(s.keys) == 0 {
		s.lastAttempt = time.Now()
		if err := s.refresh(ctx); err != nil {
			return nil, err
		}
	}
	if key, ok := s.keys[kid]; ok {
		return key, nil
	}
	return nil, errors.New("unknown signing key")
}

func (s *KeySource) fresh(kid string) (*rsa.PublicKey, bool) {
	if time.Since(s.fetchedAt) > keyCacheTTL {
		return nil, false
	}
	key, ok := s.keys[kid]
	return key, ok
}

type jwkSet struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (s *KeySource) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch google keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch google keys: status %d", resp.StatusCode)
	}

	var set jwkSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxCertsBodyBytes)).Decode(&set); err != nil {
		return fmt.Errorf("decode google keys: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		key, err := parseRSAKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = key
	}
	if len(keys) == 0 {
		return errors.New("google returned no usable signing keys")
	}
	s.keys, s.fetchedAt = keys, time.Now()
	return nil
}

func parseRSAKey(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	exp := new(big.Int).SetBytes(eBytes)
	if !exp.IsInt64() || exp.Int64() <= 1 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(exp.Int64())}, nil
}
