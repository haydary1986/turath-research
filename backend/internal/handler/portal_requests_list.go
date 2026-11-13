package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// List returns the requests the caller may see, filtered and paginated.
func (h *PortalHandler) List(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 20, 1, 50)

	where, args := scopeClause(user)
	conds := []string{where}
	addCond := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.Get("mine") == "true" {
		conds = append(conds, "r.applicant_id = $1")
	}
	if q.Get("queue") == "true" {
		// Awaiting the caller's action. A dean's college limit comes from scopeClause.
		conds = append(conds, "r.applicant_id <> $1")
		addCond("r.status = ANY(?)", portal.QueueStatuses(user.Role))
	}
	if kind := q.Get("kind"); portal.IsKind(kind) {
		addCond("r.kind = ?", kind)
	}
	if status := q.Get("status"); status != "" {
		addCond("r.status = ?", status)
	}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		addCond(`(r.request_no ILIKE ? OR r.title_ar ILIKE ? OR r.title_en ILIKE ? OR r.doi ILIKE ? OR u.name_ar ILIKE ? OR u.email ILIKE ?)`,
			"%"+likeEscaper.Replace(text)+"%")
	}
	whereSQL := " WHERE " + strings.Join(conds, " AND ")

	ctx, cancel := portalContext(r)
	defer cancel()

	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM portal_requests r JOIN users u ON u.id = r.applicant_id`+whereSQL, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count requests", err)
		return
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, requestSelect+whereSQL+
		` ORDER BY r.created_at DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to list requests", err)
		return
	}
	defer rows.Close()

	items := []requestRecord{}
	for rows.Next() {
		rec, err := scanRequest(rows)
		if err != nil {
			serverError(w, "Failed to read requests", err)
			return
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read requests", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func clampInt(raw string, fallback, lo, hi int) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return min(max(n, lo), hi)
}

type statCount struct {
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// Stats returns request counts by kind and status within the caller's visibility.
func (h *PortalHandler) Stats(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	where, args := scopeClause(user)

	ctx, cancel := portalContext(r)
	defer cancel()

	rows, err := h.db.Query(ctx, `SELECT r.kind, r.status, COUNT(*) FROM portal_requests r WHERE `+where+` GROUP BY r.kind, r.status`, args...)
	if err != nil {
		serverError(w, "Failed to compute stats", err)
		return
	}
	defer rows.Close()

	items := []statCount{}
	for rows.Next() {
		var s statCount
		if err := rows.Scan(&s.Kind, &s.Status, &s.Count); err != nil {
			serverError(w, "Failed to read stats", err)
			return
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read stats", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, items)
}
