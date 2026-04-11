package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
)

type NotificationHandler struct {
	db *pgxpool.Pool
}

func NewNotificationHandler(db *pgxpool.Pool) *NotificationHandler {
	return &NotificationHandler{db: db}
}

type notification struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	TitleAr   string  `json:"title_ar"`
	TitleEn   *string `json:"title_en"`
	Body      string  `json:"body"`
	Type      string  `json:"type"`
	Link      *string `json:"link"`
	IsRead    bool    `json:"is_read"`
	CreatedAt string  `json:"created_at"`
}

// ListMyNotifications returns notifications for the current user.
func (h *NotificationHandler) ListMyNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, user_id, title_ar, title_en, body, type, link, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, user.UserID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch notifications")
		return
	}
	defer rows.Close()

	notifs := []notification{}
	for rows.Next() {
		var n notification
		var createdAt time.Time
		if err := rows.Scan(&n.ID, &n.UserID, &n.TitleAr, &n.TitleEn, &n.Body, &n.Type, &n.Link, &n.IsRead, &createdAt); err != nil {
			continue
		}
		n.CreatedAt = createdAt.Format(time.RFC3339)
		notifs = append(notifs, n)
	}

	model.RespondJSON(w, http.StatusOK, notifs)
}

// UnreadCount returns the count of unread notifications.
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var count int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, user.UserID).Scan(&count)

	model.RespondJSON(w, http.StatusOK, map[string]int{"unread": count})
}

// MarkRead marks a notification as read.
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	_, err := h.db.Exec(ctx, `UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, user.UserID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{"marked": id})
}

// MarkAllRead marks all notifications as read for the current user.
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	_, err := h.db.Exec(ctx, `UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`, user.UserID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to mark all as read")
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]string{"status": "all_read"})
}

// CreateNotification is a utility for other services to create notifications.
// Not exposed as API — called internally.
func CreateNotification(ctx context.Context, db *pgxpool.Pool, userID, titleAr, body, notifType string, link *string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, title_ar, body, type, link)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New().String(), userID, titleAr, body, notifType, link)
	return err
}

// ── Needed: notifications table migration ────────
// This handler requires a `notifications` table — added in migration 004.

// Ensure this compiles even without the table (queries will fail gracefully).
var _ = json.Marshal
