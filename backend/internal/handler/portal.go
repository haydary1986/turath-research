package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/mailer"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

const (
	portalQueryTimeout = 8 * time.Second
	maxJSONBody        = 1 << 20
	maxFilesPerRequest = 20
)

// PortalHandler serves the scientific affairs portal: requests, files, letter templates and letters.
type PortalHandler struct {
	db        *pgxpool.Pool
	uploadDir string
	mail      *mailer.Mailer // nil ⇒ in-app notifications only
	baseURL   string         // public frontend URL used in notification emails
}

func NewPortalHandler(db *pgxpool.Pool, uploadDir string, mail *mailer.Mailer, baseURL string) *PortalHandler {
	return &PortalHandler{db: db, uploadDir: uploadDir, mail: mail, baseURL: strings.TrimRight(baseURL, "/")}
}

// emailLink turns an in-app link like /portal/requests/x into an absolute Arabic-locale URL.
func (h *PortalHandler) emailLink(link string) string {
	if h.baseURL == "" || link == "" {
		return ""
	}
	return h.baseURL + "/ar" + link
}

// requestRecord is a portal request joined with applicant, college and department names.
type requestRecord struct {
	ID               string          `json:"id"`
	RequestNo        *string         `json:"request_no"`
	Kind             string          `json:"kind"`
	Status           string          `json:"status"`
	ApplicantID      string          `json:"applicant_id"`
	ApplicantName    string          `json:"applicant_name"`
	ApplicantEmail   string          `json:"applicant_email"`
	TitleAr          *string         `json:"title_ar"`
	TitleEn          *string         `json:"title_en"`
	DOI              *string         `json:"doi"`
	CollegeID        *string         `json:"college_id"`
	DepartmentID     *string         `json:"department_id"`
	CollegeName      *string         `json:"college_name"`
	DepartmentName   *string         `json:"department_name"`
	Details          json.RawMessage `json:"details"`
	RelatedRequestID *string         `json:"related_request_id"`
	DecisionNote     *string         `json:"decision_note"`
	SubmittedAt      *time.Time      `json:"submitted_at"`
	DecidedAt        *time.Time      `json:"decided_at"`
	CreatedAt        time.Time       `json:"created_at"`
	ResearcherID     *string         `json:"researcher_id"`
	PublicationID    *string         `json:"publication_id"`
	SADecision       *string         `json:"sa_decision"`
}

const requestSelect = `
	SELECT r.id, r.request_no, r.kind, r.status, r.applicant_id,
	       COALESCE(u.name_ar, u.email), u.email,
	       r.title_ar, r.title_en, r.doi, r.college_id, r.department_id,
	       c.name_ar, d.name_ar, r.details, r.related_request_id,
	       r.decision_note, r.submitted_at, r.decided_at, r.created_at,
	       r.researcher_id, r.publication_id, r.sa_decision
	FROM portal_requests r
	JOIN users u ON u.id = r.applicant_id
	LEFT JOIN colleges c ON c.id = r.college_id
	LEFT JOIN departments d ON d.id = r.department_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequest(row rowScanner) (requestRecord, error) {
	var rec requestRecord
	err := row.Scan(
		&rec.ID, &rec.RequestNo, &rec.Kind, &rec.Status, &rec.ApplicantID,
		&rec.ApplicantName, &rec.ApplicantEmail,
		&rec.TitleAr, &rec.TitleEn, &rec.DOI, &rec.CollegeID, &rec.DepartmentID,
		&rec.CollegeName, &rec.DepartmentName, &rec.Details, &rec.RelatedRequestID,
		&rec.DecisionNote, &rec.SubmittedAt, &rec.DecidedAt, &rec.CreatedAt,
		&rec.ResearcherID, &rec.PublicationID, &rec.SADecision,
	)
	return rec, err
}

// loadRequest fetches one request; the bool is false when it does not exist.
func (h *PortalHandler) loadRequest(ctx context.Context, id string) (requestRecord, bool, error) {
	if _, err := uuid.Parse(id); err != nil {
		return requestRecord{}, false, nil
	}
	rec, err := scanRequest(h.db.QueryRow(ctx, requestSelect+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return requestRecord{}, false, nil
	}
	return rec, err == nil, err
}

// canView implements read access: the applicant, staff, and the head/dean of the owning unit.
func canView(user *middleware.UserClaims, rec requestRecord) bool {
	switch {
	case rec.ApplicantID == user.UserID:
		return true
	case rec.Status == portal.StatusDraft:
		return false
	case portal.IsStaff(user.Role):
		return true
	case user.Role == middleware.RoleDepartmentHead:
		return user.DepartmentID != "" && rec.DepartmentID != nil && *rec.DepartmentID == user.DepartmentID
	case user.Role == middleware.RoleCollegeDean:
		return user.CollegeID != "" && rec.CollegeID != nil && *rec.CollegeID == user.CollegeID
	default:
		return false
	}
}

// scopeClause returns a SQL condition (with the user id bound as $1) limiting requests to what the user may see.
func scopeClause(user *middleware.UserClaims) (string, []any) {
	args := []any{user.UserID}
	switch {
	case portal.IsStaff(user.Role):
		return "(r.status <> 'draft' OR r.applicant_id = $1)", args
	case user.Role == middleware.RoleDepartmentHead && user.DepartmentID != "":
		args = append(args, user.DepartmentID)
		return "(r.applicant_id = $1 OR (r.status <> 'draft' AND r.department_id = $2))", args
	case user.Role == middleware.RoleCollegeDean && user.CollegeID != "":
		args = append(args, user.CollegeID)
		return "(r.applicant_id = $1 OR (r.status <> 'draft' AND r.college_id = $2))", args
	default:
		return "r.applicant_id = $1", args
	}
}

// nextSequence atomically increments and returns the counter for (kind, year).
func nextSequence(ctx context.Context, tx pgx.Tx, kind string, year int) (int, error) {
	var seq int
	err := tx.QueryRow(ctx, `
		INSERT INTO portal_sequences (kind, year, last_value) VALUES ($1, $2, 1)
		ON CONFLICT (kind, year) DO UPDATE SET last_value = portal_sequences.last_value + 1
		RETURNING last_value
	`, kind, year).Scan(&seq)
	return seq, err
}

func recordEvent(ctx context.Context, tx pgx.Tx, requestID, actorID string, from *string, to, note string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO portal_request_events (request_id, actor_id, from_status, to_status, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
	`, requestID, actorID, from, to, note)
	return err
}

