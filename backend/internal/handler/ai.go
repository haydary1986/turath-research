package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/ai"
)

type AIHandler struct {
	service *ai.AnalysisService
}

func NewAIHandler(db *pgxpool.Pool, apiKey, baseURL string) *AIHandler {
	client := ai.NewDeepSeekClient(apiKey, baseURL)
	return &AIHandler{
		service: ai.NewAnalysisService(client, db),
	}
}

// CollaborationRecommendations returns AI-powered collaboration suggestions.
func (h *AIHandler) CollaborationRecommendations(w http.ResponseWriter, r *http.Request) {
	researcherID := chi.URLParam(r, "id")
	user := middleware.GetUser(r.Context())
	requestedBy := ""
	if user != nil {
		requestedBy = user.UserID
	}

	// Check cache first
	if cached, ok := h.service.GetCachedAnalysis(r.Context(), "collaboration_rec", "researcher", researcherID); ok {
		model.RespondJSON(w, http.StatusOK, map[string]any{
			"cached": true,
			"data":   cached,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	results, err := h.service.CollaborationRecommendations(ctx, researcherID, requestedBy)
	if err != nil {
		slog.Error("AI collaboration analysis failed", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]any{
		"cached": false,
		"data":   results,
	})
}

// TrendingTopics returns AI analysis of trending research topics.
func (h *AIHandler) TrendingTopics(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	requestedBy := ""
	if user != nil {
		requestedBy = user.UserID
	}

	if cached, ok := h.service.GetCachedAnalysis(r.Context(), "trend_analysis", "institution", "all"); ok {
		model.RespondJSON(w, http.StatusOK, map[string]any{
			"cached": true,
			"data":   cached,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	results, err := h.service.TrendingTopics(ctx, requestedBy)
	if err != nil {
		slog.Error("AI trending topics failed", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]any{
		"cached": false,
		"data":   results,
	})
}

// PublicationStrategy returns AI-powered publication strategy for a researcher.
func (h *AIHandler) PublicationStrategy(w http.ResponseWriter, r *http.Request) {
	researcherID := chi.URLParam(r, "id")
	user := middleware.GetUser(r.Context())
	requestedBy := ""
	if user != nil {
		requestedBy = user.UserID
	}

	if cached, ok := h.service.GetCachedAnalysis(r.Context(), "publication_strategy", "researcher", researcherID); ok {
		model.RespondJSON(w, http.StatusOK, map[string]any{
			"cached": true,
			"data":   cached,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := h.service.PublicationStrategy(ctx, researcherID, requestedBy)
	if err != nil {
		slog.Error("AI publication strategy failed", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]any{
		"cached": false,
		"data":   result,
	})
}

// GapAnalysis returns AI analysis of research gaps.
func (h *AIHandler) GapAnalysis(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	requestedBy := ""
	if user != nil {
		requestedBy = user.UserID
	}

	if cached, ok := h.service.GetCachedAnalysis(r.Context(), "gap_analysis", "institution", "all"); ok {
		model.RespondJSON(w, http.StatusOK, map[string]any{
			"cached": true,
			"data":   cached,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	result, err := h.service.GapAnalysis(ctx, requestedBy)
	if err != nil {
		slog.Error("AI gap analysis failed", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]any{
		"cached": false,
		"data":   result,
	})
}
