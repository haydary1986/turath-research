-- ══════════════════════════════════════════════════
-- Migration 4: Notifications table + Seed data for جامعة التراث
-- ══════════════════════════════════════════════════

-- ── Notifications ────────────────────────────────

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_ar   VARCHAR(500) NOT NULL,
    title_en   VARCHAR(500),
    body       TEXT NOT NULL DEFAULT '',
    type       VARCHAR(50) NOT NULL DEFAULT 'info', -- info, success, warning, alert, promotion, incentive, deadline
    link       TEXT,
    is_read    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);

-- ══════════════════════════════════════════════════
-- Seed Data: جامعة التراث
-- ══════════════════════════════════════════════════

-- Institution
INSERT INTO institutions (id, name_ar, name_en, domain)
VALUES ('11111111-1111-1111-1111-111111111111', 'جامعة التراث', 'University of Heritage', 'turath.edu.iq')
ON CONFLICT (domain) DO NOTHING;

-- Promotion requirements for جامعة التراث
INSERT INTO promotion_requirements (id, institution_id, from_rank, to_rank, field_category, min_q1_publications, min_total_publications, min_h_index, min_years_in_rank) VALUES
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'lecturer', 'assistant_prof', 'all', 1, 3, 2, 3),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'assistant_prof', 'associate_prof', 'all', 3, 8, 5, 4),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'associate_prof', 'professor', 'all', 6, 15, 10, 5)
ON CONFLICT DO NOTHING;

-- Incentive rules
INSERT INTO incentive_rules (id, institution_id, journal_classification, base_amount, corresponding_bonus, effective_from) VALUES
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q1', 3000000, 500000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q2', 2000000, 300000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q3', 1000000, 200000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'scopus_only', 500000, 100000, '2024-01-01')
ON CONFLICT DO NOTHING;
