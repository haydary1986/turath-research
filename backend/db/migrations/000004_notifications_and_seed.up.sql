-- ══════════════════════════════════════════════════
-- Migration 4: Notifications table + Seed data for جامعة التراث
-- ══════════════════════════════════════════════════

-- ── Notifications ────────────────────────────────

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_ar   VARCHAR(500) NOT NULL,
    title_en   VARCHAR(500),
    body       TEXT NOT NULL DEFAULT '',
    type       VARCHAR(50) NOT NULL DEFAULT 'info', -- info, success, warning, alert, promotion, incentive, deadline
    link       TEXT,
    is_read    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);

-- ══════════════════════════════════════════════════
-- Seed Data: جامعة التراث
-- ══════════════════════════════════════════════════

-- Institution
INSERT INTO institutions (id, name_ar, name_en, domain)
VALUES ('11111111-1111-1111-1111-111111111111', 'جامعة التراث', 'University of Heritage', 'turath.edu.iq')
ON CONFLICT (domain) DO NOTHING;

-- Colleges
INSERT INTO colleges (id, name_ar, name_en, code, institution_id) VALUES
    ('b0000001-0000-0000-0000-000000000001', 'كلية الهندسة', 'College of Engineering', 'ENG', '11111111-1111-1111-1111-111111111111'),
    ('b0000002-0000-0000-0000-000000000002', 'كلية العلوم', 'College of Science', 'SCI', '11111111-1111-1111-1111-111111111111'),
    ('b0000003-0000-0000-0000-000000000003', 'كلية تقنية المعلومات', 'College of IT', 'IT', '11111111-1111-1111-1111-111111111111'),
    ('b0000004-0000-0000-0000-000000000004', 'كلية الإدارة والاقتصاد', 'College of Business', 'BUS', '11111111-1111-1111-1111-111111111111'),
    ('b0000005-0000-0000-0000-000000000005', 'كلية القانون', 'College of Law', 'LAW', '11111111-1111-1111-1111-111111111111'),
    ('b0000006-0000-0000-0000-000000000006', 'كلية التربية', 'College of Education', 'EDU', '11111111-1111-1111-1111-111111111111')
ON CONFLICT (code) DO NOTHING;

-- Departments
INSERT INTO departments (id, name_ar, name_en, code, college_id) VALUES
    -- Engineering
    ('d0000001-0000-0000-0000-000000000001', 'هندسة الحاسوب', 'Computer Engineering', 'CE', 'b0000001-0000-0000-0000-000000000001'),
    ('d0000002-0000-0000-0000-000000000002', 'الهندسة المدنية', 'Civil Engineering', 'CIV', 'b0000001-0000-0000-0000-000000000001'),
    ('d0000003-0000-0000-0000-000000000003', 'الهندسة الكهربائية', 'Electrical Engineering', 'EE', 'b0000001-0000-0000-0000-000000000001'),
    -- Science
    ('d0000004-0000-0000-0000-000000000004', 'الرياضيات', 'Mathematics', 'MATH', 'b0000002-0000-0000-0000-000000000002'),
    ('d0000005-0000-0000-0000-000000000005', 'الفيزياء', 'Physics', 'PHYS', 'b0000002-0000-0000-0000-000000000002'),
    ('d0000006-0000-0000-0000-000000000006', 'الكيمياء', 'Chemistry', 'CHEM', 'b0000002-0000-0000-0000-000000000002'),
    -- IT
    ('d0000007-0000-0000-0000-000000000007', 'علوم الحاسوب', 'Computer Science', 'CS', 'b0000003-0000-0000-0000-000000000003'),
    ('d0000008-0000-0000-0000-000000000008', 'نظم المعلومات', 'Information Systems', 'IS', 'b0000003-0000-0000-0000-000000000003'),
    ('d0000009-0000-0000-0000-000000000009', 'الأمن السيبراني', 'Cybersecurity', 'CYB', 'b0000003-0000-0000-0000-000000000003'),
    -- Business
    ('d0000010-0000-0000-0000-000000000010', 'إدارة الأعمال', 'Business Administration', 'BA', 'b0000004-0000-0000-0000-000000000004'),
    ('d0000011-0000-0000-0000-000000000011', 'المحاسبة', 'Accounting', 'ACC', 'b0000004-0000-0000-0000-000000000004')
ON CONFLICT (code) DO NOTHING;

