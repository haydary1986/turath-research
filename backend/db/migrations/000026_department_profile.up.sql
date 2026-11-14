-- The department's vision, mission, goals and tasks (from the department's own document),
-- shown on the homepage and editable in the site editor. Fills the block only when it is
-- still empty, so an edit made in the editor is never overwritten.
UPDATE site_content
SET content = content || jsonb_build_object('department', $json${
  "vision": "تتمثل رؤية القسم في السعي نحو الريادة والتميز العلمي والبحثي عالمياً، والارتقاء بالمستوى الأكاديمي لتلبية متطلبات التطورات الحديثة.",
  "vision_en": "To pursue leadership and excellence in science and research worldwide, and to raise the academic standard to meet the demands of modern developments.",
  "mission": "تتمثل رسالة القسم بتوفير وتهيئة بيئة علمية وبحثية رصينة للباحثين والتدريسيين، ودعم حركة النشر والابتكار الأكاديمي بما يخدم المجتمع لمواكبة التطورات الحديثة في مجالات العلوم والتكنولوجيا وتطبيقها لخدمة المجتمع من خلال دعم البحث العلمي وتوثيق النتاجات وتطوير الخطط العلمية ووضع الخطط العلمية الاستراتيجية وتطوير بيئة محفزة للإبداع والابتكار.",
  "mission_en": "To provide researchers and academics with a sound scientific and research environment, and to support publishing and academic innovation that serve society and keep pace with modern developments in science and technology — by supporting scientific research, documenting output, developing scientific and strategic plans, and building an environment that encourages creativity and innovation.",
  "goals": [
    {
      "text": "تطوير وترصين البحث العلمي.",
      "text_en": "Developing and strengthening scientific research."
    },
    {
      "text": "دعم النتاج الفكري والابتكار.",
      "text_en": "Supporting intellectual output and innovation."
    },
    {
      "text": "رفع كفاءة العملية التعليمية والأكاديمية بما ينسجم مع التطورات العالمية وحاجة المجتمع وسوق العمل.",
      "text_en": "Raising the efficiency of teaching and academic work in line with global developments and the needs of society and the labour market."
    },
    {
      "text": "ترصين ودعم البحث العلمي.",
      "text_en": "Consolidating and supporting scientific research."
    },
    {
      "text": "رعاية النتاج الفكري والابتكار والتواصل الأكاديمي وإدارة الأنشطة.",
      "text_en": "Fostering intellectual output, innovation and academic communication, and managing scientific activities."
    }
  ],
  "tasks": [
    {
      "text": "تعميم الكتب الوزارية الواردة عن طريق رئاسة الجامعة.",
      "text_en": "Circulating ministry letters received through the university presidency."
    },
    {
      "text": "تغطية المؤتمرات والندوات والحلقات النقاشية التي تقيمها أقسام الكلية.",
      "text_en": "Covering the conferences, seminars and panel discussions held by the departments."
    },
    {
      "text": "متابعة البحوث المنشورة من قبل أساتذة الأقسام في المجلات العربية والعالمية وتوثيقها، وإعلام رئاسة الجامعة / مكتب المساعد العلمي بالمنشور منها في مجلات عالمية ذات معامل تأثير.",
      "text_en": "Following up and documenting research published by faculty in Arab and international journals, and informing the university presidency / office of the Vice President for Scientific Affairs of papers published in international journals with an impact factor."
    },
    {
      "text": "إحصاء الكتب المؤلَّفة من قبل أساتذة الأقسام العلمية.",
      "text_en": "Recording the books authored by faculty members."
    },
    {
      "text": "متابعة النشاطات الخارجية للتدريسيين وأعضاء الهيئة التدريسية في الكلية.",
      "text_en": "Following up the external activities of the teaching staff."
    },
    {
      "text": "إنشاء قاعدة بيانات لكافة تدريسيي الجامعة.",
      "text_en": "Building a database of all the university's academics."
    },
    {
      "text": "رفع النشاطات العلمية الخاصة بالجامعة بشكل دوري إلى الوزارة عن طريق موقع ندوتي.",
      "text_en": "Reporting the university's scientific activities to the ministry regularly through the Nadwati platform."
    },
    {
      "text": "الإجابة عن الكتب الرسمية الواردة من وزارة التعليم العالي والبحث العلمي.",
      "text_en": "Answering official letters from the Ministry of Higher Education and Scientific Research."
    },
    {
      "text": "مطابقة المناهج الدراسية المعتمدة من قبل الكليات مع ما مقرر من قبل الوزارة.",
      "text_en": "Matching the curricula adopted by the colleges against those approved by the ministry."
    },
    {
      "text": "متابعة استحداث الكليات والأقسام العلمية وللدراستين الصباحية والمسائية.",
      "text_en": "Following up the establishment of new colleges and departments, for both morning and evening study."
    }
  ]
}$json$::jsonb),
    updated_at = NOW()
WHERE id = 1
  AND COALESCE(content->'department'->>'vision', '') = ''
  AND jsonb_array_length(COALESCE(content->'department'->'tasks', '[]'::jsonb)) = 0;
