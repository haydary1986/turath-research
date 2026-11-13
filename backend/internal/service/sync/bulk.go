package sync

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	stdsync "sync"
	"time"
)

const (
	BulkStatusRunning   = "running"
	BulkStatusWaiting   = "waiting"
	BulkStatusStopped   = "stopped"
	BulkStatusCompleted = "completed"

	// Why a run is waiting, for the admin's progress view.
	bulkWaitRateLimited = "upstream_rate_limited"
	bulkWaitUnavailable = "upstream_unavailable"
	bulkWaitDailyLimit  = "daily_limit"

	// bulkBackoff is long enough for an hourly or daily budget to recover, and a retry
	// that finds it still empty spends almost nothing.
	bulkBackoff = time.Hour
	// bulkDailyWindow is the span BULK_SYNC_DAILY_LIMIT counts over.
	bulkDailyWindow = 24 * time.Hour
	// bulkMaxOutages: this many upstream failures in a row means the upstream is down.
	bulkMaxOutages = 3
	bulkPollEvery  = 30 * time.Second

	bulkCodeInternal = "sync_error"
)

// ErrBulkNotActive means there is no run to stop.
var ErrBulkNotActive = errors.New("bulk_sync_not_active")

// BulkConfig paces a run. Zero values are valid: no delay, no daily cap.
type BulkConfig struct {
	Delay      time.Duration
	DailyLimit int
}