-- Researchers (sample faculty)
INSERT INTO researchers (id, name_ar, name_en, email, title_ar, title_en, department_id, h_index, total_citations, total_publications, status) VALUES
    ('a0000001-0000-0000-0000-000000000001', 'أ.د. علي حسين الموسوي', 'Prof. Ali Hussein Al-Musawi', 'ali.musawi@turath.edu.iq', 'أستاذ', 'Professor', 'd0000007-0000-0000-0000-000000000007', 15, 890, 45, 'active'),
    ('a0000002-0000-0000-0000-000000000002', 'أ.م.د. فاطمة أحمد الجبوري', 'Dr. Fatima Ahmed Al-Jubouri', 'fatima.jubouri@turath.edu.iq', 'أستاذ مشارك', 'Associate Professor', 'd0000001-0000-0000-0000-000000000001', 11, 520, 32, 'active'),
    ('a0000003-0000-0000-0000-000000000003', 'أ.م.د. محمد كاظم الربيعي', 'Dr. Mohammed Kadhim Al-Rubaye', 'mohammed.rubaye@turath.edu.iq', 'أستاذ مشارك', 'Associate Professor', 'd0000004-0000-0000-0000-000000000004', 9, 340, 28, 'active'),
    ('a0000004-0000-0000-0000-000000000004', 'م.د. زينب عبدالله الحسيني', 'Dr. Zainab Abdullah Al-Husseini', 'zainab.husseini@turath.edu.iq', 'مدرس', 'Lecturer', 'd0000005-0000-0000-0000-000000000005', 6, 180, 18, 'active'),
    ('a0000005-0000-0000-0000-000000000005', 'أ.د. حيدر عبدالأمير العبيدي', 'Prof. Haydar Abdulameer Al-Obaidi', 'haydar.obaidi@turath.edu.iq', 'أستاذ', 'Professor', 'd0000009-0000-0000-0000-000000000009', 18, 1200, 62, 'active'),
    ('a0000006-0000-0000-0000-000000000006', 'أ.م.د. نور الهدى السعدي', 'Dr. Noor Al-Huda Al-Saadi', 'noor.saadi@turath.edu.iq', 'أستاذ مشارك', 'Associate Professor', 'd0000008-0000-0000-0000-000000000008', 8, 290, 22, 'active'),
    ('a0000007-0000-0000-0000-000000000007', 'م.د. أحمد جاسم المالكي', 'Dr. Ahmed Jasim Al-Maliki', 'ahmed.maliki@turath.edu.iq', 'مدرس', 'Lecturer', 'd0000002-0000-0000-0000-000000000002', 5, 120, 14, 'active'),
    ('a0000008-0000-0000-0000-000000000008', 'أ.م.د. سارة محمد الخفاجي', 'Dr. Sara Mohammed Al-Khafaji', 'sara.khafaji@turath.edu.iq', 'أستاذ مشارك', 'Associate Professor', 'd0000006-0000-0000-0000-000000000006', 12, 650, 38, 'active'),
    ('a0000009-0000-0000-0000-000000000009', 'م.د. حسن علي الشمري', 'Dr. Hassan Ali Al-Shamri', 'hassan.shamri@turath.edu.iq', 'مدرس', 'Lecturer', 'd0000003-0000-0000-0000-000000000003', 4, 95, 11, 'active'),
    ('a0000010-0000-0000-0000-000000000010', 'أ.د. مريم كريم العامري', 'Prof. Mariam Kareem Al-Ameri', 'mariam.ameri@turath.edu.iq', 'أستاذ', 'Professor', 'd0000010-0000-0000-0000-000000000010', 14, 780, 48, 'active')
ON CONFLICT (email) DO NOTHING;

-- Sample Journals
INSERT INTO journals (id, issn, name, publisher, sjr_ranking, impact_factor, is_open_access) VALUES
    ('f0000001-0000-0000-0000-000000000001', '0018-9340', 'IEEE Transactions on Computers', 'IEEE', 'Q1', 3.183, false),
    ('f0000002-0000-0000-0000-000000000002', '0950-7051', 'Knowledge-Based Systems', 'Elsevier', 'Q1', 8.038, false),
    ('f0000003-0000-0000-0000-000000000003', '2076-3417', 'Applied Sciences', 'MDPI', 'Q2', 2.838, true),
    ('f0000004-0000-0000-0000-000000000004', '1996-1073', 'Energies', 'MDPI', 'Q2', 3.252, true),
    ('f0000005-0000-0000-0000-000000000005', '2227-7390', 'Mathematics', 'MDPI', 'Q2', 2.258, true),
    ('f0000006-0000-0000-0000-000000000006', '0957-4174', 'Expert Systems with Applications', 'Elsevier', 'Q1', 8.665, false)
