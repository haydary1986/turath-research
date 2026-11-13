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
// users.researcher_id (set by staff, by import linking, or by the portal for a record
// the user's own request created) always counts. A plain email match counts only for
// an account whose email is verified, i.e. one that signed in with Google: anyone
// could register a password account under a colleague's address.
func ownResearcherID(ctx context.Context, db *pgxpool.Pool, userID string) (*string, error) {
	var id *string
	err := db.QueryRow(ctx, `
		SELECT COALESCE(u.researcher_id, CASE WHEN `+verifiedEmailSQL+` THEN
		       (SELECT r.id FROM researchers r WHERE lower(r.email) = lower(u.email)) END)
		FROM users u WHERE u.id = $1
	`, userID).Scan(&id)
	return id, err
}

// verifiedEmailSQL is true for a users row (alias u) whose email Google has verified.
const verifiedEmailSQL = `u.google_sub IS NOT NULL`

// mayLinkByEmail decides whether a researcher record found or created for the applicant's
// email may be tied to their account (users.researcher_id). A record the applicant's own
// request just created is theirs; an existing one is claimed only with a verified email.
func mayLinkByEmail(emailVerified, createdForApplicant bool) bool {
	return emailVerified || createdForApplicant
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
