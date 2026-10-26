package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// Activities live on PortalHandler so they share notifications, scoping and helpers.

type activityRecord struct {
	ID             string     `json:"id"`
	Type           string     `json:"activity_type"`
	TitleAr        string     `json:"title_ar"`
	Description    *string    `json:"description"`
	Presenter      *string    `json:"presenter"`
	CollegeID      *string    `json:"college_id"`
	DepartmentID   *string    `json:"department_id"`
	CollegeName    *string    `json:"college_name"`
	DepartmentName *string    `json:"department_name"`
	ActivityDate   string     `json:"activity_date"`
	Location       *string    `json:"location"`
	Attendees      int        `json:"attendees_count"`
	Status         string     `json:"status"`
	ApprovalNo     *string    `json:"approval_no"`
	DecisionNote   *string    `json:"decision_note"`
	CreatedBy      string     `json:"created_by"`
	CreatorName    string     `json:"creator_name"`
	DecidedAt      *time.Time `json:"decided_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

const activitySelect = `
	SELECT a.id, a.activity_type, a.title_ar, a.description, a.presenter,
	       a.college_id, a.department_id, c.name_ar, d.name_ar,
	       to_char(a.activity_date, 'YYYY-MM-DD'), a.location, a.attendees_count,
	       a.status, a.approval_no, a.decision_note,
	       a.created_by, COALESCE(u.name_ar, u.email), a.decided_at, a.created_at
	FROM activities a
	JOIN users u ON u.id = a.created_by
	LEFT JOIN colleges c ON c.id = a.college_id
	LEFT JOIN departments d ON d.id = a.department_id`

func scanActivity(row rowScanner) (activityRecord, error) {
	var a activityRecord
	err := row.Scan(&a.ID, &a.Type, &a.TitleAr, &a.Description, &a.Presenter,
		&a.CollegeID, &a.DepartmentID, &a.CollegeName, &a.DepartmentName,
		&a.ActivityDate, &a.Location, &a.Attendees,
		&a.Status, &a.ApprovalNo, &a.DecisionNote,
		&a.CreatedBy, &a.CreatorName, &a.DecidedAt, &a.CreatedAt)
	return a, err
}

type activityInput struct {
	activity.Input
	CollegeID    string `json:"college_id"`
	DepartmentID string `json:"department_id"`
}

// parseActivityInput validates and resolves org ids; it writes the error response itself.
func parseActivityInput(w http.ResponseWriter, r *http.Request, user *middleware.UserClaims) (activityInput, *string, *string, bool) {
	var in activityInput
	if !decodeBody(w, r, &in) {
		return in, nil, nil, false
	}
	college, ok1 := idOrNil(firstNonEmpty(in.CollegeID, user.CollegeID))
	dept, ok2 := idOrNil(firstNonEmpty(in.DepartmentID, user.DepartmentID))
	if !ok1 || !ok2 {
		model.RespondError(w, http.StatusBadRequest, "invalid identifier")
		return in, nil, nil, false
	}
	if problems := activity.Validate(in.Input, college != nil); len(problems) > 0 {
		validationError(w, problems)
		return in, nil, nil, false
	}
	return in, college, dept, true
}

// CreateActivity registers a scientific activity (any signed-in member).
func (h *PortalHandler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	in, college, dept, ok := parseActivityInput(w, r, user)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	rec, err := scanActivity(h.db.QueryRow(ctx, `
		INSERT INTO activities
			(activity_type, title_ar, description, presenter, college_id, department_id,
			 activity_date, location, attendees_count, created_by)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7::date, NULLIF($8, ''), $9, $10)
		RETURNING id, activity_type, title_ar, description, presenter,
		          college_id, department_id, NULL::varchar, NULL::varchar,
		          to_char(activity_date, 'YYYY-MM-DD'), location, attendees_count,
		          status, approval_no, decision_note, created_by, ''::varchar, decided_at, created_at
	`, in.Type, strings.TrimSpace(in.TitleAr), in.Description, in.Presenter, college, dept,
		in.ActivityDate, in.Location, in.Attendees, user.UserID))
	if err != nil {
		if isForeignKeyViolation(err) {
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
			return
		}
		serverError(w, "Failed to save activity", err)
		return
	}

	h.notifyRole(ctx, portal.RoleScientificAffairs, nil,
		"نشاط علمي جديد بانتظار الإقرار", "New scientific activity awaiting approval",
		rec.TitleAr, "/portal/activities")
	model.RespondJSON(w, http.StatusCreated, rec)
}

// UpdateActivity lets the organizer fix a pending/returned activity.
func (h *PortalHandler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	id := chi.URLParam(r, "id")
	in, college, dept, ok := parseActivityInput(w, r, user)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	rec, err := scanActivity(h.db.QueryRow(ctx, `
		UPDATE activities SET
			activity_type = $2, title_ar = $3, description = NULLIF($4, ''), presenter = NULLIF($5, ''),
			college_id = $6, department_id = $7, activity_date = $8::date,
			location = NULLIF($9, ''), attendees_count = $10, status = 'pending', updated_at = NOW()
		WHERE id = $1 AND created_by = $11 AND status IN ('pending', 'returned')
		RETURNING id, activity_type, title_ar, description, presenter,
		          college_id, department_id, NULL::varchar, NULL::varchar,
		          to_char(activity_date, 'YYYY-MM-DD'), location, attendees_count,
		          status, approval_no, decision_note, created_by, ''::varchar, decided_at, created_at
	`, id, in.Type, strings.TrimSpace(in.TitleAr), in.Description, in.Presenter, college, dept,
		in.ActivityDate, in.Location, in.Attendees, user.UserID))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Activity not found or not editable")
		return
	}
	if err != nil {
		serverError(w, "Failed to update activity", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, rec)
}

// DecideActivity is the scientific-affairs step: approve (with the official
// approval-letter number), return for edits, or reject.
func (h *PortalHandler) DecideActivity(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in struct {
		Status     string `json:"status"`
		ApprovalNo string `json:"approval_no"`
		Note       string `json:"note"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	in.ApprovalNo = strings.TrimSpace(in.ApprovalNo)
	in.Note = strings.TrimSpace(in.Note)

	var problems []string
	switch in.Status {
	case activity.StatusApproved:
		if in.ApprovalNo == "" {
			problems = append(problems, "approval_no:required")
		}
		if utf8.RuneCountInString(in.ApprovalNo) > 100 {
			problems = append(problems, "approval_no:too_long")
		}
	case activity.StatusReturned, activity.StatusRejected:
		if in.Note == "" {
			problems = append(problems, "note:required")
		}
	default:
		problems = append(problems, "status:invalid")
	}
	if utf8.RuneCountInString(in.Note) > maxNoteRunes {
		problems = append(problems, "note:too_long")
	}
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	var createdBy, titleAr string
	err := h.db.QueryRow(ctx, `
		UPDATE activities SET
			status = $2, approval_no = NULLIF($3, ''), decision_note = NULLIF($4, ''),
			decided_by = $5, decided_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'returned')
		RETURNING created_by, title_ar
	`, chi.URLParam(r, "id"), in.Status, in.ApprovalNo, in.Note, user.UserID).Scan(&createdBy, &titleAr)
	if isNoRows(err) {
		model.RespondError(w, http.StatusConflict, "Activity not found or already decided")
		return
	}
	if err != nil {
		serverError(w, "Failed to decide activity", err)
		return
	}

	titles := map[string]string{
		activity.StatusApproved: "أُقرّ النشاط العلمي وصدر كتاب الإقامة: " + in.ApprovalNo,
		activity.StatusReturned: "أُعيد النشاط العلمي للتعديل",
		activity.StatusRejected: "رُفض النشاط العلمي",
	}
	h.notifyUser(ctx, createdBy, titles[in.Status], "Activity decision",
		titleAr+"\n"+in.Note, notificationType(in.Status), "/portal/activities")
	model.RespondJSON(w, http.StatusOK, map[string]string{"decided": in.Status})
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

// ListActivities returns scoped, filtered, paginated activities.
func (h *PortalHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 20, 1, 50)

	where, args := activityScope(user)
	conds := []string{where}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
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
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		add("(a.title_ar ILIKE ? OR a.presenter ILIKE ? OR a.approval_no ILIKE ?)",
			"%"+likeEscaper.Replace(text)+"%")
	}
	whereSQL := " WHERE " + strings.Join(conds, " AND ")

	ctx, cancel := portalContext(r)
	defer cancel()

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
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read activities", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
