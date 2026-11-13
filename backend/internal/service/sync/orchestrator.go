package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
)

const (
	// SyncTimeout bounds one run; it must stay below SyncCooldown so runs never overlap.
	SyncTimeout  = 3 * time.Minute
	SyncCooldown = 5 * time.Minute

	// MaxConcurrentSyncs bounds the runs one process does at once, whoever asks.
	MaxConcurrentSyncs = 4
	doiEnrichTimeout   = 30 * time.Second

	openAlexMaxPages = 5
	finishTimeout    = 10 * time.Second
)

var (
	// ErrCooldown means a sync for this researcher started too recently.
	ErrCooldown = errors.New("sync_cooldown")
	// ErrBusy means every sync slot is taken; the caller may try again shortly.
	ErrBusy = errors.New("sync_busy")
	// errOpenAlexOtherPerson means OpenAlex filed this ORCID under a different name.
	errOpenAlexOtherPerson = errors.New("openalex author name differs from the orcid record")
)

// RunError is a run's refusal or partial result, by the code the client localizes.
type RunError struct {
	Code string
	// RateLimited: an upstream's budget ran out, so more calls now would be wasted.
	RateLimited bool
}

func (e *RunError) Error() string { return e.Code }

// Orchestrator coordinates data import from multiple academic APIs.
type Orchestrator struct {
	db       *pgxpool.Pool
	openalex *apiclient.OpenAlexClient
	crossref *apiclient.CrossRefClient
	orcid    *apiclient.ORCIDClient
	slots    chan struct{}
}

// NewOrchestrator creates a sync orchestrator with all API clients.
func NewOrchestrator(db *pgxpool.Pool, crossrefMailto, openAlexKey string) *Orchestrator {
	return &Orchestrator{
		db:       db,
		openalex: apiclient.NewOpenAlexClient(crossrefMailto, openAlexKey),
		crossref: apiclient.NewCrossRefClient(crossrefMailto),
		orcid:    apiclient.NewORCIDClient(),
		slots:    make(chan struct{}, MaxConcurrentSyncs),
	}
}

// IsSyncing reports whether a run is in flight. A run that never finished, because
// the process died, stops counting once it is older than SyncTimeout.
func IsSyncing(started, finished *time.Time, now time.Time) bool {
	if started == nil || now.Sub(*started) > SyncTimeout {
		return false
	}
	return finished == nil || finished.Before(*started)
}

// ORCIDChangeAllowed reports whether a researcher's ORCID may be replaced now. A new
// iD starts a new import, so it answers to the same limits as a manual refresh.
func ORCIDChangeAllowed(started, finished *time.Time, now time.Time) bool {
	if started == nil {
		return true
	}
	return !IsSyncing(started, finished, now) && now.Sub(*started) >= SyncCooldown
}

