-- Restores the two removed journal cards (appended at the end).
UPDATE site_content
SET content = jsonb_set(content, '{services}', COALESCE(content->'services', '[]'::jsonb) || jsonb_build_array(
        jsonb_build_object('icon', 'verify', 'link', '/portal/new/journal_verification',
            'title', 'تدقيق رصانة المجلات', 'title_en', 'Journal verification',
            'description', 'التحقق من المجلة التي نُشر فيها البحث قبل اعتماده.',
            'description_en', 'Verify the journal a paper was published in before the paper is approved.'),
        jsonb_build_object('icon', 'search', 'link', '/journals/check',
            'title', 'فحص المجلات', 'title_en', 'Journal check',
            'description', 'فحص المجلة قبل النشر والتنبيه على المجلات غير الرصينة.',
            'description_en', 'Check a journal before publishing and get warned about journals that are not reputable.')))
WHERE id = 1;
