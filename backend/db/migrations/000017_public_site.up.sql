-- Public face of the portal: the editable landing-page document and the
-- announcements (scientific activities and circulars / اعمامات) staff publish.

-- One row only: the whole landing page as a JSON document validated by the API.
CREATE TABLE site_content (
    id         SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    content    JSONB NOT NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Neutral defaults that describe the portal's own services; contact details are
-- left empty for the department to fill in.
INSERT INTO site_content (id, content) VALUES (1, $json$
{
  "hero_title": "قسم الشؤون العلمية",
  "hero_subtitle": "جامعة التراث — بوابة إلكترونية لمتابعة البحث العلمي والنشاطات العلمية وإنجاز معاملات التدريسيين والباحثين.",
  "about": "يتابع قسم الشؤون العلمية في جامعة التراث النشاط البحثي لأعضاء الهيئة التدريسية والباحثين، وينظّم إجراءات النشر والتدقيق العلمي والنشاطات العلمية في كليات الجامعة.\n\nتتيح هذه البوابة للتدريسيين تقديم طلباتهم ومتابعتها إلكترونيًا، وتنشر الإعلانات والاعمامات والنشاطات العلمية الصادرة عن القسم.",
  "services": [
    {"title": "خطط البحث العلمي", "description": "تقديم خطة البحث ومتابعة مصادقتها عبر سلسلة الموافقات."},
    {"title": "تدقيق رصانة المجلات", "description": "التحقق من المجلة التي نُشر فيها البحث قبل اعتماده."},
    {"title": "كتب الشكر والتقدير", "description": "طلب كتاب الشكر والتقدير عن البحوث المنشورة بعد تدقيقها."},
    {"title": "النشاطات العلمية", "description": "تسجيل الندوات والورش والمحاضرات ومتابعة الموافقة عليها."},
    {"title": "فحص المجلات", "description": "فحص المجلة قبل النشر والتنبيه على المجلات غير الرصينة."}
  ],
  "contact": {"email": "", "phone": "", "location": "", "hours": ""},
  "links": [
    {"label": "الموقع الرسمي لجامعة التراث", "url": "https://uoturath.edu.iq"}
  ]
}
$json$::jsonb);

CREATE TABLE announcements (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kind          VARCHAR(20) NOT NULL CHECK (kind IN ('activity', 'circular')),
    title         VARCHAR(300) NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    event_date    DATE,
    reference_no  VARCHAR(100),
    attachment_id UUID UNIQUE REFERENCES portal_files(id) ON DELETE SET NULL,
    activity_id   UUID UNIQUE REFERENCES activities(id) ON DELETE SET NULL,
    is_published  BOOLEAN NOT NULL DEFAULT FALSE,
    published_at  TIMESTAMPTZ,
    is_pinned     BOOLEAN NOT NULL DEFAULT FALSE,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (NOT is_published OR published_at IS NOT NULL)
);

CREATE INDEX idx_announcements_public ON announcements (kind, is_pinned DESC, published_at DESC) WHERE is_published;
CREATE INDEX idx_announcements_staff  ON announcements (created_at DESC);
