ALTER TABLE publication_authors DROP CONSTRAINT IF EXISTS uq_pub_authors_order;
ALTER TABLE institutions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE users DROP COLUMN IF EXISTS updated_at;
