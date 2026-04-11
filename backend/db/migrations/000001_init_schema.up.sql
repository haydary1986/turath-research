-- ══════════════════════════════════════════════════
-- Turath Research — Initial Schema
-- ══════════════════════════════════════════════════

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ── Custom Types ─────────────────────────────────

CREATE TYPE researcher_status AS ENUM ('active', 'inactive', 'emeritus', 'visiting');
CREATE TYPE publication_type AS ENUM ('journal_article', 'conference_paper', 'book_chapter', 'book', 'preprint', 'thesis', 'patent', 'report');
CREATE TYPE publication_status AS ENUM ('published', 'accepted', 'in_press', 'submitted', 'preprint');
CREATE TYPE journal_ranking AS ENUM ('Q1', 'Q2', 'Q3', 'Q4', 'unranked');
CREATE TYPE user_role AS ENUM ('admin', 'institution_admin', 'college_admin', 'department_admin', 'researcher', 'viewer');
CREATE TYPE preferred_language AS ENUM ('ar', 'en');
CREATE TYPE metric_source AS ENUM ('scopus', 'crossref', 'google_scholar', 'openalex', 'internal', 'calculated');
CREATE TYPE sync_job_type AS ENUM ('scopus_sync', 'crossref_sync', 'scholar_sync', 'orcid_sync', 'openalex_sync', 'metrics_update');
CREATE TYPE sync_target_type AS ENUM ('researcher', 'publication', 'journal');
CREATE TYPE sync_status AS ENUM ('queued', 'running', 'completed', 'failed', 'retrying');
CREATE TYPE analysis_type AS ENUM ('collaboration_rec', 'trend_analysis', 'publication_strategy', 'gap_analysis', 'funding_match');
CREATE TYPE analysis_target AS ENUM ('researcher', 'department', 'college', 'institution');
CREATE TYPE analysis_status AS ENUM ('pending', 'processing', 'completed', 'failed');

-- ── Institutions ─────────────────────────────────

