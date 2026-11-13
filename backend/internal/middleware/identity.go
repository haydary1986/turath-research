package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/identity"
)

// IdentityLookup returns a user's current authorization fields; found is false for a deleted account.
type IdentityLookup func(ctx context.Context, userID string) (identity.Current, bool, error)

// RefreshIdentity replaces the role and organization carried in the JWT with the
// database's current values (through a short per-user cache), so a role change or a
// deleted account applies without waiting for the token to expire. The token format is
// unchanged. A nil lookup leaves the claims as signed. Must be used after RequireAuth.
func RefreshIdentity(lookup IdentityLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUser(r.Context())
			if lookup == nil || claims == nil {
				next.ServeHTTP(w, r)
				return
			}
			cur, found, err := lookup(r.Context(), claims.UserID)
			if err != nil {
				slog.Error("auth: identity refresh failed", "error", err)
				model.RespondError(w, http.StatusServiceUnavailable, "auth_unavailable")
				return
			}
			if !found {
				model.RespondError(w, http.StatusUnauthorized, "account_not_found")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), Refreshed(claims, cur))))
		})
	}
}

// Refreshed returns a copy of the claims carrying the current values; the original is untouched.
func Refreshed(claims *UserClaims, cur identity.Current) *UserClaims {
	updated := *claims
	updated.Role = cur.Role
	updated.InstitutionID = cur.InstitutionID
	updated.CollegeID = cur.CollegeID
	updated.DepartmentID = cur.DepartmentID
	return &updated
}

// WithUser stores claims in the context (as RequireAuth does).
func WithUser(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}