ON CONFLICT (issn) DO NOTHING;

-- Sample Publications
INSERT INTO publications (id, doi, title_en, title_ar, publication_date, publication_type, journal_id, citation_count, language, status) VALUES
    ('e0000001-0000-0000-0000-000000000001', '10.1109/tc.2024.001', 'Deep Learning Approach for Arabic Text Classification', 'نهج التعلم العميق لتصنيف النصوص العربية', '2024-03-15', 'journal_article', 'f0000001-0000-0000-0000-000000000001', 45, 'en', 'published'),
    ('e0000002-0000-0000-0000-000000000002', '10.1016/j.knosys.2024.002', 'Cybersecurity Threat Detection Using Machine Learning', 'كشف التهديدات السيبرانية باستخدام تعلم الآلة', '2024-05-20', 'journal_article', 'f0000002-0000-0000-0000-000000000002', 32, 'en', 'published'),
    ('e0000003-0000-0000-0000-000000000003', '10.3390/app14030001', 'IoT-Based Smart Campus Energy Management System', 'نظام إدارة الطاقة الذكي للحرم الجامعي باستخدام إنترنت الأشياء', '2024-01-10', 'journal_article', 'f0000003-0000-0000-0000-000000000003', 18, 'en', 'published'),
    ('e0000004-0000-0000-0000-000000000004', '10.3390/en17050001', 'Solar Energy Optimization in Iraqi Climate Conditions', 'تحسين الطاقة الشمسية في الظروف المناخية العراقية', '2024-02-28', 'journal_article', 'f0000004-0000-0000-0000-000000000004', 22, 'en', 'published'),
    ('e0000005-0000-0000-0000-000000000005', '10.3390/math12050001', 'Novel Numerical Methods for Fractional Differential Equations', 'طرق عددية مبتكرة للمعادلات التفاضلية الكسرية', '2024-04-12', 'journal_article', 'f0000005-0000-0000-0000-000000000005', 15, 'en', 'published'),
    ('e0000006-0000-0000-0000-000000000006', '10.1016/j.eswa.2024.001', 'Blockchain-Based Academic Credential Verification', 'التحقق من الشهادات الأكاديمية باستخدام البلوكتشين', '2024-06-05', 'journal_article', 'f0000006-0000-0000-0000-000000000006', 28, 'en', 'published'),
    ('e0000007-0000-0000-0000-000000000007', '10.1109/tc.2024.002', 'Quantum Computing Applications in Cryptography', 'تطبيقات الحوسبة الكمية في التشفير', '2024-07-18', 'journal_article', 'f0000001-0000-0000-0000-000000000001', 12, 'en', 'published'),
    ('e0000008-0000-0000-0000-000000000008', '10.3390/app14080001', 'AI-Driven Water Quality Monitoring in Tigris River', 'مراقبة جودة المياه في نهر دجلة بالذكاء الاصطناعي', '2024-08-22', 'journal_article', 'f0000003-0000-0000-0000-000000000003', 8, 'en', 'published'),
    ('e0000009-0000-0000-0000-000000000009', '10.1016/j.knosys.2024.003', 'Natural Language Processing for Iraqi Arabic Dialect', 'معالجة اللغة الطبيعية للهجة العراقية', '2024-09-10', 'journal_article', 'f0000002-0000-0000-0000-000000000002', 35, 'en', 'published'),
    ('e0000010-0000-0000-0000-000000000010', '10.3390/math12100001', 'Optimization Algorithms for Supply Chain Management', 'خوارزميات التحسين لإدارة سلسلة التوريد', '2024-10-01', 'journal_article', 'f0000005-0000-0000-0000-000000000005', 11, 'en', 'published')
ON CONFLICT (doi) DO NOTHING;

