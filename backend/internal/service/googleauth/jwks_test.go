package googleauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func jwksServer(t *testing.T, key *rsa.PublicKey, hits *atomic.Int32) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kid": "kid-1",
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}, {
		"kid": "ec-key", "kty": "EC",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestKeySourceFetchesCachesAndThrottles(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var hits atomic.Int32
	srv := jwksServer(t, &priv.PublicKey, &hits)

	source := NewKeySource()
	source.url = srv.URL

	got, err := source.Key(context.Background(), "kid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 || got.E != priv.E {
		t.Fatal("returned key does not match the published one")
	}

	if _, err := source.Key(context.Background(), "kid-1"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("known key must be served from cache, got %d fetches", hits.Load())
	}

	for i := 0; i < 3; i++ {
		if _, err := source.Key(context.Background(), "missing"); err == nil {
			t.Fatal("unknown key id must fail")
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("unknown ids must not hammer Google, got %d fetches", hits.Load())
	}
}

func TestKeySourceReportsFetchFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	source := NewKeySource()
	source.url = srv.URL
	if _, err := source.Key(context.Background(), "kid-1"); err == nil {
		t.Fatal("expected an error when the key endpoint fails")
	}
}
