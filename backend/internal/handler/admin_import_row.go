package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/haydary1986/turath-research/backend/internal/service/importer"
	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
)

// Problem code for a row that was saved although its ORCID cell was not applied.
const importCodeORCIDKept = "orcid_kept"

// rowResult is what one committed row changed.
type rowResult struct {
	inserted, userLinked, orcidKept bool
}

func (r *importReport) addResult(line int, result rowResult) {
	if result.inserted {
		r.Created++
	} else {
		r.Updated++
	}
	if result.userLinked {
		r.UsersLinked++
	}
	if result.orcidKept {
		r.Problems = append(r.Problems, importer.Problem{Line: line, Code: importCodeORCIDKept})
	}
}

// importFailureCode names a row's failure for the report. The log gets only the line
// and the SQLSTATE: driver messages can quote the row's values, which are personal data.
func importFailureCode(line int, err error) string {
	var conflict profileConflict
	if errors.As(err, &conflict) {
		return string(conflict)
	}
	if code := takenCodeFor(err); code != "" {
		return code
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		slog.Error("researcher import: row failed", "line", line, "sqlstate", pgErr.Code, "constraint", pgErr.ConstraintName)
	} else {
		slog.Error("researcher import: row failed", "line", line, "context_error", errors.Is(err, context.DeadlineExceeded))
	}
	return "save_failed"
}

// applyRow saves one sheet line in its own transaction, so a bad row fails alone.
func (h *AdminImportHandler) applyRow(ctx context.Context, orgs *orgCache, row importer.Row) (rowResult, error) {
	var result rowResult
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit

	collegeID, err := ensureCollege(ctx, tx, orgs, row.College)
	if err != nil {
		return result, err
	}
	departmentID, err := ensureDepartment(ctx, tx, orgs, row.Department, collegeID)
	if err != nil {
		return result, err
	}

	researcherID, orcid, inserted, kept, err := resolveImportTarget(ctx, tx, row)
	if err != nil {
		return result, err
	}
	result.inserted, result.orcidKept = inserted, kept

	if inserted {
		err = insertImportedResearcher(ctx, tx, researcherID, row, orcid, departmentID)
	} else {
		err = mergeImportedResearcher(ctx, tx, researcherID, row, orcid, departmentID)
	}
	if err != nil {
		return result, err
	}

	if row.ORCID != "" {
		if _, err := tx.Exec(ctx, `UPDATE researchers SET orcid_attested = TRUE WHERE id = $1 AND orcid_id = $2`, researcherID, row.ORCID); err != nil {
			return result, err
		}
	}

	// Link an existing portal account with the same university email.
	tag, err := tx.Exec(ctx, `
		UPDATE users SET researcher_id = $1,
			college_id = COALESCE(users.college_id, $2), department_id = COALESCE(users.department_id, $3)
		WHERE lower(email) = $4 AND researcher_id IS NULL
	`, researcherID, collegeID, departmentID, row.Email)
	if err != nil {
		return result, err
	}
	result.userLinked = tag.RowsAffected() > 0
	return result, tx.Commit(ctx)
}

// resolveImportTarget finds the row's researcher and refuses identifiers another one
// holds. A filled ORCID stays, because replacing one must go through the checks that
// reset imported data; the returned orcid is what may be written.
func resolveImportTarget(ctx context.Context, tx pgx.Tx, row importer.Row) (id, orcid string, inserted, orcidKept bool, err error) {
	var stored *string
	err = tx.QueryRow(ctx, `
		SELECT id, orcid_id FROM researchers WHERE lower(email) = $1 ORDER BY created_at LIMIT 1 FOR UPDATE
	`, row.Email).Scan(&id, &stored)
	switch {
	case isNoRows(err):
		id, inserted, orcid = uuid.NewString(), true, row.ORCID
	case err != nil:
		return "", "", false, false, err
	case stored == nil:
		orcid = row.ORCID
	default:
		orcidKept = row.ORCID != "" && canonicalStoredORCID(stored) != row.ORCID
	}

	code, err := claimedBy(ctx, tx, id, researchlinks.Links{
		ORCID: orcid, GoogleScholarID: row.Links.GoogleScholarID, ScopusAuthorID: row.Links.ScopusAuthorID,
	})
	if err != nil {
		return "", "", false, false, err
	}
	if code != "" {
		return "", "", false, false, profileConflict(code)
	}
	return id, orcid, inserted, orcidKept, nil
}

func insertImportedResearcher(ctx context.Context, tx pgx.Tx, id string, row importer.Row, orcid string, departmentID *string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO researchers (id, name_ar, name_en, email, title_ar, department_id, orcid_id, status,
			google_scholar_url, google_scholar_id, scopus_url, scopus_author_id, researchgate_url, linkedin_url,
			web_of_science_url, phone, phone_secondary, recovery_email, employee_no, account_status)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, NULLIF($7, ''), 'active',
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''),
			NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, ''), NULLIF($18, ''), NULLIF($19, ''))
	`, id, row.NameAr, row.NameEn, row.Email, row.TitleAr, departmentID, orcid,
		row.Links.GoogleScholarURL, row.Links.GoogleScholarID, row.Links.ScopusURL, row.Links.ScopusAuthorID,
		row.Links.ResearchGateURL, row.Links.LinkedInURL, row.Links.WebOfScienceURL,
		row.Phone, row.PhoneSecondary, row.RecoveryEmail, row.EmployeeNo, row.AccountStatus)
	return err
}

// mergeImportedResearcher overwrites only what the row carries: an empty cell keeps the stored value.
func mergeImportedResearcher(ctx context.Context, tx pgx.Tx, id string, row importer.Row, orcid string, departmentID *string) error {
	_, err := tx.Exec(ctx, `
		UPDATE researchers SET
			name_ar            = $2,
			name_en            = COALESCE(NULLIF($3, ''), name_en),
			title_ar           = COALESCE(NULLIF($4, ''), title_ar),
			department_id      = COALESCE($5, department_id),
			orcid_id           = COALESCE(orcid_id, NULLIF($6, '')),
			google_scholar_url = COALESCE(NULLIF($7, ''), google_scholar_url),
			google_scholar_id  = COALESCE(NULLIF($8, ''), google_scholar_id),
			scopus_url         = COALESCE(NULLIF($9, ''), scopus_url),
			scopus_author_id   = COALESCE(NULLIF($10, ''), scopus_author_id),
			researchgate_url   = COALESCE(NULLIF($11, ''), researchgate_url),
			linkedin_url       = COALESCE(NULLIF($12, ''), linkedin_url),
			web_of_science_url = COALESCE(NULLIF($13, ''), web_of_science_url),
			phone              = COALESCE(NULLIF($14, ''), phone),
			phone_secondary    = COALESCE(NULLIF($15, ''), phone_secondary),
			recovery_email     = COALESCE(NULLIF($16, ''), recovery_email),
			employee_no        = COALESCE(NULLIF($17, ''), employee_no),
			account_status     = COALESCE(NULLIF($18, ''), account_status),
			updated_at         = NOW()
		WHERE id = $1
	`, id, row.NameAr, row.NameEn, row.TitleAr, departmentID, orcid,
		row.Links.GoogleScholarURL, row.Links.GoogleScholarID, row.Links.ScopusURL, row.Links.ScopusAuthorID,
		row.Links.ResearchGateURL, row.Links.LinkedInURL, row.Links.WebOfScienceURL,
		row.Phone, row.PhoneSecondary, row.RecoveryEmail, row.EmployeeNo, row.AccountStatus)
	return err
}