// notifyUser stores an in-app notification and mirrors it to the user's university email.
// Failures are logged, never surfaced to the caller.
func (h *PortalHandler) notifyUser(ctx context.Context, userID, titleAr, titleEn, body, kind, link string) {
	_, err := h.db.Exec(ctx, `
		INSERT INTO notifications (user_id, title_ar, title_en, body, type, link)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, titleAr, titleEn, body, kind, link)
	if err != nil {
		slog.Error("portal: notify user failed", "user_id", userID, "error", err)
	}
	if h.mail.Enabled() {
		var email string
		if err := h.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err == nil {
			h.mail.SendAsync(email, titleAr, emailBody(body, h.emailLink(link)))
		}
	}
}

func emailBody(body, link string) string {
	if link == "" {
		return body
	}
	return body + "\n\n" + link
}

// notifyRole alerts every user holding `role` (college deans only within collegeID),
// in-app and by email. Used to wake the next actor in the approval chain.
func (h *PortalHandler) notifyRole(ctx context.Context, role string, collegeID *string, titleAr, titleEn, body, link string) {
	where := `role = $5::user_role`
	args := []any{titleAr, titleEn, body, link, role}
	if collegeID != nil {
		where += ` AND college_id = $6`
		args = append(args, *collegeID)
	}
	_, err := h.db.Exec(ctx, `
		INSERT INTO notifications (user_id, title_ar, title_en, body, type, link)
		SELECT id, $1, $2, $3, 'info', $4 FROM users WHERE `+where, args...)
	if err != nil {
		slog.Error("portal: notify role failed", "role", role, "error", err)
		return
	}
	if !h.mail.Enabled() {
		return
	}
	emailWhere := `role = $1::user_role`
	emailArgs := []any{role}
	if collegeID != nil {
		emailWhere += ` AND college_id = $2`
		emailArgs = append(emailArgs, *collegeID)
	}
	rows, err := h.db.Query(ctx, `SELECT email FROM users WHERE `+emailWhere, emailArgs...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var email string
		if rows.Scan(&email) == nil {
			h.mail.SendAsync(email, titleAr, emailBody(body, h.emailLink(link)))
		}
	}
}

var statusTitlesAr = map[string]string{
	portal.StatusPendingDean:        "تم استلام الطلب لدى عمادة الكلية",
	portal.StatusPending:            "منح العميد الموافقة الأولية — الطلب لدى الشؤون العلمية",
	portal.StatusPendingVP:          "رُفعت توصية الشؤون العلمية إلى المساعد العلمي",
	portal.StatusReturned:           "أُعيد الطلب للتعديل",
	portal.StatusApproved:           "تمت المصادقة على الطلب",
	portal.StatusRejected:           "رُفض الطلب",
	portal.StatusVerified:           "تم تدقيق الطلب",
	portal.StatusNotEligible:        "الطلب غير مستوفٍ",
	portal.StatusAwaitingSignatures: "الطلب بانتظار توقيع اللجنة",
	portal.StatusCompleted:          "اكتمل الطلب",
	portal.StatusIssued:             "صدر كتاب الشكر والتقدير",
}

func requestLink(id string) string { return "/portal/requests/" + id }

func requestLabel(rec requestRecord) string {
	if rec.RequestNo != nil {
		return *rec.RequestNo
	}
	return rec.ID
}

// ── Shared request/response helpers ──────────────

func portalContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), portalQueryTimeout)
}

// authedUser returns the caller or writes a 401.
func authedUser(w http.ResponseWriter, r *http.Request) *middleware.UserClaims {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
	}
	return user
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func isUniqueViolation(err error) bool     { return isPgCode(err, "23505") }
func isForeignKeyViolation(err error) bool { return isPgCode(err, "23503") }

func serverError(w http.ResponseWriter, msg string, err error) {
	slog.Error("portal: "+msg, "error", err)
	model.RespondError(w, http.StatusInternalServerError, msg)
}

func validationError(w http.ResponseWriter, problems []string) {
	model.RespondError(w, http.StatusUnprocessableEntity, "validation:"+strings.Join(problems, ","))
}

func blankToNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// idOrNil validates an optional UUID string; ok is false when it is present but malformed.
func idOrNil(s string) (id *string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, true
	}
	if _, err := uuid.Parse(s); err != nil {
		return nil, false
	}
	return &s, true
}
