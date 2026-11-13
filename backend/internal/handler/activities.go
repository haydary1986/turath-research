package handler

import (
	"context"
	"net/http"
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
	Category       string     `json:"category"`
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
	CategoryDetail *string    `json:"category_detail"`
	ItemCount      *int       `json:"item_count"`
	Notes          *string    `json:"notes"`
	Status         string     `json:"status"`
	ApprovalNo     *string    `json:"approval_no"`
	DecisionNote   *string    `json:"decision_note"`
	CreatedBy      string     `json:"created_by"`
	CreatorName    string     `json:"creator_name"`
	DecidedAt      *time.Time `json:"decided_at"`
	CreatedAt      time.Time  `json:"created_at"`
	// CanEdit tells the caller whether they may change the record now (author or responsibility holder).
	CanEdit bool `json:"can_edit"`
}

const activitySelect = `
	SELECT a.id, a.category, a.activity_type, a.title_ar, a.description, a.presenter,
	       a.college_id, a.department_id, c.name_ar, d.name_ar,
	       to_char(a.activity_date, 'YYYY-MM-DD'), a.location, a.attendees_count,
	       a.target_audience, a.organizer, a.attendance_mode,
	       a.category_detail, a.item_count, a.notes,
	       a.status, a.approval_no, a.decision_note,
	       a.created_by, COALESCE(u.name_ar, u.email), a.decided_at, a.created_at
	FROM activities a
	JOIN users u ON u.id = a.created_by
	LEFT JOIN colleges c ON c.id = a.college_id
	LEFT JOIN departments d ON d.id = a.department_id`

// activityReturning lines up with scanActivity for INSERT/UPDATE … RETURNING (names are filled by the list).
const activityReturning = `
	RETURNING id, category, activity_type, title_ar, description, presenter,
	          college_id, department_id, NULL::varchar, NULL::varchar,
	          to_char(activity_date, 'YYYY-MM-DD'), location, attendees_count,
	          target_audience, organizer, attendance_mode,
	          category_detail, item_count, notes,
	          status, approval_no, decision_note, created_by, ''::varchar, decided_at, created_at`

