-- Director photos are portal files owned by the site document; remove them with the block.
DELETE FROM portal_files WHERE role = 'site_director_photo';

UPDATE site_content
SET content = (content - 'director' - 'colleges' - 'socials' - 'faq' - 'stats' - 'hide_publications')
    || jsonb_build_object('services', COALESCE((
        SELECT jsonb_agg(s - 'icon' - 'link') FROM jsonb_array_elements(content->'services') s), '[]'::jsonb))
WHERE id = 1;

DROP INDEX IF EXISTS idx_journal_reference_type_name;
DROP INDEX IF EXISTS idx_journal_reference_slug;
ALTER TABLE journal_reference_list DROP COLUMN IF EXISTS slug;
