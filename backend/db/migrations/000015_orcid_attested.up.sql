-- Set when an administrator imports an ORCID from an institutional source; a
-- researcher's own later change clears it. Lets a university-affiliated record pass
-- when Arabic and Latin spellings of the name fail the automatic comparison.
ALTER TABLE researchers ADD COLUMN orcid_attested BOOLEAN NOT NULL DEFAULT FALSE;
