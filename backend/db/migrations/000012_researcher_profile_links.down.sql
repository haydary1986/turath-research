ALTER TABLE publication_authors DROP COLUMN IF EXISTS source;

ALTER TABLE researchers
    DROP COLUMN IF EXISTS google_scholar_url,
    DROP COLUMN IF EXISTS scopus_url,
    DROP COLUMN IF EXISTS researchgate_url,
    DROP COLUMN IF EXISTS linkedin_url,
    DROP COLUMN IF EXISTS openalex_id,
    DROP COLUMN IF EXISTS h_index_source,
    DROP COLUMN IF EXISTS last_synced_at,
    DROP COLUMN IF EXISTS last_sync_error,
    DROP COLUMN IF EXISTS last_sync_started_at,
    DROP COLUMN IF EXISTS last_sync_finished_at;
