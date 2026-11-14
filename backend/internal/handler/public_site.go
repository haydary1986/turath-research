package handler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
	"github.com/haydary1986/turath-research/backend/internal/service/publicsite"
)

// PublicHandler serves the anonymous, aggregate-only parts of the public site: the
// live statistics, the latest publications and the public journal pages.
type PublicHandler struct {
	db       *pgxpool.Pool
	journals *journalcheck.Service
	// institutionWorks counts the university's works in OpenAlex; nil disables it.
	institutionWorks InstitutionWorksCounter

	stats        cached[map[string]int64]
	publications cached[[]publicPublication]

	// Uncached journal lookups cost three upstream calls each: a visitor gets a few,
	// and all anonymous visitors together share a ceiling.
	checkPerIP  *journalcheck.UserThrottle
	checkGlobal *journalcheck.UserThrottle
}

const (
	publicCacheTTL = 10 * time.Minute

	publicChecksPerIP       = 10
	publicChecksGlobal      = 300
	publicChecksInterval    = 10 * time.Minute
	publicChecksGlobalKey   = "*"
	publicStatsCacheControl = "public, max-age=60"
)

// InstitutionWorksCounter returns the number of works an external index attributes to the university.
type InstitutionWorksCounter func(ctx context.Context) (int, error)

// institutionWorksTimeout keeps a slow index from holding up the public statistics.
const institutionWorksTimeout = 8 * time.Second

// NewPublicHandler builds a PublicHandler.
func NewPublicHandler(db *pgxpool.Pool, journals *journalcheck.Service, institutionWorks InstitutionWorksCounter) *PublicHandler {
	return &PublicHandler{
		db: db, journals: journals, institutionWorks: institutionWorks,
		checkPerIP:  journalcheck.NewUserThrottle(publicChecksPerIP, publicChecksInterval),
		checkGlobal: journalcheck.NewUserThrottle(publicChecksGlobal, publicChecksInterval),
	}
}

// cached holds one value computed at most once per publicCacheTTL.
type cached[T any] struct {
	mu    sync.Mutex
	value T
	at    time.Time
}

// get returns the cached value, recomputing it when stale. A failed load keeps no value.
func (c *cached[T]) get(ctx context.Context, load func(context.Context) (T, error)) (T, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < publicCacheTTL {
		return c.value, c.at, nil
	}
	value, err := load(ctx)
	if err != nil {
		var zero T
		return zero, time.Time{}, err
	}
	c.value, c.at = value, time.Now()
	return value, c.at, nil
}

// clientIP is the caller's address as set by the RealIP middleware.
func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// ── Live statistics ─────────────────────────────

// statsQuery computes every statistic in one round trip. What each one counts:
//   - academics: researchers whose status is active.
//   - publications: published records with a title that are linked to at least one researcher;
//     computeStats raises it to the university's OpenAlex total when that is larger.
//   - indexed: distinct papers (by DOI, else by record) from journal-verification requests
//     that passed review (verified onward) with index_db scopus or wos, plus linked
//     publications that carry a Scopus record id.
//   - colleges: rows of the colleges table.
//   - activities: scientific activities approved and dated in the current calendar year.
const statsQuery = `
	WITH linked AS (
		SELECT p.id, p.doi, p.scopus_id FROM publications p
		WHERE p.status = 'published'
		  AND COALESCE(NULLIF(btrim(p.title_en), ''), NULLIF(btrim(p.title_ar), '')) IS NOT NULL
		  AND EXISTS (SELECT 1 FROM publication_authors pa JOIN researchers r ON r.id = pa.researcher_id
		              WHERE pa.publication_id = p.id)
	)
	SELECT
		(SELECT COUNT(*) FROM researchers WHERE status = 'active'),
		(SELECT COUNT(*) FROM linked),
		(SELECT COUNT(*) FROM (
			SELECT COALESCE(lower(doi), 'request:' || id::text) FROM portal_requests
			WHERE kind = 'journal_verification'
			  AND status IN ('verified', 'awaiting_signatures', 'completed')
			  AND details->>'index_db' IN ('scopus', 'wos')
			UNION
			SELECT COALESCE(lower(doi), 'publication:' || id::text) FROM linked WHERE scopus_id IS NOT NULL
		) indexed),
		(SELECT COUNT(*) FROM colleges),
		(SELECT COUNT(*) FROM activities
		 WHERE status = 'approved'
		   AND activity_date >= date_trunc('year', CURRENT_DATE)::date
		   AND activity_date < (date_trunc('year', CURRENT_DATE) + INTERVAL '1 year')::date)`

