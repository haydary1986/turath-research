package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/googleauth"
	"github.com/haydary1986/turath-research/backend/internal/service/loginaudit"
)

// unusablePasswordHash can never match a bcrypt comparison, so Google-only accounts cannot use password login.
const unusablePasswordHash = "!google"

var errGoogleAccountMismatch = errors.New("email is linked to a different google account")

type authConfigResponse struct {
	GoogleClientID   string `json:"google_client_id"`
	AllowedDomain    string `json:"allowed_domain"`
	PasswordRegister bool   `json:"password_register"`
}

// Config tells the login page how sign-in is set up (public, contains no secrets).
func (h *AuthHandler) Config(w http.ResponseWriter, _ *http.Request) {
	model.RespondJSON(w, http.StatusOK, authConfigResponse{
		GoogleClientID:   h.cfg.Google.ClientID,
		AllowedDomain:    h.cfg.Google.AllowedDomain,
		PasswordRegister: h.cfg.Auth.AllowPasswordRegister,
	})
}

// GoogleLogin signs a person in with a Google ID token, creating a researcher account on first sign-in.
// Only accounts of the configured university domain are accepted.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	attempt := loginaudit.Event{Method: loginaudit.MethodGoogle}
	fail := func(status int, message, reason string) {
		attempt.Reason = reason
		recordLogin(r, h.db, attempt)
		model.RespondError(w, status, message)
	}
	if h.verifier == nil {
		fail(http.StatusServiceUnavailable, "google_not_configured", loginaudit.ReasonNotConfigured)
		return
	}
	var req struct {
		Credential string `json:"credential"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(http.StatusBadRequest, "Invalid request body", loginaudit.ReasonInvalidRequest)
		return
	}
	if req.Credential == "" {
		fail(http.StatusBadRequest, "credential is required", loginaudit.ReasonInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// The credential itself is never logged or stored; only the verified email is.
	identity, err := h.verifier.Verify(ctx, req.Credential)
	attempt.Email = identity.Email
	switch {
	case errors.Is(err, googleauth.ErrDomainNotAllowed):
		slog.Warn("google sign-in rejected: domain not allowed")
		fail(http.StatusForbidden, "domain_not_allowed", loginaudit.ReasonDomainNotAllowed)
		return
	case err != nil:
		slog.Warn("google sign-in rejected: invalid token", "error", err)
		fail(http.StatusUnauthorized, "invalid_google_token", loginaudit.ReasonInvalidToken)
		return
	}

	user, err := h.findOrCreateGoogleUser(ctx, identity)
	if errors.Is(err, errGoogleAccountMismatch) {
		fail(http.StatusForbidden, "account_conflict", loginaudit.ReasonAccountConflict)
		return
	}
	if err != nil {
		slog.Error("google sign-in: user lookup failed", "error", err)
		fail(http.StatusInternalServerError, "Sign-in failed", loginaudit.ReasonServerError)
		return
	}
	attempt.UserID = &user.ID

	token, err := h.generateToken(user)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		fail(http.StatusInternalServerError, "Failed to generate token", loginaudit.ReasonServerError)
		return
	}
	attempt.Success = true
	recordLogin(r, h.db, attempt)
	model.RespondJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

const googleUserSelect = `
	SELECT id, email, name_ar, name_en, role::text, institution_id, college_id, department_id, google_sub
	FROM users
	WHERE google_sub = $1 OR lower(email) = $2
	ORDER BY (google_sub = $1) DESC NULLS LAST
	LIMIT 1`

// findOrCreateGoogleUser links the Google account to an existing user with the same email,
// or creates a new researcher. The stable Google "sub" is the identity key once linked.
func (h *AuthHandler) findOrCreateGoogleUser(ctx context.Context, id googleauth.Identity) (userInfo, error) {
	user, linkedSub, found, err := h.lookupGoogleUser(ctx, id)
	if err != nil {
		return userInfo{}, err
	}

	created := !found
	if !found {
		name := id.Name
		if name == "" {
			name = strings.SplitN(id.Email, "@", 2)[0]
		}
		_, err := h.db.Exec(ctx, `
			INSERT INTO users (email, password_hash, name_ar, role, google_sub)
			VALUES ($1, $2, $3, 'researcher', $4)
		`, id.Email, unusablePasswordHash, name, id.Subject)
		if err != nil && !isUniqueViolation(err) {
			return userInfo{}, err
		}
		// A concurrent first sign-in may have inserted the row; read it back either way.
		user, linkedSub, found, err = h.lookupGoogleUser(ctx, id)
		if err != nil {
			return userInfo{}, err
		}
		if !found {
			return userInfo{}, errors.New("user vanished after insert")
		}
	}

	if linkedSub != nil && *linkedSub != id.Subject {
		return userInfo{}, errGoogleAccountMismatch
	}
	if linkedSub == nil {
		if _, err := h.db.Exec(ctx, `UPDATE users SET google_sub = $2 WHERE id = $1 AND google_sub IS NULL`, user.ID, id.Subject); err != nil {
			return userInfo{}, err
		}
	}
	if _, err := h.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID); err != nil {
		slog.Error("google sign-in: failed to record last login", "user_id", user.ID, "error", err)
	}
	// The verified university email ties the account to its imported record (college,
	// department, official name), so the dean sees the academic's requests from day one.
	if h.linkImportedResearcher(ctx, user.ID, created) {
		if linked, _, ok, err := h.lookupGoogleUser(ctx, id); err == nil && ok {
			user = linked
		}
	}
	return user, nil
}

func (h *AuthHandler) lookupGoogleUser(ctx context.Context, id googleauth.Identity) (userInfo, *string, bool, error) {
	var u userInfo
	var sub *string
	err := h.db.QueryRow(ctx, googleUserSelect, id.Subject, id.Email).Scan(
		&u.ID, &u.Email, &u.NameAr, &u.NameEn, &u.Role, &u.InstitutionID, &u.CollegeID, &u.DepartmentID, &sub,
	)
	if isNoRows(err) {
		return userInfo{}, nil, false, nil
	}
	if err != nil {
		return userInfo{}, nil, false, err
	}
	return u, sub, true, nil
}