// BulkRun is the persisted state of one pass over every researcher with an ORCID.
// It holds counts and an opaque cursor only, so it is safe to log and to show.
type BulkRun struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	OnlyFailed      bool           `json:"only_failed"`
	Cursor          *string        `json:"-"`
	Processed       int            `json:"processed"`
	Succeeded       int            `json:"succeeded"`
	Failed          int            `json:"failed"`
	Skipped         int            `json:"skipped"`
	Failures        map[string]int `json:"failures"`
	WaitReason      *string        `json:"wait_reason"`
	ResumeAt        *time.Time     `json:"resume_at"`
	WindowStartedAt time.Time      `json:"-"`
	WindowCount     int            `json:"-"`
	StartedAt       time.Time      `json:"started_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	FinishedAt      *time.Time     `json:"finished_at"`
}

func (r BulkRun) waitingFor(reason string, until time.Time) BulkRun {
	r.Status, r.WaitReason, r.ResumeAt = BulkStatusWaiting, &reason, &until
	return r
}

func (r BulkRun) resumed() BulkRun {
	r.Status, r.WaitReason, r.ResumeAt = BulkStatusRunning, nil, nil
	return r
}

// admitted spends one unit of the daily cap, or says the run must wait for the window.
func (r BulkRun) admitted(now time.Time, limit int) (BulkRun, bool) {
	if now.Sub(r.WindowStartedAt) >= bulkDailyWindow {
		r.WindowStartedAt, r.WindowCount = now, 0
	}
	if limit > 0 && r.WindowCount >= limit {
		return r.waitingFor(bulkWaitDailyLimit, r.WindowStartedAt.Add(bulkDailyWindow)), false
	}
	r.WindowCount++
	return r, true
}

// settled records a researcher's result. A rate limit leaves the cursor in place, so
// that researcher is retried once the budget is back; anything else moves on.
func (r BulkRun) settled(id string, err error, outages int, now time.Time) (BulkRun, int) {
	var runErr *RunError
	if errors.As(err, &runErr) && runErr.RateLimited {
		return r.waitingFor(bulkWaitRateLimited, now.Add(bulkBackoff)), 0
	}

	r.Cursor = &id
	r.Processed++
	switch {
	case err == nil:
		r.Succeeded++
		return r, 0
	case errors.Is(err, ErrCooldown):
		r.Skipped++
		return r, outages
	}

	code := bulkCodeInternal
	if runErr != nil {
		code = runErr.Code
	}
	r.Failed++
	r.Failures = maps.Clone(r.Failures)
	if r.Failures == nil {
		r.Failures = map[string]int{}
	}
	r.Failures[code]++

	if code != codeORCIDUnavailable && code != codeOpenAlexUnavailable && code != codeSourcesUnavailable {
		return r, 0
	}
	outages++
	if outages >= bulkMaxOutages {
		return r.waitingFor(bulkWaitUnavailable, now.Add(bulkBackoff)), 0
	}
	return r, outages
}

// bulkBackend is what a run needs from the outside world.
type bulkBackend interface {
	loadRun(ctx context.Context, id string) (BulkRun, error)
	saveRun(ctx context.Context, run BulkRun) error
	nextResearcher(ctx context.Context, after *string, onlyFailed bool) (string, bool, error)
	syncOne(ctx context.Context, researcherID string) error
	// cooldownWait is how long until a refused researcher can run, or 0 when its last run
	// succeeded and so needs no repeat. A run cut off by a restart lands in the first case.
	cooldownWait(ctx context.Context, researcherID string) (time.Duration, error)
}

// BulkRunner walks the researchers one at a time. Everything needed to continue is
// stored after each one, so a restart loses at most the researcher in flight.
// The backend runs as one process, so one in-memory guard is enough.
type BulkRunner struct {
	backend bulkBackend
	store   *bulkStore
	cfg     BulkConfig
	now     func() time.Time
	sleep   func(context.Context, time.Duration)

	mu      stdsync.Mutex
	base    context.Context
	running bool
	again   string
}

// NewBulkRunner creates a runner that does nothing until started or resumed.
func NewBulkRunner(db bulkDB, orchestrator *Orchestrator, cfg BulkConfig) *BulkRunner {
	store := &bulkStore{db: db, orchestrator: orchestrator}
	return &BulkRunner{
		backend: store, store: store, cfg: cfg,
		now: time.Now, sleep: sleepContext, base: context.Background(),
	}
}

func sleepContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// launch starts the loop unless one is running, which then re-reads the run itself.
func (b *BulkRunner) launch(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		b.again = runID
		return
	}
	b.running = true
	go b.supervise(b.base, runID)
}

func (b *BulkRunner) supervise(ctx context.Context, runID string) {
	for {
		b.safeLoop(ctx, runID)
		b.mu.Lock()
		runID = b.again
		b.again = ""
		if runID == "" || ctx.Err() != nil {
			b.running = false
			b.mu.Unlock()
			return
		}
		b.mu.Unlock()
	}
}

func (b *BulkRunner) safeLoop(ctx context.Context, runID string) {
	defer func() {
		if failure := recover(); failure != nil {
			slog.Error("bulk sync panicked", "run", runID)
		}
	}()
	b.loop(ctx, runID)
}

func (b *BulkRunner) loop(ctx context.Context, runID string) {
	outages := 0
	for ctx.Err() == nil {
		run, err := b.backend.loadRun(ctx, runID)
		if err != nil {
			slog.Error("bulk sync cannot read its run", "run", runID, "error", err)
			b.sleep(ctx, bulkPollEvery)
			continue
		}
		if run.Status == BulkStatusStopped || run.Status == BulkStatusCompleted {
			return
		}
		if run.Status == BulkStatusWaiting {
			if run.ResumeAt != nil && run.ResumeAt.After(b.now()) {
				b.sleep(ctx, min(run.ResumeAt.Sub(b.now()), bulkPollEvery))
				continue
			}
			run = run.resumed()
		}
		var delay time.Duration
		run, outages, delay = b.step(ctx, run, outages)
		if ctx.Err() != nil {
			return // the cursor stays, so this researcher runs again after the restart
		}
		if err := b.backend.saveRun(ctx, run); err != nil {
			slog.Error("bulk sync progress not saved", "run", runID, "error", err)
			delay = bulkPollEvery
		}
		if run.Status == BulkStatusCompleted {
			return
		}
		b.sleep(ctx, delay)
	}
}

// step settles the next researcher, and returns how long to pause before another.
func (b *BulkRunner) step(ctx context.Context, run BulkRun, outages int) (BulkRun, int, time.Duration) {
	admitted, ok := run.admitted(b.now(), b.cfg.DailyLimit)
	if !ok {
		return admitted, outages, 0
	}
	id, found, err := b.backend.nextResearcher(ctx, run.Cursor, run.OnlyFailed)
	if err != nil {
		slog.Error("bulk sync cannot read the next researcher", "run", run.ID, "error", err)
		return run, outages, bulkPollEvery
	}
	if !found {
		now := b.now()
		run.Status, run.FinishedAt, run.WaitReason, run.ResumeAt = BulkStatusCompleted, &now, nil, nil
		return run, outages, 0
	}
	settled, outages := admitted.settled(id, b.syncWithCooldown(ctx, id), outages, b.now())
	return settled, outages, b.cfg.Delay
}

func (b *BulkRunner) syncWithCooldown(ctx context.Context, id string) error {
	err := b.backend.syncOne(ctx, id)
	if !errors.Is(err, ErrCooldown) {
		return err
	}
	wait, waitErr := b.backend.cooldownWait(ctx, id)
	if waitErr != nil {
		return waitErr
	}
	if wait <= 0 {
		return err
	}
	b.sleep(ctx, wait)
	return b.backend.syncOne(ctx, id)
}
