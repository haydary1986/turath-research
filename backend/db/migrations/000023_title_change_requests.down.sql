-- NOT VALID keeps existing title-change rows (and their history) instead of deleting them;
-- only new rows are held to the narrower list.
ALTER TABLE form_templates DROP CONSTRAINT form_templates_kind_check;
ALTER TABLE form_templates ADD CONSTRAINT form_templates_kind_check
    CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter')) NOT VALID;

ALTER TABLE portal_requests DROP CONSTRAINT portal_requests_kind_check;
ALTER TABLE portal_requests ADD CONSTRAINT portal_requests_kind_check
    CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter')) NOT VALID;
