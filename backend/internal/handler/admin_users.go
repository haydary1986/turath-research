package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

// The system runs with six roles; the legacy enum values (dean_research,
// center_director, viewer) are kept in the database type but no longer assignable.
var assignableRoles = map[string]struct{}{
	middleware.RoleAdmin:             {},
	middleware.RoleVPResearch:        {},
	middleware.RoleScientificAffairs: {},
	middleware.RoleCollegeDean:       {},
	middleware.RoleDepartmentHead:    {},
	middleware.RoleResearcher:        {},
}

// AdminUserHandler lets administrators assign roles and organizational units.
type AdminUserHandler struct {
	db            *pgxpool.Pool
	invalidate    func(userID string) // drops the user's cached role so a change applies at once
	allowedDomain string              // new accounts must use this university email domain
}

func NewAdminUserHandler(db *pgxpool.Pool, invalidate func(userID string), allowedDomain string) *AdminUserHandler {
	return &AdminUserHandler{db: db, invalidate: invalidate, allowedDomain: allowedDomain}
}

type adminUser struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	NameAr       *string `json:"name_ar"`
	Role         string  `json:"role"`
	CollegeID    *string `json:"college_id"`
	DepartmentID *string `json:"department_id"`
	// Responsibilities counts the user's data-entry responsibilities («مسؤوليات الإدخال»).
	Responsibilities int `json:"responsibilities_count"`
}

// List returns users, optionally filtered by name or email.
func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 20, 1, 100)
	pattern := "%" + likeEscaper.Replace(strings.TrimSpace(q.Get("q"))) + "%"

	ctx, cancel := portalContext(r)
	defer cancel()

	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email ILIKE $1 OR name_ar ILIKE $1`, pattern).Scan(&total); err != nil {
		serverError(w, "Failed to count users", err)
		return
	}
	rows, err := h.db.Query(ctx, `
		SELECT id, email, name_ar, role::text, college_id, department_id,
		       (SELECT COUNT(*) FROM entry_responsibilities e WHERE e.user_id = users.id)
		FROM users WHERE email ILIKE $1 OR name_ar ILIKE $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, pattern, limit, (page-1)*limit)
	if err != nil {
		serverError(w, "Failed to list users", err)
		return
	}
	defer rows.Close()

	items := []adminUser{}
	for rows.Next() {
		var u adminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.NameAr, &u.Role, &u.CollegeID, &u.DepartmentID, &u.Responsibilities); err != nil {
			serverError(w, "Failed to read users", err)
			return
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read users", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}

// Update changes a user's role and college/department. Administrators cannot change their own role.
func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	caller := authedUser(w, r)
	if caller == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	var in struct {
		Role         string `json:"role"`
		CollegeID    string `json:"college_id"`
		DepartmentID string `json:"department_id"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	if _, ok := assignableRoles[in.Role]; !ok {
		model.RespondError(w, http.StatusBadRequest, "Unknown role")
		return
	}
	if id == caller.UserID && in.Role != caller.Role {
		model.RespondError(w, http.StatusConflict, "cannot_change_own_role")
		return
	}
	college, ok1 := idOrNil(in.CollegeID)
	dept, ok2 := idOrNil(in.DepartmentID)
	if !ok1 || !ok2 {
		model.RespondError(w, http.StatusBadRequest, "invalid identifier")
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	if err := h.updateUser(ctx, id, in.Role, college, dept); err != nil {
		switch {
		case isNoRows(err):
			model.RespondError(w, http.StatusNotFound, "User not found")
		case isForeignKeyViolation(err):
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
		default:
			serverError(w, "Failed to update user", err)
		}
		return
	}
	if h.invalidate != nil {
		h.invalidate(id)
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"updated": id})
}

func (h *AdminUserHandler) updateUser(ctx context.Context, id, role string, college, dept *string) error {
	var updated string
	return h.db.QueryRow(ctx, `
		UPDATE users SET role = $2::user_role, college_id = $3, department_id = $4
		WHERE id = $1 RETURNING id
	`, id, role, college, dept).Scan(&updated)
}
