-- ══════════════════════════════════════════════════
-- Migration 12: researcher-managed profile links and automatic sync state
-- ══════════════════════════════════════════════════

ALTER TABLE researchers
    ADD COLUMN google_scholar_url    TEXT,
    ADD COLUMN scopus_url            TEXT,
    ADD COLUMN researchgate_url      TEXT,
    ADD COLUMN linkedin_url          TEXT,
    ADD COLUMN openalex_id           VARCHAR(30),
    ADD COLUMN h_index_source        VARCHAR(20),
    ADD COLUMN last_synced_at        TIMESTAMPTZ,
    ADD COLUMN last_sync_error       VARCHAR(50),
    ADD COLUMN last_sync_started_at  TIMESTAMPTZ,
    ADD COLUMN last_sync_finished_at TIMESTAMPTZ;

-- 'sync' marks authorship an import may later withdraw; NULL (portal, manual) is never touched.
ALTER TABLE publication_authors ADD COLUMN source VARCHAR(20);

-- The old importer attributed works by author name, and it is the only writer of
-- citation_count_updated_at. Marking its links lets the next ORCID sync withdraw namesakes.
UPDATE publication_authors pa
SET source = 'sync'
FROM publications p
WHERE p.id = pa.publication_id
  AND p.citation_count_updated_at IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM portal_requests r WHERE r.publication_id = p.id);
