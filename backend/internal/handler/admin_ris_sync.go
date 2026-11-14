package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/rissync"
)

// AdminRISSyncHandler shows and drives the two-way profile sync with the RIS.
type AdminRISSyncHandler struct {
	db       *pgxpool.Pool
	engine   *rissync.Engine
	schedule bool
	interval time.Duration
}

// NewAdminRISSyncHandler builds the handler; a nil engine reports "not configured".
func NewAdminRISSyncHandler(db *pgxpool.Pool, engine *rissync.Engine, schedule bool, interval time.Duration) *AdminRISSyncHandler {
	return &AdminRISSyncHandler{db: db, engine: engine, schedule: schedule, interval: interval}
}

type risSyncRun struct {
	ID         string          `json:"id"`
	Mode       string          `json:"mode"`
	Trigger    string          `json:"trigger"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at"`
	Stats      json.RawMessage `json:"stats"`
	Failed     bool            `json:"failed"`
}

type risSyncStatus struct {
	Configured      bool         `json:"configured"`
	Scheduled       bool         `json:"scheduled"`
	IntervalMinutes int          `json:"interval_minutes"`
	OpenReviews     int          `json:"open_reviews"`
	LinkedProfiles  int          `json:"linked_profiles"`
	Runs            []risSyncRun `json:"runs"`
}

// Status returns the configuration, the open review count and the latest runs.
func (h *AdminRISSyncHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	out := risSyncStatus{Configured: h.engine != nil, Scheduled: h.engine != nil && h.schedule,
		IntervalMinutes: int(h.interval.Minutes()), Runs: []risSyncRun{}}
	if err := h.db.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM ris_sync_reviews WHERE resolved_at IS NULL),
		(SELECT COUNT(*) FROM ris_sync_links)`).Scan(&out.OpenReviews, &out.LinkedProfiles); err != nil {
		serverError(w, "Failed to load sync status", err)
		return
	}
	rows, err := h.db.Query(ctx, `SELECT id, mode, trigger, started_at, finished_at, stats, error IS NOT NULL
		FROM ris_sync_runs ORDER BY started_at DESC LIMIT 10`)
	if err != nil {
		serverError(w, "Failed to load sync runs", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var run risSyncRun
		if err := rows.Scan(&run.ID, &run.Mode, &run.Trigger, &run.StartedAt, &run.FinishedAt, &run.Stats, &run.Failed); err != nil {
			serverError(w, "Failed to read sync runs", err)
			return
		}
		out.Runs = append(out.Runs, run)
	}
	model.RespondJSON(w, http.StatusOK, out)
}

// Start begins a dry run or an applied run in the background.
func (h *AdminRISSyncHandler) Start(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in struct {
		Mode string `json:"mode"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	if in.Mode != rissync.ModeDryRun && in.Mode != rissync.ModeApply {
		validationError(w, []string{"mode:invalid"})
		return
	}
	if h.engine == nil {
		model.RespondError(w, http.StatusConflict, "ris_not_configured")
		return
	}
	runID, err := h.engine.StartAsync(in.Mode, user.UserID)
	if errors.Is(err, rissync.ErrBusy) {
		model.RespondError(w, http.StatusConflict, "ris_sync_busy")
		return
	}
	if err != nil {
		serverError(w, "Failed to start the sync", err)
		return
	}
	model.RespondJSON(w, http.StatusAccepted, map[string]string{"run_id": runID})
}

type risSyncChange struct {
	ResearcherID   *string `json:"researcher_id"`
	ResearcherName *string `json:"researcher_name"`
	Field          string  `json:"field"`
	Action         string  `json:"action"`
	OldValue       *string `json:"old_value"`
	NewValue       *string `json:"new_value"`
	Applied        bool    `json:"applied"`
	Note           *string `json:"note"`
}

// Changes lists what one run changed (or, for a dry run, would change).
func (h *AdminRISSyncHandler) Changes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		SELECT c.researcher_id, r.name_ar, c.field, c.action, c.old_value, c.new_value, c.applied, c.note
		FROM ris_sync_changes c LEFT JOIN researchers r ON r.id = c.researcher_id
		WHERE c.run_id = $1 ORDER BY r.name_ar, c.field LIMIT 2000`, chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, "Failed to load changes", err)
		return
	}
	defer rows.Close()
	out := []risSyncChange{}
	for rows.Next() {
		var c risSyncChange
		if err := rows.Scan(&c.ResearcherID, &c.ResearcherName, &c.Field, &c.Action, &c.OldValue, &c.NewValue, &c.Applied, &c.Note); err != nil {
			serverError(w, "Failed to read changes", err)
			return
		}
		out = append(out, c)
	}
	model.RespondJSON(w, http.StatusOK, out)
}

type risSyncReview struct {
	ID             string    `json:"id"`
	ResearcherID   string    `json:"researcher_id"`
	ResearcherName string    `json:"researcher_name"`
	Field          string    `json:"field"`
	PortalValue    string    `json:"portal_value"`
	RISValue       string    `json:"ris_value"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
}

// Reviews lists the differences waiting for an admin's decision.
func (h *AdminRISSyncHandler) Reviews(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		SELECT v.id, v.researcher_id, r.name_ar, v.field, v.portal_value, v.ris_value, v.reason, v.created_at
		FROM ris_sync_reviews v JOIN researchers r ON r.id = v.researcher_id
		WHERE v.resolved_at IS NULL ORDER BY r.name_ar, v.field`)
	if err != nil {
		serverError(w, "Failed to load reviews", err)
		return
	}
	defer rows.Close()
	out := []risSyncReview{}
	for rows.Next() {
		var v risSyncReview
		if err := rows.Scan(&v.ID, &v.ResearcherID, &v.ResearcherName, &v.Field, &v.PortalValue, &v.RISValue, &v.Reason, &v.CreatedAt); err != nil {
			serverError(w, "Failed to read reviews", err)
			return
		}
		out = append(out, v)
	}
	model.RespondJSON(w, http.StatusOK, out)
}

// Resolve applies the admin's choice for one review.
func (h *AdminRISSyncHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var in struct {
		Use string `json:"use"`
	}
	if !decodeBody(w, r, &in) {
		return
	}
	if in.Use != rissync.UsePortal && in.Use != rissync.UseRIS {
		validationError(w, []string{"use:invalid"})
		return
	}
	if h.engine == nil {
		model.RespondError(w, http.StatusConflict, "ris_not_configured")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	err := h.engine.Resolve(ctx, chi.URLParam(r, "id"), in.Use, user.UserID)
	switch {
	case errors.Is(err, rissync.ErrReviewNotFound):
		model.RespondError(w, http.StatusNotFound, "review_not_found")
	case err != nil:
		serverError(w, "Failed to resolve the review", err)
	default:
		model.RespondJSON(w, http.StatusOK, map[string]string{"resolved": in.Use})
	}
}
