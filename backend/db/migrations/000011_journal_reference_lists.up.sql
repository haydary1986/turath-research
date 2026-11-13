-- ══════════════════════════════════════════════════
-- Migration 11: automatic journal screening
-- ══════════════════════════════════════════════════

-- Public predatory / hijacked lists, replaced per list_type on each refresh.
CREATE TABLE journal_reference_list (
    id             BIGSERIAL PRIMARY KEY,
    name_key       TEXT NOT NULL,
    name           TEXT NOT NULL,
    url            TEXT,
    url_host       TEXT,
    list_type      VARCHAR(30) NOT NULL
        CHECK (list_type IN ('predatory_journal', 'predatory_publisher', 'hijacked')),
    authentic_name TEXT,
    authentic_url  TEXT,
    source         VARCHAR(255) NOT NULL,
    fetched_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_journal_reference_name ON journal_reference_list(list_type, name_key);
CREATE INDEX idx_journal_reference_host ON journal_reference_list(url_host) WHERE url_host IS NOT NULL;

-- A failed fetch updates only this row; the list rows above stay untouched.
CREATE TABLE journal_reference_refresh (
    list_type       VARCHAR(30) PRIMARY KEY,
    last_attempt_at TIMESTAMPTZ NOT NULL,
    last_success_at TIMESTAMPTZ,
    last_error      TEXT,
    row_count       INTEGER NOT NULL DEFAULT 0
);

-- Answers from OpenAlex / Crossref / DOAJ. Failures are never cached.
CREATE TABLE journal_lookup_cache (
    source     VARCHAR(30) NOT NULL,
    lookup_key TEXT NOT NULL,
    payload    JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (source, lookup_key)
);

CREATE INDEX idx_journal_lookup_expiry ON journal_lookup_cache(expires_at);