CREATE TABLE institutions (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_ar    VARCHAR(255) NOT NULL,
    name_en    VARCHAR(255),
    domain     VARCHAR(255) UNIQUE,
    logo_url   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Colleges ─────────────────────────────────────

CREATE TABLE colleges (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_ar             VARCHAR(255) NOT NULL,
    name_en             VARCHAR(255),
    code                VARCHAR(50) UNIQUE,
    institution_id      UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    dean_researcher_id  UUID,  -- FK added after researchers table
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Departments ──────────────────────────────────

CREATE TABLE departments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_ar             VARCHAR(255) NOT NULL,
    name_en             VARCHAR(255),
    code                VARCHAR(50) UNIQUE,
    college_id          UUID NOT NULL REFERENCES colleges(id) ON DELETE CASCADE,
    head_researcher_id  UUID,  -- FK added after researchers table
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Researchers ──────────────────────────────────

CREATE TABLE researchers (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    orcid_id              VARCHAR(50) UNIQUE,
    scopus_author_id      VARCHAR(50) UNIQUE,
    google_scholar_id     VARCHAR(50) UNIQUE,
    name_ar               VARCHAR(255) NOT NULL,
    name_en               VARCHAR(255),
    email                 VARCHAR(255) UNIQUE NOT NULL,
    title_ar              VARCHAR(100),
    title_en              VARCHAR(100),
    department_id         UUID REFERENCES departments(id) ON DELETE SET NULL,
    h_index               INTEGER DEFAULT 0,
    h_index_updated_at    TIMESTAMPTZ,
    total_citations       INTEGER DEFAULT 0,
    total_publications    INTEGER DEFAULT 0,
    avatar_url            TEXT,
    bio_ar                TEXT,
    bio_en                TEXT,
    research_interests    JSONB DEFAULT '[]',
    status                researcher_status DEFAULT 'active',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add deferred FKs for dean/head
ALTER TABLE colleges ADD CONSTRAINT fk_colleges_dean
    FOREIGN KEY (dean_researcher_id) REFERENCES researchers(id) ON DELETE SET NULL;

ALTER TABLE departments ADD CONSTRAINT fk_departments_head
    FOREIGN KEY (head_researcher_id) REFERENCES researchers(id) ON DELETE SET NULL;

-- ── Journals ─────────────────────────────────────

CREATE TABLE journals (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issn               VARCHAR(20) UNIQUE,
    eissn              VARCHAR(20) UNIQUE,
    name               VARCHAR(500) NOT NULL,
    publisher          VARCHAR(255),
    country            VARCHAR(100),
    language           VARCHAR(10),
    sjr_ranking        journal_ranking DEFAULT 'unranked',
    impact_factor      DECIMAL(8,3),
    impact_factor_year INTEGER,
    subject_areas      JSONB DEFAULT '[]',
    is_open_access     BOOLEAN DEFAULT FALSE,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Research Fields ──────────────────────────────

CREATE TABLE research_fields (
    id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_ar   VARCHAR(255) NOT NULL,
    name_en   VARCHAR(255),
    parent_id UUID REFERENCES research_fields(id) ON DELETE SET NULL,
    asjc_code VARCHAR(20)
);

-- ── Publications ─────────────────────────────────

CREATE TABLE publications (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    doi                       VARCHAR(255) UNIQUE,
    scopus_id                 VARCHAR(50) UNIQUE,
    title_ar                  TEXT,
    title_en                  TEXT,
    abstract_ar               TEXT,
    abstract_en               TEXT,
    publication_date          DATE,
    publication_type          publication_type DEFAULT 'journal_article',
    journal_id                UUID REFERENCES journals(id) ON DELETE SET NULL,
    conference_name           VARCHAR(500),
    volume                    VARCHAR(50),
    issue                     VARCHAR(50),
    pages                     VARCHAR(50),
    url                       TEXT,
    pdf_url                   TEXT,
    language                  VARCHAR(10) DEFAULT 'ar',
    status                    publication_status DEFAULT 'published',
    citation_count            INTEGER DEFAULT 0,
    citation_count_updated_at TIMESTAMPTZ,
    metadata                  JSONB DEFAULT '{}',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Publication Authors ──────────────────────────

CREATE TABLE publication_authors (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    publication_id    UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    researcher_id     UUID REFERENCES researchers(id) ON DELETE SET NULL,
    author_name       VARCHAR(255) NOT NULL,
    author_order      INTEGER NOT NULL,
    is_corresponding  BOOLEAN DEFAULT FALSE,
    affiliation_text  TEXT
);

-- ── Publication Fields ───────────────────────────

CREATE TABLE publication_fields (
    publication_id UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    field_id       UUID NOT NULL REFERENCES research_fields(id) ON DELETE CASCADE,
    PRIMARY KEY (publication_id, field_id)
);

-- ── Publication Metrics (time-series) ────────────

CREATE TABLE publication_metrics (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    publication_id  UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    recorded_at     DATE NOT NULL,
    citation_count  INTEGER DEFAULT 0,
    download_count  INTEGER DEFAULT 0,
    view_count      INTEGER DEFAULT 0,
    altmetric_score DECIMAL(10,2) DEFAULT 0,
    source          metric_source NOT NULL,
    UNIQUE (publication_id, recorded_at, source)
);

-- ── Researcher Metrics (time-series) ─────────────

CREATE TABLE researcher_metrics (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id       UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    recorded_at         DATE NOT NULL,
    h_index             INTEGER DEFAULT 0,
    total_citations     INTEGER DEFAULT 0,
    total_publications  INTEGER DEFAULT 0,
    i10_index           INTEGER DEFAULT 0,
    source              metric_source NOT NULL,
    UNIQUE (researcher_id, recorded_at, source)
);

-- ── Department Metrics (monthly aggregate) ───────

CREATE TABLE department_metrics_monthly (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id       UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    year                INTEGER NOT NULL,
    month               INTEGER NOT NULL,
    total_publications  INTEGER DEFAULT 0,
    total_citations     INTEGER DEFAULT 0,
    avg_h_index         DECIMAL(6,2) DEFAULT 0,
    q1_publications     INTEGER DEFAULT 0,
    q2_publications     INTEGER DEFAULT 0,
    open_access_count   INTEGER DEFAULT 0,
    UNIQUE (department_id, year, month)
);

-- ── Users ────────────────────────────────────────

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email               VARCHAR(255) UNIQUE NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    name_ar             VARCHAR(255),
    name_en             VARCHAR(255),
    role                user_role DEFAULT 'viewer',
    researcher_id       UUID REFERENCES researchers(id) ON DELETE SET NULL,
    institution_id      UUID REFERENCES institutions(id) ON DELETE SET NULL,
    preferred_language  preferred_language DEFAULT 'ar',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Sync Jobs ────────────────────────────────────

CREATE TABLE sync_jobs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_type      sync_job_type NOT NULL,
    target_type   sync_target_type NOT NULL,
    target_id     UUID NOT NULL,
    status        sync_status DEFAULT 'queued',
    attempts      INTEGER DEFAULT 0,
    max_attempts  INTEGER DEFAULT 3,
    error_message TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── AI Analyses ──────────────────────────────────

CREATE TABLE ai_analyses (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_type    analysis_type NOT NULL,
    target_type      analysis_target NOT NULL,
    target_id        UUID NOT NULL,
    input_data       JSONB DEFAULT '{}',
    result           JSONB DEFAULT '{}',
    model_used       VARCHAR(100),
    tokens_used      INTEGER DEFAULT 0,
    confidence_score DECIMAL(4,3),
    status           analysis_status DEFAULT 'pending',
    requested_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at       TIMESTAMPTZ
);

-- ══════════════════════════════════════════════════
-- Indexes
-- ══════════════════════════════════════════════════

-- Researchers
CREATE INDEX idx_researchers_department ON researchers(department_id);
CREATE INDEX idx_researchers_status ON researchers(status);
CREATE INDEX idx_researchers_name_trgm ON researchers USING GIN (name_ar gin_trgm_ops);

-- Publications
CREATE INDEX idx_publications_date ON publications(publication_date DESC);
CREATE INDEX idx_publications_type ON publications(publication_type);
CREATE INDEX idx_publications_journal ON publications(journal_id);
CREATE INDEX idx_publications_title_trgm ON publications USING GIN (
    (COALESCE(title_ar, '') || ' ' || COALESCE(title_en, '')) gin_trgm_ops
);

-- Publication authors
CREATE INDEX idx_pub_authors_publication ON publication_authors(publication_id);
CREATE INDEX idx_pub_authors_researcher ON publication_authors(researcher_id);

-- Journals
CREATE INDEX idx_journals_ranking ON journals(sjr_ranking);

-- Metrics
CREATE INDEX idx_pub_metrics_pub_date ON publication_metrics(publication_id, recorded_at DESC);
CREATE INDEX idx_researcher_metrics_date ON researcher_metrics(researcher_id, recorded_at DESC);

-- Sync jobs
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status, created_at);

-- AI analyses
CREATE INDEX idx_ai_analyses_target ON ai_analyses(target_type, target_id, analysis_type);
CREATE INDEX idx_ai_analyses_status ON ai_analyses(status);

-- Users
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_institution ON users(institution_id);
