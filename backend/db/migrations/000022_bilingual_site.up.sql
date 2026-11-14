-- Bilingual public site: every editable text of the landing-page document gets an
-- optional English counterpart stored beside the Arabic (the *_en keys), and each
-- announcement records the language it was written in.
--
-- The Arabic stays exactly as it is. English is filled in only where the stored
-- Arabic is still one of the seeded defaults (000017 / 000021) and no English has
-- been entered yet; text the department has edited gets an empty English value,
-- and the public pages fall back to the Arabic for it.

-- ── Announcements ────────────────────────────────
ALTER TABLE announcements
    ADD COLUMN lang VARCHAR(2) NOT NULL DEFAULT 'ar' CHECK (lang IN ('ar', 'en'));

-- ── Landing-page document ────────────────────────
CREATE TEMP TABLE site_en_defaults (ar TEXT PRIMARY KEY, en TEXT NOT NULL);

INSERT INTO site_en_defaults (ar, en) VALUES
    ('قسم الشؤون العلمية', 'Scientific Affairs Department'),
    ('جامعة التراث — بوابة إلكترونية لمتابعة البحث العلمي والنشاطات العلمية وإنجاز معاملات التدريسيين والباحثين.',
     'Al-Turath University — an online portal for following scientific research and scientific activities and for completing the requests of faculty members and researchers.'),
    (E'يتابع قسم الشؤون العلمية في جامعة التراث النشاط البحثي لأعضاء الهيئة التدريسية والباحثين، وينظّم إجراءات النشر والتدقيق العلمي والنشاطات العلمية في كليات الجامعة.\n\nتتيح هذه البوابة للتدريسيين تقديم طلباتهم ومتابعتها إلكترونيًا، وتنشر الإعلانات والتعاميم والنشاطات العلمية الصادرة عن القسم.',
     E'The Scientific Affairs Department of Al-Turath University follows the research activity of faculty members and researchers, and organises publication procedures, scientific review and scientific activities across the university''s colleges.\n\nThrough this portal, faculty members submit and track their requests online, and the department publishes its announcements, circulars and scientific activities.'),
    (E'يتابع قسم الشؤون العلمية في جامعة التراث النشاط البحثي لأعضاء الهيئة التدريسية والباحثين، وينظّم إجراءات النشر والتدقيق العلمي والنشاطات العلمية في كليات الجامعة.\n\nتتيح هذه البوابة للتدريسيين تقديم طلباتهم ومتابعتها إلكترونيًا، وتنشر الإعلانات والاعمامات والنشاطات العلمية الصادرة عن القسم.',
     E'The Scientific Affairs Department of Al-Turath University follows the research activity of faculty members and researchers, and organises publication procedures, scientific review and scientific activities across the university''s colleges.\n\nThrough this portal, faculty members submit and track their requests online, and the department publishes its announcements, circulars and scientific activities.'),
    -- Services
    ('خطط البحث العلمي', 'Research plans'),
    ('تقديم خطة البحث ومتابعة مصادقتها عبر سلسلة الموافقات.', 'Submit a research plan and follow its endorsement through the approval chain.'),
    ('تدقيق رصانة المجلات', 'Journal verification'),
    ('التحقق من المجلة التي نُشر فيها البحث قبل اعتماده.', 'Verify the journal a paper was published in before the paper is approved.'),
    ('كتب الشكر والتقدير', 'Appreciation letters'),
    ('طلب كتاب الشكر والتقدير عن البحوث المنشورة بعد تدقيقها.', 'Request an appreciation letter for published papers once they have been verified.'),
    ('النشاطات العلمية', 'Scientific activities'),
    ('تسجيل الندوات والورش والمحاضرات ومتابعة الموافقة عليها.', 'Register seminars, workshops and lectures and follow their approval.'),
    ('فحص المجلات', 'Journal check'),
    ('فحص المجلة قبل النشر والتنبيه على المجلات غير الرصينة.', 'Check a journal before publishing and get warned about journals that are not reputable.'),
    ('دليل المجلات المفترسة', 'Predatory journals guide'),
    ('قوائم المجلات والناشرين المفترسين والمجلات المختطفة وعلامات التحذير.', 'Lists of predatory journals and publishers and of hijacked journals, and the warning signs to look for.'),
    -- Quick links and director block
    ('الموقع الرسمي لجامعة التراث', 'Al-Turath University official website'),
    ('كلمة مدير قسم الشؤون العلمية', 'Message from the Director of Scientific Affairs'),
    -- Colleges
    ('كلية طب الأسنان', 'College of Dentistry'),
    ('كلية الصيدلة', 'College of Pharmacy'),
    ('كلية التمريض', 'College of Nursing'),
    ('كلية التقنيات الصحية والطبية', 'College of Health and Medical Technologies'),
    ('كلية الهندسة', 'College of Engineering'),
    ('الكلية التقنية الهندسية', 'Technical College of Engineering'),
    ('كلية العلوم', 'College of Science'),
    ('كلية القانون', 'College of Law'),
    ('كلية الآداب', 'College of Arts'),
    ('كلية الفنون الجميلة', 'College of Fine Arts'),
    ('كلية الإدارة والاقتصاد', 'College of Administration and Economics'),
    ('كلية التربية', 'College of Education'),
    ('كلية التربية البدنية وعلوم الرياضة', 'College of Physical Education and Sport Sciences'),
    ('كلية الإعلام', 'College of Media'),
    -- FAQ
    ('لماذا يفشل تسجيل الدخول إلى البوابة؟', 'Why does signing in to the portal fail?'),
    ('يكون الدخول بحساب Google Workspace الجامعي فقط، أي البريد الذي ينتهي بـ @uoturath.edu.iq. إذا كان المتصفح مسجّلًا بحساب Gmail شخصي فاختر الحساب الجامعي من نافذة Google أو سجّل الخروج من الحساب الشخصي ثم أعد المحاولة. إذا استمرت المشكلة فتواصل مع قسم الشؤون العلمية.',
     'Sign-in works only with your university Google Workspace account, that is, an email ending in @uoturath.edu.iq. If the browser is signed in with a personal Gmail account, choose the university account in the Google window, or sign out of the personal account and try again. If the problem continues, contact the Scientific Affairs Department.'),
    ('كيف أضيف معرّف ORCID إلى ملفي؟', 'How do I add my ORCID iD to my profile?'),
    ('بعد تسجيل الدخول افتح صفحة «ملفي البحثي» وأدخل معرّف ORCID بصيغته الكاملة (مثل 0000-0002-1825-0097) ثم احفظ. تُستخدم بياناتك المنشورة في ORCID لمزامنة بحوثك تلقائيًا مع البوابة.',
     'After signing in, open the "My research profile" page, enter your ORCID iD in full (for example 0000-0002-1825-0097) and save. The data you have published on ORCID is used to sync your publications with the portal automatically.'),
    ('ما الذي أحتاجه لتقديم خطة بحث؟', 'What do I need to submit a research plan?'),
    ('تحتاج إلى الاستمارة الرسمية لخطة البحث موقّعة من اللجنة العلمية في القسم، ترفعها مع الطلب في البوابة. يمر الطلب بعد ذلك بمراحل الموافقة: عميد الكلية ثم الشؤون العلمية ثم المساعد العلمي.',
     'You need the official research plan form signed by the department''s scientific committee, which you upload with the request on the portal. The request then goes through the approval stages: the college dean, then Scientific Affairs, then the Vice President for Scientific Affairs.'),
    ('كيف يعمل فحص المجلات المفترسة؟', 'How does the predatory journal check work?'),
    ('يقارن الفحص اسم المجلة أو ISSN أو رابطها بقوائم عامة للمجلات والناشرين المفترسين والمجلات المختطفة، وبمصادر مثل OpenAlex و Crossref و DOAJ. النتيجة استرشادية وليست حكمًا: المرجع في الاعتماد قوائم وزارة التعليم العالي والبحث العلمي وقرار الشؤون العلمية.',
     'The check compares the journal''s name, ISSN or website with public lists of predatory journals, predatory publishers and hijacked journals, and with sources such as OpenAlex, Crossref and DOAJ. The result is advisory, not a ruling: the lists of the Ministry of Higher Education and Scientific Research and the decision of Scientific Affairs prevail.');

