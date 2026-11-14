-- Drops the English counterparts of the landing-page texts and the announcement language.

CREATE FUNCTION pg_temp.strip_keys(items JSONB, keys TEXT[]) RETURNS JSONB
LANGUAGE sql IMMUTABLE AS $fn$
    SELECT COALESCE(jsonb_agg(item - keys ORDER BY ord), '[]'::jsonb)
    FROM jsonb_array_elements(COALESCE(items, '[]'::jsonb)) WITH ORDINALITY AS t(item, ord)
$fn$;

UPDATE site_content SET content = (content - ARRAY['hero_title_en', 'hero_subtitle_en', 'about_en'])
    || jsonb_build_object(
        'services', pg_temp.strip_keys(content->'services', ARRAY['title_en', 'description_en']),
        'contact', COALESCE(content->'contact', '{}'::jsonb) - ARRAY['location_en', 'hours_en'],
        'links', pg_temp.strip_keys(content->'links', ARRAY['label_en']),
        'colleges', pg_temp.strip_keys(content->'colleges', ARRAY['label_en']),
        'stats', pg_temp.strip_keys(content->'stats', ARRAY['label_en']),
        'faq', pg_temp.strip_keys(content->'faq', ARRAY['question_en', 'answer_en']),
        'director', COALESCE(content->'director', '{}'::jsonb) - ARRAY['title_en', 'name_en', 'position_en', 'message_en'])
WHERE id = 1;

DROP FUNCTION pg_temp.strip_keys(JSONB, TEXT[]);

ALTER TABLE announcements DROP COLUMN IF EXISTS lang;
