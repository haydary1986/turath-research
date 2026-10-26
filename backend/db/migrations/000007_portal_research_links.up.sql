-- Link portal requests to the researcher registry and to the publication
-- created when a journal verification completes.
ALTER TABLE portal_requests ADD COLUMN researcher_id UUID REFERENCES researchers(id) ON DELETE SET NULL;
ALTER TABLE portal_requests ADD COLUMN publication_id UUID REFERENCES publications(id) ON DELETE SET NULL;

CREATE INDEX idx_portal_requests_researcher ON portal_requests(researcher_id);
