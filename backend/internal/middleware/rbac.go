package middleware

import (
	"net/http"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

// Role constants matching the database enum.
const (
	RoleAdmin             = "admin"
	RoleVPResearch        = "vp_research"
	RoleDeanResearch      = "dean_research"
	RoleScientificAffairs = "scientific_affairs"
	RoleCollegeDean       = "college_dean"
	RoleCenterDirector    = "center_director"
	RoleDepartmentHead    = "department_head"
	RoleResearcher        = "researcher"
	RoleViewer            = "viewer"
)

// RequireRole restricts access to users with one of the allowed roles.
// Must be used after RequireAuth in the middleware chain.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				model.RespondError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			// Admin always has access.
			if user.Role == RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := roleSet[user.Role]; !ok {
				model.RespondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyOf is an alias for RequireRole for readability.
var RequireAnyOf = RequireRole
