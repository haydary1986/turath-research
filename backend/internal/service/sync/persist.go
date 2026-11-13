package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/haydary1986/turath-research/backend/internal/service/researchlinks"
)

// persist stores a run in one transaction, holding the researcher row so a concurrent
// ORCID change either waits for it or makes it abort with errSuperseded.
func (o *Orchestrator) persist(ctx context.Context, researcherID, orcid string, startedAt time.Time, fetched fetchResult) (saved, failed int, err error) {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // a no-op after commit

	var currentORCID *string
	var currentStart *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT orcid_id, last_sync_started_at FROM researchers WHERE id = $1 FOR UPDATE`, researcherID,
	).Scan(&currentORCID, &currentStart); err != nil {
		return 0, 0, err
	}
	if currentORCID == nil || currentStart == nil || !currentStart.Equal(startedAt) {
		return 0, 0, errSuperseded
	}
	if id, ok := researchlinks.ParseORCID(*currentORCID); !ok || id != orcid {
		return 0, 0, errSuperseded
	}

	kept := make([]string, 0, len(fetched.works))
	for _, w := range fetched.works {
		id, err := upsertInSavepoint(ctx, tx, researcherID, w)
		if err != nil {
			failed++
			slog.Warn("failed to upsert publication", "researcher", researcherID, "doi", w.DOI, "error", err)
			continue
		}
		kept = append(kept, id)
	}

	// Withdraw old links only on a complete, non-empty answer: an outage, a truncated
	// list or an emptied record must not erase a researcher's list.
	if fetched.completeAnswer() && failed == 0 && len(kept) > 0 {
		if err := withdrawSyncedLinks(ctx, tx, researcherID, kept); err != nil {
			return 0, failed, err
		}
	}
	if err := updateTotals(ctx, tx, researcherID, fetched); err != nil {
		return 0, failed, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, failed, err
	}
	return len(kept), failed, nil
}

func upsertInSavepoint(ctx context.Context, tx pgx.Tx, researcherID string, w importedWork) (string, error) {
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer savepoint.Rollback(ctx) //nolint:errcheck // a no-op after commit
	id, err := upsertPublication(ctx, savepoint, researcherID, w)
	if err != nil {
		return "", err
	}
	return id, savepoint.Commit(ctx)
}

func findPublication(ctx context.Context, tx pgx.Tx, researcherID string, w importedWork) (string, error) {
	var id string
	if w.DOI != "" {
		err := tx.QueryRow(ctx, `SELECT id FROM publications WHERE lower(doi) = $1`, w.DOI).Scan(&id)
		if err == nil || !errors.Is(err, pgx.ErrNoRows) {
			return id, err
		}
	}
	// DOI-less works would otherwise be inserted again on every sync. The lookup stays
	// inside this researcher's list so one author's title can never claim another's paper.
	err := tx.QueryRow(ctx, `
		SELECT p.id FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		WHERE pa.researcher_id = $1
		  AND lower(btrim(COALESCE(p.title_en, p.title_ar))) = lower(btrim($2))
		  AND (p.doi IS NULL OR $3 = '')
		LIMIT 1
	`, researcherID, w.Title, w.DOI).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// upsertPublication links the researcher to a work. Publications are shared between
// authors and a work list is only what one person asserts, so an existing row keeps
// its metadata: nothing from a sync may change it except OpenAlex's citation count.
func upsertPublication(ctx context.Context, tx pgx.Tx, researcherID string, w importedWork) (string, error) {
	id, err := findPublication(ctx, tx, researcherID, w)
	if err != nil {
		return "", fmt.Errorf("find publication: %w", err)
	}
	if id == "" {
		id, err = insertPublication(ctx, tx, w)
		if err != nil {
			return "", fmt.Errorf("insert publication: %w", err)
		}
	} else if w.FromOpenAlex {
		if _, err = tx.Exec(ctx, `
			UPDATE publications SET
				citation_count = GREATEST(COALESCE(citation_count, 0), $2),
				citation_count_updated_at = NOW(),
				updated_at = NOW()
			WHERE id = $1
		`, id, w.CitationCount); err != nil {
			return "", fmt.Errorf("update citation count: %w", err)
		}
	}
	return id, linkAuthor(ctx, tx, id, researcherID)
}

// insertPublication creates a row, stamping citation_count_updated_at: that stamp is
// what marks a publication as import-created in withdrawSyncedLinks. A concurrent
// insert of the same DOI is treated like any existing row.
func insertPublication(ctx context.Context, tx pgx.Tx, w importedWork) (string, error) {
	titleAr, titleEn := splitTitle(w.Title)
	language := w.Language
	if language == "" || len(language) > 10 {
		language = "en"
		if titleAr != nil {
			language = "ar"
		}
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO publications (doi, title_ar, title_en, abstract_en, publication_date, publication_type,
		                          citation_count, citation_count_updated_at, language, metadata)
		VALUES (NULLIF($1, ''), $2, $3, NULLIF($4, ''), NULLIF($5, '')::date, $6::publication_type,
		        $7, NOW(), $8, jsonb_strip_nulls(jsonb_build_object('journal_name', NULLIF($9::text, ''))))
		ON CONFLICT (doi) DO UPDATE SET
			citation_count = CASE WHEN $10 THEN GREATEST(COALESCE(publications.citation_count, 0), EXCLUDED.citation_count)
			                      ELSE publications.citation_count END,
			citation_count_updated_at = CASE WHEN $10 THEN NOW() ELSE publications.citation_count_updated_at END
		RETURNING id
	`, w.DOI, titleAr, titleEn, w.AbstractEn, w.PublicationDate, publicationType(w.Type),
		w.CitationCount, language, w.Journal, w.FromOpenAlex).Scan(&id)
	return id, err
}

