package handler

import (
	"context"
	"log/slog"
)

// linkImportedResearcherSQL ties a Google-verified account to the imported researcher
// record with the same university email: researcher_id, department and its college, and
// the institution. Values an admin already set on the account are kept. A brand-new
// account ($2) also takes the official names from the record, since Google's display
// name is often an English or nickname form. A record already linked to another account
// is left alone.
const linkImportedResearcherSQL = `
	UPDATE users u SET
		researcher_id  = r.id,
		department_id  = COALESCE(u.department_id, r.department_id),
		college_id     = COALESCE(u.college_id, d.college_id),
		institution_id = COALESCE(u.institution_id, (SELECT id FROM institutions ORDER BY created_at LIMIT 1)),
		name_ar        = CASE WHEN $2 AND COALESCE(btrim(r.name_ar), '') <> '' THEN r.name_ar ELSE u.name_ar END,
		name_en        = COALESCE(NULLIF(btrim(u.name_en), ''), NULLIF(btrim(r.name_en), ''))
	FROM researchers r
	LEFT JOIN departments d ON d.id = r.department_id
	WHERE u.id = $1
	  AND u.researcher_id IS NULL
	  AND lower(r.email) = lower(u.email)
	  AND NOT EXISTS (SELECT 1 FROM users o WHERE o.researcher_id = r.id AND o.id <> u.id)`

// linkImportedResearcher runs linkImportedResearcherSQL for a user who just signed in with
// Google. It reports whether a link was made; failures are logged, never fatal to sign-in.
func (h *AuthHandler) linkImportedResearcher(ctx context.Context, userID string, created bool) bool {
	tag, err := h.db.Exec(ctx, linkImportedResearcherSQL, userID, created)
	if err != nil {
		slog.Error("google sign-in: failed to link the imported researcher record", "user_id", userID, "error", err)
		return false
	}
	return tag.RowsAffected() > 0
}
