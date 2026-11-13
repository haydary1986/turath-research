package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/middleware"
	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/googleauth"
	"github.com/haydary1986/turath-research/backend/internal/service/loginaudit"
)

type AuthHandler struct {
	db       *pgxpool.Pool
	cfg      *config.Config
	verifier *googleauth.Verifier // nil when Google sign-in is not configured
}

func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	h := &AuthHandler{db: db, cfg: cfg}
	if cfg.Google.ClientID != "" {
		h.verifier = googleauth.NewVerifier(cfg.Google.ClientID, cfg.Google.AllowedDomain, googleauth.NewKeySource().Key)
	}
	return h
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	NameAr   string `json:"name_ar"`
	NameEn   string `json:"name_en"`
}

type authResponse struct {
	Token string   `json:"token"`
	User  userInfo `json:"user"`
}

type userInfo struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	NameAr        *string `json:"name_ar"`
	NameEn        *string `json:"name_en"`
	Role          string  `json:"role"`
	InstitutionID *string `json:"institution_id"`
	CollegeID     *string `json:"college_id"`
	DepartmentID  *string `json:"department_id"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	attempt := loginaudit.Event{Method: loginaudit.MethodPassword}
	fail := func(status int, message, reason string) {
		attempt.Reason = reason
		recordLogin(r, h.db, attempt)
		model.RespondError(w, status, message)
	}

	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody)).Decode(&req); err != nil {
		fail(http.StatusBadRequest, "Invalid request body", loginaudit.ReasonInvalidRequest)
		return
	}
	attempt.Email = req.Email

	if req.Email == "" || req.Password == "" {
		fail(http.StatusBadRequest, "Email and password are required", loginaudit.ReasonInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user userInfo
	var passwordHash string
	err := h.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name_ar, name_en, role,
		       institution_id, college_id, department_id
		FROM users WHERE email = $1
	`, req.Email).Scan(
		&user.ID, &user.Email, &passwordHash, &user.NameAr, &user.NameEn,
		&user.Role, &user.InstitutionID, &user.CollegeID, &user.DepartmentID,
	)
	if err != nil {
		reason := loginaudit.ReasonUnknownUser
		if !isNoRows(err) {
			slog.Error("login: user lookup failed", "error", err)
			reason = loginaudit.ReasonServerError
		}
		fail(http.StatusUnauthorized, "Invalid email or password", reason)
		return
	}
	attempt.UserID = &user.ID

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		reason := loginaudit.ReasonWrongPassword
		if passwordHash == unusablePasswordHash {
			reason = loginaudit.ReasonGoogleOnlyAccount
		}
		fail(http.StatusUnauthorized, "Invalid email or password", reason)
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		fail(http.StatusInternalServerError, "Failed to generate token", loginaudit.ReasonServerError)
		return
	}

	if _, err := h.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID); err != nil {
		slog.Error("login: failed to record last login", "user_id", user.ID, "error", err)
	}
	attempt.Success = true
	recordLogin(r, h.db, attempt)
	model.RespondJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Auth.AllowPasswordRegister {
		model.RespondError(w, http.StatusForbidden, "registration_disabled")
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		model.RespondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	if len(req.Password) < 8 {
		model.RespondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := uuid.New().String()
	_, err = h.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name_ar, name_en, role)
		VALUES ($1, $2, $3, $4, $5, 'researcher')
	`, id, req.Email, string(hash), nilIfEmpty(req.NameAr), nilIfEmpty(req.NameEn))

	if err != nil {
		model.RespondError(w, http.StatusConflict, "Email already registered")
		return
	}

	user := userInfo{
		ID:     id,
		Email:  req.Email,
		NameAr: ptrIfNotEmpty(req.NameAr),
		NameEn: ptrIfNotEmpty(req.NameEn),
		Role:   "researcher",
	}

	token, err := h.generateToken(user)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		model.RespondError(w, http.StatusInternalServerError, "Registration succeeded but token generation failed")
		return
	}

	model.RespondJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUser(r.Context())
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user userInfo
	err := h.db.QueryRow(ctx, `
		SELECT id, email, name_ar, name_en, role,
		       institution_id, college_id, department_id
		FROM users WHERE id = $1
	`, claims.UserID).Scan(
		&user.ID, &user.Email, &user.NameAr, &user.NameEn,
		&user.Role, &user.InstitutionID, &user.CollegeID, &user.DepartmentID,
	)
	if err != nil {
		model.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	model.RespondJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) generateToken(user userInfo) (string, error) {
	now := time.Now()
	claims := middleware.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.cfg.JWT.Expiry)),
		},
		UserID:        user.ID,
		Email:         user.Email,
		Role:          user.Role,
		InstitutionID: derefOr(user.InstitutionID, ""),
		CollegeID:     derefOr(user.CollegeID, ""),
		DepartmentID:  derefOr(user.DepartmentID, ""),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWT.Secret))
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}
