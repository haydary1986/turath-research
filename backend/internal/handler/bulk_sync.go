package handler

import (
	"errors"
	"net/http"

	"github.com/haydary1986/turath-research/backend/internal/model"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

// BulkSyncHandler lets an admin sync every researcher with an ORCID, one at a time.
type BulkSyncHandler struct {
	runner *syncsvc.BulkRunner
}

func NewBulkSyncHandler(runner *syncsvc.BulkRunner) *BulkSyncHandler {
	return &BulkSyncHandler{runner: runner}
}

// Progress reports how far the latest pass got. Counts only, no researcher data.
func (h *BulkSyncHandler) Progress(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	progress, err := h.runner.Progress(ctx)
	if err != nil {
		serverError(w, "bulk sync progress", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, progress)
}

// Start continues the unfinished pass or begins a new one, and returns immediately.
func (h *BulkSyncHandler) Start(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	var body struct {
		OnlyFailed bool `json:"only_failed"`
		Fresh      bool `json:"fresh"`
	}
	if !decodeOptionalBody(w, r, &body) {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	err := h.runner.Start(ctx, syncsvc.BulkStartOptions{OnlyFailed: body.OnlyFailed, Fresh: body.Fresh, UserID: user.UserID})
	if err != nil {
		serverError(w, "bulk sync start", err)
		return
	}
	h.respondProgress(w, r, http.StatusAccepted)
}

// Stop halts the pass after the researcher in flight.
func (h *BulkSyncHandler) Stop(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	err := h.runner.Stop(ctx)
	if errors.Is(err, syncsvc.ErrBulkNotActive) {
		model.RespondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		serverError(w, "bulk sync stop", err)
		return
	}
	h.respondProgress(w, r, http.StatusOK)
}

func (h *BulkSyncHandler) respondProgress(w http.ResponseWriter, r *http.Request, status int) {
	ctx, cancel := portalContext(r)
	defer cancel()
	progress, err := h.runner.Progress(ctx)
	if err != nil {
		serverError(w, "bulk sync progress", err)
		return
	}
	model.RespondJSON(w, status, progress)
}
