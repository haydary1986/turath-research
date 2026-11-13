package handler

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

// isResearchStaff matches the staffOnly route group and the client's isStaffRole.
func isResearchStaff(role string) bool {
	return role == middleware.RoleAdmin || role == middleware.RoleScientificAffairs || role == middleware.RoleVPResearch
}

// ownResearcherID resolves a user's researcher record, without creating one.
func ownResearcherID(ctx context.Context, db *pgxpool.Pool, userID string) (*string, error) {
	var id *string
	err := db.QueryRow(ctx, `
		SELECT COALESCE(u.researcher_id, (SELECT r.id FROM researchers r WHERE lower(r.email) = lower(u.email)))
		FROM users u WHERE u.id = $1
	`, userID).Scan(&id)
	return id, err
}

// requireActsFor admits staff and the researcher themselves, and answers otherwise.
func requireActsFor(ctx context.Context, w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, researcherID string) bool {
	user := authedUser(w, r)
	if user == nil {
		return false
	}
	if isResearchStaff(user.Role) {
		return true
	}
	own, err := ownResearcherID(ctx, db, user.UserID)
	if err != nil {
		serverError(w, "researcher access check", err)
		return false
	}
	if own == nil || *own != researcherID {
		model.RespondError(w, http.StatusForbidden, "Insufficient permissions")
		return false
	}
	return true
}

// callerIsResearchStaff tells whether personal contact data may be shown to the caller.
func callerIsResearchStaff(r *http.Request) bool {
	user := middleware.GetUser(r.Context())
	return user != nil && isResearchStaff(user.Role)
}
