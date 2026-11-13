package handler

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
)

// Public, read-only view of the predatory/hijacked reference lists and an anonymous
// journal check. Everything here is aggregate list data: the staff watchlist is only
// reflected in a check's verdict, never in its name, note or classification fields.

const (
	maxPublicListPage    = 100
	maxPublicSitemapPage = 5000
	maxRelatedEntries    = 6
	maxNeighbourEntries  = 3
	maxPublicQueryRunes  = 200
	letterOther          = "other"
)

var (
	slugShape   = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,100}$`)
	letterShape = regexp.MustCompile(`^[a-z]$`)
)

type publicJournalEntry struct {
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	ListType      string    `json:"list_type"`
	Hosts         []string  `json:"hosts"`
	AuthenticName *string   `json:"authentic_name"`
	AuthenticURL  *string   `json:"authentic_url"`
	Source        string    `json:"source"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type journalEntryLink struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	ListType string `json:"list_type"`
}

type journalEntryDetail struct {
	Entry           publicJournalEntry `json:"entry"`
	SameHost        []journalEntryLink `json:"same_host"`
	Neighbours      []journalEntryLink `json:"neighbours"`
	ListRefreshedAt *time.Time         `json:"list_refreshed_at"`
}

type journalLetter struct {
	Letter string `json:"letter"`
	Count  int    `json:"count"`
}

type journalSitemapEntry struct {
	Slug      string    `json:"slug"`
	FetchedAt time.Time `json:"fetched_at"`
}

// One listed journal or publisher spans several rows (aliases, alternate fake sites);
// they share a slug and are grouped back into one entry here.
const groupedEntrySelect = `
	SELECT slug, list_type, min(name),
	       COALESCE(array_agg(DISTINCT url_host ORDER BY url_host) FILTER (WHERE url_host <> ''), '{}'),
	       max(authentic_name), max(authentic_url), min(source), max(fetched_at)
	FROM journal_reference_list`

func scanJournalEntry(row rowScanner) (publicJournalEntry, error) {
	var e publicJournalEntry
	err := row.Scan(&e.Slug, &e.ListType, &e.Name, &e.Hosts, &e.AuthenticName, &e.AuthenticURL, &e.Source, &e.FetchedAt)
	return e, err
}

// publicListType reads the required "type" parameter; ok is false after writing a 400.
func publicListType(w http.ResponseWriter, r *http.Request) (string, bool) {
	listType := r.URL.Query().Get("type")
	if _, ok := referenceListTypes[listType]; !ok {
		model.RespondError(w, http.StatusBadRequest, "type must be predatory_journal, predatory_publisher or hijacked")
		return "", false
	}
	return listType, true
}

// PublicJournalStatus reports list sizes and refresh times (no authentication).
func (h *PublicHandler) PublicJournalStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := portalContext(r)
	defer cancel()
	statuses, err := h.journals.Statuses(ctx)
	if err != nil {
		serverError(w, "Failed to read list status", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondJSON(w, http.StatusOK, statuses)
}

// PublicJournalCheck runs the advisory check for anonymous visitors. Uncached external
// lookups are throttled per IP and overall; local lists are always consulted.
func (h *PublicHandler) PublicJournalCheck(w http.ResponseWriter, r *http.Request) {
	query, problems := journalQuery(r)
	if len(problems) > 0 {
		validationError(w, problems)
		return
	}
	ip := clientIP(r)
	query.AllowExternal = func() bool {
		return h.checkPerIP.Allow(ip) && h.checkGlobal.Allow(publicChecksGlobalKey)
	}
	ctx, cancel := context.WithTimeout(r.Context(), journalCheckTimeout)
	defer cancel()
	result, err := h.journals.Check(ctx, query)
	if err != nil {
		serverError(w, "Failed to check journal", err)
		return
	}
	model.RespondJSON(w, http.StatusOK, publicCheckResult(result))
}

// publicCheckResult drops the staff watchlist's internals (entry name, note,
// classification, how it matched); the verdict and reason codes still reflect it.
func publicCheckResult(in journalcheck.Result) journalcheck.Result {
	out := in
	out.Matched = false
	out.Classification = ""
	out.Name = ""
	out.Note = nil
	out.MatchedBy = ""
	return out
}

// PublicJournalEntries pages through one list, filtered by first letter or text (no authentication).
func (h *PublicHandler) PublicJournalEntries(w http.ResponseWriter, r *http.Request) {
	listType, ok := publicListType(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), 50, 1, maxPublicListPage)

	where := " WHERE list_type = $1"
	args := []any{listType}
	switch letter := strings.ToLower(q.Get("letter")); {
	case letterShape.MatchString(letter):
		args = append(args, letter+"%")
		where += " AND lower(name) LIKE $" + strconv.Itoa(len(args))
	case letter == letterOther:
		where += " AND lower(name) !~ '^[a-z]'"
	}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		if len([]rune(text)) > maxPublicQueryRunes {
			model.RespondError(w, http.StatusBadRequest, "query is too long")
			return
		}
		args = append(args, "%"+likeEscaper.Replace(text)+"%")
		n := strconv.Itoa(len(args))
		where += " AND (name ILIKE $" + n + " OR url_host ILIKE $" + n + " OR authentic_name ILIKE $" + n + ")"
	}

	ctx, cancel := portalContext(r)
	defer cancel()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(DISTINCT slug) FROM journal_reference_list`+where, args...).Scan(&total); err != nil {
		serverError(w, "Failed to count list", err)
		return
	}
	args = append(args, limit, (page-1)*limit)
	items, err := h.queryEntries(ctx, groupedEntrySelect+where+` GROUP BY slug, list_type
		ORDER BY lower(min(name)), slug LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}

