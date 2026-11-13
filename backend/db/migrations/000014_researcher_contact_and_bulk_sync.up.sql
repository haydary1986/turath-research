-- ══════════════════════════════════════════════════
-- Migration 14: researcher contact data, Web of Science link, resumable bulk ORCID sync
-- ══════════════════════════════════════════════════

ALTER TABLE researchers
    ADD COLUMN phone              VARCHAR(11) CHECK (phone ~ '^07[0-9]{9}$'),
    ADD COLUMN phone_secondary    VARCHAR(11) CHECK (phone_secondary ~ '^07[0-9]{9}$'),
    ADD COLUMN recovery_email     VARCHAR(254),
    ADD COLUMN web_of_science_url TEXT,
    ADD COLUMN employee_no        VARCHAR(50),
    ADD COLUMN account_status     VARCHAR(20) CHECK (account_status IN ('linked', 'needs_review'));

CREATE TABLE bulk_sync_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status          VARCHAR(20) NOT NULL CHECK (status IN ('running', 'waiting', 'stopped', 'completed')),
    only_failed     BOOLEAN NOT NULL DEFAULT FALSE,
    -- Researchers are visited in id order; everything up to here is settled.
    cursor_id       UUID,
    processed       INT NOT NULL DEFAULT 0,
    succeeded       INT NOT NULL DEFAULT 0,
    failed          INT NOT NULL DEFAULT 0,
    skipped         INT NOT NULL DEFAULT 0,
    failures        JSONB NOT NULL DEFAULT '{}',
    wait_reason     VARCHAR(30),
    resume_at       TIMESTAMPTZ,
    -- The daily cap's window, kept here so a restart does not forgive it.
    window_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    window_count    INT NOT NULL DEFAULT 0,
    started_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX bulk_sync_runs_one_active ON bulk_sync_runs ((TRUE)) WHERE status IN ('running', 'waiting');
