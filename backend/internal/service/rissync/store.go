package rissync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

// PortalResearcher is a portal researcher with the synced fields in canonical form.
type PortalResearcher struct {
	ID        string
	Email     string
	Values    map[string]string
	UpdatedAt time.Time
}

// errTaken means the value already belongs to another portal researcher (a unique id).
var errTaken = errors.New("value belongs to another researcher")

func loadPortal(ctx context.Context, db *pgxpool.Pool) (map[string]PortalResearcher, error) {
	rows, err := db.Query(ctx, `
		SELECT id, lower(btrim(email)), COALESCE(name_ar, ''), COALESCE(name_en, ''), COALESCE(phone, ''),
		       COALESCE(orcid_id, ''), COALESCE(google_scholar_url, ''), COALESCE(scopus_url, ''),
		       COALESCE(researchgate_url, ''), COALESCE(linkedin_url, ''), COALESCE(web_of_science_url, ''),
		       COALESCE(department_id::text, ''), updated_at
		FROM researchers WHERE COALESCE(btrim(email), '') <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]PortalResearcher{}
	for rows.Next() {
		var p PortalResearcher
		var nameAr, nameEn, phone, orcid, scholar, scopus, rg, li, wos, dept string
		if err := rows.Scan(&p.ID, &p.Email, &nameAr, &nameEn, &phone, &orcid, &scholar, &scopus, &rg, &li, &wos, &dept, &p.UpdatedAt); err != nil {
			return nil, err
		}
		link := func(field, raw string) string {
			v, _ := CanonicalLink(field, raw) // an invalid stored value counts as empty
			return v
		}
		p.Values = map[string]string{
			FieldNameAr: CanonicalName(nameAr), FieldNameEn: CanonicalName(nameEn),
			FieldPhone: CanonicalPhone(phone), FieldDepartment: dept,
			FieldORCID: link(FieldORCID, orcid), FieldScholar: link(FieldScholar, scholar),
			FieldScopus: link(FieldScopus, scopus), FieldResearchGate: link(FieldResearchGate, rg),
			FieldLinkedIn: link(FieldLinkedIn, li), FieldWebOfScience: link(FieldWebOfScience, wos),
		}
		out[p.Email] = p
	}
	return out, rows.Err()
}

func loadPortalOrgs(ctx context.Context, db *pgxpool.Pool) (colleges, departments []Org, err error) {
	rows, err := db.Query(ctx, `SELECT id::text, name_ar, '' FROM colleges
		UNION ALL SELECT id::text, name_ar, COALESCE(college_id::text, '') FROM departments`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.NameAr, &o.CollegeID); err != nil {
			return nil, nil, err
		}
		if o.CollegeID == "" {
			colleges = append(colleges, o)
		} else {
			departments = append(departments, o)
		}
	}
	return colleges, departments, rows.Err()
}

// applyToPortal writes one field into the portal researcher, with the side effects the
// profile page applies: link ids are stored with their URL, a changed ORCID resets what
// was imported for the old one, and a department move also moves the linked account.
func applyToPortal(ctx context.Context, db *pgxpool.Pool, researcherID, field, value string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if err := writePortalField(ctx, tx, researcherID, field, value); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errTaken
		}
		return err
	}
	return tx.Commit(ctx)
}

func writePortalField(ctx context.Context, tx pgx.Tx, id, field, value string) error {
	exec := func(sql string, args ...any) error {
		_, err := tx.Exec(ctx, sql, append([]any{id}, args...)...)
		return err
	}
	switch field {
	case FieldNameAr:
		if value == "" {
			return nil // the Arabic name is required in the portal; never cleared from RIS
		}
		return exec(`UPDATE researchers SET name_ar = $2, updated_at = NOW() WHERE id = $1`, value)
	case FieldNameEn:
		return exec(`UPDATE researchers SET name_en = NULLIF($2, ''), updated_at = NOW() WHERE id = $1`, value)
	case FieldPhone:
		return exec(`UPDATE researchers SET phone = NULLIF($2, ''), updated_at = NOW() WHERE id = $1`, value)
	case FieldDepartment:
		if err := exec(`UPDATE researchers SET department_id = NULLIF($2, '')::uuid, updated_at = NOW() WHERE id = $1`, value); err != nil {
			return err
		}
		return exec(`UPDATE users SET department_id = NULLIF($2, '')::uuid,
			college_id = (SELECT college_id FROM departments WHERE id = NULLIF($2, '')::uuid)
			WHERE researcher_id = $1`, value)
	case FieldScholar:
		links, _ := researchlinks.Parse(researchlinks.Input{GoogleScholarURL: value})
		return exec(`UPDATE researchers SET google_scholar_url = NULLIF($2, ''), google_scholar_id = NULLIF($3, ''),
			updated_at = NOW() WHERE id = $1`, links.GoogleScholarURL, links.GoogleScholarID)
	case FieldScopus:
		links, _ := researchlinks.Parse(researchlinks.Input{ScopusURL: value})
		return exec(`UPDATE researchers SET scopus_url = NULLIF($2, ''), scopus_author_id = NULLIF($3, ''),
			updated_at = NOW() WHERE id = $1`, links.ScopusURL, links.ScopusAuthorID)
	case FieldResearchGate:
		return exec(`UPDATE researchers SET researchgate_url = NULLIF($2, ''), updated_at = NOW() WHERE id = $1`, value)
	case FieldLinkedIn:
		return exec(`UPDATE researchers SET linkedin_url = NULLIF($2, ''), updated_at = NOW() WHERE id = $1`, value)
	case FieldWebOfScience:
		return exec(`UPDATE researchers SET web_of_science_url = NULLIF($2, ''), updated_at = NOW() WHERE id = $1`, value)
	case FieldORCID:
		// What was imported for the previous ORCID belongs to that identity, not this one.
		if err := syncsvc.ResetImported(ctx, tx, id); err != nil {
			return err
		}
		return exec(`UPDATE researchers SET orcid_id = NULLIF($2, ''), orcid_attested = FALSE, updated_at = NOW() WHERE id = $1`, value)
	}
	return fmt.Errorf("unknown field %q", field)
}

// risUpdate is how one field is written into RIS: researcher columns or a profile link.
func risUpdate(field, value string, risDept Org) (columns map[string]any, platform string, skip string) {
	switch field {
	case FieldNameAr:
		if value == "" {
			return nil, "", "empty_required"
		}
		return map[string]any{"full_name_ar": value}, "", ""
	case FieldNameEn:
		if value == "" {
			return nil, "", "empty_required" // RIS requires an English name
		}
		return map[string]any{"full_name_en": value}, "", ""
	case FieldPhone:
		return map[string]any{"private_phone": nullable(value)}, "", ""
	case FieldDepartment:
		if value == "" {
			return map[string]any{"department_id": nil}, "", ""
		}
		if risDept.ID == "" {
			return nil, "", "department_unmapped"
		}
		return map[string]any{"department_id": risDept.ID, "college_id": nullable(risDept.CollegeID)}, "", ""
	}
	return nil, field, ""
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
