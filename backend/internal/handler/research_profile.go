package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

const profilePublicationsLimit = 500

// ResearchProfileHandler lets each user maintain their own research links and read
// the data imported from them.
type ResearchProfileHandler struct {
	db           *pgxpool.Pool
	orchestrator *syncsvc.Orchestrator
}

func NewResearchProfileHandler(db *pgxpool.Pool, orchestrator *syncsvc.Orchestrator) *ResearchProfileHandler {
	return &ResearchProfileHandler{db: db, orchestrator: orchestrator}
}

type profilePublication struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Journal       *string `json:"journal"`
	Year          *int    `json:"year"`
	DOI           *string `json:"doi"`
	CitationCount int     `json:"citation_count"`
}

type researchProfile struct {
	Linked            bool                 `json:"linked"`
	GoogleScholarURL  *string              `json:"google_scholar_url"`
	ScopusURL         *string              `json:"scopus_url"`
	ResearchGateURL   *string              `json:"researchgate_url"`
	LinkedInURL       *string              `json:"linkedin_url"`
	WebOfScienceURL   *string              `json:"web_of_science_url"`
	Phone             *string              `json:"phone"`
	PhoneSecondary    *string              `json:"phone_secondary"`
	RecoveryEmail     *string              `json:"recovery_email"`
	ORCID             *string              `json:"orcid"`
	ORCIDURL          *string              `json:"orcid_url"`
	OpenAlexID        *string              `json:"openalex_id"`
	HIndex            int                  `json:"h_index"`
	HIndexSource      *string              `json:"h_index_source"`
	TotalCitations    int                  `json:"total_citations"`
	TotalPublications int                  `json:"total_publications"`
	LastSyncedAt      *time.Time           `json:"last_synced_at"`
	LastSyncError     *string              `json:"last_sync_error"`
	Syncing           bool                 `json:"syncing"`
	Publications      []profilePublication `json:"publications"`
}

const profileSelect = `
	SELECT r.google_scholar_url, r.google_scholar_id, r.scopus_url, r.scopus_author_id,
	       r.researchgate_url, r.linkedin_url, r.web_of_science_url,
	       r.phone, r.phone_secondary, r.recovery_email, r.orcid_id, r.openalex_id,
	       COALESCE(r.h_index, 0), r.h_index_source, COALESCE(r.total_citations, 0),
	       COALESCE(r.total_publications, 0), r.last_synced_at, r.last_sync_error,
	       r.last_sync_started_at, r.last_sync_finished_at
	FROM researchers r WHERE r.id = $1`

func scanProfile(row pgx.Row) (researchProfile, error) {
	var p researchProfile
	var scholarID, scopusID *string
	var started, finished *time.Time
	if err := row.Scan(&p.GoogleScholarURL, &scholarID, &p.ScopusURL, &scopusID,
		&p.ResearchGateURL, &p.LinkedInURL, &p.WebOfScienceURL,
		&p.Phone, &p.PhoneSecondary, &p.RecoveryEmail, &p.ORCID, &p.OpenAlexID,
		&p.HIndex, &p.HIndexSource, &p.TotalCitations, &p.TotalPublications,
		&p.LastSyncedAt, &p.LastSyncError, &started, &finished); err != nil {
		return p, err
	}
	p.Linked = true
	p.Syncing = syncsvc.IsSyncing(started, finished, time.Now())
	// Imported records hold only the id; without a link here, saving the form would erase it.
	if p.GoogleScholarURL == nil && scholarID != nil {
		link := researchlinks.ScholarURL(*scholarID)
		p.GoogleScholarURL = &link
	}
	if p.ScopusURL == nil && scopusID != nil {
		link := researchlinks.ScopusURL(*scopusID)
		p.ScopusURL = &link
	}
	if p.ORCID != nil {
		if id, ok := researchlinks.ParseORCID(*p.ORCID); ok {
			link := researchlinks.ORCIDURL(id)
			p.ORCID, p.ORCIDURL = &id, &link
		}
	}
	return p, nil
}

// findResearcherID resolves the caller's record without creating one, so merely
// opening the page does not add a row to the researcher register.
func (h *ResearchProfileHandler) findResearcherID(ctx context.Context, userID string) (*string, error) {
	return ownResearcherID(ctx, h.db, userID)
}

func (h *ResearchProfileHandler) loadProfile(ctx context.Context, researcherID *string) (researchProfile, error) {
	if researcherID == nil {
		return researchProfile{Publications: []profilePublication{}}, nil
	}
	profile, err := scanProfile(h.db.QueryRow(ctx, profileSelect, *researcherID))
	if err != nil {
		return profile, err
	}
	rows, err := h.db.Query(ctx, `
		SELECT p.id, COALESCE(p.title_ar, p.title_en, ''), COALESCE(j.name, p.metadata->>'journal_name'),
		       EXTRACT(YEAR FROM p.publication_date)::int, p.doi, COALESCE(p.citation_count, 0)
		FROM publications p
		LEFT JOIN journals j ON j.id = p.journal_id
		WHERE EXISTS (SELECT 1 FROM publication_authors pa WHERE pa.publication_id = p.id AND pa.researcher_id = $1)
		ORDER BY p.publication_date DESC NULLS LAST, p.citation_count DESC NULLS LAST
		LIMIT $2
	`, *researcherID, profilePublicationsLimit)
	if err != nil {
		return profile, err
	}
	profile.Publications, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (profilePublication, error) {
		var pub profilePublication
		err := row.Scan(&pub.ID, &pub.Title, &pub.Journal, &pub.Year, &pub.DOI, &pub.CitationCount)
		return pub, err
	})
	return profile, err
}

