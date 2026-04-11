package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := map[string]string{
		"status":   "ok",
		"postgres": "ok",
		"redis":    "ok",
	}

	if err := h.db.Ping(ctx); err != nil {
		slog.Error("health check: postgres ping failed", "error", err)
		status["postgres"] = "unhealthy"
		status["status"] = "degraded"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		slog.Error("health check: redis ping failed", "error", err)
		status["redis"] = "unhealthy"
		status["status"] = "degraded"
	}

	if status["status"] == "ok" {
		model.RespondJSON(w, http.StatusOK, status)
	} else {
		model.RespondJSON(w, http.StatusServiceUnavailable, status)
	}
}
