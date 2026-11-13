package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/contactinfo"
)

// researcherContact is personal data: only requireActsFor's callers (the researcher,
// scientific affairs, the VP and admin) may receive it. Lists never carry these fields.
type researcherContact struct {
	Email          string  `json:"email"`
	Phone          *string `json:"phone"`
	PhoneSecondary *string `json:"phone_secondary"`
	RecoveryEmail  *string `json:"recovery_email"`
	EmployeeNo     *string `json:"employee_no"`
	AccountStatus  *string `json:"account_status"`
}

// Contact returns one researcher's contact data to the permitted callers.
func (h *ResearcherHandler) Contact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		model.RespondError(w, http.StatusNotFound, "not_found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	if !requireActsFor(ctx, w, r, h.db, id) {
		return
	}

	var c researcherContact
	err := h.db.QueryRow(ctx, `
		SELECT email, phone, phone_secondary, recovery_email, employee_no, account_status
		FROM researchers WHERE id = $1
	`, id).Scan(&c.Email, &c.Phone, &c.PhoneSecondary, &c.RecoveryEmail, &c.EmployeeNo, &c.AccountStatus)
	if isNoRows(err) {
		model.RespondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		serverError(w, "researcher contact", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, c)
}

// contactUpdate is the contact part of the owner's form. A nil field was not sent and
// stays as it is; an empty one clears the value.
type contactUpdate struct {
	Phone          *string `json:"phone"`
	PhoneSecondary *string `json:"phone_secondary"`
	RecoveryEmail  *string `json:"recovery_email"`
}

// normalized returns a canonical copy and every problem as "field:code".
func (c contactUpdate) normalized() (contactUpdate, []string) {
	var problems []string
	parse := func(field, code string, raw *string, parser func(string) (string, bool)) *string {
		if raw == nil {
			return nil
		}
		value, ok := parser(*raw)
		if !ok {
			problems = append(problems, field+":"+code)
		}
		return &value
	}
	return contactUpdate{
		Phone:          parse(contactinfo.FieldPhone, contactinfo.CodeInvalidPhone, c.Phone, contactinfo.ParsePhone),
		PhoneSecondary: parse(contactinfo.FieldSecondary, contactinfo.CodeInvalidPhone, c.PhoneSecondary, contactinfo.ParsePhone),
		RecoveryEmail:  parse(contactinfo.FieldRecovery, contactinfo.CodeInvalidEmail, c.RecoveryEmail, contactinfo.ParseEmail),
	}, problems
}
