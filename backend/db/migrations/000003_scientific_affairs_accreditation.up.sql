-- ══════════════════════════════════════════════════
-- Migration 3: Scientific Affairs, Accreditation, Workflows
-- New roles, promotion system, incentives, performance,
-- sabbaticals, conferences, agreements, chairs, centers,
-- accreditation evidence, KPIs, approval workflows
-- ══════════════════════════════════════════════════

-- ── Drop old user_role and recreate with new roles ──

ALTER TABLE users ALTER COLUMN role DROP DEFAULT;
ALTER TABLE users ALTER COLUMN role TYPE VARCHAR(50);
DROP TYPE IF EXISTS user_role;

CREATE TYPE user_role AS ENUM (
    'admin', 'vp_research', 'dean_research', 'scientific_affairs',
    'college_dean', 'center_director', 'department_head',
    'researcher', 'viewer'
);

ALTER TABLE users ALTER COLUMN role TYPE user_role USING role::user_role;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'viewer';

-- Add new columns to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS college_id UUID REFERENCES colleges(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS can_approve_promotions BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS can_approve_budgets BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_college ON users(college_id);
CREATE INDEX IF NOT EXISTS idx_users_department ON users(department_id);

-- ── Custom types for new tables ─────────────────

CREATE TYPE academic_rank AS ENUM ('lecturer', 'assistant_prof', 'associate_prof', 'professor');
CREATE TYPE field_category AS ENUM ('sciences', 'engineering', 'humanities', 'medical', 'all');

CREATE TYPE promotion_status AS ENUM (
    'draft', 'submitted', 'dept_review', 'college_review',
    'scientific_affairs_review', 'vp_review', 'council_review',
    'approved', 'rejected', 'returned'
);

CREATE TYPE sabbatical_status AS ENUM (
    'draft', 'submitted', 'dept_review', 'college_review',
    'scientific_affairs_review', 'vp_review',
    'approved', 'rejected', 'active', 'completed'
);

CREATE TYPE conference_status AS ENUM (
    'draft', 'submitted', 'dept_review', 'college_review',
    'approved', 'rejected', 'attended', 'report_pending', 'completed'
);

CREATE TYPE participation_type AS ENUM ('presenter', 'keynote', 'attendee', 'organizer', 'chair');

CREATE TYPE journal_classification AS ENUM ('Q1', 'Q2', 'Q3', 'Q4', 'scopus_only', 'other');

CREATE TYPE incentive_status AS ENUM (
    'pending_review', 'eligible', 'ineligible', 'approved', 'paid', 'rejected'
);

CREATE TYPE performance_status AS ENUM ('auto_generated', 'dept_reviewed', 'college_reviewed', 'finalized');

CREATE TYPE agreement_type AS ENUM (
    'mou', 'research_collaboration', 'student_exchange',
    'joint_supervision', 'industrial_partnership', 'funding_agreement'
);

CREATE TYPE agreement_status AS ENUM (
    'draft', 'legal_review', 'pending_signature',
    'active', 'expiring_soon', 'expired', 'terminated'
);

CREATE TYPE chair_status AS ENUM ('active', 'vacant', 'expired', 'suspended');
CREATE TYPE center_status AS ENUM ('active', 'under_establishment', 'suspended', 'closed');
CREATE TYPE center_role AS ENUM ('director', 'co_director', 'principal_investigator', 'researcher', 'affiliate');

CREATE TYPE accreditation_body AS ENUM ('ncaaa', 'abet', 'aacsb', 'qs', 'the', 'arwu', 'custom');
CREATE TYPE evidence_type AS ENUM ('report', 'data', 'screenshot', 'document', 'link', 'auto');
CREATE TYPE evidence_status AS ENUM ('draft', 'submitted', 'accepted', 'needs_update');
CREATE TYPE recommendation_type AS ENUM ('observation', 'recommendation', 'weakness', 'strength');
CREATE TYPE recommendation_status AS ENUM ('open', 'in_progress', 'completed', 'overdue', 'closed');

CREATE TYPE kpi_category AS ENUM (
    'research_output', 'research_quality', 'funding', 'collaboration',
    'innovation', 'community_impact', 'student_research', 'sdg'
);
CREATE TYPE kpi_period AS ENUM ('monthly', 'quarterly', 'semester', 'annual');
CREATE TYPE kpi_source AS ENUM ('auto_calculated', 'manual', 'mixed');
CREATE TYPE kpi_scope AS ENUM ('institution', 'college', 'department');

CREATE TYPE workflow_type AS ENUM (
    'promotion', 'sabbatical', 'conference', 'incentive',
    'agreement', 'research_project'
);
CREATE TYPE approver_role AS ENUM (
    'department_head', 'college_dean', 'scientific_affairs',
    'dean_research', 'vp_research', 'council'
);
CREATE TYPE workflow_decision AS ENUM ('pending', 'approved', 'rejected', 'returned');

CREATE TYPE performance_criterion AS ENUM (
    'q1_pubs', 'q2_pubs', 'other_pubs', 'citations',
    'h_index', 'funding', 'supervision', 'conferences',
    'community', 'committees'
);

-- ══════════════════════════════════════════════════
-- Promotion System
-- ══════════════════════════════════════════════════

CREATE TABLE promotion_requirements (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id         UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    from_rank              academic_rank NOT NULL,
    to_rank                academic_rank NOT NULL,
    field_category         field_category DEFAULT 'all',
    min_q1_publications    INTEGER DEFAULT 0,
    min_total_publications INTEGER DEFAULT 0,
    min_h_index            INTEGER DEFAULT 0,
    min_years_in_rank      INTEGER DEFAULT 0,
    min_supervised_phd     INTEGER DEFAULT 0,
    min_supervised_masters INTEGER DEFAULT 0,
    requires_book          BOOLEAN DEFAULT FALSE,
    min_external_funding   DECIMAL(15,2) DEFAULT 0,
    additional_criteria    JSONB DEFAULT '{}',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, from_rank, to_rank, field_category)
);

