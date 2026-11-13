package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haydary1986/turath-research/backend/internal/service/identity"
)

func serveWith(t *testing.T, lookup IdentityLookup, claims *UserClaims) (int, *UserClaims) {
	t.Helper()
	var seen *UserClaims
	h := RefreshIdentity(lookup)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(WithUser(req.Context(), claims))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, seen
}

func TestRefreshIdentityUsesDatabaseValues(t *testing.T) {
	signed := &UserClaims{UserID: "u1", Email: "a@x", Role: RoleAdmin, CollegeID: "old"}
	lookup := func(context.Context, string) (identity.Current, bool, error) {
		return identity.Current{Role: RoleResearcher, CollegeID: "c2", DepartmentID: "d2"}, true, nil
	}
	code, seen := serveWith(t, lookup, signed)
	if code != http.StatusOK || seen.Role != RoleResearcher || seen.CollegeID != "c2" || seen.DepartmentID != "d2" {
		t.Fatalf("demoted admin must act as researcher: %d %+v", code, seen)
	}
	if seen.UserID != "u1" || seen.Email != "a@x" {
		t.Error("identity fields stay as signed")
	}
	if signed.Role != RoleAdmin {
		t.Error("the signed claims are not mutated")
	}
}

func TestRefreshIdentityRejectsDeletedUsers(t *testing.T) {
	lookup := func(context.Context, string) (identity.Current, bool, error) { return identity.Current{}, false, nil }
	if code, _ := serveWith(t, lookup, &UserClaims{UserID: "gone", Role: RoleAdmin}); code != http.StatusUnauthorized {
		t.Errorf("deleted account: %d", code)
	}
	failing := func(context.Context, string) (identity.Current, bool, error) {
		return identity.Current{}, false, errors.New("db down")
	}
	if code, _ := serveWith(t, failing, &UserClaims{UserID: "u", Role: RoleAdmin}); code != http.StatusServiceUnavailable {
		t.Errorf("lookup failure fails closed: %d", code)
	}
	if code, seen := serveWith(t, nil, &UserClaims{UserID: "u", Role: RoleAdmin}); code != http.StatusOK || seen.Role != RoleAdmin {
		t.Error("without a lookup the signed claims are used")
	}
}
