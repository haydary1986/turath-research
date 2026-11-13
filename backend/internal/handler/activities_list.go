package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// loadScopes reads a user's data-entry responsibilities. It runs on every request
// that needs them, so a grant or a withdrawal applies immediately (never via the JWT).
func loadScopes(ctx context.Context, db *pgxpool.Pool, userID string) ([]activity.Scope, error) {
	rows, err := db.Query(ctx,
		`SELECT category, COALESCE(college_id::text, '') FROM entry_responsibilities WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scopes []activity.Scope
	for rows.Next() {
		var s activity.Scope
		if err := rows.Scan(&s.Category, &s.CollegeID); err != nil {
			return nil, err
		}
		scopes = append(scopes, s)
	}
	return scopes, rows.Err()
}

// loadActor is the caller with their current responsibilities.
func loadActor(ctx context.Context, db *pgxpool.Pool, user *middleware.UserClaims) (activity.Actor, error) {
	scopes, err := loadScopes(ctx, db, user.UserID)
	return activity.Actor{UserID: user.UserID, Role: user.Role, Scopes: scopes}, err
}

// activityScope limits visibility: own records, the head's department,
// the dean's college; staff and admin see everything.
func activityScope(user *middleware.UserClaims) (string, []any) {
	args := []any{user.UserID}
	switch {
	case portal.IsStaff(user.Role):
		return "TRUE", nil
	case user.Role == middleware.RoleCollegeDean && user.CollegeID != "":
		args = append(args, user.CollegeID)
		return "(a.created_by = $1 OR a.college_id = $2)", args
	case user.Role == middleware.RoleDepartmentHead && user.DepartmentID != "":
		args = append(args, user.DepartmentID)
		return "(a.created_by = $1 OR a.department_id = $2)", args
	default:
		return "a.created_by = $1", args
	}
}

// responsibilityClause widens visibility to every record a holder is responsible for:
// whole categories for university-wide scopes, category and college pairs otherwise.
// Placeholders continue after the given args; an empty clause means no responsibility.
func responsibilityClause(scopes []activity.Scope, args []any) (string, []any) {
	var university []string
	var parts []string
	for _, s := range scopes {
		if s.CollegeID == "" {
			university = append(university, s.Category)
			continue
		}
		args = append(args, s.Category, s.CollegeID)
		n := len(args)
		parts = append(parts, "(a.category = $"+strconv.Itoa(n-1)+" AND a.college_id = $"+strconv.Itoa(n)+")")
	}
	if len(university) > 0 {
		args = append(args, university)
		parts = append([]string{"a.category = ANY($" + strconv.Itoa(len(args)) + "::text[])"}, parts...)
	}
	switch len(parts) {
	case 0:
		return "", args
	case 1:
		return parts[0], args
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

// visibleActivities combines the role scope with the caller's responsibilities.
func visibleActivities(user *middleware.UserClaims, scopes []activity.Scope) (string, []any) {
	where, args := activityScope(user)
	if where == "TRUE" {
		return where, args
	}
	clause, args := responsibilityClause(scopes, args)
	if clause == "" {
		return where, args
	}
	return "(" + where + " OR " + clause + ")", args
}

// ListActivities returns scoped, filtered, paginated activities.
func (h *PortalHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 20, 1, 50)

	ctx, cancel := portalContext(r)
	defer cancel()
	actor, err := loadActor(ctx, h.db, user)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}

	where, args := visibleActivities(user, actor.Scopes)
	conds := []string{where}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if c := q.Get("category"); activity.IsCategory(c) {
		add("a.category = ?", c)
	}
	if t := q.Get("type"); activity.IsType(t) {
		add("a.activity_type = ?", t)
	}
	if s := q.Get("status"); s != "" {
		add("a.status = ?", s)
	}
	if y, err := strconv.Atoi(q.Get("year")); err == nil && y > 2000 && y < 2100 {
		add("EXTRACT(YEAR FROM a.activity_date) = ?", y)
	}
	if college := q.Get("college_id"); college != "" {
		if id, ok := idOrNil(college); ok && id != nil {
			add("a.college_id = ?", *id)
		}
	}
	if q.Get("mine") == "true" {
		add("a.created_by = ?", user.UserID)
	}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		add("(a.title_ar ILIKE ? OR a.presenter ILIKE ? OR a.approval_no ILIKE ? OR a.category_detail ILIKE ?)",
			"%"+likeEscaper.Replace(text)+"%")
	}
	whereSQL := " WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := h.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM activities a JOIN users u ON u.id = a.created_by`+whereSQL, args...,
	).Scan(&total); err != nil {
		serverError(w, "Failed to count activities", err)
		return
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, activitySelect+whereSQL+
		` ORDER BY a.activity_date DESC, a.created_at DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to list activities", err)
		return
	}
	defer rows.Close()

	items := []activityRecord{}
	for rows.Next() {
		rec, err := scanActivity(rows)
		if err != nil {
			serverError(w, "Failed to read activities", err)
			return
		}
		rec.CanEdit = activity.CanEdit(actor, activity.Record{
			CreatedBy: rec.CreatedBy, Category: rec.Category, CollegeID: rec.CollegeID, Status: rec.Status,
		})
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read activities", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
