-- ══════════════════════════════════════════════════
-- Migration 9: scientific activities register + journal watchlist
-- ══════════════════════════════════════════════════

-- Seminars, workshops, panels… reported by colleges; scientific affairs
-- approves and records the official approval-letter number (كتاب الإقامة).
CREATE TABLE activities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    activity_type   VARCHAR(30) NOT NULL
        CHECK (activity_type IN ('seminar', 'workshop', 'panel', 'lecture', 'course', 'conference', 'other')),
    title_ar        VARCHAR(500) NOT NULL,
    description     TEXT,
    presenter       VARCHAR(500),
    college_id      UUID REFERENCES colleges(id) ON DELETE SET NULL,
    department_id   UUID REFERENCES departments(id) ON DELETE SET NULL,
    activity_date   DATE NOT NULL,
    location        VARCHAR(255),
    attendees_count INTEGER NOT NULL DEFAULT 0 CHECK (attendees_count >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'returned', 'rejected')),
    approval_no     VARCHAR(100),       -- عدد كتاب الإقامة الصادر من الشؤون العلمية
    decision_note   TEXT,
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    decided_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    decided_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activities_owner   ON activities(created_by, created_at DESC);
CREATE INDEX idx_activities_college ON activities(college_id, activity_date DESC);
CREATE INDEX idx_activities_status  ON activities(status, created_at DESC);

-- Accredited / predatory / suspended journals watchlist, matched on
-- normalized ISSN and on a folded name key.
CREATE TABLE journal_screening (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issn           VARCHAR(9),
    name           VARCHAR(500) NOT NULL,
    name_key       VARCHAR(500) NOT NULL,
    publisher      VARCHAR(255),
    classification VARCHAR(20) NOT NULL
        CHECK (classification IN ('accredited', 'predatory', 'suspended')),
    note           TEXT,
    source         VARCHAR(255),
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_journal_screening_issn ON journal_screening(issn) WHERE issn IS NOT NULL;
CREATE INDEX idx_journal_screening_name ON journal_screening(name_key);
