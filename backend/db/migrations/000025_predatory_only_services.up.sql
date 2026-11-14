-- The homepage keeps one journal card, "Predatory journals" (explanation and links);
-- the journal verification and journal check cards are removed. Other cards, and their
-- order, are untouched.
UPDATE site_content
SET content = jsonb_set(content, '{services}', COALESCE((
        SELECT jsonb_agg(
                   CASE WHEN svc->>'link' = '/journals/predatory'
                        THEN svc || jsonb_build_object(
                            'title', 'المجلات المفترسة',
                            'title_en', 'Predatory journals',
                            'description', 'شرح مختصر للمجلات المفترسة والمختطفة وروابط القوائم المعتمدة.',
                            'description_en', 'A short explanation of predatory and hijacked journals, with links to the trusted lists.')
                        ELSE svc END
                   ORDER BY ord)
        FROM jsonb_array_elements(content->'services') WITH ORDINALITY AS t(svc, ord)
        WHERE COALESCE(svc->>'link', '') NOT IN ('/journals/check', '/portal/new/journal_verification')
    ), '[]'::jsonb)),
    updated_at = NOW()
WHERE id = 1 AND jsonb_typeof(content->'services') = 'array';
