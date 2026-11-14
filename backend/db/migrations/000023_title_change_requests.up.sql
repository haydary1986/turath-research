-- A researcher may ask scientific affairs to change the title of a registered research.
ALTER TABLE portal_requests DROP CONSTRAINT portal_requests_kind_check;
ALTER TABLE portal_requests ADD CONSTRAINT portal_requests_kind_check
    CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter', 'title_change'));

ALTER TABLE form_templates DROP CONSTRAINT form_templates_kind_check;
ALTER TABLE form_templates ADD CONSTRAINT form_templates_kind_check
    CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter', 'title_change'));
