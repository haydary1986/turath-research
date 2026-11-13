package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// bulkDB is the part of the pool the store uses.
type bulkDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const bulkEligible = `orcid_id IS NOT NULL AND btrim(orcid_id) <> ''
	AND ($1::uuid IS NULL OR id > $1::uuid)
	AND (NOT $2::boolean OR last_synced_at IS NULL OR last_sync_error IS NOT NULL)`

const bulkRunColumns = `id, status, only_failed, cursor_id, processed, succeeded, failed, skipped, failures,
	wait_reason, resume_at, window_started_at, window_count, started_at, updated_at, finished_at`

type bulkStore struct {
	db           bulkDB
	orchestrator *Orchestrator
}

func scanBulkRun(row pgx.Row) (BulkRun, error) {
	var r BulkRun
	err := row.Scan(&r.ID, &r.Status, &r.OnlyFailed, &r.Cursor, &r.Processed, &r.Succeeded, &r.Failed, &r.Skipped,
		&r.Failures, &r.WaitReason, &r.ResumeAt, &r.WindowStartedAt, &r.WindowCount, &r.StartedAt, &r.UpdatedAt, &r.FinishedAt)
	return r, err
}

func (s *bulkStore) loadRun(ctx context.Context, id string) (BulkRun, error) {
	return scanBulkRun(s.db.QueryRow(ctx, `SELECT `+bulkRunColumns+` FROM bulk_sync_runs WHERE id = $1`, id))
}

// saveRun never overrides a stop the admin made while a researcher was in flight.
func (s *bulkStore) saveRun(ctx context.Context, r BulkRun) error {
	_, err := s.db.Exec(ctx, `
		UPDATE bulk_sync_runs SET
			status = CASE WHEN status = 'stopped' THEN status ELSE $2 END,
			wait_reason = CASE WHEN status = 'stopped' THEN NULL ELSE $3 END,
			resume_at = CASE WHEN status = 'stopped' THEN NULL ELSE $4::timestamptz END,
			finished_at = CASE WHEN status = 'stopped' THEN NULL ELSE $5::timestamptz END,
			cursor_id = $6, processed = $7, succeeded = $8, failed = $9, skipped = $10, failures = $11,
			window_started_at = $12, window_count = $13, updated_at = NOW()
		WHERE id = $1
	`, r.ID, r.Status, r.WaitReason, r.ResumeAt, r.FinishedAt, r.Cursor, r.Processed, r.Succeeded, r.Failed, r.Skipped,
		r.Failures, r.WindowStartedAt, r.WindowCount)
	return err
}

func (s *bulkStore) nextResearcher(ctx context.Context, after *string, onlyFailed bool) (string, bool, error) {
	var id string
	err := s.db.QueryRow(ctx, `SELECT id FROM researchers WHERE `+bulkEligible+` ORDER BY id LIMIT 1`, after, onlyFailed).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

func (s *bulkStore) syncOne(ctx context.Context, researcherID string) error {
	return s.orchestrator.syncNow(ctx, researcherID)
}

func (s *bulkStore) cooldownWait(ctx context.Context, researcherID string) (time.Duration, error) {
	var started, synced *time.Time
	var now time.Time
	if err := s.db.QueryRow(ctx,
		`SELECT last_sync_started_at, last_synced_at, NOW() FROM researchers WHERE id = $1`, researcherID,
	).Scan(&started, &synced, &now); err != nil {
		return 0, err
	}
	if started == nil || (synced != nil && !synced.Before(*started)) {
		return 0, nil
	}
	return started.Add(SyncCooldown + time.Second).Sub(now), nil
}

// syncNow runs one researcher to the end on the caller's goroutine, waiting for a
// slot, so a long pass shares the limit with people refreshing their own pages.
func (o *Orchestrator) syncNow(ctx context.Context, researcherID string) (err error) {
	select {
	case o.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer o.release()
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("sync panicked: %v", failure)
		}
	}()
	startedAt, ok, err := o.Claim(ctx, researcherID)
	if err != nil {
		return fmt.Errorf("claim sync: %w", err)
	}
	if !ok {
		return ErrCooldown
	}
	runCtx, cancel := context.WithTimeout(ctx, SyncTimeout)
	defer cancel()
	return o.Run(runCtx, researcherID, startedAt)
}