func (h *PublicHandler) queryEntries(ctx context.Context, sql string, args ...any) ([]publicJournalEntry, error) {
	rows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []publicJournalEntry{}
	for rows.Next() {
		e, err := scanJournalEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

// PublicJournalLetters counts the entries of one list by first letter, for the A–Z index.
func (h *PublicHandler) PublicJournalLetters(w http.ResponseWriter, r *http.Request) {
	listType, ok := publicListType(w, r)
	if !ok {
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		SELECT letter, COUNT(*) FROM (
			SELECT slug, CASE WHEN lower(left(min(name), 1)) ~ '^[a-z]$' THEN lower(left(min(name), 1)) ELSE $2::text END AS letter
			FROM journal_reference_list WHERE list_type = $1 GROUP BY slug
		) g GROUP BY letter ORDER BY letter = $2::text, letter`, listType, letterOther)
	if err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	defer rows.Close()
	out := []journalLetter{}
	for rows.Next() {
		var l journalLetter
		if err := rows.Scan(&l.Letter, &l.Count); err != nil {
			serverError(w, "Failed to read list", err)
			return
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondJSON(w, http.StatusOK, out)
}

// PublicJournalEntry returns one listed journal or publisher with related entries.
// A slug that left the list is a 404, so removed entries stop being published.
func (h *PublicHandler) PublicJournalEntry(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !slugShape.MatchString(slug) {
		model.RespondError(w, http.StatusNotFound, "Entry not found")
		return
	}
	ctx, cancel := portalContext(r)
	defer cancel()
	entries, err := h.queryEntries(ctx, groupedEntrySelect+` WHERE slug = $1 GROUP BY slug, list_type ORDER BY list_type LIMIT 1`, slug)
	if err != nil {
		serverError(w, "Failed to load entry", err)
		return
	}
	if len(entries) == 0 {
		model.RespondError(w, http.StatusNotFound, "Entry not found")
		return
	}
	detail := journalEntryDetail{Entry: entries[0], SameHost: []journalEntryLink{}, Neighbours: []journalEntryLink{}}
	if detail.SameHost, err = h.sameHostEntries(ctx, detail.Entry); err == nil {
		detail.Neighbours, err = h.neighbourEntries(ctx, detail.Entry)
	}
	if err == nil {
		err = h.db.QueryRow(ctx, `SELECT last_success_at FROM journal_reference_refresh WHERE list_type = $1`,
			detail.Entry.ListType).Scan(&detail.ListRefreshedAt)
		if isNoRows(err) {
			err = nil
		}
	}
	if err != nil {
		serverError(w, "Failed to load related entries", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondJSON(w, http.StatusOK, detail)
}

func (h *PublicHandler) sameHostEntries(ctx context.Context, e publicJournalEntry) ([]journalEntryLink, error) {
	if len(e.Hosts) == 0 {
		return []journalEntryLink{}, nil
	}
	return h.queryLinks(ctx, `
		SELECT slug, min(name), list_type FROM journal_reference_list
		WHERE url_host = ANY($1) AND slug <> $2
		GROUP BY slug, list_type ORDER BY lower(min(name)), slug LIMIT $3`, e.Hosts, e.Slug, maxRelatedEntries)
}

func (h *PublicHandler) neighbourEntries(ctx context.Context, e publicJournalEntry) ([]journalEntryLink, error) {
	return h.queryLinks(ctx, `
		WITH g AS (
			SELECT slug, min(name) AS name, list_type FROM journal_reference_list
			WHERE list_type = $1 GROUP BY slug, list_type
		)
		(SELECT slug, name, list_type FROM g WHERE (lower(name), slug) < (lower($2), $3)
		 ORDER BY lower(name) DESC, slug DESC LIMIT $4)
		UNION ALL
		(SELECT slug, name, list_type FROM g WHERE (lower(name), slug) > (lower($2), $3)
		 ORDER BY lower(name), slug LIMIT $4)`, e.ListType, e.Name, e.Slug, maxNeighbourEntries)
}

func (h *PublicHandler) queryLinks(ctx context.Context, sql string, args ...any) ([]journalEntryLink, error) {
	rows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (journalEntryLink, error) {
		var l journalEntryLink
		err := row.Scan(&l.Slug, &l.Name, &l.ListType)
		return l, err
	})
}

// PublicJournalSitemap lists the slugs of one list for the sitemap, in stable order.
func (h *PublicHandler) PublicJournalSitemap(w http.ResponseWriter, r *http.Request) {
	listType, ok := publicListType(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page := clampInt(q.Get("page"), 1, 1, 100000)
	limit := clampInt(q.Get("limit"), maxPublicSitemapPage, 1, maxPublicSitemapPage)
	ctx, cancel := portalContext(r)
	defer cancel()
	var total int
	if err := h.db.QueryRow(ctx, `SELECT COUNT(DISTINCT slug) FROM journal_reference_list WHERE list_type = $1`,
		listType).Scan(&total); err != nil {
		serverError(w, "Failed to count list", err)
		return
	}
	rows, err := h.db.Query(ctx, `
		SELECT slug, max(fetched_at) FROM journal_reference_list WHERE list_type = $1
		GROUP BY slug ORDER BY slug LIMIT $2 OFFSET $3`, listType, limit, (page-1)*limit)
	if err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (journalSitemapEntry, error) {
		var e journalSitemapEntry
		err := row.Scan(&e.Slug, &e.FetchedAt)
		return e, err
	})
	if err != nil {
		serverError(w, "Failed to read list", err)
		return
	}
	w.Header().Set("Cache-Control", publicStatsCacheControl)
	model.RespondPaginated(w, http.StatusOK, items, total, page, limit)
}
