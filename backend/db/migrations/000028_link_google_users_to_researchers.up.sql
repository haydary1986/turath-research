-- Accounts that already signed in with Google (verified university email) are tied to the
-- imported researcher record with the same email, as new sign-ins now are: researcher,
-- department and its college, institution. Values already on an account are kept, and a
-- record linked to another account is left alone.
UPDATE users u SET
    researcher_id  = r.id,
    department_id  = COALESCE(u.department_id, r.department_id),
    college_id     = COALESCE(u.college_id, d.college_id),
    institution_id = COALESCE(u.institution_id, (SELECT id FROM institutions ORDER BY created_at LIMIT 1)),
    name_en        = COALESCE(NULLIF(btrim(u.name_en), ''), NULLIF(btrim(r.name_en), ''))
FROM researchers r
LEFT JOIN departments d ON d.id = r.department_id
WHERE u.google_sub IS NOT NULL
  AND u.researcher_id IS NULL
  AND lower(r.email) = lower(u.email)
  AND NOT EXISTS (SELECT 1 FROM users o WHERE o.researcher_id = r.id AND o.id <> u.id);