func scanActivity(row rowScanner) (activityRecord, error) {
	var a activityRecord
	err := row.Scan(&a.ID, &a.Category, &a.Type, &a.TitleAr, &a.Description, &a.Presenter,
		&a.CollegeID, &a.DepartmentID, &a.CollegeName, &a.DepartmentName,
		&a.ActivityDate, &a.Location, &a.Attendees,
		&a.TargetAudience, &a.Organizer, &a.AttendanceMode,
		&a.CategoryDetail, &a.ItemCount, &a.Notes,
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
// Members' own work defaults to their college; an institutional record with no college
// belongs to the whole university, so it never inherits one.
func parseActivityInput(w http.ResponseWriter, r *http.Request, user *middleware.UserClaims) (activityInput, *string, *string, bool) {
	var in activityInput
	if !decodeBody(w, r, &in) {
		return in, nil, nil, false
	}
	in.Input = activity.Normalize(in.Input)
	userCollege, userDept := user.CollegeID, user.DepartmentID
	if activity.IsInstitutional(in.Category) {
		userCollege, userDept = "", ""
	}
	college, ok1 := idOrNil(firstNonEmpty(in.CollegeID, userCollege))
	dept, ok2 := idOrNil(resolveActivityDepartment(in.CollegeID, in.DepartmentID, userCollege, userDept))
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

// activityArgs are the stored columns of an input in the order of the INSERT and UPDATE statements ($2…$17).
func activityArgs(in activityInput, college, dept *string) []any {
	return []any{
		in.Category, in.Type, strings.TrimSpace(in.TitleAr), in.Description, in.Presenter, college, dept,
		in.ActivityDate, in.Location, in.Attendees, in.TargetAudience, in.Organizer, in.AttendanceMode,
		in.CategoryDetail, in.ItemCount, strings.TrimSpace(in.Notes),
	}
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
	if college == nil {
		validationError(w, []string{"department:wrong_college"})
		return false
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

// CreateActivity registers a record of the FS 003 register. Members file their own
// work; institutional categories need the matching responsibility or scientific affairs.
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
	actor, err := loadActor(ctx, h.db, user)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	if !activity.CanEnter(actor, in.Category, college) {
		model.RespondError(w, http.StatusForbidden, "activity_category_forbidden")
		return
	}
	if !h.departmentFitsCollege(ctx, w, college, dept) {
		return
	}

	args := append(activityArgs(in, college, dept), user.UserID)
	rec, err := scanActivity(h.db.QueryRow(ctx, `
		INSERT INTO activities
			(category, activity_type, title_ar, description, presenter, college_id, department_id,
			 activity_date, location, attendees_count, target_audience, organizer, attendance_mode,
			 category_detail, item_count, notes, created_by)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8::date, NULLIF($9, ''), $10,
		        NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), $15, NULLIF($16, ''), $17)`+
		activityReturning, args...))
	if err != nil {
		if isForeignKeyViolation(err) {
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
			return
		}
		serverError(w, "Failed to save activity", err)
		return
	}
	rec.CanEdit = true

	h.notifyActivityReviewers(ctx, user,
		"نشاط علمي جديد بانتظار الاعتماد", "New scientific activity awaiting approval", rec.TitleAr)
	model.RespondJSON(w, http.StatusCreated, rec)
}

// UpdateActivity lets the author, or a holder of the record's responsibility, fix a
// pending/returned record. Every edit sends it back to pending.
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

	current, found, err := h.loadActivityAccess(ctx, id)
	if err != nil {
		serverError(w, "Failed to load activity", err)
		return
	}
	actor, err := loadActor(ctx, h.db, user)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	if !found || !activity.CanEdit(actor, current) {
		model.RespondError(w, http.StatusNotFound, "Activity not found or not editable")
		return
	}
	if !activity.CanMoveTo(actor, current, in.Category, college) {
		model.RespondError(w, http.StatusForbidden, "activity_category_forbidden")
		return
	}
	if !h.departmentFitsCollege(ctx, w, college, dept) {
		return
	}

	args := append([]any{id}, activityArgs(in, college, dept)...)
	rec, err := scanActivity(h.db.QueryRow(ctx, `
		UPDATE activities SET
			category = $2, activity_type = $3, title_ar = $4, description = NULLIF($5, ''), presenter = NULLIF($6, ''),
			college_id = $7, department_id = $8, activity_date = $9::date,
			location = NULLIF($10, ''), attendees_count = $11,
			target_audience = NULLIF($12, ''), organizer = NULLIF($13, ''), attendance_mode = NULLIF($14, ''),
			category_detail = NULLIF($15, ''), item_count = $16, notes = NULLIF($17, ''),
			status = 'pending', updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'returned')`+activityReturning, args...))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "Activity not found or not editable")
		return
	}
	if err != nil {
		if isForeignKeyViolation(err) {
			model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
			return
		}
		serverError(w, "Failed to update activity", err)
		return
	}
	rec.CanEdit = true
	if current.Status == activity.StatusReturned {
		h.notifyActivityReviewers(ctx, user,
			"أُعيد تقديم نشاط علمي بعد تعديله", "Returned scientific activity resubmitted", rec.TitleAr)
	}
	model.RespondJSON(w, http.StatusOK, rec)
}

// loadActivityAccess reads what the edit rules need; found is false for an unknown id.
func (h *PortalHandler) loadActivityAccess(ctx context.Context, id string) (activity.Record, bool, error) {
	var rec activity.Record
	err := h.db.QueryRow(ctx, `SELECT created_by, category, college_id, status FROM activities WHERE id = $1`, id).
		Scan(&rec.CreatedBy, &rec.Category, &rec.CollegeID, &rec.Status)
	if isNoRows(err) {
		return rec, false, nil
	}
	return rec, err == nil, err
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