// Get returns the caller's links, figures and publications.
func (h *ResearchProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	researcherID, err := h.findResearcherID(ctx, user.UserID)
	if err != nil {
		serverError(w, "research profile lookup", err)
		return
	}
	profile, err := h.loadProfile(ctx, researcherID)
	if err != nil {
		serverError(w, "research profile load", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, profile)
}

// storedORCIDCanonical folds legacy spellings in SQL, so one that a migration could
// not rewrite (it would collide with another row) still counts as taken.
const storedORCIDCanonical = `upper(btrim(regexp_replace(btrim(orcid_id), '^(https?://)?(www\.)?orcid\.org/', '', 'i'), '/'))`

// claimedBy reports which identifier, if any, another researcher already holds.
// links.ORCID is canonical, and empty fields claim nothing.
func claimedBy(ctx context.Context, tx pgx.Tx, researcherID string, links researchlinks.Links) (string, error) {
	var code string
	err := tx.QueryRow(ctx, `
		SELECT CASE
			WHEN $2 <> '' AND `+storedORCIDCanonical+` = $2 THEN 'orcid_taken'
			WHEN $3 <> '' AND google_scholar_id = $3 THEN 'google_scholar_taken'
			ELSE 'scopus_taken'
		END
		FROM researchers
		WHERE id <> $1 AND (($2 <> '' AND `+storedORCIDCanonical+` = $2)
			OR ($3 <> '' AND google_scholar_id = $3)
			OR ($4 <> '' AND scopus_author_id = $4))
		LIMIT 1
	`, researcherID, links.ORCID, links.GoogleScholarID, links.ScopusAuthorID).Scan(&code)
	if isNoRows(err) {
		return "", nil
	}
	return code, err
}

// canonicalStoredORCID reads a stored value, leaving an unparseable one as it is so
// that replacing it still registers as a change.
func canonicalStoredORCID(stored *string) string {
	if stored == nil {
		return ""
	}
	if id, ok := researchlinks.ParseORCID(*stored); ok {
		return id
	}
	return *stored
}

func takenCodeFor(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return ""
	}
	switch pgErr.ConstraintName {
	case "researchers_orcid_id_key":
		return "orcid_taken"
	case "researchers_google_scholar_id_key":
		return "google_scholar_taken"
	case "researchers_scopus_author_id_key":
		return "scopus_taken"
	}
	return ""
}

// saveLinks writes the form; it reports the researcher and whether the ORCID changed.
func (h *ResearchProfileHandler) saveLinks(ctx context.Context, userID string, links researchlinks.Links, contact contactUpdate) (researcherID string, orcidChanged bool, err error) {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit

	rec := requestRecord{ApplicantID: userID}
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(name_ar, ''), email, department_id FROM users WHERE id = $1`, userID,
	).Scan(&rec.ApplicantName, &rec.ApplicantEmail, &rec.DepartmentID); err != nil {
		return "", false, err
	}
	researcherID, err = ensureResearcher(ctx, tx, rec)
	if err != nil {
		return "", false, err
	}
	if researcherID == "" {
		// A researcher record with this email exists, but the account's email is not
		// verified: links must never be written into a record the caller may not own.
		return "", false, profileConflict("researcher_not_linked")
	}

	var stored *string
	var started, finished *time.Time
	var now time.Time
	if err := tx.QueryRow(ctx, `
		SELECT orcid_id, last_sync_started_at, last_sync_finished_at, NOW() FROM researchers WHERE id = $1 FOR UPDATE
	`, researcherID).Scan(&stored, &started, &finished, &now); err != nil {
		return "", false, err
	}
	orcidChanged = canonicalStoredORCID(stored) != links.ORCID

	code, err := claimedBy(ctx, tx, researcherID, links)
	if err != nil {
		return "", false, err
	}
	if code != "" {
		return "", false, profileConflict(code)
	}
	// Removing an ORCID starts nothing, so only entering a new one waits. The cooldown
	// survives the removal, which closes the clear-then-enter way around it.
	if orcidChanged && links.ORCID != "" && !syncsvc.ORCIDChangeAllowed(started, finished, now) {
		return "", false, errORCIDChangeTooSoon
	}
	if orcidChanged {
		if err := syncsvc.ResetImported(ctx, tx, researcherID); err != nil {
			return "", false, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE researchers SET
			google_scholar_url = NULLIF($2, ''), google_scholar_id = NULLIF($3, ''),
			scopus_url = NULLIF($4, ''), scopus_author_id = NULLIF($5, ''),
			researchgate_url = NULLIF($6, ''), linkedin_url = NULLIF($7, ''),
			orcid_attested = orcid_attested AND orcid_id IS NOT DISTINCT FROM NULLIF($8, ''),
			orcid_id = NULLIF($8, ''), web_of_science_url = NULLIF($9, ''),
			phone = CASE WHEN $10::text IS NULL THEN phone ELSE NULLIF($10::text, '') END,
			phone_secondary = CASE WHEN $11::text IS NULL THEN phone_secondary ELSE NULLIF($11::text, '') END,
			recovery_email = CASE WHEN $12::text IS NULL THEN recovery_email ELSE NULLIF($12::text, '') END,
			updated_at = NOW()
		WHERE id = $1
	`, researcherID, links.GoogleScholarURL, links.GoogleScholarID, links.ScopusURL, links.ScopusAuthorID,
		links.ResearchGateURL, links.LinkedInURL, links.ORCID, links.WebOfScienceURL,
		contact.Phone, contact.PhoneSecondary, contact.RecoveryEmail); err != nil {
		if code := takenCodeFor(err); code != "" {
			return "", false, profileConflict(code)
		}
		return "", false, err
	}
	return researcherID, orcidChanged, tx.Commit(ctx)
}

