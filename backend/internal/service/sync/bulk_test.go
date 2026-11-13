package sync

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeBulk stores one run in memory; time passes only when the runner sleeps.
type fakeBulk struct {
	run      BulkRun
	ids      []string
	results  map[string][]error
	synced   []string
	clock    time.Time
	slept    []time.Duration
	waits    map[string]time.Duration
	onSynced func(id string)
}

func (f *fakeBulk) loadRun(context.Context, string) (BulkRun, error) { return f.run, nil }
func (f *fakeBulk) saveRun(_ context.Context, r BulkRun) error {
	if f.run.Status == BulkStatusStopped {
		r.Status = BulkStatusStopped
	}
	f.run = r
	return nil
}
func (f *fakeBulk) nextResearcher(_ context.Context, after *string, _ bool) (string, bool, error) {
	for _, id := range f.ids {
		if after == nil || id > *after {
			return id, true, nil
		}
	}
	return "", false, nil
}
func (f *fakeBulk) syncOne(_ context.Context, id string) error {
	f.synced = append(f.synced, id)
	if f.onSynced != nil {
		f.onSynced(id)
	}
	queue := f.results[id]
	if len(queue) == 0 {
		return nil
	}
	f.results[id] = queue[1:]
	return queue[0]
}
func (f *fakeBulk) cooldownWait(_ context.Context, id string) (time.Duration, error) {
	return f.waits[id], nil
}

func newFakeRunner(ids []string, cfg BulkConfig) (*BulkRunner, *fakeBulk) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := &fakeBulk{
		run:     BulkRun{ID: "run", Status: BulkStatusRunning, WindowStartedAt: start},
		ids:     ids,
		results: map[string][]error{},
		waits:   map[string]time.Duration{},
		clock:   start,
	}
	runner := &BulkRunner{
		backend: fake, cfg: cfg,
		now: func() time.Time { return fake.clock },
		sleep: func(_ context.Context, d time.Duration) {
			fake.slept = append(fake.slept, d)
			fake.clock = fake.clock.Add(d)
		},
	}
	return runner, fake
}

func assertSynced(t *testing.T, fake *fakeBulk, want ...string) {
	t.Helper()
	if len(fake.synced) != len(want) {
		t.Fatalf("synced %v, want %v", fake.synced, want)
	}
	for i := range want {
		if fake.synced[i] != want[i] {
			t.Fatalf("synced %v, want %v", fake.synced, want)
		}
	}
}

func TestBulkRunsSequentiallyWithDelayAndCounts(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a", "b", "c"}, BulkConfig{Delay: 5 * time.Second})
	fake.results["b"] = []error{&RunError{Code: CodeORCIDNameMismatch}}
	runner.loop(context.Background(), "run")

	assertSynced(t, fake, "a", "b", "c")
	run := fake.run
	if run.Status != BulkStatusCompleted || run.FinishedAt == nil {
		t.Errorf("status = %s", run.Status)
	}
	if run.Processed != 3 || run.Succeeded != 2 || run.Failed != 1 || run.Failures[CodeORCIDNameMismatch] != 1 {
		t.Errorf("counts = %+v", run)
	}
	for _, d := range fake.slept {
		if d != 5*time.Second {
			t.Errorf("slept %v", fake.slept)
		}
	}
	if len(fake.slept) != 3 {
		t.Errorf("slept %d times", len(fake.slept))
	}
}

func TestBulkResumesAfterTheCursor(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a", "b", "c"}, BulkConfig{})
	cursor := "a"
	fake.run.Cursor, fake.run.Processed = &cursor, 1
	runner.loop(context.Background(), "run")
	assertSynced(t, fake, "b", "c")
	if fake.run.Processed != 3 {
		t.Errorf("processed = %d", fake.run.Processed)
	}
}

