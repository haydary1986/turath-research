package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
)

const maxResponsibilities = 60

// ResponsibilityHandler manages data-entry responsibilities («مسؤوليات الإدخال»):
// the admin assigns them per user, and every user reads their own.
type ResponsibilityHandler struct {
	db         *pgxpool.Pool
	invalidate func(userID string) // responsibilities are read per request; this also drops the cached identity
}

func NewResponsibilityHandler(db *pgxpool.Pool, invalidate func(userID string)) *ResponsibilityHandler {
	return &ResponsibilityHandler{db: db, invalidate: invalidate}
}

type responsibility struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	CollegeID   *string `json:"college_id"` // null: the whole university
	CollegeName *string `json:"college_name"`
}

type responsibilityInput struct {
	Category  string `json:"category"`
	CollegeID string `json:"college_id"`
}

func (h *ResponsibilityHandler) list(ctx context.Context, userID string) ([]responsibility, error) {
	rows, err := h.db.Query(ctx, `
		SELECT e.id, e.category, e.college_id, c.name_ar
		FROM entry_responsibilities e
		LEFT JOIN colleges c ON c.id = e.college_id
		WHERE e.user_id = $1
		ORDER BY e.category, c.name_ar NULLS FIRST`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []responsibility{}
	for rows.Next() {
		var it responsibility
		if err := rows.Scan(&it.ID, &it.Category, &it.CollegeID, &it.CollegeName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Mine returns the caller's own responsibilities (drives «مسؤولياتي» and the export button).
func (h *ResponsibilityHandler) Mine(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	items, err := h.list(ctx, user.UserID)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// List returns one user's responsibilities (admin).
func (h *ResponsibilityHandler) List(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	items, err := h.list(ctx, id)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// normalizeResponsibilities validates and de-duplicates the requested set.
// It returns "field:code" problems, numbered by position («items.2.category:invalid»).
func normalizeResponsibilities(in []responsibilityInput) ([]activity.Scope, []string) {
	var problems []string
	if len(in) > maxResponsibilities {
		return nil, []string{"items:too_many"}
	}
	seen := map[activity.Scope]struct{}{}
	out := make([]activity.Scope, 0, len(in))
	for i, it := range in {
		prefix := "items." + strconv.Itoa(i) + "."
		if !activity.IsCategory(it.Category) {
			problems = append(problems, prefix+"category:invalid")
			continue
		}
		college, ok := idOrNil(it.CollegeID)
		if !ok {
			problems = append(problems, prefix+"college_id:invalid")
			continue
		}
		scope := activity.Scope{Category: it.Category}
		if college != nil {
			scope.CollegeID = *college
		}
		if _, dup := seen[scope]; dup {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, problems
}

// Replace sets a user's whole responsibility set in one transaction (admin).
// Body: {"items": [{"category": "international_agreements", "college_id": ""}]}.
func (h *ResponsibilityHandler) Replace(w http.ResponseWriter, r *http.Request) {
	caller := authedUser(w, r)
	if caller == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	var body struct {
		Items []responsibilityInput `json:"items"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	scopes, problems := normalizeResponsibilities(body.Items)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	if err := h.replace(ctx, id, caller.UserID, scopes); err != nil {
		switch {
		case isNoRows(err):
			model.RespondError(w, http.StatusNotFound, "User not found")
		case isForeignKeyViolation(err):
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
		default:
			serverError(w, "Failed to save responsibilities", err)
		}
		return
	}
	if h.invalidate != nil {
		h.invalidate(id)
	}
	items, err := h.list(ctx, id)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}

func (h *ResponsibilityHandler) replace(ctx context.Context, userID, grantedBy string, scopes []activity.Scope) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var locked string
	// Locks the user row so two concurrent saves cannot interleave their delete/insert.
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&locked); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM entry_responsibilities WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, s := range scopes {
		var college *string
		if s.CollegeID != "" {
			college = &s.CollegeID
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO entry_responsibilities (user_id, category, college_id, granted_by)
			VALUES ($1, $2, $3, $4)`, userID, s.Category, college, grantedBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