-- Link publications to authors
INSERT INTO publication_authors (id, publication_id, researcher_id, author_name, author_order, is_corresponding)
SELECT uuid_generate_v4(), pub_id, res_id, author_name, author_order, is_corr
FROM (VALUES
    ('e0000001-0000-0000-0000-000000000001'::uuid, 'a0000001-0000-0000-0000-000000000001'::uuid, 'Ali Hussein Al-Musawi', 1, true),
    ('e0000001-0000-0000-0000-000000000001'::uuid, 'a0000002-0000-0000-0000-000000000002'::uuid, 'Fatima Ahmed Al-Jubouri', 2, false),
    ('e0000002-0000-0000-0000-000000000002'::uuid, 'a0000005-0000-0000-0000-000000000005'::uuid, 'Haydar Abdulameer Al-Obaidi', 1, true),
    ('e0000002-0000-0000-0000-000000000002'::uuid, 'a0000001-0000-0000-0000-000000000001'::uuid, 'Ali Hussein Al-Musawi', 2, false),
    ('e0000003-0000-0000-0000-000000000003'::uuid, 'a0000009-0000-0000-0000-000000000009'::uuid, 'Hassan Ali Al-Shamri', 1, true),
    ('e0000004-0000-0000-0000-000000000004'::uuid, 'a0000009-0000-0000-0000-000000000009'::uuid, 'Hassan Ali Al-Shamri', 1, false),
    ('e0000004-0000-0000-0000-000000000004'::uuid, 'a0000004-0000-0000-0000-000000000004'::uuid, 'Zainab Abdullah Al-Husseini', 2, true),
    ('e0000005-0000-0000-0000-000000000005'::uuid, 'a0000003-0000-0000-0000-000000000003'::uuid, 'Mohammed Kadhim Al-Rubaye', 1, true),
    ('e0000006-0000-0000-0000-000000000006'::uuid, 'a0000005-0000-0000-0000-000000000005'::uuid, 'Haydar Abdulameer Al-Obaidi', 1, true),
    ('e0000006-0000-0000-0000-000000000006'::uuid, 'a0000006-0000-0000-0000-000000000006'::uuid, 'Noor Al-Huda Al-Saadi', 2, false),
    ('e0000007-0000-0000-0000-000000000007'::uuid, 'a0000005-0000-0000-0000-000000000005'::uuid, 'Haydar Abdulameer Al-Obaidi', 1, true),
    ('e0000008-0000-0000-0000-000000000008'::uuid, 'a0000008-0000-0000-0000-000000000008'::uuid, 'Sara Mohammed Al-Khafaji', 1, true),
    ('e0000008-0000-0000-0000-000000000008'::uuid, 'a0000004-0000-0000-0000-000000000004'::uuid, 'Zainab Abdullah Al-Husseini', 2, false),
    ('e0000009-0000-0000-0000-000000000009'::uuid, 'a0000001-0000-0000-0000-000000000001'::uuid, 'Ali Hussein Al-Musawi', 1, true),
    ('e0000009-0000-0000-0000-000000000009'::uuid, 'a0000006-0000-0000-0000-000000000006'::uuid, 'Noor Al-Huda Al-Saadi', 2, false),
    ('e0000010-0000-0000-0000-000000000010'::uuid, 'a0000010-0000-0000-0000-000000000010'::uuid, 'Mariam Kareem Al-Ameri', 1, true),
    ('e0000010-0000-0000-0000-000000000010'::uuid, 'a0000003-0000-0000-0000-000000000003'::uuid, 'Mohammed Kadhim Al-Rubaye', 2, false)
) AS v(pub_id, res_id, author_name, author_order, is_corr)
WHERE EXISTS (SELECT 1 FROM publications WHERE id = v.pub_id);

-- Promotion requirements for جامعة التراث
INSERT INTO promotion_requirements (id, institution_id, from_rank, to_rank, field_category, min_q1_publications, min_total_publications, min_h_index, min_years_in_rank) VALUES
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'lecturer', 'assistant_prof', 'all', 1, 3, 2, 3),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'assistant_prof', 'associate_prof', 'all', 3, 8, 5, 4),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'associate_prof', 'professor', 'all', 6, 15, 10, 5)
ON CONFLICT DO NOTHING;

-- Incentive rules
INSERT INTO incentive_rules (id, institution_id, journal_classification, base_amount, corresponding_bonus, effective_from) VALUES
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q1', 3000000, 500000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q2', 2000000, 300000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'Q3', 1000000, 200000, '2024-01-01'),
    (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'scopus_only', 500000, 100000, '2024-01-01')
ON CONFLICT DO NOTHING;

-- Admin user for جامعة التراث
-- Password: Admin1234! — hash generated via bcrypt cost 10
-- Note: Register via API for proper hashing; this is a bootstrap user
INSERT INTO users (id, email, password_hash, name_ar, name_en, role, institution_id)
VALUES (
    'aa000001-0000-0000-0000-000000000001',
    'admin@turath.edu.iq',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'مدير النظام',
    'System Admin',
    'admin',
    '11111111-1111-1111-1111-111111111111'
) ON CONFLICT (email) DO NOTHING;
