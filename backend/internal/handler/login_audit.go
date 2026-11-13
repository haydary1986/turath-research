package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/loginaudit"
)

const loginAuditTimeout = 3 * time.Second

// recordLogin stores one sign-in attempt. It runs on its own short deadline, detached
// from the request, so a client hanging up still leaves a trace; failures are logged only.
func recordLogin(r *http.Request, db *pgxpool.Pool, e loginaudit.Event) {
	if db == nil {
		return
	}
	e.IP = r.RemoteAddr // already the client's address: RealIP runs first
	e.UserAgent = r.UserAgent()
	e = loginaudit.Sanitize(e)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), loginAuditTimeout)
	defer cancel()
	var reason *string
	if e.Reason != "" {
		reason = &e.Reason
	}
	_, err := db.Exec(ctx, `
		INSERT INTO login_events (user_id, email, method, success, reason, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.UserID, e.Email, e.Method, e.Success, reason, e.IP, e.UserAgent)
	if err != nil {
		slog.Error("auth: failed to record login event", "method", e.Method, "error", err)
	}
}

// PruneLoginEvents deletes login events older than the cutoff (retention job).
func PruneLoginEvents(db *pgxpool.Pool) loginaudit.Pruner {
	return func(ctx context.Context, cutoff time.Time) (int64, error) {
		tag, err := db.Exec(ctx, `DELETE FROM login_events WHERE created_at < $1`, cutoff)
		return tag.RowsAffected(), err
	}
}

// AuditHandler serves the admin's login log and account-activity views.
type AuditHandler struct {
	db *pgxpool.Pool
}

func NewAuditHandler(db *pgxpool.Pool) *AuditHandler {
	return &AuditHandler{db: db}
}

type loginEvent struct {
	ID        int64     `json:"id"`
	UserID    *string   `json:"user_id"`
	UserName  *string   `json:"user_name"`
	Email     string    `json:"email"`
	Method    string    `json:"method"`
	Result    string    `json:"result"` // success | failure
	Reason    *string   `json:"reason"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// loginEventFilters turns the query into SQL conditions; dates are Baghdad days.
func loginEventFilters(q map[string][]string) ([]string, []any, []string) {
	get := func(k string) string {
		if v := q[k]; len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	var conds, problems []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if text := get("q"); text != "" {
		add("(e.email ILIKE ? OR u.name_ar ILIKE ? OR u.email ILIKE ?)", "%"+likeEscaper.Replace(text)+"%")
	}
	if id := get("user_id"); id != "" {
		if v, ok := idOrNil(id); ok && v != nil {
			add("e.user_id = ?", *v)
		} else {
			problems = append(problems, "user_id:invalid")
		}
	}
	switch m := get("method"); m {
	case "":
	case loginaudit.MethodPassword, loginaudit.MethodGoogle:
		add("e.method = ?", m)
	default:
		problems = append(problems, "method:invalid")
	}
	switch get("result") {
	case "":
	case "success":
		conds = append(conds, "e.success")
	case "failure":
		conds = append(conds, "NOT e.success")
	default:
		problems = append(problems, "result:invalid")
	}
	for _, bound := range [...]struct{ field, cmp string }{{"from", ">="}, {"to", "<="}} {
		field, cmp := bound.field, bound.cmp
		raw := get(field)
		if raw == "" {
			continue
		}
		if _, err := time.Parse(reportDate, raw); err != nil {
			problems = append(problems, field+":invalid_date")
			continue
		}
		add("(e.created_at AT TIME ZONE 'Asia/Baghdad')::date "+cmp+" ?::date", raw)
	}
	return conds, args, problems
}

// ListLoginEvents returns the filtered, paginated login log (admin).
// Query: q (email or name), user_id, method, result (success|failure), from, to, page, limit.
func (h *AuditHandler) ListLoginEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 25, 1, 100)
	conds, args, problems := loginEventFilters(q)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	const from = ` FROM login_events e LEFT JOIN users u ON u.id = e.user_id`

	ctx, cancel := portalContext(r)
	defer cancel()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*)`+from+where, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count login events", err)
		return
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, `
		SELECT e.id, e.user_id, u.name_ar, e.email, e.method, e.success, e.reason, e.ip, e.user_agent, e.created_at`+
		from+where+` ORDER BY e.created_at DESC, e.id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to list login events", err)
		return
	}
	defer rows.Close()
	items := []loginEvent{}
	for rows.Next() {
		var e loginEvent
		var success bool
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.Email, &e.Method, &success,
			&e.Reason, &e.IP, &e.UserAgent, &e.CreatedAt); err != nil {
			serverError(w, "Failed to read login events", err)
			return
		}
		e.Result = "failure"
		if success {
			e.Result = "success"
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read login events", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
