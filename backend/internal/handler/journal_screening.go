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

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

// Journal watchlist (accredited / predatory / suspended), managed by staff and
// consulted automatically when verification requests are displayed.

type screeningRecord struct {
	ID             string    `json:"id"`
	ISSN           *string   `json:"issn"`
	Name           string    `json:"name"`
	Publisher      *string   `json:"publisher"`
	Classification string    `json:"classification"`
	Note           *string   `json:"note"`
	Source         *string   `json:"source"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const screeningSelect = `
	SELECT id, issn, name, publisher, classification, note, source, updated_at
	FROM journal_screening`

func scanScreening(row rowScanner) (screeningRecord, error) {
	var s screeningRecord
	err := row.Scan(&s.ID, &s.ISSN, &s.Name, &s.Publisher, &s.Classification, &s.Note, &s.Source, &s.UpdatedAt)
	return s, err
}

type screeningInput struct {
	ISSN           string `json:"issn"`
	Name           string `json:"name"`
	Publisher      string `json:"publisher"`
	Classification string `json:"classification"`
	Note           string `json:"note"`
	Source         string `json:"source"`
}

func (in *screeningInput) problems() (normalizedISSN string, errs []string) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		errs = append(errs, "journal_name:required")
	}
	if utf8.RuneCountInString(in.Name) > 500 {
		errs = append(errs, "journal_name:too_long")
	}
	if !screening.IsClassification(in.Classification) {
		errs = append(errs, "classification:invalid")
	}
	if strings.TrimSpace(in.ISSN) != "" {
		normalizedISSN = screening.NormalizeISSN(in.ISSN)
		if normalizedISSN == "" {
			errs = append(errs, "issn:malformed")
		}
	}
	return normalizedISSN, errs
}

// ListScreening returns the watchlist, filterable by classification and text.
func (h *PortalHandler) ListScreening(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 50, 1, 100)

	conds := []string{"TRUE"}
	args := []any{}
	if c := q.Get("classification"); screening.IsClassification(c) {
		args = append(args, c)
		conds = append(conds, "classification = $1")
	}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		args = append(args, "%"+likeEscaper.Replace(text)+"%")
		n := len(args)
		conds = append(conds, "(name ILIKE $"+itoa(n)+" OR issn ILIKE $"+itoa(n)+" OR publisher ILIKE $"+itoa(n)+")")
	}
	whereSQL := " WHERE " + strings.Join(conds, " AND ")

	ctx, cancel := portalContext(r)
	defer cancel()

	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(*) FROM journal_screening`+whereSQL, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count watchlist", err)
		return
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := h.db.Query(ctx, screeningSelect+whereSQL+
		` ORDER BY updated_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to list watchlist", err)
		return
	}
	defer rows.Close()
	items := []screeningRecord{}
	for rows.Next() {
		rec, err := scanScreening(rows)
		if err != nil {
			serverError(w, "Failed to read watchlist", err)
			return
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read watchlist", err)
		return
	}
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}

// UpsertScreening creates or (with {id}) edits a watchlist entry.
func (h *PortalHandler) UpsertScreening(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in screeningInput
	if !decodeBody(w, r, &in) {
		return
	}
	issn, problems := in.problems()
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	nameKey := screening.NameKey(in.Name)

	ctx, cancel := portalContext(r)
	defer cancel()

	id := chi.URLParam(r, "id")
	var rec screeningRecord
	var err error
	if id == "" {
		rec, err = scanScreening(h.db.QueryRow(ctx, `
			INSERT INTO journal_screening (issn, name, name_key, publisher, classification, note, source, created_by)
			VALUES (NULLIF($1, ''), $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), NULLIF($7, ''), $8)
			RETURNING id, issn, name, publisher, classification, note, source, updated_at
		`, issn, in.Name, nameKey, in.Publisher, in.Classification, in.Note, in.Source, user.UserID))
	} else {
		if _, perr := uuid.Parse(id); perr != nil {
			model.RespondError(w, http.StatusNotFound, "Entry not found")
			return
		}
		rec, err = scanScreening(h.db.QueryRow(ctx, `
			UPDATE journal_screening SET
				issn = NULLIF($2, ''), name = $3, name_key = $4, publisher = NULLIF($5, ''),
				classification = $6, note = NULLIF($7, ''), source = NULLIF($8, ''), updated_at = NOW()
			WHERE id = $1
			RETURNING id, issn, name, publisher, classification, note, source, updated_at
		`, id, issn, in.Name, nameKey, in.Publisher, in.Classification, in.Note, in.Source))
	}
	switch {
	case isNoRows(err):
		model.RespondError(w, http.StatusNotFound, "Entry not found")
	case isUniqueViolation(err):
		model.RespondError(w, http.StatusConflict, "duplicate_issn")
	case err != nil:
		serverError(w, "Failed to save entry", err)
	default:
		model.RespondJSON(w, http.StatusOK, rec)
	}
}

// DeleteScreening removes a watchlist entry.
func (h *PortalHandler) DeleteScreening(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "Entry not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	tag, err := h.db.Exec(ctx, `DELETE FROM journal_screening WHERE id = $1`, id)
	if err != nil {
		serverError(w, "Failed to delete entry", err)
		return
	}
	if tag.RowsAffected() == 0 {
		model.RespondError(w, http.StatusNotFound, "Entry not found")
		return
	}
	model.RespondJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

const (
	journalCheckTimeout = 20 * time.Second
	maxCheckFieldRunes  = 500
	maxCheckURLRunes    = 2000

	// A person filling a form causes a handful of uncached lookups; each costs three
	// upstream calls, and a repeated query is served from the cache at no cost.
	uncachedChecksPerUser  = 30
	uncachedChecksInterval = 10 * time.Minute
)

// journalQuery reads and bounds the check parameters.
func journalQuery(r *http.Request) (journalcheck.Query, []string) {
	values := r.URL.Query()
	query := journalcheck.Query{
		ISSN: values.Get("issn"), ISSNOnline: values.Get("issn_online"), Name: values.Get("name"),
		Publisher: values.Get("publisher"), URL: values.Get("url"),
	}
	var problems []string
	if utf8.RuneCountInString(query.Name) > maxCheckFieldRunes {
		problems = append(problems, "journal_name:too_long")
	}
	if utf8.RuneCountInString(query.Publisher) > maxCheckFieldRunes {
		problems = append(problems, "publisher:too_long")
	}
	if utf8.RuneCountInString(query.URL) > maxCheckURLRunes {
		problems = append(problems, "journal_url:too_long")
	}
	return query, problems
}

// CheckJournal returns advisory evidence: manual watchlist, reference lists and
// the external indexes.
func (h *PortalHandler) CheckJournal(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	query, problems := journalQuery(r)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	query.AllowExternal = func() bool { return h.journalThrottle.Allow(user.UserID) }
	ctx, cancel := context.WithTimeout(r.Context(), journalCheckTimeout)
	defer cancel()
	result, err := h.journals.Check(ctx, query)
	if err != nil {
		serverError(w, "Failed to check journal", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, result)
}

// JournalListStatus reports reference-list sizes and refresh times.
func (h *PortalHandler) JournalListStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	statuses, err := h.journals.Statuses(ctx)
	if err != nil {
		serverError(w, "Failed to read list status", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, statuses)
}

func itoa(n int) string { return strconv.Itoa(n) }
