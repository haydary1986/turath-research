package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
)

// Orchestrator coordinates data import from multiple academic APIs.
type Orchestrator struct {
	db       *pgxpool.Pool
	openalex *apiclient.OpenAlexClient
	crossref *apiclient.CrossRefClient
	orcid    *apiclient.ORCIDClient
}

// NewOrchestrator creates a sync orchestrator with all API clients.
func NewOrchestrator(db *pgxpool.Pool, crossrefMailto string) *Orchestrator {
	return &Orchestrator{
		db:       db,
		openalex: apiclient.NewOpenAlexClient(crossrefMailto),
		crossref: apiclient.NewCrossRefClient(crossrefMailto),
		orcid:    apiclient.NewORCIDClient(),
	}
}

// SyncResearcher imports publications for a researcher from all sources.
func (o *Orchestrator) SyncResearcher(ctx context.Context, researcherID string) error {
	// Fetch researcher info
	var nameEn, orcidID *string
	err := o.db.QueryRow(ctx,
		`SELECT name_en, orcid_id FROM researchers WHERE id = $1`, researcherID,
	).Scan(&nameEn, &orcidID)
	if err != nil {
		return fmt.Errorf("researcher not found: %w", err)
	}

	// Create sync job
	jobID := uuid.New().String()
	_, _ = o.db.Exec(ctx, `
		INSERT INTO sync_jobs (id, job_type, target_type, target_id, status, started_at)
		VALUES ($1, 'openalex_sync', 'researcher', $2, 'running', NOW())
	`, jobID, researcherID)

	var importedCount int
	var syncErr error

	// Run sources in parallel using goroutines
	var wg sync.WaitGroup
	var mu sync.Mutex
	works := make(map[string]importedWork) // keyed by DOI for dedup

	// Source 1: OpenAlex (by name)
	if nameEn != nil && *nameEn != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := o.openalex.SearchWorksByAuthorName(ctx, *nameEn, 1, 50)
			if err != nil {
				slog.Warn("openalex sync failed", "researcher", researcherID, "error", err)
				return
			}
			mu.Lock()
			for _, w := range result.Results {
				doi := w.DOI
				if doi == "" {
					doi = w.ID // use OpenAlex ID as fallback key
				}
				works[doi] = importedWork{
					DOI:             w.DOI,
					TitleEn:         w.DisplayName,
					PublicationDate: w.PublicationDate,
					CitationCount:   w.CitedByCount,
					Source:          "openalex",
					Language:        w.Language,
					AbstractEn:      apiclient.ReconstructAbstract(w.AbstractInvIdx),
				}
			}
			mu.Unlock()
			slog.Info("openalex sync", "researcher", researcherID, "found", len(result.Results))
		}()
	}

	// Source 2: ORCID (by ORCID ID)
	if orcidID != nil && *orcidID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worksResult, err := o.orcid.GetWorks(ctx, *orcidID)
			if err != nil {
				slog.Warn("orcid sync failed", "researcher", researcherID, "error", err)
				return
			}
			mu.Lock()
			for _, group := range worksResult.Group {
				for _, ws := range group.WorkSummary {
					doi := apiclient.ExtractDOI(ws)
					title := ""
					if ws.Title != nil {
						title = ws.Title.Title.Value
					}
					key := doi
					if key == "" {
						key = title // fallback dedup by title
					}
					if key == "" {
						continue
					}
					if _, exists := works[key]; !exists {
						works[key] = importedWork{
							DOI:     doi,
							TitleEn: title,
							Source:  "orcid",
						}
					}
				}
			}
			mu.Unlock()
			slog.Info("orcid sync", "researcher", researcherID, "groups", len(worksResult.Group))
		}()
	}

	wg.Wait()

	// Persist deduplicated works
	for _, w := range works {
		if err := o.upsertPublication(ctx, researcherID, w); err != nil {
			slog.Warn("failed to upsert publication", "doi", w.DOI, "error", err)
			continue
		}
		importedCount++
	}

	// Update sync job status
	status := "completed"
	errMsg := ""
	if syncErr != nil {
		status = "failed"
		errMsg = syncErr.Error()
	}
	_, _ = o.db.Exec(ctx, `
		UPDATE sync_jobs SET status = $2, completed_at = NOW(), error_message = $3
		WHERE id = $1
	`, jobID, status, errMsg)

	// Update researcher totals
	_, _ = o.db.Exec(ctx, `
		UPDATE researchers SET
			total_publications = (SELECT COUNT(*) FROM publication_authors WHERE researcher_id = $1),
			total_citations = COALESCE((
				SELECT SUM(p.citation_count) FROM publications p
				JOIN publication_authors pa ON pa.publication_id = p.id
				WHERE pa.researcher_id = $1
			), 0),
			updated_at = NOW()
		WHERE id = $1
	`, researcherID)

	slog.Info("sync complete", "researcher", researcherID, "imported", importedCount, "total_deduped", len(works))
	return syncErr
}

