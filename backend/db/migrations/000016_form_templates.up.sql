-- Official blank forms (PDF) that applicants download, sign and attach, one per request kind.
CREATE TABLE form_templates (
    kind        VARCHAR(30) PRIMARY KEY
        CHECK (kind IN ('research_plan', 'journal_verification', 'appreciation_letter')),
    file_id     UUID NOT NULL REFERENCES portal_files(id) ON DELETE CASCADE,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