// linkAuthor adds the researcher once, at the next free position: co-authors from this
// university share one publication, and UNIQUE(publication_id, author_order) would
// otherwise reject the second of them. name_en is often empty for portal-created records.
func linkAuthor(ctx context.Context, tx pgx.Tx, publicationID, researcherID string) error {
	if _, err := tx.Exec(ctx, `SELECT 1 FROM publications WHERE id = $1 FOR UPDATE`, publicationID); err != nil {
		return fmt.Errorf("lock publication: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO publication_authors (publication_id, researcher_id, author_name, author_order, source)
		SELECT $1::uuid, r.id, COALESCE(NULLIF(r.name_en, ''), r.name_ar),
		       (SELECT COALESCE(MAX(author_order), 0) + 1 FROM publication_authors WHERE publication_id = $1::uuid),
		       'sync'
		FROM researchers r
		WHERE r.id = $2::uuid
		  AND NOT EXISTS (SELECT 1 FROM publication_authors WHERE publication_id = $1::uuid AND researcher_id = $2::uuid)
	`, publicationID, researcherID); err != nil {
		return fmt.Errorf("link author: %w", err)
	}
	return nil
}

// withdrawSyncedLinks removes imported authorship absent from keep, then deletes a
// publication left authorless if an import created it and nothing else cites it. A
// row staff entered by hand survives a self-asserted link being withdrawn from it.
func withdrawSyncedLinks(ctx context.Context, tx pgx.Tx, researcherID string, keep []string) error {
	rows, err := tx.Query(ctx, `
		DELETE FROM publication_authors
		WHERE researcher_id = $1 AND source = 'sync' AND NOT (publication_id = ANY($2::text[]::uuid[]))
		RETURNING publication_id::text
	`, researcherID, keep)
	if err != nil {
		return fmt.Errorf("withdraw links: %w", err)
	}
	removed, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("withdraw links: %w", err)
	}
	if len(removed) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM publications p
		WHERE p.id = ANY($1::text[]::uuid[])
		  AND p.citation_count_updated_at IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM publication_authors pa WHERE pa.publication_id = p.id)
		  AND NOT EXISTS (SELECT 1 FROM portal_requests r WHERE r.publication_id = p.id)
		  AND NOT EXISTS (SELECT 1 FROM publication_incentives i WHERE i.publication_id = p.id)
	`, removed); err != nil {
		return fmt.Errorf("delete orphan publications: %w", err)
	}
	return nil
}

// updateTotals trusts OpenAlex's summary when it knows the author. When OpenAlex or the
// identity check could not answer, the earlier figures stay; when OpenAlex answered
// "unknown" or a name check refused it, none can be claimed.
func updateTotals(ctx context.Context, tx pgx.Tx, researcherID string, fetched fetchResult) error {
	const count = `(SELECT COUNT(DISTINCT publication_id) FROM publication_authors WHERE researcher_id = $1)`
	const citationSum = `COALESCE((
		SELECT SUM(p.citation_count) FROM publications p
		WHERE EXISTS (SELECT 1 FROM publication_authors pa WHERE pa.publication_id = p.id AND pa.researcher_id = $1)
	), 0)`

	var err error
	switch author := fetched.author; {
	case fetched.identityUnknown:
		_, err = tx.Exec(ctx, `
			UPDATE researchers SET total_publications = `+count+`, updated_at = NOW() WHERE id = $1
		`, researcherID)
	case author != nil:
		_, err = tx.Exec(ctx, `
			UPDATE researchers SET
				total_publications = `+count+`,
				total_citations = $2, h_index = $3, h_index_source = 'openalex', h_index_updated_at = NOW(),
				openalex_id = $4, updated_at = NOW()
			WHERE id = $1
		`, researcherID, author.CitedByCount, author.SummaryStats.HIndex, openAlexShortID(author.ID))
	case fetched.openalexErr == nil:
		_, err = tx.Exec(ctx, `
			UPDATE researchers SET
				total_publications = `+count+`, total_citations = `+citationSum+`,
				h_index = 0, h_index_source = NULL, openalex_id = NULL, updated_at = NOW()
			WHERE id = $1
		`, researcherID)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE researchers SET total_publications = `+count+`, updated_at = NOW() WHERE id = $1
		`, researcherID)
	}
	if err != nil {
		return fmt.Errorf("update totals: %w", err)
	}
	return nil
}

// ResetImported forgets everything derived from a previous ORCID, inside the caller's
// transaction. last_sync_started_at stays, so clearing an ORCID and entering another
// cannot dodge the cooldown; a run still in flight aborts on the changed ORCID.
func ResetImported(ctx context.Context, tx pgx.Tx, researcherID string) error {
	if err := withdrawSyncedLinks(ctx, tx, researcherID, []string{}); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE researchers SET
			total_publications = (SELECT COUNT(DISTINCT publication_id) FROM publication_authors WHERE researcher_id = $1),
			total_citations = 0, h_index = 0, h_index_source = NULL, h_index_updated_at = NULL,
			openalex_id = NULL, last_synced_at = NULL, last_sync_error = NULL,
			last_sync_finished_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, researcherID)
	if err != nil {
		return fmt.Errorf("reset imported data: %w", err)
	}
	return nil
}
