package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

// Account activity buckets. "Today" is the Baghdad calendar day.
const (
	onlineWindow   = "5 minutes"
	baghdadToday   = `(date_trunc('day', NOW() AT TIME ZONE 'Asia/Baghdad') AT TIME ZONE 'Asia/Baghdad')`
	activityOnline = `u.last_seen_at >= NOW() - INTERVAL '` + onlineWindow + `'`
	activityToday  = `u.last_seen_at >= ` + baghdadToday
	activity30d    = `u.last_seen_at >= NOW() - INTERVAL '30 days'`
	activityNever  = `(u.last_seen_at IS NULL AND u.last_login_at IS NULL)`
)

// accountActivityFilter maps the ?status= values onto their conditions.
var accountActivityFilter = map[string]string{
	"online": activityOnline,
	"today":  activityToday,
	"30d":    activity30d,
	"never":  activityNever,
}

type accountSummary struct {
	Total               int `json:"total"`
	Online              int `json:"online"`
	Today               int `json:"today"`
	Last30              int `json:"last_30_days"`
	Never               int `json:"never_signed_in"`
	OnlineWindowMinutes int `json:"online_window_minutes"`
}

// AccountsSummary counts accounts by recent activity (admin).
func (h *AuditHandler) AccountsSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	s := accountSummary{OnlineWindowMinutes: 5}
	err := h.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE `+activityOnline+`),
		       COUNT(*) FILTER (WHERE `+activityToday+`),
		       COUNT(*) FILTER (WHERE `+activity30d+`),
		       COUNT(*) FILTER (WHERE `+activityNever+`)
		FROM users u`).Scan(&s.Total, &s.Online, &s.Today, &s.Last30, &s.Never)
	if err != nil {
		serverError(w, "Failed to count accounts", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, s)
}

type accountActivity struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	NameAr      *string    `json:"name_ar"`
	Role        string     `json:"role"`
	CollegeName *string    `json:"college_name"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Online      bool       `json:"online"`
}

// ListAccounts lists users with their last activity, most recent first (admin).
// Query: status (online|today|30d|never), q, page, limit.
func (h *AuditHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 25, 1, 100)

	var conds []string
	var args []any
	if status := q.Get("status"); status != "" {
		cond, ok := accountActivityFilter[status]
		if !ok {
			validationError(w, []string{"status:invalid"})
			return
		}
		conds = append(conds, cond)
	}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		args = append(args, "%"+likeEscaper.Replace(text)+"%")
		conds = append(conds, "(u.email ILIKE $1 OR u.name_ar ILIKE $1)")
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM users u`+where, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count accounts", err)
		return
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, `
		SELECT u.id, u.email, u.name_ar, u.role::text, c.name_ar, u.last_seen_at, u.last_login_at, `+activityOnline+`
		FROM users u LEFT JOIN colleges c ON c.id = u.college_id`+where+`
		ORDER BY u.last_seen_at DESC NULLS LAST, u.last_login_at DESC NULLS LAST, u.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to list accounts", err)
		return
	}
	defer rows.Close()
	items := []accountActivity{}
	for rows.Next() {
		var a accountActivity
		var online *bool
		if err := rows.Scan(&a.ID, &a.Email, &a.NameAr, &a.Role, &a.CollegeName, &a.LastSeenAt, &a.LastLoginAt, &online); err != nil {
			serverError(w, "Failed to read accounts", err)
			return
		}
		a.Online = online != nil && *online
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read accounts", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
