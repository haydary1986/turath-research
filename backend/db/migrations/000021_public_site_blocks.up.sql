-- Public site, second pass: new landing-page blocks (statistics overrides,
-- director's message, FAQ, college sites, social links), service icons/links,
-- the «التعاميم» wording, and stable slugs for the public journal-list pages.

-- ── Reference-list slugs ──────────────────────────
-- The refresher replaces every row of a list daily (new ids each time), so the
-- public URL of an entry is derived from what the entry *is*: a readable part
-- from the name plus a short hash of list type and listed name. Alias rows (one
-- per name key) and a hijacked journal's alternate fake site share the slug, so
-- each listed journal or publisher has one page. The same entry keeps its slug
-- across refreshes; a removed entry's slug stops resolving.
ALTER TABLE journal_reference_list
    ADD COLUMN slug TEXT GENERATED ALWAYS AS (
        trim(BOTH '-' FROM
            left(trim(BOTH '-' FROM regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')), 80)
            || '-' || left(md5(list_type || '|' || lower(name)), 10))
    ) STORED;

CREATE INDEX idx_journal_reference_slug ON journal_reference_list (slug);
CREATE INDEX idx_journal_reference_type_name ON journal_reference_list (list_type, lower(name));

-- ── Landing-page document ─────────────────────────
UPDATE site_content SET content = content
    || jsonb_build_object(
        'about', replace(replace(content->>'about', 'الإعلانات والاعمامات', 'الإعلانات والتعاميم'), 'الاعمامات', 'التعاميم'),
        'services', $json$[
            {"title": "خطط البحث العلمي", "description": "تقديم خطة البحث ومتابعة مصادقتها عبر سلسلة الموافقات.", "icon": "research", "link": "/portal/new/research_plan"},
            {"title": "تدقيق رصانة المجلات", "description": "التحقق من المجلة التي نُشر فيها البحث قبل اعتماده.", "icon": "verify", "link": "/portal/new/journal_verification"},
            {"title": "كتب الشكر والتقدير", "description": "طلب كتاب الشكر والتقدير عن البحوث المنشورة بعد تدقيقها.", "icon": "award", "link": "/portal/new/appreciation_letter"},
            {"title": "النشاطات العلمية", "description": "تسجيل الندوات والورش والمحاضرات ومتابعة الموافقة عليها.", "icon": "calendar", "link": "/portal/activities"},
            {"title": "فحص المجلات", "description": "فحص المجلة قبل النشر والتنبيه على المجلات غير الرصينة.", "icon": "search", "link": "/journals/check"},
            {"title": "دليل المجلات المفترسة", "description": "قوائم المجلات والناشرين المفترسين والمجلات المختطفة وعلامات التحذير.", "icon": "shield", "link": "/journals/predatory"}
        ]$json$::jsonb,
        'director', jsonb_build_object(
            'title', 'كلمة مدير قسم الشؤون العلمية', 'name', '', 'position', '', 'message', '', 'photo_id', ''),
        'colleges', $json$[
            {"label": "كلية طب الأسنان", "url": "https://tucod.uoturath.edu.iq"},
            {"label": "كلية الصيدلة", "url": "https://tucop.uoturath.edu.iq"},
            {"label": "كلية التمريض", "url": "https://tucon.uoturath.edu.iq"},
            {"label": "كلية التقنيات الصحية والطبية", "url": "https://tucomht.uoturath.edu.iq"},
            {"label": "كلية الهندسة", "url": "https://tucoe.uoturath.edu.iq"},
            {"label": "الكلية التقنية الهندسية", "url": "https://tucoet.uoturath.edu.iq"},
            {"label": "كلية العلوم", "url": "https://tucos.uoturath.edu.iq"},
            {"label": "كلية القانون", "url": "https://tucol.uoturath.edu.iq"},
            {"label": "كلية الآداب", "url": "https://tucoli.uoturath.edu.iq"},
            {"label": "كلية الفنون الجميلة", "url": "https://tucoa.uoturath.edu.iq"},
            {"label": "كلية الإدارة والاقتصاد", "url": "https://tucomae.uoturath.edu.iq"},
            {"label": "كلية التربية", "url": "https://tucoedu.uoturath.edu.iq"},
            {"label": "كلية التربية البدنية وعلوم الرياضة", "url": "https://tucope.uoturath.edu.iq"},
            {"label": "كلية الإعلام", "url": "https://tucom.uoturath.edu.iq"}
        ]$json$::jsonb,
        'socials', '[]'::jsonb,
        'faq', $json$[
            {"question": "لماذا يفشل تسجيل الدخول إلى البوابة؟", "answer": "يكون الدخول بحساب Google Workspace الجامعي فقط، أي البريد الذي ينتهي بـ @uoturath.edu.iq. إذا كان المتصفح مسجّلًا بحساب Gmail شخصي فاختر الحساب الجامعي من نافذة Google أو سجّل الخروج من الحساب الشخصي ثم أعد المحاولة. إذا استمرت المشكلة فتواصل مع قسم الشؤون العلمية."},
            {"question": "كيف أضيف معرّف ORCID إلى ملفي؟", "answer": "بعد تسجيل الدخول افتح صفحة «ملفي البحثي» وأدخل معرّف ORCID بصيغته الكاملة (مثل 0000-0002-1825-0097) ثم احفظ. تُستخدم بياناتك المنشورة في ORCID لمزامنة بحوثك تلقائيًا مع البوابة."},
            {"question": "ما الذي أحتاجه لتقديم خطة بحث؟", "answer": "تحتاج إلى الاستمارة الرسمية لخطة البحث موقّعة من اللجنة العلمية في القسم، ترفعها مع الطلب في البوابة. يمر الطلب بعد ذلك بمراحل الموافقة: عميد الكلية ثم الشؤون العلمية ثم المساعد العلمي."},
            {"question": "كيف يعمل فحص المجلات المفترسة؟", "answer": "يقارن الفحص اسم المجلة أو ISSN أو رابطها بقوائم عامة للمجلات والناشرين المفترسين والمجلات المختطفة، وبمصادر مثل OpenAlex و Crossref و DOAJ. النتيجة استرشادية وليست حكمًا: المرجع في الاعتماد قوائم وزارة التعليم العالي والبحث العلمي وقرار الشؤون العلمية."}
        ]$json$::jsonb,
        'stats', '[]'::jsonb,
        'hide_publications', false
    )
WHERE id = 1;
