package handler

import (
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

const maxUserNameRunes = 255

type createUserInput struct {
	Email        string `json:"email"`
	NameAr       string `json:"name_ar"`
	Role         string `json:"role"`
	CollegeID    string `json:"college_id"`
	DepartmentID string `json:"department_id"`
}

// validateNewUser normalizes the input and returns "field:code" problems. Accounts are
// pre-created for university emails only: the owner signs in with Google, which links
// the account by email and keeps the role assigned here.
func validateNewUser(in createUserInput, allowedDomain string) (createUserInput, []string) {
	out := in
	out.Email = strings.ToLower(strings.TrimSpace(in.Email))
	out.NameAr = strings.TrimSpace(in.NameAr)
	var problems []string
	addr, err := mail.ParseAddress(out.Email)
	switch {
	case out.Email == "":
		problems = append(problems, "email:required")
	case err != nil || addr.Address != out.Email:
		problems = append(problems, "email:invalid")
	case allowedDomain != "" && !strings.HasSuffix(out.Email, "@"+allowedDomain):
		problems = append(problems, "email:domain_not_allowed")
	}
	if out.NameAr == "" {
		problems = append(problems, "name_ar:required")
	} else if utf8.RuneCountInString(out.NameAr) > maxUserNameRunes {
		problems = append(problems, "name_ar:too_long")
	}
	if _, ok := assignableRoles[in.Role]; !ok {
		problems = append(problems, "role:invalid")
	}
	return out, problems
}

// Create pre-registers a university account with its role and unit (admin only).
// It links the account to the researcher record holding the same email, if any.
func (h *AdminUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in createUserInput
	if !decodeBody(w, r, &in) {
		return
	}
	in, problems := validateNewUser(in, h.allowedDomain)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	college, ok1 := idOrNil(in.CollegeID)
	dept, ok2 := idOrNil(in.DepartmentID)
	if !ok1 || !ok2 {
		model.RespondError(w, http.StatusBadRequest, "invalid identifier")
		return
	}

	ctx, cancel := portalContext(r)
	defer cancel()

	var created adminUser
	err := h.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name_ar, role, college_id, department_id, researcher_id, institution_id)
		VALUES ($1::varchar, $2, $3, $4::user_role, $5, $6,
			(SELECT id FROM researchers WHERE lower(email) = $1::varchar LIMIT 1),
			(SELECT id FROM institutions ORDER BY created_at LIMIT 1))
		RETURNING id, email, name_ar, role::text, college_id, department_id
	`, in.Email, unusablePasswordHash, in.NameAr, in.Role, college, dept).
		Scan(&created.ID, &created.Email, &created.NameAr, &created.Role, &created.CollegeID, &created.DepartmentID)
	switch {
	case isUniqueViolation(err):
		model.RespondError(w, http.StatusConflict, "email_taken")
	case isForeignKeyViolation(err):
		model.RespondError(w, http.StatusBadRequest, "invalid_college_or_department")
	case err != nil:
		serverError(w, "Failed to create user", err)
	default:
		model.RespondJSON(w, http.StatusCreated, created)
	}
}
