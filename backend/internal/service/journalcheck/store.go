package journalcheck

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore implements Store on PostgreSQL.
type PGStore struct {
	db *pgxpool.Pool
}

// NewPGStore builds a PGStore.
func NewPGStore(db *pgxpool.Pool) *PGStore {
	return &PGStore{db: db}
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ReplaceList swaps one list atomically, so readers never see it half-loaded.
func (p *PGStore) ReplaceList(ctx context.Context, listType string, entries []RefEntry, at time.Time) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			slog.Error("journal list rollback failed", "error", rbErr)
		}
	}()
	if _, err := tx.Exec(ctx, `DELETE FROM journal_reference_list WHERE list_type = $1`, listType); err != nil {
		return fmt.Errorf("clear list: %w", err)
	}
	rows := make([][]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []any{
			e.NameKey, e.Name, nullable(e.URL), nullable(e.URLHost), listType,
			nullable(e.AuthenticName), nullable(e.AuthenticURL), e.Source, e.FetchedAt,
		})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"journal_reference_list"},
		[]string{"name_key", "name", "url", "url_host", "list_type", "authentic_name", "authentic_url", "source", "fetched_at"},
		pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("insert list: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO journal_reference_refresh (list_type, last_attempt_at, last_success_at, last_error, row_count)
		VALUES ($1, $2, $2, NULL, $3)
		ON CONFLICT (list_type) DO UPDATE SET
			last_attempt_at = $2, last_success_at = $2, last_error = NULL, row_count = $3
	`, listType, at, len(entries)); err != nil {
		return fmt.Errorf("record refresh: %w", err)
	}
	return tx.Commit(ctx)
}

// RecordRefreshFailure notes a failed attempt without touching the list rows.
func (p *PGStore) RecordRefreshFailure(ctx context.Context, listType, message string, at time.Time) error {
	_, err := p.db.Exec(ctx, `
		INSERT INTO journal_reference_refresh (list_type, last_attempt_at, last_error)
		VALUES ($1, $2, $3)
		ON CONFLICT (list_type) DO UPDATE SET last_attempt_at = $2, last_error = $3
	`, listType, at, message)
	return err
}

// ListStatuses counts live rows, so the figure stays true whatever the refresh row says.
func (p *PGStore) ListStatuses(ctx context.Context) ([]ListStatus, error) {
	rows, err := p.db.Query(ctx, `
		SELECT t.list_type, COALESCE(c.n, 0), r.last_attempt_at, r.last_success_at, r.last_error IS NOT NULL
		FROM unnest($1::text[]) AS t(list_type)
		LEFT JOIN (SELECT list_type, COUNT(*) AS n FROM journal_reference_list GROUP BY list_type) c USING (list_type)
		LEFT JOIN journal_reference_refresh r ON r.list_type = t.list_type
	`, ListTypes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ListStatus{}
	for rows.Next() {
		var status ListStatus
		var failed *bool
		if err := rows.Scan(&status.ListType, &status.Count, &status.LastAttemptAt, &status.LastSuccessAt, &failed); err != nil {
			return nil, err
		}
		status.LastFailed = failed != nil && *failed
		out = append(out, status)
	}
	return out, rows.Err()
}

// FindReferences returns candidate rows; MatchHits decides how each one matched.
func (p *PGStore) FindReferences(ctx context.Context, nameKeys, publisherKeys, hosts []string) ([]RefEntry, error) {
	rows, err := p.db.Query(ctx, `
		SELECT list_type, name_key, name, COALESCE(url, ''), COALESCE(url_host, ''),
		       COALESCE(authentic_name, ''), COALESCE(authentic_url, ''), source, fetched_at
		FROM journal_reference_list
		WHERE (list_type <> 'predatory_publisher' AND name_key = ANY($1))
		   OR (list_type = 'predatory_publisher' AND name_key = ANY($2))
		   OR url_host = ANY($3)
		LIMIT 50
	`, nameKeys, publisherKeys, hosts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RefEntry{}
	for rows.Next() {
		var e RefEntry
		if err := rows.Scan(&e.ListType, &e.NameKey, &e.Name, &e.URL, &e.URLHost, &e.AuthenticName, &e.AuthenticURL, &e.Source, &e.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// FindManual returns watchlist rows matching any ISSN or name key.
func (p *PGStore) FindManual(ctx context.Context, issns, nameKeys []string) ([]ManualRow, error) {
	rows, err := p.db.Query(ctx, `
		SELECT COALESCE(issn, ''), name_key, name, classification, note
		FROM journal_screening
		WHERE issn = ANY($1) OR name_key = ANY($2)
		ORDER BY updated_at DESC
	`, issns, nameKeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ManualRow{}
	for rows.Next() {
		var row ManualRow
		if err := rows.Scan(&row.ISSN, &row.NameKey, &row.Name, &row.Classification, &row.Note); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CacheGet returns an unexpired payload.
func (p *PGStore) CacheGet(ctx context.Context, source, key string, now time.Time) ([]byte, bool, error) {
	var payload []byte
	err := p.db.QueryRow(ctx, `
		SELECT payload FROM journal_lookup_cache WHERE source = $1 AND lookup_key = $2 AND expires_at > $3
	`, source, key, now).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

// CachePut stores or replaces a payload.
func (p *PGStore) CachePut(ctx context.Context, source, key string, payload []byte, now time.Time, ttl time.Duration) error {
	_, err := p.db.Exec(ctx, `
		INSERT INTO journal_lookup_cache (source, lookup_key, payload, fetched_at, expires_at)
		VALUES ($1, $2, $3::jsonb, $4, $5)
		ON CONFLICT (source, lookup_key) DO UPDATE SET
			payload = EXCLUDED.payload, fetched_at = EXCLUDED.fetched_at, expires_at = EXCLUDED.expires_at
	`, source, key, string(payload), now, now.Add(ttl))
	return err
}

// PurgeCache removes expired lookups.
func (p *PGStore) PurgeCache(ctx context.Context, now time.Time) error {
	_, err := p.db.Exec(ctx, `DELETE FROM journal_lookup_cache WHERE expires_at <= $1`, now)
	return err
}
