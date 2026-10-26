package handler

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// This file ties the portal to the research registry so data is entered once:
// a submitted request belongs to a researcher record, and a completed journal
// verification materializes as a publication with authorship and counters.

// ensureResearcher returns the researcher record for the applicant, creating and
// linking one (users.researcher_id) on first use. Runs inside the caller's tx.
func ensureResearcher(ctx context.Context, tx pgx.Tx, rec requestRecord) (string, error) {
	// Already linked through the user account?
	var researcherID *string
	if err := tx.QueryRow(ctx,
		`SELECT researcher_id FROM users WHERE id = $1`, rec.ApplicantID,
	).Scan(&researcherID); err != nil {
		return "", err
	}
	if researcherID != nil {
		return *researcherID, nil
	}

	email := strings.ToLower(rec.ApplicantEmail)

	// A researcher with the same university email may already exist (e.g. imported).
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM researchers WHERE lower(email) = $1`, email).Scan(&id)
	if err != nil && !isNoRows(err) {
		return "", err
	}
	if isNoRows(err) {
		name := rec.ApplicantName
		if name == "" {
			name = strings.SplitN(email, "@", 2)[0]
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO researchers (name_ar, email, department_id, status)
			VALUES ($1, $2, $3, 'active')
			ON CONFLICT (email) DO UPDATE SET email = researchers.email
			RETURNING id
		`, name, email, rec.DepartmentID).Scan(&id)
		if err != nil {
			return "", err
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET researcher_id = $2 WHERE id = $1 AND researcher_id IS NULL`,
		rec.ApplicantID, id,
	); err != nil {
		return "", err
	}
	return id, nil
}

var quartileToRanking = map[string]string{"Q1": "Q1", "Q2": "Q2", "Q3": "Q3", "Q4": "Q4"}

type journalDetails struct {
	name, publisher, issn, quartile, impactFactor, articleURL, publicationDate string
}

func journalDetailsOf(raw json.RawMessage) (journalDetails, error) {
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		return journalDetails{}, err
	}
	get := func(k string) string {
		s, _ := d[k].(string)
		return strings.TrimSpace(s)
	}
	return journalDetails{
		name:            get("journal_name"),
		publisher:       get("publisher"),
		issn:            get("issn"),
		quartile:        get("quartile"),
		impactFactor:    get("impact_factor"),
		articleURL:      get("article_url"),
		publicationDate: get("publication_date"),
	}, nil
}

// ensureJournal finds the journal by ISSN, then by name, and creates it otherwise.
func ensureJournal(ctx context.Context, tx pgx.Tx, d journalDetails) (*string, error) {
	if d.name == "" {
		return nil, nil
	}
	var id string
	if d.issn != "" {
		err := tx.QueryRow(ctx, `SELECT id FROM journals WHERE issn = $1 OR eissn = $1`, d.issn).Scan(&id)
		if err == nil {
			return &id, nil
		}
		if !isNoRows(err) {
			return nil, err
		}
	}
	err := tx.QueryRow(ctx, `SELECT id FROM journals WHERE lower(name) = lower($1)`, d.name).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if !isNoRows(err) {
		return nil, err
	}

	ranking := "unranked"
	if r, ok := quartileToRanking[d.quartile]; ok {
		ranking = r
	}
	var impact *float64
	if f, err := strconv.ParseFloat(d.impactFactor, 64); err == nil && f > 0 && f < 1000 {
		impact = &f
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO journals (name, publisher, issn, sjr_ranking, impact_factor)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4::journal_ranking, $5)
		ON CONFLICT (issn) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, d.name, d.publisher, d.issn, ranking, impact).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// materializePublication creates (or finds, by DOI) the publication for a completed
// journal verification, records authorship and refreshes the researcher's counter.
// Runs inside the transition transaction; failures roll the whole transition back.
func materializePublication(ctx context.Context, tx pgx.Tx, rec requestRecord) error {
	if rec.Kind != portal.KindJournalVerification || rec.PublicationID != nil {
		return nil
	}

	researcherID, err := ensureResearcher(ctx, tx, rec)
	if err != nil {
		return err
	}
	details, err := journalDetailsOf(rec.Details)
	if err != nil {
		return err
	}
	journalID, err := ensureJournal(ctx, tx, details)
	if err != nil {
		return err
	}

	var pubID string
	found := false
	if rec.DOI != nil {
		err := tx.QueryRow(ctx, `SELECT id FROM publications WHERE doi = $1`, *rec.DOI).Scan(&pubID)
		if err != nil && !isNoRows(err) {
			return err
		}
		found = err == nil
	}
	if !found {
		language := "ar"
		if rec.TitleEn != nil {
			language = "en"
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO publications (doi, title_ar, title_en, publication_date, publication_type,
			                          journal_id, url, language, status)
			VALUES ($1, $2, $3, NULLIF($4, '')::date, 'journal_article', $5, NULLIF($6, ''), $7, 'published')
			RETURNING id
		`, rec.DOI, rec.TitleAr, rec.TitleEn, details.publicationDate, journalID,
			details.articleURL, language).Scan(&pubID); err != nil {
			return err
		}
	}

	var hasAuthor bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM publication_authors WHERE publication_id = $1 AND researcher_id = $2)
	`, pubID, researcherID).Scan(&hasAuthor); err != nil {
		return err
	}
	if !hasAuthor {
		if _, err := tx.Exec(ctx, `
			INSERT INTO publication_authors (publication_id, researcher_id, author_name, author_order, is_corresponding)
			VALUES ($1, $2, $3, 1, TRUE)
		`, pubID, researcherID, rec.ApplicantName); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE researchers SET
			total_publications = (SELECT COUNT(*) FROM publication_authors WHERE researcher_id = $1),
			updated_at = NOW()
		WHERE id = $1
	`, researcherID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE portal_requests SET publication_id = $2, researcher_id = $3 WHERE id = $1`,
		rec.ID, pubID, researcherID)
	return err
}
