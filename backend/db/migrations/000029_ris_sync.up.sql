-- Two-way sync of researcher profiles with the RIS (ris.uoturath.edu.iq).

-- Which RIS researcher each portal researcher is (matched by university email).
CREATE TABLE ris_sync_links (
    researcher_id     UUID PRIMARY KEY REFERENCES researchers(id) ON DELETE CASCADE,
    ris_researcher_id UUID NOT NULL UNIQUE,
    linked_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The value both systems last agreed on, per researcher and field. A side that moved
-- away from it since is the one that changed.
CREATE TABLE ris_sync_state (
    researcher_id UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    field         VARCHAR(30) NOT NULL,
    value         TEXT NOT NULL,
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (researcher_id, field)
);

CREATE TABLE ris_sync_runs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mode        VARCHAR(10) NOT NULL CHECK (mode IN ('dry_run', 'apply')),
    trigger     VARCHAR(10) NOT NULL CHECK (trigger IN ('schedule', 'manual')),
    started_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    stats       JSONB NOT NULL DEFAULT '{}',
    error       TEXT
);
CREATE INDEX idx_ris_sync_runs_started ON ris_sync_runs(started_at DESC);

-- Every copy the sync made (or, in a dry run, would make).
CREATE TABLE ris_sync_changes (
    id            BIGSERIAL PRIMARY KEY,
    run_id        UUID NOT NULL REFERENCES ris_sync_runs(id) ON DELETE CASCADE,
    researcher_id UUID REFERENCES researchers(id) ON DELETE SET NULL,
    field         VARCHAR(30) NOT NULL,
    action        VARCHAR(20) NOT NULL,
    old_value     TEXT,
    new_value     TEXT,
    applied       BOOLEAN NOT NULL DEFAULT FALSE,
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ris_sync_changes_run ON ris_sync_changes(run_id);
CREATE INDEX idx_ris_sync_changes_researcher ON ris_sync_changes(researcher_id, created_at DESC);

-- Differences an admin must decide; the field is not synced while one is open.
CREATE TABLE ris_sync_reviews (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    field         VARCHAR(30) NOT NULL,
    portal_value  TEXT NOT NULL,
    ris_value     TEXT NOT NULL,
    reason        VARCHAR(40) NOT NULL DEFAULT 'conflict',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at   TIMESTAMPTZ,
    resolved_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    resolution    VARCHAR(10) CHECK (resolution IN ('portal', 'ris'))
);
CREATE UNIQUE INDEX uq_ris_sync_reviews_open ON ris_sync_reviews(researcher_id, field) WHERE resolved_at IS NULL;
