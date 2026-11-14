package rissync

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// minInterval keeps a mistyped setting from hammering the RIS.
const minInterval = 2 * time.Minute

// RunSchedule applies the sync every interval until ctx ends. A run that finds another
// still in progress is skipped, not queued.
func (e *Engine) RunSchedule(ctx context.Context, interval time.Duration) {
	if e == nil {
		return
	}
	interval = max(interval, minInterval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runID, stats, err := e.Run(ctx, ModeApply, TriggerSchedule, "")
			switch {
			case errors.Is(err, ErrBusy):
				slog.Info("ris sync: previous run still in progress, skipping")
			case err != nil:
				slog.Error("ris sync: scheduled run failed", "run_id", runID, "error", err)
			case stats.ToRIS+stats.ToPortal+stats.Reviews > 0:
				slog.Info("ris sync: changes applied", "run_id", runID, "to_ris", stats.ToRIS,
					"to_portal", stats.ToPortal, "reviews", stats.Reviews, "failed", stats.Failed)
			}
		}
	}
}

// StartAsync begins one run in the background and returns its id once the run row exists,
// so a request handler can answer before the run finishes.
func (e *Engine) StartAsync(mode, actorID string) (string, error) {
	started := make(chan string, 1)
	failed := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		_, _, err := e.runNotify(ctx, mode, TriggerManual, actorID, started)
		if err != nil {
			slog.Error("ris sync: manual run failed", "error", err)
			failed <- err
		}
	}()
	select {
	case id := <-started:
		return id, nil
	case err := <-failed:
		return "", err
	case <-time.After(30 * time.Second):
		return "", errors.New("ris sync: run did not start")
	}
}
