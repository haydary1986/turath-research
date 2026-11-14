-- Adds the ministry's Nadwati platform (seminars and conferences) to the site's quick links,
-- shown on the homepage. Appends only: existing links stay, nothing is added twice, and the
-- editor's limit of 8 links is respected.
UPDATE site_content
SET content = jsonb_set(
        content, '{links}',
        COALESCE(content->'links', '[]'::jsonb) || jsonb_build_array(jsonb_build_object(
            'label', 'منصة ندوتي — وزارة التعليم العالي والبحث العلمي',
            'label_en', 'Nadwati platform — Ministry of Higher Education and Scientific Research',
            'url', 'https://nadwati.rdd.edu.iq/'))),
    updated_at = NOW()
WHERE id = 1
  AND jsonb_array_length(COALESCE(content->'links', '[]'::jsonb)) < 8
  AND NOT EXISTS (
      SELECT 1 FROM jsonb_array_elements(COALESCE(content->'links', '[]'::jsonb)) AS link
      WHERE link->>'url' ILIKE '%nadwati.rdd.edu.iq%');