CREATE TABLE promotion_applications (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id      UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    current_rank       academic_rank NOT NULL,
    target_rank        academic_rank NOT NULL,
    application_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    q1_publications    INTEGER DEFAULT 0,
    q2_publications    INTEGER DEFAULT 0,
    total_publications INTEGER DEFAULT 0,
    h_index_at_apply   INTEGER DEFAULT 0,
    supervised_phd     INTEGER DEFAULT 0,
    supervised_masters INTEGER DEFAULT 0,
    external_funding   DECIMAL(15,2) DEFAULT 0,
    has_book           BOOLEAN DEFAULT FALSE,
    years_in_rank      INTEGER DEFAULT 0,
    eligibility_score  DECIMAL(5,2) DEFAULT 0,
    status             promotion_status DEFAULT 'draft',
    rejection_reason   TEXT,
    documents_url      TEXT,
    submitted_at       TIMESTAMPTZ,
    decided_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_promotion_apps_researcher ON promotion_applications(researcher_id);
CREATE INDEX idx_promotion_apps_status ON promotion_applications(status);

-- ══════════════════════════════════════════════════
-- Publication Incentives
-- ══════════════════════════════════════════════════

CREATE TABLE incentive_rules (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id              UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    journal_classification      journal_classification NOT NULL,
    base_amount                 DECIMAL(10,2) NOT NULL,
    corresponding_bonus         DECIMAL(10,2) DEFAULT 0,
    international_collab_bonus  DECIMAL(10,2) DEFAULT 0,
    max_authors_share           INTEGER DEFAULT 5,
    effective_from              DATE NOT NULL,
    effective_to                DATE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE publication_incentives (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    publication_id          UUID NOT NULL REFERENCES publications(id) ON DELETE CASCADE,
    researcher_id           UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    journal_classification  journal_classification NOT NULL,
    incentive_amount        DECIMAL(10,2) NOT NULL,
    is_corresponding        BOOLEAN DEFAULT FALSE,
    author_share            DECIMAL(5,2) DEFAULT 100,
    status                  incentive_status DEFAULT 'pending_review',
    review_notes            TEXT,
    reviewed_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    payment_batch           VARCHAR(100),
    paid_at                 TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incentives_researcher ON publication_incentives(researcher_id);
CREATE INDEX idx_incentives_status ON publication_incentives(status);
CREATE INDEX idx_incentives_publication ON publication_incentives(publication_id);

-- ══════════════════════════════════════════════════
-- Annual Performance Reviews
-- ══════════════════════════════════════════════════

CREATE TABLE performance_criteria_weights (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id    UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    criterion         performance_criterion NOT NULL,
    weight_percentage DECIMAL(5,2) NOT NULL,
    effective_year    INTEGER NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (institution_id, criterion, effective_year)
);

CREATE TABLE annual_performance_reviews (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id         UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    review_year           INTEGER NOT NULL,
    q1_publications       INTEGER DEFAULT 0,
    q2_publications       INTEGER DEFAULT 0,
    total_publications    INTEGER DEFAULT 0,
    citations_this_year   INTEGER DEFAULT 0,
    h_index_current       INTEGER DEFAULT 0,
    funding_acquired      DECIMAL(15,2) DEFAULT 0,
    students_supervised   INTEGER DEFAULT 0,
    conferences_attended  INTEGER DEFAULT 0,
    community_service     INTEGER DEFAULT 0,
    committees_membership INTEGER DEFAULT 0,
    weighted_score        DECIMAL(6,2) DEFAULT 0,
    rank_in_department    INTEGER,
    rank_in_college       INTEGER,
    reviewer_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewer_notes        TEXT,
    dean_notes            TEXT,
    status                performance_status DEFAULT 'auto_generated',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (researcher_id, review_year)
);

CREATE INDEX idx_perf_reviews_researcher ON annual_performance_reviews(researcher_id);
CREATE INDEX idx_perf_reviews_year ON annual_performance_reviews(review_year);

-- ══════════════════════════════════════════════════
-- Sabbatical Requests
-- ══════════════════════════════════════════════════

CREATE TABLE sabbatical_requests (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id         UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    destination           VARCHAR(500),
    destination_country   VARCHAR(100),
    research_plan         TEXT,
    start_date            DATE,
    end_date              DATE,
    status                sabbatical_status DEFAULT 'draft',
    outputs_report        TEXT,
    publications_during   INTEGER DEFAULT 0,
    productivity_before   DECIMAL(6,2) DEFAULT 0,
    productivity_during   DECIMAL(6,2) DEFAULT 0,
    productivity_after    DECIMAL(6,2) DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sabbaticals_researcher ON sabbatical_requests(researcher_id);
CREATE INDEX idx_sabbaticals_status ON sabbatical_requests(status);

-- ══════════════════════════════════════════════════
-- Conference Participations
-- ══════════════════════════════════════════════════

CREATE TABLE conference_participations (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    researcher_id           UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    conference_name         VARCHAR(500) NOT NULL,
    conference_location     VARCHAR(255),
    conference_country      VARCHAR(100),
    conference_date_start   DATE,
    conference_date_end     DATE,
    conference_url          TEXT,
    paper_title             TEXT,
    publication_id          UUID REFERENCES publications(id) ON DELETE SET NULL,
    participation_type      participation_type DEFAULT 'presenter',
    is_international        BOOLEAN DEFAULT TRUE,
    estimated_cost          DECIMAL(10,2) DEFAULT 0,
    approved_budget         DECIMAL(10,2) DEFAULT 0,
    status                  conference_status DEFAULT 'draft',
    post_report             TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conferences_researcher ON conference_participations(researcher_id);
CREATE INDEX idx_conferences_status ON conference_participations(status);

-- ══════════════════════════════════════════════════
-- Agreements & Partnerships
-- ══════════════════════════════════════════════════

CREATE TABLE agreements (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id          UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    partner_name_ar         VARCHAR(500),
    partner_name_en         VARCHAR(500),
    partner_country         VARCHAR(100),
    agreement_type          agreement_type NOT NULL,
    title_ar                VARCHAR(500),
    title_en                VARCHAR(500),
    description             TEXT,
    start_date              DATE,
    end_date                DATE,
    auto_renew              BOOLEAN DEFAULT FALSE,
    status                  agreement_status DEFAULT 'draft',
    responsible_id          UUID REFERENCES researchers(id) ON DELETE SET NULL,
    outputs_count           INTEGER DEFAULT 0,
    document_url            TEXT,
    renewal_reminder_days   INTEGER DEFAULT 90,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agreements_institution ON agreements(institution_id);
CREATE INDEX idx_agreements_status ON agreements(status);

-- ══════════════════════════════════════════════════
-- Research Chairs
-- ══════════════════════════════════════════════════

CREATE TABLE research_chairs (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id     UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    name_ar            VARCHAR(500) NOT NULL,
    name_en            VARCHAR(500),
    sponsor_name       VARCHAR(500),
    chair_holder_id    UUID REFERENCES researchers(id) ON DELETE SET NULL,
    department_id      UUID REFERENCES departments(id) ON DELETE SET NULL,
    annual_budget      DECIMAL(15,2) DEFAULT 0,
    start_date         DATE,
    end_date           DATE,
    status             chair_status DEFAULT 'active',
    total_publications INTEGER DEFAULT 0,
    total_patents      INTEGER DEFAULT 0,
    total_funding      DECIMAL(15,2) DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ══════════════════════════════════════════════════
-- Research Centers
-- ══════════════════════════════════════════════════

CREATE TABLE research_centers (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id    UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    name_ar           VARCHAR(500) NOT NULL,
    name_en           VARCHAR(500),
    director_id       UUID REFERENCES researchers(id) ON DELETE SET NULL,
    focus_areas       JSONB DEFAULT '[]',
    annual_budget     DECIMAL(15,2) DEFAULT 0,
    status            center_status DEFAULT 'active',
    established_date  DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE research_center_members (
    center_id     UUID NOT NULL REFERENCES research_centers(id) ON DELETE CASCADE,
    researcher_id UUID NOT NULL REFERENCES researchers(id) ON DELETE CASCADE,
    role          center_role DEFAULT 'researcher',
    joined_at     DATE DEFAULT CURRENT_DATE,
    PRIMARY KEY (center_id, researcher_id)
);

-- ══════════════════════════════════════════════════
-- Accreditation & Quality Assurance
-- ══════════════════════════════════════════════════

CREATE TABLE accreditation_evidence (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id        UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    accreditation_body    accreditation_body NOT NULL,
    standard_code         VARCHAR(50) NOT NULL,
    standard_name_ar      VARCHAR(500),
    standard_name_en      VARCHAR(500),
    evidence_type         evidence_type NOT NULL,
    evidence_description  TEXT,
    evidence_url          TEXT,
    auto_generated        BOOLEAN DEFAULT FALSE,
    college_id            UUID REFERENCES colleges(id) ON DELETE SET NULL,
    cycle_year            INTEGER,
    status                evidence_status DEFAULT 'draft',
    uploaded_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accred_evidence_body ON accreditation_evidence(accreditation_body, standard_code);
CREATE INDEX idx_accred_evidence_college ON accreditation_evidence(college_id);

CREATE TABLE accreditation_recommendations (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id          UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    accreditation_body      accreditation_body NOT NULL,
    college_id              UUID REFERENCES colleges(id) ON DELETE SET NULL,
    visit_date              DATE,
    recommendation_type     recommendation_type NOT NULL,
    recommendation_text     TEXT NOT NULL,
    improvement_plan        TEXT,
    responsible_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    deadline                DATE,
    completion_percentage   INTEGER DEFAULT 0,
    evidence_of_action      TEXT,
    status                  recommendation_status DEFAULT 'open',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accred_recs_status ON accreditation_recommendations(status);

-- ══════════════════════════════════════════════════
-- Strategic KPIs
-- ══════════════════════════════════════════════════

CREATE TABLE strategic_kpis (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id        UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    kpi_name_ar           VARCHAR(500) NOT NULL,
    kpi_name_en           VARCHAR(500),
    category              kpi_category NOT NULL,
    target_value          DECIMAL(15,2),
    minimum_value         DECIMAL(15,2),
    excellence_value      DECIMAL(15,2),
    actual_value          DECIMAL(15,2),
    unit                  VARCHAR(50),
    measurement_period    kpi_period DEFAULT 'annual',
    strategic_plan_ref    VARCHAR(100),
    data_source           kpi_source DEFAULT 'auto_calculated',
    scope_type            kpi_scope DEFAULT 'institution',
    scope_id              UUID,
    year                  INTEGER NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kpis_institution_year ON strategic_kpis(institution_id, year);
CREATE INDEX idx_kpis_category ON strategic_kpis(category);

-- ══════════════════════════════════════════════════
-- Approval Workflows
-- ══════════════════════════════════════════════════

CREATE TABLE approval_workflows (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_type  workflow_type NOT NULL,
    request_id     UUID NOT NULL,
    step_order     INTEGER NOT NULL,
    step_name      VARCHAR(255) NOT NULL,
    approver_role  approver_role NOT NULL,
    approver_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    decision       workflow_decision DEFAULT 'pending',
    comments       TEXT,
    decided_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_request ON approval_workflows(workflow_type, request_id);
CREATE INDEX idx_workflows_approver ON approval_workflows(approver_id, decision);
CREATE INDEX idx_workflows_pending ON approval_workflows(decision) WHERE decision = 'pending';