-- English for an Arabic text: the stored English when there is one, else the default
-- translation when the Arabic is still a seeded default, else ''.
CREATE FUNCTION pg_temp.site_en(ar TEXT, current_en TEXT) RETURNS TEXT
LANGUAGE sql STABLE AS $fn$
    SELECT COALESCE(NULLIF(current_en, ''), (SELECT en FROM site_en_defaults d WHERE d.ar = $1), '')
$fn$;

-- Adds English keys to every object of a JSON array: pairs is {english_key: arabic_key}.
CREATE FUNCTION pg_temp.site_en_list(items JSONB, pairs JSONB) RETURNS JSONB
LANGUAGE sql STABLE AS $fn$
    SELECT COALESCE(jsonb_agg(
        item || (SELECT COALESCE(jsonb_object_agg(p.key, pg_temp.site_en(item->>(p.value #>> '{}'), item->>p.key)), '{}'::jsonb)
                 FROM jsonb_each(pairs) p)
        ORDER BY ord), '[]'::jsonb)
    FROM jsonb_array_elements(COALESCE(items, '[]'::jsonb)) WITH ORDINALITY AS t(item, ord)
$fn$;

UPDATE site_content SET content = content || jsonb_build_object(
    'hero_title_en', pg_temp.site_en(content->>'hero_title', content->>'hero_title_en'),
    'hero_subtitle_en', pg_temp.site_en(content->>'hero_subtitle', content->>'hero_subtitle_en'),
    'about_en', pg_temp.site_en(content->>'about', content->>'about_en'),
    'services', pg_temp.site_en_list(content->'services', '{"title_en": "title", "description_en": "description"}'),
    'contact', COALESCE(content->'contact', '{}'::jsonb) || jsonb_build_object(
        'location_en', pg_temp.site_en(content->'contact'->>'location', content->'contact'->>'location_en'),
        'hours_en', pg_temp.site_en(content->'contact'->>'hours', content->'contact'->>'hours_en')),
    'links', pg_temp.site_en_list(content->'links', '{"label_en": "label"}'),
    'colleges', pg_temp.site_en_list(content->'colleges', '{"label_en": "label"}'),
    'stats', pg_temp.site_en_list(content->'stats', '{"label_en": "label"}'),
    'faq', pg_temp.site_en_list(content->'faq', '{"question_en": "question", "answer_en": "answer"}'),
    'director', COALESCE(content->'director', '{}'::jsonb) || jsonb_build_object(
        'title_en', pg_temp.site_en(content->'director'->>'title', content->'director'->>'title_en'),
        'name_en', pg_temp.site_en(content->'director'->>'name', content->'director'->>'name_en'),
        'position_en', pg_temp.site_en(content->'director'->>'position', content->'director'->>'position_en'),
        'message_en', pg_temp.site_en(content->'director'->>'message', content->'director'->>'message_en'))
)
WHERE id = 1;

DROP FUNCTION pg_temp.site_en_list(JSONB, JSONB);
DROP FUNCTION pg_temp.site_en(TEXT, TEXT);
DROP TABLE site_en_defaults;
