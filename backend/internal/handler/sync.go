package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

type SyncHandler struct {
	db           *pgxpool.Pool
	orchestrator *syncsvc.Orchestrator
}

func NewSyncHandler(db *pgxpool.Pool, orchestrator *syncsvc.Orchestrator) *SyncHandler {
	return &SyncHandler{db: db, orchestrator: orchestrator}
}

// respondSyncStart turns a refused start into the code the client localizes.
func respondSyncStart(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, syncsvc.ErrCooldown), errors.Is(err, syncsvc.ErrBusy):
		model.RespondError(w, http.StatusTooManyRequests, err.Error())
	default:
		serverError(w, "sync start", err)
	}
	return false
}

// SyncResearcher starts a sync for one researcher (staff only).
func (h *SyncHandler) SyncResearcher(w http.ResponseWriter, r *http.Request) {
	researcherID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(researcherID); err != nil {
		model.RespondError(w, http.StatusNotFound, "Researcher not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !respondSyncStart(w, h.orchestrator.Start(ctx, researcherID)) {
		return
	}
	model.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message":       "Sync started",
		"researcher_id": researcherID,
	})
}

// SyncByDOI enriches a publication by DOI via CrossRef (staff only).
func (h *SyncHandler) SyncByDOI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DOI string `json:"doi"`
	}
	if !decodeBody(w, r, &req) {
		return
	}
	doi := syncsvc.NormalizeDOI(req.DOI)
	if doi == "" {
		model.RespondError(w, http.StatusBadRequest, "doi is required")
		return
	}
	if !respondSyncStart(w, h.orchestrator.StartDOI(doi)) {
		return
	}
	model.RespondJSON(w, http.StatusAccepted, map[string]string{
		"message": "DOI enrichment started",
		"doi":     doi,
	})
}

// ListSyncJobs returns recent sync jobs.
func (h *SyncHandler) ListSyncJobs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, job_type, target_type, target_id, status, attempts,
		       error_message, started_at, completed_at, created_at
		FROM sync_jobs
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch sync jobs")
		return
	}
	defer rows.Close()

	type syncJobRow struct {
		ID           string  `json:"id"`
		JobType      string  `json:"job_type"`
		TargetType   string  `json:"target_type"`
		TargetID     string  `json:"target_id"`
		Status       string  `json:"status"`
		Attempts     int     `json:"attempts"`
		ErrorMessage *string `json:"error_message"`
		StartedAt    *string `json:"started_at"`
		CompletedAt  *string `json:"completed_at"`
		CreatedAt    string  `json:"created_at"`
	}

	jobs := []syncJobRow{}
	for rows.Next() {
		var j syncJobRow
		var startedAt, completedAt, createdAt *time.Time
		if err := rows.Scan(&j.ID, &j.JobType, &j.TargetType, &j.TargetID,
			&j.Status, &j.Attempts, &j.ErrorMessage, &startedAt, &completedAt, &createdAt); err != nil {
			continue
		}
		if startedAt != nil {
			s := startedAt.Format(time.RFC3339)
			j.StartedAt = &s
		}
		if completedAt != nil {
			s := completedAt.Format(time.RFC3339)
			j.CompletedAt = &s
		}
		if createdAt != nil {
			j.CreatedAt = createdAt.Format(time.RFC3339)
		}
		jobs = append(jobs, j)
	}

	model.RespondJSON(w, http.StatusOK, jobs)
}