type importedWork struct {
	DOI             string
	TitleEn         string
	PublicationDate string
	CitationCount   int
	Source          string
	Language        string
	AbstractEn      string
}

func (o *Orchestrator) upsertPublication(ctx context.Context, researcherID string, w importedWork) error {
	// Check if publication already exists by DOI
	var existingID string
	if w.DOI != "" {
		err := o.db.QueryRow(ctx, `SELECT id FROM publications WHERE doi = $1`, w.DOI).Scan(&existingID)
		if err == nil {
			// Publication exists — just ensure the author link exists
			o.ensureAuthorLink(ctx, existingID, researcherID)
			// Update citation count if higher
			_, _ = o.db.Exec(ctx, `
				UPDATE publications SET citation_count = GREATEST(citation_count, $2),
				       citation_count_updated_at = NOW(), updated_at = NOW()
				WHERE id = $1
			`, existingID, w.CitationCount)
			return nil
		}
	}

	// Insert new publication
	pubID := uuid.New().String()
	lang := w.Language
	if lang == "" {
		lang = "en"
	}

	_, err := o.db.Exec(ctx, `
		INSERT INTO publications (id, doi, title_en, abstract_en, publication_date,
		                          citation_count, citation_count_updated_at, language)
		VALUES ($1, $2, $3, $4, $5::date, $6, NOW(), $7)
		ON CONFLICT (doi) DO UPDATE SET
			citation_count = GREATEST(publications.citation_count, EXCLUDED.citation_count),
			citation_count_updated_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`, pubID, nilIfEmptyStr(w.DOI), nilIfEmptyStr(w.TitleEn), nilIfEmptyStr(w.AbstractEn),
		nilIfEmptyStr(w.PublicationDate), w.CitationCount, lang)

	if err != nil {
		return fmt.Errorf("insert publication: %w", err)
	}

	o.ensureAuthorLink(ctx, pubID, researcherID)
	return nil
}

func (o *Orchestrator) ensureAuthorLink(ctx context.Context, pubID, researcherID string) {
	_, _ = o.db.Exec(ctx, `
		INSERT INTO publication_authors (id, publication_id, researcher_id, author_name, author_order)
		SELECT uuid_generate_v4(), $1, $2, r.name_en, 1
		FROM researchers r WHERE r.id = $2
		ON CONFLICT DO NOTHING
	`, pubID, researcherID)
}

func nilIfEmptyStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// SyncByDOI enriches a publication using CrossRef data.
func (o *Orchestrator) SyncByDOI(ctx context.Context, doi string) error {
	work, err := o.crossref.GetWorkByDOI(ctx, doi)
	if err != nil {
		return fmt.Errorf("crossref lookup: %w", err)
	}

	title := work.GetFirstTitle()
	pubDate := work.GetPublicationDate()

	_, err = o.db.Exec(ctx, `
		UPDATE publications SET
			title_en = COALESCE(NULLIF($2, ''), title_en),
			abstract_en = COALESCE(NULLIF($3, ''), abstract_en),
			publication_date = COALESCE($4::date, publication_date),
			citation_count = GREATEST(citation_count, $5),
			volume = COALESCE(NULLIF($6, ''), volume),
			issue = COALESCE(NULLIF($7, ''), issue),
			pages = COALESCE(NULLIF($8, ''), pages),
			url = COALESCE(NULLIF($9, ''), url),
			updated_at = NOW()
		WHERE doi = $1
	`, doi, title, work.Abstract, nilIfEmptyStr(pubDate),
		work.IsReferencedBy, work.Volume, work.Issue, work.Page, work.URL)

	return err
}
