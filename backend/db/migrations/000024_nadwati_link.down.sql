-- Removes only the Nadwati link this migration added.
UPDATE site_content
SET content = jsonb_set(content, '{links}', COALESCE((
        SELECT jsonb_agg(link) FROM jsonb_array_elements(content->'links') AS link
        WHERE link->>'url' <> 'https://nadwati.rdd.edu.iq/'), '[]'::jsonb))
WHERE id = 1 AND content ? 'links';
