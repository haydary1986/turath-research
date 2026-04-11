package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

type contextKey string

const userContextKey contextKey = "user"

// UserClaims represents the JWT payload.
type UserClaims struct {
	jwt.RegisteredClaims
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	InstitutionID string `json:"institution_id,omitempty"`
	CollegeID    string `json:"college_id,omitempty"`
	DepartmentID string `json:"department_id,omitempty"`
}

// RequireAuth validates the JWT token and injects claims into context.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				model.RespondError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				model.RespondError(w, http.StatusUnauthorized, "Bearer token required")
				return
			}

			claims := &UserClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				model.RespondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser extracts the user claims from request context.
func GetUser(ctx context.Context) *UserClaims {
	claims, ok := ctx.Value(userContextKey).(*UserClaims)
	if !ok {
		return nil
	}
	return claims
}