// errORCIDChangeTooSoon carries the code the client localizes.
var errORCIDChangeTooSoon = errors.New("orcid_change_cooldown")

type profileConflict string

func (c profileConflict) Error() string { return string(c) }

// Update validates and stores the links, then syncs when there is something new to fetch.
func (h *ResearchProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var input struct {
		researchlinks.Input
		contactUpdate
	}
	if !decodeBody(w, r, &input) {
		return
	}
	links, problems := researchlinks.Parse(input.Input)
	contact, contactProblems := input.contactUpdate.normalized()
	problems = append(problems, contactProblems...)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	researcherID, orcidChanged, err := h.saveLinks(ctx, user.UserID, links, contact)
	var conflict profileConflict
	if errors.As(err, &conflict) {
		model.RespondError(w, http.StatusConflict, string(conflict))
		return
	}
	if errors.Is(err, errORCIDChangeTooSoon) {
		model.RespondError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	if err != nil {
		serverError(w, "research profile save", err)
		return
	}

	profile, err := h.loadProfile(ctx, &researcherID)
	if err != nil {
		serverError(w, "research profile load", err)
		return
	}
	// A record that has an ORCID but was never synced (imported by an admin) also starts now.
	if links.ORCID != "" && (orcidChanged || profile.LastSyncedAt == nil) && !profile.Syncing {
		switch err := h.orchestrator.Start(ctx, researcherID); {
		case err == nil:
			profile.Syncing, profile.LastSyncError = true, nil
		case errors.Is(err, syncsvc.ErrCooldown):
			// The links are saved; the next refresh picks the data up.
		case errors.Is(err, syncsvc.ErrBusy):
			busy := syncsvc.ErrBusy.Error()
			profile.LastSyncError = &busy
		default:
			serverError(w, "research profile sync claim", err)
			return
		}
	}
	model.RespondJSON(w, http.StatusOK, profile)
}

// Sync starts a refresh of the caller's data.
func (h *ResearchProfileHandler) Sync(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	researcherID, err := h.findResearcherID(ctx, user.UserID)
	if err != nil {
		serverError(w, "research profile lookup", err)
		return
	}
	var orcid *string
	if researcherID != nil {
		if err := h.db.QueryRow(ctx, `SELECT orcid_id FROM researchers WHERE id = $1`, *researcherID).Scan(&orcid); err != nil {
			serverError(w, "research profile lookup", err)
			return
		}
	}
	if orcid == nil || *orcid == "" {
		validationError(w, []string{researchlinks.FieldORCID + ":orcid_required"})
		return
	}

	if !respondSyncStart(w, h.orchestrator.Start(ctx, *researcherID)) {
		return
	}
	model.RespondJSON(w, http.StatusAccepted, map[string]bool{"syncing": true})
}

// StaffLinks returns one researcher's profile links for the staff-facing detail page.
func (h *ResearchProfileHandler) StaffLinks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "not_found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()

	profile, err := scanProfile(h.db.QueryRow(ctx, profileSelect, id))
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		serverError(w, "researcher links", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, map[string]any{
		"google_scholar_url": profile.GoogleScholarURL,
		"scopus_url":         profile.ScopusURL,
		"researchgate_url":   profile.ResearchGateURL,
		"linkedin_url":       profile.LinkedInURL,
		"web_of_science_url": profile.WebOfScienceURL,
		"orcid_url":          profile.ORCIDURL,
		"last_synced_at":     profile.LastSyncedAt,
		"last_sync_error":    profile.LastSyncError,
	})
}
