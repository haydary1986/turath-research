-- ══════════════════════════════════════════════════
-- Migration 13: store every ORCID in its bare canonical form
-- ══════════════════════════════════════════════════

UPDATE researchers SET orcid_id = NULL WHERE btrim(orcid_id) = '';

-- Rows that do not reduce to a well-formed id stay as they are, for staff to correct.
-- When two rows reduce to one id only the first is rewritten: UNIQUE would otherwise
-- abort the deploy, and the handlers fold the leftover spelling when they compare.
WITH cleaned AS (
    SELECT id, orcid_id,
           upper(btrim(regexp_replace(
               regexp_replace(orcid_id, '[[:space:]‎‏‪-‮⁦-⁩﻿​]', '', 'g'),
               '^(https?://)?(www\.)?orcid\.org/', '', 'i'), '/')) AS canonical,
           created_at
    FROM researchers
    WHERE orcid_id IS NOT NULL
),
ranked AS (
    SELECT id, orcid_id, canonical,
           row_number() OVER (PARTITION BY canonical ORDER BY (orcid_id = canonical) DESC, created_at, id) AS position
    FROM cleaned
    WHERE canonical ~ '^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{3}[0-9X]$'
)
UPDATE researchers r
SET orcid_id = ranked.canonical
FROM ranked
WHERE r.id = ranked.id AND ranked.position = 1 AND ranked.orcid_id <> ranked.canonical;