// BulkProgress is what the admin sees: the latest run and how much is left.
type BulkProgress struct {
	Run       *BulkRun `json:"run"`
	Eligible  int      `json:"eligible"`
	Remaining int      `json:"remaining"`
	Active    bool     `json:"active"`
}

func (s *bulkStore) latestRun(ctx context.Context) (*BulkRun, error) {
	run, err := scanBulkRun(s.db.QueryRow(ctx, `SELECT `+bulkRunColumns+` FROM bulk_sync_runs ORDER BY started_at DESC LIMIT 1`))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *bulkStore) count(ctx context.Context, after *string, onlyFailed bool) (int, error) {
	var total int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM researchers WHERE `+bulkEligible, after, onlyFailed).Scan(&total)
	return total, err
}

// Progress reports the latest run. It starts nothing and calls no upstream.
func (b *BulkRunner) Progress(ctx context.Context) (BulkProgress, error) {
	var progress BulkProgress
	var err error
	if progress.Eligible, err = b.store.count(ctx, nil, false); err != nil {
		return progress, err
	}
	if progress.Run, err = b.store.latestRun(ctx); err != nil {
		return progress, err
	}
	progress.Remaining = progress.Eligible
	if run := progress.Run; run != nil {
		progress.Active = run.Status == BulkStatusRunning || run.Status == BulkStatusWaiting
		if run.Status == BulkStatusCompleted {
			progress.Remaining = 0
		} else if progress.Remaining, err = b.store.count(ctx, run.Cursor, run.OnlyFailed); err != nil {
			return progress, err
		}
	}
	return progress, nil
}

// BulkStartOptions choose what a new pass covers.
type BulkStartOptions struct {
	// OnlyFailed covers researchers never synced or whose last run ended with a problem.
	OnlyFailed bool
	// Fresh starts over rather than resuming a stopped pass.
	Fresh  bool
	UserID string
}

// Start continues the active or stopped pass, or begins a new one.
func (b *BulkRunner) Start(ctx context.Context, opts BulkStartOptions) error {
	if latest, err := b.store.latestRun(ctx); err != nil {
		return err
	} else if latest != nil {
		switch {
		case latest.Status == BulkStatusRunning || latest.Status == BulkStatusWaiting:
			b.launch(latest.ID)
			return nil
		case latest.Status == BulkStatusStopped && !opts.Fresh:
			if _, err := b.store.db.Exec(ctx, `
				UPDATE bulk_sync_runs SET status = 'running', updated_at = NOW() WHERE id = $1 AND status = 'stopped'
			`, latest.ID); err != nil {
				return err
			}
			b.launch(latest.ID)
			return nil
		}
	}
	var id string
	err := b.store.db.QueryRow(ctx, `
		INSERT INTO bulk_sync_runs (status, only_failed, started_by) VALUES ('running', $1, $2) RETURNING id
	`, opts.OnlyFailed, opts.UserID).Scan(&id)
	if err != nil {
		return err
	}
	b.launch(id)
	return nil
}

// Stop halts the pass after the researcher in flight; Start continues it from there.
func (b *BulkRunner) Stop(ctx context.Context) error {
	tag, err := b.store.db.Exec(ctx, `
		UPDATE bulk_sync_runs SET status = 'stopped', wait_reason = NULL, resume_at = NULL, updated_at = NOW()
		WHERE status IN ('running', 'waiting')
	`)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBulkNotActive
	}
	return nil
}

// Resume continues a pass a restart interrupted. ctx ends the loop at shutdown.
func (b *BulkRunner) Resume(ctx context.Context) {
	b.mu.Lock()
	b.base = ctx
	b.mu.Unlock()

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	latest, err := b.store.latestRun(readCtx)
	if err != nil {
		slog.Error("bulk sync resume check failed", "error", err)
		return
	}
	if latest != nil && (latest.Status == BulkStatusRunning || latest.Status == BulkStatusWaiting) {
		slog.Info("bulk sync resuming", "run", latest.ID, "processed", latest.Processed)
		b.launch(latest.ID)
	}
}
