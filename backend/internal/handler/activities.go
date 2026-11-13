package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
	TargetAudience *string    `json:"target_audience"`
	Organizer      *string    `json:"organizer"`
	AttendanceMode *string    `json:"attendance_mode"`
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
	       a.target_audience, a.organizer, a.attendance_mode,
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
		&a.TargetAudience, &a.Organizer, &a.AttendanceMode,
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
	dept, ok2 := idOrNil(resolveActivityDepartment(in.CollegeID, in.DepartmentID, user.CollegeID, user.DepartmentID))
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

// resolveActivityDepartment fills an empty department from the user's own, but only
// for the user's own college: their department means nothing inside another one.
func resolveActivityDepartment(inCollege, inDept, userCollege, userDept string) string {
	if strings.TrimSpace(inDept) != "" {
		return inDept
	}
	chosen := strings.TrimSpace(firstNonEmpty(inCollege, userCollege))
	if chosen != "" && strings.EqualFold(chosen, strings.TrimSpace(userCollege)) {
		return userDept
	}
	return ""
}

// departmentFitsCollege writes the error response itself when dept is outside college.
func (h *PortalHandler) departmentFitsCollege(ctx context.Context, w http.ResponseWriter, college, dept *string) bool {
	if dept == nil {
		return true
	}
	var fits bool
	err := h.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM departments WHERE id = $1 AND college_id = $2)`, *dept, college).Scan(&fits)
	if err != nil {
		serverError(w, "Failed to check department", err)
		return false
	}
	if !fits {
		validationError(w, []string{"department:wrong_college"})
	}
	return fits
}

// activityReviewers says who to alert about an activity awaiting a decision. Authors
// cannot decide their own, so a scientific-affairs author needs someone else told:
// the other officers, and admins as the escalation path.
func activityReviewers(authorRole, authorID string) (excludeUser string, alsoAdmins bool) {
	if authorRole == portal.RoleScientificAffairs {
		return authorID, true
	}
	return "", false
}

func (h *PortalHandler) notifyActivityReviewers(ctx context.Context, author *middleware.UserClaims, titleAr, titleEn, body string) {
	const link = "/portal/activities"
	exclude, alsoAdmins := activityReviewers(author.Role, author.UserID)
	h.notifyRoleExcept(ctx, portal.RoleScientificAffairs, nil, exclude, titleAr, titleEn, body, link)
	if alsoAdmins {
		h.notifyRole(ctx, portal.RoleAdmin, nil, titleAr, titleEn, body, link)
	}
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
	if !h.departmentFitsCollege(ctx, w, college, dept) {
		return
	}

	rec, err := scanActivity(h.db.QueryRow(ctx, `
		INSERT INTO activities
			(activity_type, title_ar, description, presenter, college_id, department_id,
			 activity_date, location, attendees_count, target_audience, organizer, attendance_mode, created_by)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7::date, NULLIF($8, ''), $9,
		        NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), $13)
		RETURNING id, activity_type, title_ar, description, presenter,
		          college_id, department_id, NULL::varchar, NULL::varchar,
		          to_char(activity_date, 'YYYY-MM-DD'), location, attendees_count,
		          target_audience, organizer, attendance_mode,
		          status, approval_no, decision_note, created_by, ''::varchar, decided_at, created_at
	`, in.Type, strings.TrimSpace(in.TitleAr), in.Description, in.Presenter, college, dept,
		in.ActivityDate, in.Location, in.Attendees,
		in.TargetAudience, in.Organizer, in.AttendanceMode, user.UserID))
	if err != nil {
		if isForeignKeyViolation(err) {
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
			return
		}
		serverError(w, "Failed to save activity", err)
		return
	}

	h.notifyActivityReviewers(ctx, user,
		"نشاط علمي جديد بانتظار الاعتماد", "New scientific activity awaiting approval", rec.TitleAr)
	model.RespondJSON(w, http.StatusCreated, rec)
}

// UpdateActivity lets the organizer fix a pending/returned activity.
func (h *PortalHandler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Activity not found or not editable")
		return
	}
	in, college, dept, ok := parseActivityInput(w, r, user)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	if !h.departmentFitsCollege(ctx, w, college, dept) {
		return
	}

	var previous string
	if err := h.db.QueryRow(ctx, `SELECT status FROM activities WHERE id = $1`, id).Scan(&previous); err != nil && !isNoRows(err) {
		serverError(w, "Failed to load activity", err)
		return
	}

	rec, err := scanActivity(h.db.QueryRow(ctx, `
		UPDATE activities SET
			activity_type = $2, title_ar = $3, description = NULLIF($4, ''), presenter = NULLIF($5, ''),
			college_id = $6, department_id = $7, activity_date = $8::date,
			location = NULLIF($9, ''), attendees_count = $10,
			target_audience = NULLIF($11, ''), organizer = NULLIF($12, ''), attendance_mode = NULLIF($13, ''),
			status = 'pending', updated_at = NOW()
		WHERE id = $1 AND created_by = $14 AND status IN ('pending', 'returned')
		RETURNING id, activity_type, title_ar, description, presenter,
		          college_id, department_id, NULL::varchar, NULL::varchar,
		          to_char(activity_date, 'YYYY-MM-DD'), location, attendees_count,
		          target_audience, organizer, attendance_mode,
		          status, approval_no, decision_note, created_by, ''::varchar, decided_at, created_at
	`, id, in.Type, strings.TrimSpace(in.TitleAr), in.Description, in.Presenter, college, dept,
		in.ActivityDate, in.Location, in.Attendees,
		in.TargetAudience, in.Organizer, in.AttendanceMode, user.UserID))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Activity not found or not editable")
		return
	}
	if err != nil {
		serverError(w, "Failed to update activity", err)
		return
	}
	if previous == activity.StatusReturned {
		h.notifyActivityReviewers(ctx, user,
			"أُعيد تقديم نشاط علمي بعد تعديله", "Returned scientific activity resubmitted", rec.TitleAr)
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
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Activity not found")
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

	var owner string
	err := h.db.QueryRow(ctx, `SELECT created_by FROM activities WHERE id = $1`, id).Scan(&owner)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Activity not found")
		return
	}
	if err != nil {
		serverError(w, "Failed to load activity", err)
		return
	}
	if !mayDecideActivity(user, owner) {
		model.RespondError(w, http.StatusForbidden, "cannot_decide_own_activity")
		return
	}

	var createdBy, titleAr string
	err = h.db.QueryRow(ctx, `
		UPDATE activities SET
			status = $2, approval_no = NULLIF($3, ''), decision_note = NULLIF($4, ''),
			decided_by = $5, decided_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'returned')
		RETURNING created_by, title_ar
	`, id, in.Status, in.ApprovalNo, in.Note, user.UserID).Scan(&createdBy, &titleAr)
	if isNoRows(err) {
		model.RespondError(w, http.StatusConflict, "activity_already_decided")
		return
	}
	if err != nil {
		serverError(w, "Failed to decide activity", err)
		return
	}

	titles := map[string]string{
		activity.StatusApproved: "اعتُمد النشاط العلمي، وعدد كتاب الإقامة: " + in.ApprovalNo,
		activity.StatusReturned: "أُعيد النشاط العلمي للتعديل",
		activity.StatusRejected: "رُفض النشاط العلمي",
	}
	h.notifyUser(ctx, createdBy, titles[in.Status], "Activity decision",
		titleAr+"\n"+in.Note, notificationType(in.Status), "/portal/activities")
	model.RespondJSON(w, http.StatusOK, map[string]string{"decided": in.Status})
}

// mayDecideActivity stops reviewers approving their own activity. Admin is the escalation path.
func mayDecideActivity(user *middleware.UserClaims, createdBy string) bool {
	return user.Role == portal.RoleAdmin || user.UserID != createdBy
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
