-- ══════════════════════════════════════════════════
-- Migration 19: data-entry responsibilities («مسؤوليات الإدخال»).
-- A user is made responsible for one FS 003 category, either for the whole
-- university (college_id NULL) or for one college. Holders enter, see, edit and
-- export every record of the category in their scope; approval stays with
-- scientific affairs. The API reads this table on every request.
-- ══════════════════════════════════════════════════

CREATE TABLE entry_responsibilities (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   VARCHAR(40) NOT NULL CHECK (category IN (
        'infrastructure', 'workshops', 'courses', 'local_research', 'indexed_research', 'international_research', 'books', 'patents', 'conferences', 'international_agreements', 'local_agreements', 'fellowships', 'community_service'
    )),
    college_id UUID REFERENCES colleges(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_entry_responsibilities
    ON entry_responsibilities (user_id, category, college_id) NULLS NOT DISTINCT;
CREATE INDEX idx_entry_responsibilities_category ON entry_responsibilities (category, college_id);
