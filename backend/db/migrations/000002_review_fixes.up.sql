-- Fixes from code review

-- Add updated_at to users and institutions
ALTER TABLE users ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE institutions ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Add unique constraint on publication_authors to prevent duplicate positions
ALTER TABLE publication_authors
    ADD CONSTRAINT uq_pub_authors_order UNIQUE (publication_id, author_order);
