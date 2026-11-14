-- External research tools for the footer: where to look a journal up in Scopus and
-- Web of Science, its quartile (SCImago) and impact factor, plus author and open-access
-- lookups. Fills the list only while it is empty; staff edit it in the site editor.
UPDATE site_content
SET content = content || jsonb_build_object('tools', $json$[
  {
    "label": "البحث عن مجلات Scopus",
    "label_en": "Scopus journal search",
    "url": "https://www.scopus.com/sources"
  },
  {
    "label": "ترتيب المجلات والربع (Q) — SCImago",
    "label_en": "Journal rank and quartile (Q) — SCImago",
    "url": "https://www.scimagojr.com/journalrank.php"
  },
  {
    "label": "قائمة مجلات Web of Science",
    "label_en": "Web of Science Master Journal List",
    "url": "https://mjl.clarivate.com/home"
  },
  {
    "label": "معامل التأثير — Journal Citation Reports",
    "label_en": "Impact factor — Journal Citation Reports",
    "url": "https://jcr.clarivate.com/"
  },
  {
    "label": "البحث عن باحث في Scopus",
    "label_en": "Scopus author search",
    "url": "https://www.scopus.com/freelookup/form/author.uri"
  },
  {
    "label": "مجلات الوصول المفتوح — DOAJ",
    "label_en": "Open access journals — DOAJ",
    "url": "https://doaj.org/"
  },
  {
    "label": "بوابة المجلات العلمية العراقية",
    "label_en": "Iraqi scientific journals portal",
    "url": "https://jor.rdd.edu.iq/"
  },
  {
    "label": "سجل ORCID للباحثين",
    "label_en": "ORCID researcher ID",
    "url": "https://orcid.org/"
  }
]$json$::jsonb),
    updated_at = NOW()
WHERE id = 1 AND jsonb_array_length(COALESCE(content->'tools', '[]'::jsonb)) = 0;
