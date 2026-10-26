package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const (
	maxTitleRunes = 500
	maxDOIRunes   = 255
	maxNoteRunes  = 2000
)

type requestInput struct {
	Kind             string         `json:"kind"`
	TitleAr          string         `json:"title_ar"`
	TitleEn          string         `json:"title_en"`
	DOI              string         `json:"doi"`
	CollegeID        string         `json:"college_id"`
	DepartmentID     string         `json:"department_id"`
	RelatedRequestID string         `json:"related_request_id"`
	Details          map[string]any `json:"details"`
}

type normalizedInput struct {
	titleAr, titleEn, doi              *string
	collegeID, departmentID, relatedID *string
	details                            []byte
}

// normalizeInput trims and validates the shape of the input. Completeness is checked on submit, not here,
// so applicants can save partial drafts. It returns a message when the input is unusable.
func normalizeInput(kind string, in requestInput, user *middleware.UserClaims) (normalizedInput, string) {
	var n normalizedInput
	if utf8.RuneCountInString(in.TitleAr) > maxTitleRunes || utf8.RuneCountInString(in.TitleEn) > maxTitleRunes {
		return n, "title is too long"
	}
	doi := portal.NormalizeDOI(in.DOI)
	if utf8.RuneCountInString(doi) > maxDOIRunes {
		return n, "doi is too long"
	}
	n.titleAr, n.titleEn, n.doi = blankToNil(in.TitleAr), blankToNil(in.TitleEn), blankToNil(doi)

	college, ok1 := idOrNil(firstNonEmpty(in.CollegeID, user.CollegeID))
	dept, ok2 := idOrNil(firstNonEmpty(in.DepartmentID, user.DepartmentID))
	related, ok3 := idOrNil(in.RelatedRequestID)
	if !ok1 || !ok2 || !ok3 {
		return n, "invalid identifier"
	}
	if related != nil && kind != portal.KindAppreciationLetter {
		return n, "related_request_id is only valid for appreciation letters"
	}
	n.collegeID, n.departmentID, n.relatedID = college, dept, related

	details := in.Details
	if details == nil {
		details = map[string]any{}
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return n, "invalid details"
	}
	n.details = raw
	return n, ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// relatedVerificationOK reports whether the applicant owns a completed journal verification with this id.
func (h *PortalHandler) relatedVerificationOK(ctx context.Context, applicantID, relatedID string) (bool, error) {
	var ok bool
	err := h.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM portal_requests
			WHERE id = $1 AND kind = 'journal_verification' AND status = 'completed' AND applicant_id = $2
		)`, relatedID, applicantID).Scan(&ok)
	return ok, err
}

// Create saves a new draft request owned by the caller.
func (h *PortalHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in requestInput
	if !decodeBody(w, r, &in) {
		return
	}
	if !portal.IsKind(in.Kind) {
		model.RespondError(w, http.StatusBadRequest, "Unknown request kind")
		return
	}
	n, msg := normalizeInput(in.Kind, in, user)
	if msg != "" {
		model.RespondError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	if n.relatedID != nil {
		ok, err := h.relatedVerificationOK(ctx, user.UserID, *n.relatedID)
		if err != nil {
			serverError(w, "Failed to check related request", err)
			return
		}
		if !ok {
			model.RespondError(w, http.StatusBadRequest, "related_request_invalid")
			return
		}
	}

	var id string
	err := h.db.QueryRow(ctx, `
		INSERT INTO portal_requests
			(kind, applicant_id, title_ar, title_en, doi, college_id, department_id, details, related_request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		RETURNING id
	`, in.Kind, user.UserID, n.titleAr, n.titleEn, n.doi, n.collegeID, n.departmentID, string(n.details), n.relatedID).Scan(&id)
	if h.writeSaveError(w, err) {
		return
	}
	h.respondRequest(ctx, w, http.StatusCreated, id)
}

// Update edits a draft or returned request. Only the applicant may do so.
func (h *PortalHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in requestInput
	if !decodeBody(w, r, &in) {
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || rec.ApplicantID != user.UserID {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}
	if !isEditable(rec.Status) {
		model.RespondError(w, http.StatusConflict, "Request can no longer be edited")
		return
	}

	n, msg := normalizeInput(rec.Kind, in, user)
	if msg != "" {
		model.RespondError(w, http.StatusBadRequest, msg)
		return
	}
	if n.relatedID != nil {
		ok, err := h.relatedVerificationOK(ctx, user.UserID, *n.relatedID)
		if err != nil {
			serverError(w, "Failed to check related request", err)
			return
		}
		if !ok {
			model.RespondError(w, http.StatusBadRequest, "related_request_invalid")
			return
		}
	}

	_, err = h.db.Exec(ctx, `
		UPDATE portal_requests SET
			title_ar = $2, title_en = $3, doi = $4, college_id = $5, department_id = $6,
			details = $7::jsonb, related_request_id = $8, updated_at = NOW()
		WHERE id = $1 AND status IN ('draft', 'returned')
	`, rec.ID, n.titleAr, n.titleEn, n.doi, n.collegeID, n.departmentID, string(n.details), n.relatedID)
	if h.writeSaveError(w, err) {
		return
	}
	h.respondRequest(ctx, w, http.StatusOK, rec.ID)
}

// writeSaveError maps database constraint errors to client errors. It returns true when it wrote a response.
func (h *PortalHandler) writeSaveError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case isUniqueViolation(err):
		model.RespondError(w, http.StatusConflict, "duplicate_doi")
	case isForeignKeyViolation(err):
		model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
	default:
		serverError(w, "Failed to save request", err)
	}
	return true
}

func (h *PortalHandler) respondRequest(ctx context.Context, w http.ResponseWriter, status int, id string) {
	rec, found, err := h.loadRequest(ctx, id)
	if err != nil || !found {
		serverError(w, "Failed to load request", err)
		return
	}
	model.RespondJSON(w, status, rec)
}

func isEditable(status string) bool {
	return status == portal.StatusDraft || status == portal.StatusReturned
}

// Submit validates a draft/returned request and sends it to scientific affairs.
func (h *PortalHandler) Submit(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || rec.ApplicantID != user.UserID {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}
	if !isEditable(rec.Status) {
		model.RespondError(w, http.StatusConflict, "Request was already submitted")
		return
	}

	problems, err := h.submissionProblems(ctx, rec)
	if err != nil {
		serverError(w, "Failed to validate request", err)
		return
	}
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}

	if err := h.markSubmitted(ctx, rec, user.UserID); err != nil {
		if isUniqueViolation(err) {
			model.RespondError(w, http.StatusConflict, "duplicate_doi")
			return
		}
		serverError(w, "Failed to submit request", err)
		return
	}

	updated, _, err := h.loadRequest(ctx, rec.ID)
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	label := requestLabel(updated)
	h.notifyUser(ctx, user.UserID, "تم إرسال الطلب بنجاح", "Request submitted", "رقم الطلب: "+label, "success", requestLink(rec.ID))
	h.notifyRole(ctx, portal.RoleCollegeDean, updated.CollegeID,
		"طلب جديد بانتظار الموافقة الأولية", "New request awaiting initial approval",
		"رقم الطلب: "+label, requestLink(rec.ID))
	model.RespondJSON(w, http.StatusOK, updated)
}

func (h *PortalHandler) markSubmitted(ctx context.Context, rec requestRecord, actorID string) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	requestNo := rec.RequestNo
	if requestNo == nil {
		seq, err := nextSequence(ctx, tx, rec.Kind, time.Now().Year())
		if err != nil {
			return err
		}
		no := portal.FormatRequestNo(rec.Kind, time.Now().Year(), seq)
		requestNo = &no
	}
	researcherID, err := ensureResearcher(ctx, tx, rec)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE portal_requests
		SET status = 'pending_dean', request_no = $2, submitted_at = NOW(), decision_note = NULL,
		    researcher_id = $3, updated_at = NOW()
		WHERE id = $1 AND status IN ('draft', 'returned')
	`, rec.ID, *requestNo, researcherID); err != nil {
		return err
	}
	from := rec.Status
	if err := recordEvent(ctx, tx, rec.ID, actorID, &from, portal.StatusPendingDean, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// submissionProblems lists everything missing before the request can be submitted.
func (h *PortalHandler) submissionProblems(ctx context.Context, rec requestRecord) ([]string, error) {
	var problems []string

	hasTitle := rec.TitleAr != nil || rec.TitleEn != nil
	if rec.Kind == portal.KindResearchPlan {
		hasTitle = rec.TitleAr != nil && rec.TitleEn != nil
	}
	if !hasTitle {
		problems = append(problems, "title:required")
	}
	// The dean's initial approval is college-scoped, so a college is mandatory.
	if rec.CollegeID == nil {
		problems = append(problems, "college:required")
	}

	var details map[string]any
	if err := json.Unmarshal(rec.Details, &details); err != nil {
		return nil, err
	}
	problems = append(problems, portal.ValidateDetails(rec.Kind, details)...)

	hasVerified := false
	if rec.RelatedRequestID != nil {
		ok, err := h.relatedVerificationOK(ctx, rec.ApplicantID, *rec.RelatedRequestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			problems = append(problems, "related_request:invalid")
		}
		hasVerified = ok
	}

	present, err := h.attachmentRoles(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	for _, role := range portal.RequiredAttachments(rec.Kind, hasVerified) {
		if !present[role] {
			problems = append(problems, "attachment:"+role)
		}
	}
	return problems, nil
}

func (h *PortalHandler) attachmentRoles(ctx context.Context, requestID string) (map[string]bool, error) {
	rows, err := h.db.Query(ctx, `SELECT DISTINCT role FROM portal_files WHERE request_id = $1`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := map[string]bool{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles[role] = true
	}
	return roles, rows.Err()
}

// Transition lets staff move a submitted request along its workflow.
func (h *PortalHandler) Transition(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in struct {
		To       string `json:"to"`
		Note     string `json:"note"`
		Decision string `json:"decision"` // required when to=pending_vp: the recommended outcome
	}
	if !decodeBody(w, r, &in) {
		return
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > maxNoteRunes {
		model.RespondError(w, http.StatusBadRequest, "note is too long")
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}
	if !mayActOn(user, rec) {
		model.RespondError(w, http.StatusForbidden, "Insufficient permissions")
		return
	}
	if in.To == portal.StatusIssued || !portal.CanTransition(rec.Kind, rec.Status, in.To) {
		model.RespondError(w, http.StatusConflict, "Transition not allowed")
		return
	}

	var saDecision *string
	switch {
	case in.To == portal.StatusPendingVP:
		// Scientific affairs raises its decision as a recommendation to the VP.
		if !portal.IsFinalDecision(rec.Kind, in.Decision) {
			validationError(w, []string{"decision:required"})
			return
		}
		saDecision = &in.Decision
	case rec.Status == portal.StatusPendingVP && portal.IsFinalDecision(rec.Kind, in.To):
		// The VP confirms exactly what scientific affairs recommended, or sends it back.
		if rec.SADecision == nil || *rec.SADecision != in.To {
			model.RespondError(w, http.StatusConflict, "decision_mismatch")
			return
		}
	}

	needsNote := portal.RequiresNote(in.To) ||
		(rec.Status == portal.StatusPendingVP && in.To == portal.StatusPending) // VP sending back must say why
	if needsNote && note == "" {
		validationError(w, []string{"note:required"})
		return
	}
	if in.To == portal.StatusCompleted {
		present, err := h.attachmentRoles(ctx, rec.ID)
		if err != nil {
			serverError(w, "Failed to check attachments", err)
			return
		}
		if !present[portal.RoleSignedForm] {
			validationError(w, []string{"attachment:" + portal.RoleSignedForm})
			return
		}
	}

	if err := h.applyTransition(ctx, rec, user.UserID, in.To, note, saDecision); err != nil {
		if err == errStatusChanged {
			model.RespondError(w, http.StatusConflict, "Request status changed, reload and retry")
			return
		}
		serverError(w, "Failed to update request", err)
		return
	}

	// Wake the next actor in the chain.
	label := requestLabel(rec)
	switch in.To {
	case portal.StatusPending:
		h.notifyRole(ctx, portal.RoleScientificAffairs, nil,
			"طلب بانتظار تدقيق الشؤون العلمية", "Request awaiting scientific affairs review",
			"رقم الطلب: "+label, requestLink(rec.ID))
	case portal.StatusPendingVP:
		h.notifyRole(ctx, portal.RoleVPResearch, nil,
			"توصية بانتظار تأكيد المساعد العلمي", "Recommendation awaiting VP confirmation",
			"رقم الطلب: "+label, requestLink(rec.ID))
	}
	h.notifyUser(ctx, rec.ApplicantID, statusTitlesAr[in.To], "Request status updated", note+"\nرقم الطلب: "+requestLabel(rec), notificationType(in.To), requestLink(rec.ID))
	h.respondRequest(ctx, w, http.StatusOK, rec.ID)
}

// mayActOn checks whether the caller is the actor of the request's current stage:
// admin always; the college dean only for requests of their own college; nobody
// reviews their own request; the department head never decides.
func mayActOn(user *middleware.UserClaims, rec requestRecord) bool {
	if rec.ApplicantID == user.UserID {
		return false
	}
	if user.Role == portal.RoleAdmin {
		return true
	}
	required := portal.RequiredRole(rec.Status)
	if required == "" || user.Role != required {
		return false
	}
	if required == portal.RoleCollegeDean {
		return user.CollegeID != "" && rec.CollegeID != nil && *rec.CollegeID == user.CollegeID
	}
	return true
}

var errStatusChanged = errorString("status changed concurrently")

type errorString string

func (e errorString) Error() string { return string(e) }

func (h *PortalHandler) applyTransition(ctx context.Context, rec requestRecord, actorID, to, note string, saDecision *string) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	tag, err := tx.Exec(ctx, `
		UPDATE portal_requests
		SET status = $3::varchar, decision_note = NULLIF($4, ''), updated_at = NOW(),
		    sa_decision = COALESCE($5, sa_decision),
		    decided_at = CASE WHEN $3::varchar IN ('approved', 'rejected', 'not_eligible', 'completed') THEN NOW() ELSE decided_at END
		WHERE id = $1 AND status = $2
	`, rec.ID, rec.Status, to, note, saDecision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errStatusChanged
	}
	if to == portal.StatusCompleted {
		if err := materializePublication(ctx, tx, rec); err != nil {
			return err
		}
	}
	from := rec.Status
	if err := recordEvent(ctx, tx, rec.ID, actorID, &from, to, note); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func notificationType(status string) string {
	switch status {
	case portal.StatusRejected, portal.StatusNotEligible:
		return "alert"
	case portal.StatusReturned:
		return "warning"
	case portal.StatusApproved, portal.StatusVerified, portal.StatusCompleted, portal.StatusIssued:
		return "success"
	default:
		return "info"
	}
}

// ── Reading ──────────────────────────────────────

type attachmentInfo struct {
	ID           string    `json:"id"`
	Role         string    `json:"role"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	UploadedBy   *string   `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type eventInfo struct {
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       *string   `json:"note"`
	ActorName  *string   `json:"actor_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type requestDetail struct {
	Request            requestRecord    `json:"request"`
	Attachments        []attachmentInfo `json:"attachments"`
	Events             []eventInfo      `json:"events"`
	AllowedTransitions []string         `json:"allowed_transitions"`
	CanEdit            bool             `json:"can_edit"`
	LetterID           *string          `json:"letter_id"`
	JournalFlag        *journalFlag     `json:"journal_flag,omitempty"`
}

// Get returns one request with its attachments, history and the actions the caller may take.
func (h *PortalHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	rec, found, err := h.loadRequest(ctx, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load request", err)
		return
	}
	if !found || !canView(user, rec) {
		model.RespondError(w, http.StatusNotFound, "Request not found")
		return
	}

	detail := requestDetail{Request: rec, AllowedTransitions: []string{}}
	if detail.Attachments, err = h.listAttachments(ctx, rec.ID); err != nil {
		serverError(w, "Failed to load attachments", err)
		return
	}
	if detail.Events, err = h.listEvents(ctx, rec.ID); err != nil {
		serverError(w, "Failed to load history", err)
		return
	}
	if err := h.db.QueryRow(ctx, `SELECT id FROM issued_letters WHERE request_id = $1`, rec.ID).Scan(&detail.LetterID); err != nil && !isNoRows(err) {
		serverError(w, "Failed to load letter", err)
		return
	}

	if rec.Kind != portal.KindResearchPlan {
		var details map[string]any
		if json.Unmarshal(rec.Details, &details) == nil {
			issn, _ := details["issn"].(string)
			name, _ := details["journal_name"].(string)
			if flag, err := h.screenJournal(ctx, issn, name); err == nil && flag.Matched {
				detail.JournalFlag = &flag
			}
		}
	}

	detail.CanEdit = rec.ApplicantID == user.UserID && isEditable(rec.Status)
	if mayActOn(user, rec) {
		for _, next := range portal.NextStatuses(rec.Kind, rec.Status) {
			if next == portal.StatusIssued {
				continue // issued goes through the dedicated letter endpoint
			}
			// At the VP stage only the recommended outcome (or sending back) is offered.
			if rec.Status == portal.StatusPendingVP && portal.IsFinalDecision(rec.Kind, next) &&
				(rec.SADecision == nil || *rec.SADecision != next) {
				continue
			}
			detail.AllowedTransitions = append(detail.AllowedTransitions, next)
		}
	}
	model.RespondJSON(w, http.StatusOK, detail)
}

func (h *PortalHandler) listAttachments(ctx context.Context, requestID string) ([]attachmentInfo, error) {
	rows, err := h.db.Query(ctx, `
		SELECT id, role, original_name, mime_type, size_bytes, uploaded_by, created_at
		FROM portal_files WHERE request_id = $1 ORDER BY created_at
	`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []attachmentInfo{}
	for rows.Next() {
		var a attachmentInfo
		if err := rows.Scan(&a.ID, &a.Role, &a.OriginalName, &a.MimeType, &a.SizeBytes, &a.UploadedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (h *PortalHandler) listEvents(ctx context.Context, requestID string) ([]eventInfo, error) {
	rows, err := h.db.Query(ctx, `
		SELECT e.from_status, e.to_status, e.note, COALESCE(u.name_ar, u.email), e.created_at
		FROM portal_request_events e LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.request_id = $1 ORDER BY e.created_at
	`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []eventInfo{}
	for rows.Next() {
		var e eventInfo
		if err := rows.Scan(&e.FromStatus, &e.ToStatus, &e.Note, &e.ActorName, &e.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

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