func (h *PublicHandler) computeStats(ctx context.Context) (map[string]int64, error) {
	var academics, publications, indexed, colleges, activities int64
	err := h.db.QueryRow(ctx, statsQuery).Scan(&academics, &publications, &indexed, &colleges, &activities)
	if err != nil {
		return nil, err
	}
	publications = publishedCount(publications, h.countInstitutionWorks(ctx))
	return map[string]int64{
		publicsite.StatAcademics: academics, publicsite.StatPublications: publications,
		publicsite.StatIndexed: indexed, publicsite.StatColleges: colleges, publicsite.StatActivities: activities,
	}, nil
}

// countInstitutionWorks asks OpenAlex for the university's total output; -1 when unavailable.
func (h *PublicHandler) countInstitutionWorks(ctx context.Context) int64 {
	if h.institutionWorks == nil {
		return -1
	}
	ctx, cancel := context.WithTimeout(ctx, institutionWorksTimeout)
	defer cancel()
	n, err := h.institutionWorks(ctx)
	if err != nil {
		slog.Warn("public stats: institution works count unavailable, using portal records", "error", err)
		return -1
	}
	return int64(n)
}

// publishedCount is the larger of the portal's own records and the external index total, so
// the figure covers research not yet linked to a researcher and never drops when the index is down.
func publishedCount(local, external int64) int64 {
	return max(local, external)
}

// publicStat is one statistic as visitors see it. Number is the computed value;
// when staff set a manual value, Value carries it and Number is null.
type publicStat struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	LabelEn string `json:"label_en"`
	Value   string `json:"value"`
	Number  *int64 `json:"number"`
}

// staffStat adds the editor's view: the stored setting and the computed number.
type staffStat struct {
	publicsite.StatSetting
	Computed int64 `json:"computed"`
}

type staffStatsResponse struct {
	Items      []staffStat `json:"items"`
	ComputedAt time.Time   `json:"computed_at"`
}

// loadStats returns the stored settings (in display order) and the computed numbers.
func (h *PublicHandler) loadStats(r *http.Request) ([]publicsite.StatSetting, map[string]int64, time.Time, error) {
	ctx, cancel := portalContext(r)
	defer cancel()
	content, _, err := loadSiteContent(ctx, h.db)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	settings := content.Stats
	if settings == nil {
		settings = publicsite.NormalizeContent(publicsite.Content{}).Stats
	}
	numbers, at, err := h.stats.get(ctx, h.computeStats)
	return settings, numbers, at, err
}

// PublicStats returns the visible statistics in display order (no authentication).
func (h *PublicHandler) PublicStats(w http.ResponseWriter, r *http.Request) {
	settings, numbers, _, err := h.loadStats(r)
	if err != nil {
		serverError(w, "Failed to compute statistics", err)
		return
	}
	out := []publicStat{}
	for _, s := range settings {
		if s.Hidden {
			continue
		}
		stat := publicStat{Key: s.Key, Label: s.Label, LabelEn: s.LabelEn, Value: s.Value}
		if s.Value == "" {
			n := numbers[s.Key]
			stat.Number = &n
		}
		out = append(out, stat)
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondJSON(w, http.StatusOK, out)
}

// StaffStats returns every statistic with its setting and computed number (site editors).
func (h *PublicHandler) StaffStats(w http.ResponseWriter, r *http.Request) {
	settings, numbers, at, err := h.loadStats(r)
	if err != nil {
		serverError(w, "Failed to compute statistics", err)
		return
	}
	items := make([]staffStat, 0, len(settings))
	for _, s := range settings {
		items = append(items, staffStat{StatSetting: s, Computed: numbers[s.Key]})
	}
	model.RespondJSON(w, http.StatusOK, staffStatsResponse{Items: items, ComputedAt: at})
}
