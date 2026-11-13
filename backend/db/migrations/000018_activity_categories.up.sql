-- ══════════════════════════════════════════════════
-- Migration 18: every scientific-activity record belongs to one category of the
-- ministry form FS 003 «استمارة تقدم عمل الجامعة او الهيئة».
-- ══════════════════════════════════════════════════

ALTER TABLE activities ADD COLUMN category        VARCHAR(40);
-- The category's own form column: venue, journal, publisher, counter-party, country…
ALTER TABLE activities ADD COLUMN category_detail VARCHAR(500);
-- «العدد» of the infrastructure sheet.
ALTER TABLE activities ADD COLUMN item_count      INTEGER CHECK (item_count IS NULL OR item_count > 0);
-- «الملاحظات» of the form.
ALTER TABLE activities ADD COLUMN notes           TEXT;

-- Existing records: seminars, workshops, panels and lectures go to sheet 2, courses to
-- sheet 3, conferences to sheet 10, and "other" to community service (sheet 14), the
-- form's catch-all for activities outside the academic programme. Their venue is the
-- form column of all four sheets.
UPDATE activities SET
    category = CASE activity_type
        WHEN 'course'     THEN 'courses'
        WHEN 'conference' THEN 'conferences'
        WHEN 'other'      THEN 'community_service'
        ELSE 'workshops'
    END,
    category_detail = location;

ALTER TABLE activities ALTER COLUMN category SET NOT NULL;
ALTER TABLE activities ADD CONSTRAINT activities_category_check CHECK (category IN (
    'infrastructure', 'workshops', 'courses', 'local_research', 'indexed_research', 'international_research', 'books', 'patents', 'conferences', 'international_agreements', 'local_agreements', 'fellowships', 'community_service'
));
ALTER TABLE activities ADD CONSTRAINT activities_infrastructure_count_check
    CHECK (category <> 'infrastructure' OR item_count IS NOT NULL);

-- Institutional records (infrastructure, agreements) may belong to the whole university.
-- Personal ones still need a college; the API enforces that, as it did before.

CREATE INDEX idx_activities_category ON activities(category, college_id, activity_date DESC);