// Claim atomically marks a run as started; ok is false when the cooldown refused it.
func (o *Orchestrator) Claim(ctx context.Context, researcherID string) (startedAt time.Time, ok bool, err error) {
	err = o.db.QueryRow(ctx, `
		UPDATE researchers SET last_sync_started_at = NOW(), last_sync_error = NULL
		WHERE id = $1
		  AND (last_sync_started_at IS NULL OR last_sync_started_at < NOW() - make_interval(secs => $2))
		RETURNING last_sync_started_at
	`, researcherID, SyncCooldown.Seconds()).Scan(&startedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	return startedAt, err == nil, err
}

func (o *Orchestrator) tryAcquire() bool {
	select {
	case o.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (o *Orchestrator) release() { <-o.slots }

// runDetached outlives the request that asked for the work and holds a slot until done.
func (o *Orchestrator) runDetached(timeout time.Duration, label string, work func(context.Context) error) {
	go func() {
		defer o.release()
		defer func() {
			if failure := recover(); failure != nil {
				slog.Error("background sync panicked", "work", label, "panic", fmt.Sprint(failure))
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := work(ctx); err != nil {
			slog.Warn("background sync ended with a problem", "work", label, "error", err)
		}
	}()
}

// Start claims a researcher's sync and runs it in the background. It returns ErrBusy
// when no slot is free and ErrCooldown when the claim is refused; a slot is taken
// first so a refused start never burns the researcher's cooldown.
func (o *Orchestrator) Start(ctx context.Context, researcherID string) error {
	if !o.tryAcquire() {
		return ErrBusy
	}
	startedAt, ok, err := o.Claim(ctx, researcherID)
	if err != nil || !ok {
		o.release()
		if err != nil {
			return fmt.Errorf("claim sync: %w", err)
		}
		return ErrCooldown
	}
	o.runDetached(SyncTimeout, "researcher:"+researcherID, func(ctx context.Context) error {
		return o.Run(ctx, researcherID, startedAt)
	})
	return nil
}

// StartDOI enriches one publication in the background, under the same slot limit.
func (o *Orchestrator) StartDOI(doi string) error {
	if !o.tryAcquire() {
		return ErrBusy
	}
	o.runDetached(doiEnrichTimeout, "doi", func(ctx context.Context) error {
		return o.SyncByDOI(ctx, doi)
	})
	return nil
}

// Run imports a claimed researcher's works. Attribution comes only from identifiers
// the researcher supplied: their ORCID record and the OpenAlex author it resolves to.
func (o *Orchestrator) Run(ctx context.Context, researcherID string, startedAt time.Time) error {
	var stored, nameEn *string
	var nameAr string
	var attested bool
	if err := o.db.QueryRow(ctx,
		`SELECT orcid_id, name_ar, name_en, orcid_attested FROM researchers WHERE id = $1`, researcherID,
	).Scan(&stored, &nameAr, &nameEn, &attested); err != nil {
		return fmt.Errorf("researcher not found: %w", err)
	}
	researcherNames := []string{nameAr}
	if nameEn != nil {
		researcherNames = append(researcherNames, *nameEn)
	}
	orcid, valid := "", false
	if stored != nil {
		orcid, valid = researchlinks.ParseORCID(*stored)
	}
	if !valid {
		o.finish(researcherID, startedAt, syncOutcome{Code: codeNoORCID})
		return &RunError{Code: codeNoORCID}
	}

	jobID := o.startJob(ctx, researcherID)
	fetched := o.fetch(ctx, researcherID, orcid, researcherNames, attested)
	saved, failed, err := o.persist(ctx, researcherID, orcid, startedAt, fetched)
	if errors.Is(err, errSuperseded) {
		o.finishJob(jobID, "completed", codeSuperseded)
		return nil
	}
	if err != nil {
		slog.Error("sync persist failed", "researcher", researcherID, "error", err)
	}

	outcome := outcomeFor(fetched, failed)
	if err != nil {
		outcome = syncOutcome{Code: codeSaveFailed}
	}
	o.finish(researcherID, startedAt, outcome)
	status := "completed"
	if !outcome.Succeeded {
		status = "failed"
	}
	o.finishJob(jobID, status, outcome.Code)
	slog.Info("sync complete", "researcher", researcherID, "saved", saved, "failed", failed, "code", outcome.Code)
	if outcome.Code != "" {
		return &RunError{Code: outcome.Code, RateLimited: errors.Is(fetched.orcidErr, apiclient.ErrRateLimited) ||
			errors.Is(fetched.openalexErr, apiclient.ErrRateLimited)}
	}
	return nil
}

// fetch checks identity before it believes either source: the ORCID record must carry
// the researcher's name, and the OpenAlex author must carry the ORCID record's name.
func (o *Orchestrator) fetch(ctx context.Context, researcherID, orcid string, researcherNames []string, attested bool) fetchResult {
	var result fetchResult
	person, err := o.orcid.GetPerson(ctx, orcid)
	if err != nil {
		result.orcidErr, result.identityUnknown = err, true
		slog.Warn("orcid identity check failed", "researcher", researcherID, "error", err)
		return result
	}
	orcidNames := person.Names()
	if code := orcidIdentityCode(researcherNames, orcidNames); code != "" && !o.attestedByUniversity(ctx, researcherID, orcid, attested) {
		result.identityCode = code
		slog.Warn("orcid record refused by the name check", "researcher", researcherID, "code", code)
		return result
	}

	var orcidWorks, openalexWorks []importedWork
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		orcidWorks, result.orcidErr = o.fetchORCID(ctx, orcid)
		if result.orcidErr != nil {
			slog.Warn("orcid sync failed", "researcher", researcherID, "error", result.orcidErr)
		}
	}()
	go func() {
		defer wg.Done()
		var openalexErr error
		result.author, openalexWorks, result.truncated, openalexErr = o.fetchOpenAlex(ctx, orcid, orcidNames)
		switch {
		case errors.Is(openalexErr, errOpenAlexOtherPerson):
			result.identityCode = CodeOpenAlexNameMismatch
			slog.Warn("openalex author refused by the name check", "researcher", researcherID)
		case openalexErr != nil:
			result.openalexErr = openalexErr
			slog.Warn("openalex sync failed", "researcher", researcherID, "error", openalexErr)
		}
	}()
	wg.Wait()
	result.works = mergeWorks(openalexWorks, orcidWorks)
	return result
}

// attestedByUniversity lets an administrator-imported ORCID pass a failed name
// comparison, only when the record itself names the university as an employer.
func (o *Orchestrator) attestedByUniversity(ctx context.Context, researcherID, orcid string, attested bool) bool {
	if !attested {
		return false
	}
	employments, err := o.orcid.GetEmployments(ctx, orcid)
	if err != nil {
		slog.Warn("orcid employment check failed", "researcher", researcherID, "error", err)
		return false
	}
	return listsUniversity(employments)
}

func (o *Orchestrator) fetchORCID(ctx context.Context, orcid string) ([]importedWork, error) {
	group, err := o.orcid.GetWorks(ctx, orcid)
	if err != nil {
		return nil, err
	}
	works := []importedWork{}
	for _, g := range group.Group {
		for _, summary := range g.WorkSummary {
			works = append(works, workFromORCID(summary))
		}
	}
	return works, nil
}

// fetchOpenAlex may return an author together with an error: the summary figures
// are then still usable even though the work list is incomplete.
func (o *Orchestrator) fetchOpenAlex(ctx context.Context, orcid string, orcidNames []string) (*apiclient.OpenAlexPerson, []importedWork, bool, error) {
	author, err := o.openalex.FindAuthorByORCID(ctx, orcid)
	if err != nil || author == nil {
		return nil, nil, false, err
	}
	authorID := openAlexShortID(author.ID)
	if authorID == "" {
		return nil, nil, false, fmt.Errorf("openalex returned unusable author id %q", author.ID)
	}
	if got, ok := researchlinks.ParseORCID(author.ORCID); !ok || got != orcid {
		return nil, nil, false, fmt.Errorf("openalex returned another author's orcid %q", author.ORCID)
	}
	if !openAlexTrusted(orcidNames, author) {
		return nil, nil, false, errOpenAlexOtherPerson
	}
	raw, truncated, err := o.openalex.ListWorksByAuthor(ctx, authorID, openAlexMaxPages)
	if err != nil {
		return author, nil, false, err
	}
	works := make([]importedWork, 0, len(raw))
	for _, w := range raw {
		works = append(works, workFromOpenAlex(w))
	}
	return author, works, truncated, nil
}

func (o *Orchestrator) startJob(ctx context.Context, researcherID string) string {
	jobID := uuid.New().String()
	if _, err := o.db.Exec(ctx, `
		INSERT INTO sync_jobs (id, job_type, target_type, target_id, status, started_at)
		VALUES ($1, 'openalex_sync', 'researcher', $2, 'running', NOW())
	`, jobID, researcherID); err != nil {
		slog.Warn("sync job not recorded", "researcher", researcherID, "error", err)
	}
	return jobID
}

// finishJob and finish use their own context: a run that hit its deadline must still
// record that it ended, or the page would keep showing it as running.
func (o *Orchestrator) finishJob(jobID, status, code string) {
	ctx, cancel := context.WithTimeout(context.Background(), finishTimeout)
	defer cancel()
	if _, err := o.db.Exec(ctx, `
		UPDATE sync_jobs SET status = $2, completed_at = NOW(), error_message = NULLIF($3, '')
		WHERE id = $1
	`, jobID, status, code); err != nil {
		slog.Warn("sync job not updated", "job", jobID, "error", err)
	}
}

// finish matches on startedAt so a superseded run cannot overwrite a newer run's state.
func (o *Orchestrator) finish(researcherID string, startedAt time.Time, outcome syncOutcome) {
	ctx, cancel := context.WithTimeout(context.Background(), finishTimeout)
	defer cancel()
	if _, err := o.db.Exec(ctx, `
		UPDATE researchers SET
			last_sync_finished_at = NOW(),
			last_sync_error = NULLIF($3, ''),
			last_synced_at = CASE WHEN $4 THEN NOW() ELSE last_synced_at END
		WHERE id = $1 AND last_sync_started_at = $2
	`, researcherID, startedAt, outcome.Code, outcome.Succeeded); err != nil {
		slog.Error("sync state not recorded", "researcher", researcherID, "error", err)
	}
}

// SyncByDOI fills gaps in a publication from CrossRef. It never replaces a value
// already stored and leaves citation counts to OpenAlex.
func (o *Orchestrator) SyncByDOI(ctx context.Context, doi string) error {
	work, err := o.crossref.GetWorkByDOI(ctx, doi)
	if err != nil {
		return fmt.Errorf("crossref lookup: %w", err)
	}

	_, err = o.db.Exec(ctx, `
		UPDATE publications SET
			title_en = COALESCE(title_en, NULLIF($2, '')),
			abstract_en = COALESCE(abstract_en, NULLIF($3, '')),
			publication_date = COALESCE(publication_date, $4::date),
			volume = COALESCE(volume, NULLIF($5, '')),
			issue = COALESCE(issue, NULLIF($6, '')),
			pages = COALESCE(pages, NULLIF($7, '')),
			url = COALESCE(url, NULLIF($8, '')),
			updated_at = NOW()
		WHERE lower(doi) = $1
	`, doi, work.GetFirstTitle(), work.Abstract, nilIfEmptyStr(work.GetPublicationDate()),
		work.Volume, work.Issue, work.Page, work.URL)

	return err
}

// NormalizeDOI reduces any spelling to the bare lowercase form; "" when unusable.
func NormalizeDOI(raw string) string { return normalizeDOI(raw) }

func nilIfEmptyStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
