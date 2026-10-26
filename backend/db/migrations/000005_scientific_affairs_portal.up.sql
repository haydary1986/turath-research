-- ══════════════════════════════════════════════════
-- Migration 5: Scientific Affairs Portal
-- Research plan registration, journal verification, appreciation letters
-- ══════════════════════════════════════════════════

-- Per-kind, per-year counters. Incremented atomically (see handler/portal_numbering.go).
CREATE TABLE portal_sequences (
    kind       VARCHAR(30) NOT NULL,
    year       INTEGER     NOT NULL,
    last_value INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (kind, year)
);

-- ── Requests (one table for all three workflows) ──

CREATE TABLE portal_requests (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_no         VARCHAR(30) UNIQUE,  -- assigned on submission, NULL while draft
    kind               VARCHAR(30) NOT NULL
        CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter')),
    status             VARCHAR(30) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'pending', 'returned', 'approved', 'rejected', 'verified',
                          'not_eligible', 'awaiting_signatures', 'completed', 'issued')),
    applicant_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title_ar           VARCHAR(500),
    title_en           VARCHAR(500),
    doi                VARCHAR(255),        -- normalized (lowercase, no URL prefix)
    college_id         UUID REFERENCES colleges(id) ON DELETE SET NULL,
    department_id      UUID REFERENCES departments(id) ON DELETE SET NULL,
    details            JSONB NOT NULL DEFAULT '{}',
    related_request_id UUID REFERENCES portal_requests(id) ON DELETE SET NULL,
    decision_note      TEXT,
    submitted_at       TIMESTAMPTZ,
    decided_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_requests_applicant ON portal_requests(applicant_id, created_at DESC);
CREATE INDEX idx_portal_requests_status    ON portal_requests(kind, status, created_at DESC);
CREATE INDEX idx_portal_requests_college   ON portal_requests(college_id);
CREATE INDEX idx_portal_requests_dept      ON portal_requests(department_id);

-- One live request per kind and DOI (a rejected / not-eligible one may be re-filed).
CREATE UNIQUE INDEX uq_portal_requests_live_doi
    ON portal_requests(kind, doi)
    WHERE doi IS NOT NULL AND status NOT IN ('rejected', 'not_eligible');

-- ── Audit trail ──────────────────────────────────

CREATE TABLE portal_request_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id  UUID NOT NULL REFERENCES portal_requests(id) ON DELETE CASCADE,
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    from_status VARCHAR(30),
    to_status   VARCHAR(30) NOT NULL,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_events_request ON portal_request_events(request_id, created_at);

-- ── Files (request attachments and template assets) ──

CREATE TABLE portal_files (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id    UUID REFERENCES portal_requests(id) ON DELETE CASCADE, -- NULL for template assets
    role          VARCHAR(40) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    stored_name   VARCHAR(100) NOT NULL UNIQUE,
    mime_type     VARCHAR(100) NOT NULL,
    size_bytes    BIGINT NOT NULL,
    uploaded_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_files_request ON portal_files(request_id);

-- ── Letter templates (uploaded letterhead + editable fields) ──

CREATE TABLE document_templates (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kind               VARCHAR(30) NOT NULL DEFAULT 'appreciation_letter',
    name               VARCHAR(255) NOT NULL,
    header_line_1      VARCHAR(255) NOT NULL DEFAULT '',
    header_line_2      VARCHAR(255) NOT NULL DEFAULT '',
    title_text         VARCHAR(255) NOT NULL DEFAULT '',
    body_text          TEXT NOT NULL,
    closing_text       VARCHAR(255) NOT NULL DEFAULT '',
    signer_name        VARCHAR(255) NOT NULL DEFAULT '',
    signer_title       VARCHAR(255) NOT NULL DEFAULT '',
    background_file_id UUID REFERENCES portal_files(id) ON DELETE SET NULL,
    signature_file_id  UUID REFERENCES portal_files(id) ON DELETE SET NULL,
    is_active          BOOLEAN NOT NULL DEFAULT FALSE,
    created_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- At most one active template per kind.
CREATE UNIQUE INDEX uq_document_templates_active ON document_templates(kind) WHERE is_active;

-- ── Issued letters (immutable snapshot of what was printed) ──

CREATE TABLE issued_letters (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id         UUID NOT NULL UNIQUE REFERENCES portal_requests(id) ON DELETE RESTRICT,
    template_id        UUID REFERENCES document_templates(id) ON DELETE SET NULL,
    letter_year        INTEGER NOT NULL,
    letter_seq         INTEGER NOT NULL,
    letter_no          VARCHAR(30) NOT NULL,
    issued_date        DATE NOT NULL DEFAULT CURRENT_DATE,
    recipient_name     VARCHAR(255) NOT NULL,
    college_name       VARCHAR(255) NOT NULL DEFAULT '',
    department_name    VARCHAR(255) NOT NULL DEFAULT '',
    header_line_1      VARCHAR(255) NOT NULL DEFAULT '',
    header_line_2      VARCHAR(255) NOT NULL DEFAULT '',
    title_text         VARCHAR(255) NOT NULL DEFAULT '',
    body_text          TEXT NOT NULL,
    closing_text       VARCHAR(255) NOT NULL DEFAULT '',
    signer_name        VARCHAR(255) NOT NULL DEFAULT '',
    signer_title       VARCHAR(255) NOT NULL DEFAULT '',
    background_file_id UUID REFERENCES portal_files(id) ON DELETE SET NULL,
    signature_file_id  UUID REFERENCES portal_files(id) ON DELETE SET NULL,
    issued_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (letter_year, letter_seq)
);

-- ── Default template so the portal works before a letterhead is uploaded ──

INSERT INTO document_templates
    (name, header_line_1, header_line_2, title_text, body_text, closing_text, signer_name, signer_title, is_active)
VALUES (
    'القالب الافتراضي',
    'جامعة التراث',
    'مكتب رئيس الجامعة',
    'كتاب شكر وتقدير',
    E'تقديرًا لجهودكم العلمية المتميزة ونشر بحثكم الموسوم:\n"{{research_title}}"\nفي مجلة ({{journal_name}}) المفهرسة في قاعدة {{index_db}}.\nنسأل الله لكم المزيد من العطاء وخدمة المسيرة العلمية في جامعتكم.',
    'مع التقدير',
    '',
    'رئيس جامعة التراث',
    TRUE
);