func TestBulkRateLimitWaitsAndRetriesTheSameResearcher(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a", "b"}, BulkConfig{})
	fake.results["a"] = []error{&RunError{Code: codeOpenAlexUnavailable, RateLimited: true}}
	var duringWait BulkRun
	fake.onSynced = func(id string) {
		if id == "a" && fake.run.WaitReason != nil {
			duringWait = fake.run
		}
	}
	runner.loop(context.Background(), "run")

	assertSynced(t, fake, "a", "a", "b")
	if duringWait.WaitReason == nil || *duringWait.WaitReason != bulkWaitRateLimited {
		t.Errorf("the run never recorded its wait: %+v", duringWait)
	}
	if fake.run.Failed != 0 || fake.run.Succeeded != 2 || fake.run.Processed != 2 {
		t.Errorf("a rate limit must not count as a result: %+v", fake.run)
	}
	var total time.Duration
	for _, d := range fake.slept {
		total += d
	}
	if total < bulkBackoff {
		t.Errorf("waited only %v", total)
	}
}

func TestBulkDailyLimitHoldsTheRunUntilTheWindowEnds(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a", "b", "c"}, BulkConfig{DailyLimit: 2})
	start := fake.clock
	var cAt time.Time
	fake.onSynced = func(id string) {
		if id == "c" {
			cAt = fake.clock
		}
	}
	runner.loop(context.Background(), "run")
	assertSynced(t, fake, "a", "b", "c")
	if cAt.Sub(start) < bulkDailyWindow {
		t.Errorf("third researcher ran after only %v", cAt.Sub(start))
	}
}

func TestBulkRepeatedOutagesPauseTheRun(t *testing.T) {
	down := &RunError{Code: codeOpenAlexUnavailable}
	runner, fake := newFakeRunner([]string{"a", "b", "c", "d"}, BulkConfig{})
	for _, id := range []string{"a", "b", "c"} {
		fake.results[id] = []error{down}
	}
	var dAt time.Time
	fake.onSynced = func(id string) {
		if id == "d" {
			dAt = fake.clock
		}
	}
	start := fake.clock
	runner.loop(context.Background(), "run")
	assertSynced(t, fake, "a", "b", "c", "d")
	if dAt.Sub(start) < bulkBackoff {
		t.Errorf("no pause after three outages: %v", dAt.Sub(start))
	}
	if fake.run.Failed != 3 || fake.run.Succeeded != 1 {
		t.Errorf("counts = %+v", fake.run)
	}
}

func TestBulkCooldownSkipsAFreshResearcherAndWaitsForACutOffOne(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a-fresh", "b-cut"}, BulkConfig{})
	fake.results["a-fresh"] = []error{ErrCooldown}
	fake.results["b-cut"] = []error{ErrCooldown}
	fake.waits["b-cut"] = time.Minute
	runner.loop(context.Background(), "run")
	assertSynced(t, fake, "a-fresh", "b-cut", "b-cut")
	if fake.run.Skipped != 1 || fake.run.Succeeded != 1 {
		t.Errorf("counts = %+v", fake.run)
	}
}

func TestBulkStopWinsOverTheResearcherInFlight(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a", "b"}, BulkConfig{})
	fake.onSynced = func(string) { fake.run.Status = BulkStatusStopped }
	runner.loop(context.Background(), "run")
	assertSynced(t, fake, "a")
	if fake.run.Status != BulkStatusStopped || fake.run.Processed != 1 {
		t.Errorf("run = %+v", fake.run)
	}
}

func TestBulkShutdownLeavesTheCursorForTheRestart(t *testing.T) {
	runner, fake := newFakeRunner([]string{"a"}, BulkConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	fake.onSynced = func(string) { cancel() }
	fake.results["a"] = []error{errors.New("context canceled")}
	runner.loop(ctx, "run")
	if fake.run.Cursor != nil || fake.run.Processed != 0 {
		t.Errorf("an interrupted researcher must run again: %+v", fake.run)
	}
}

func TestBulkRunIsNotMutatedBySettling(t *testing.T) {
	original := BulkRun{Failures: map[string]int{"x": 1}}
	updated, _ := original.settled("a", &RunError{Code: "x"}, 0, time.Now())
	if original.Failures["x"] != 1 || updated.Failures["x"] != 2 || original.Processed != 0 {
		t.Errorf("original %+v, updated %+v", original, updated)
	}
}
