# خطة نظام إدارة البحث العلمي المؤسسي (تُراث)

## نظرة عامة

منصة ويب متكاملة لإدارة البحث العلمي على مستوى الجامعة، تشمل:
- **تتبع الأداء البحثي**: استيراد تلقائي من المصادر الأكاديمية العالمية (Scopus, CrossRef, ORCID, OpenAlex)
- **لوحات متابعة متعددة المستويات**: باحث، قسم، كلية، مؤسسة
- **ذكاء اصطناعي**: تحليلات وتوصيات بحثية مخصصة عبر DeepSeek
- **إدارة الشؤون العلمية**: ترقيات، مكافآت نشر، تقييم أداء، تفرغ، مؤتمرات، اتفاقيات
- **الاعتماد وضمان الجودة**: سجل شواهد، متابعة توصيات، تقارير تقييم ذاتي تلقائية
- **سير عمل متعدد المستويات**: لوحات مخصصة وموافقات إلكترونية لكل دور (باحث، رئيس قسم، عميد، شؤون علمية، المساعد العلمي)

**البنية**: Next.js (Frontend) + Go (Backend) — فصل كامل بين الواجهة والخادم

**المراحل**: 8 مراحل — 29 أسبوع (~7 أشهر)

---

## 1. التقنيات المستخدمة (Tech Stack)

### الواجهة الأمامية (Frontend — Next.js)
| التقنية | الغرض |
|---------|-------|
| **Next.js 14+ (App Router)** | إطار العمل مع SSR/SSG |
| **TypeScript** | أمان الأنواع |
| **Tailwind CSS v4** | التنسيق مع دعم RTL عبر CSS logical properties |
| **Shadcn/ui + Radix UI** | مكتبة المكونات (دعم ممتاز لـ RTL) |
| **next-intl** | الترجمة (عربي/إنجليزي) |
| **Recharts / Nivo** | الرسوم البيانية |
| **TanStack Query v5** | إدارة حالة السيرفر (يتكلم مع Go API) |
| **TanStack Table** | جداول بيانات متقدمة (فلترة، ترتيب، تصفح) |
| **React Hook Form + Zod** | النماذج والتحقق |

### الواجهة الخلفية (Backend — Go)
| التقنية | الغرض |
|---------|-------|
| **Go 1.22+** | لغة الخادم الرئيسية |
| **Chi Router** | HTTP router خفيف وسريع |
| **sqlc** | توليد Go code من SQL (type-safe، أسرع من ORM) |
| **golang-migrate** | ترحيل قاعدة البيانات |
| **pgx v5** | PostgreSQL driver عالي الأداء |
| **golang-jwt/jwt** | مصادقة JWT |
| **goroutines + channels** | معالجة خلفية بدون BullMQ |
| **go-redis** | تخزين مؤقت فقط |
| **gonum** | حسابات رياضية وتحليل شبكات الاستشهاد |
| **openai-go** | عميل DeepSeek (متوافق مع OpenAI) |

### قاعدة البيانات والبنية التحتية
| التقنية | الغرض |
|---------|-------|
| **PostgreSQL 16** | قاعدة البيانات الرئيسية |
| **Redis** | تخزين مؤقت + rate limiting |
| **Docker + Docker Compose** | بيئة التطوير والإنتاج |

### الذكاء الاصطناعي
| التقنية | الغرض |
|---------|-------|
| **DeepSeek API** | جميع ميزات الذكاء الاصطناعي |
| **DeepSeek-V3** | معظم التحليلات ($0.27/MTok input, $1.10/MTok output) |
| **DeepSeek-R1** | التحليلات العميقة (تحليل الفجوات، كشف التلاعب) |

> **لماذا DeepSeek؟** أرخص بـ 10-50x مقارنة بـ Claude/GPT. واجهة API متوافقة مع OpenAI SDK.

> **لماذا Go بدلاً من Node.js للخادم؟**
> - أسرع 10-50x في المعالجة الثقيلة (تحليل شبكات الاستشهاد، كشف التلاعب)
> - goroutines تلغي الحاجة لـ BullMQ + Redis queues + worker processes
> - استهلاك ذاكرة أقل 5-10x (50MB vs 500MB)
> - binary واحد للنشر — لا يحتاج Node.js runtime
> - مصمم للتزامن — مزامنة 10+ APIs خارجية بالتوازي

---

## 2. هيكل النظام (Architecture)

```
┌──────────────────────────────────────────────────────────┐
│              Frontend (Next.js — Vercel/Static)           │
│                                                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐ │
│  │  لوحات   │ │  ملفات   │ │ الرسوم   │ │  تحليلات   │ │
│  │ المتابعة │ │ الباحثين │ │ البيانية │ │    AI      │ │
│  └──────────┘ └──────────┘ └──────────┘ └────────────┘ │
│                                                          │
│  RTL (next-intl) + Shadcn/ui + Tailwind logical props   │
└────────────────────────┬─────────────────────────────────┘
                         │ REST API (JSON)
┌────────────────────────▼─────────────────────────────────┐
│                 Backend (Go — Single Binary)              │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │ Chi Router  │  │ JWT Auth     │  │ Middleware      │ │
│  │ REST API    │  │ RBAC         │  │ (rate, cors,    │ │
│  │ /api/v1/*   │  │              │  │  i18n, log)     │ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │              Service Layer                          │ │
│  │  ┌─────────┐ ┌────────┐ ┌────────┐ ┌───────────┐ │ │
│  │  │Researcher│ │Publish │ │Metrics │ │    AI     │ │ │
│  │  │ Service  │ │Service │ │Service │ │  Service  │ │ │
│  │  └─────────┘ └────────┘ └────────┘ └───────────┘ │ │
│  │  ┌───────────┐ ┌──────────┐ ┌──────────────────┐ │ │
│  │  │ Citation  │ │Plagiarism│ │   Predatory      │ │ │
│  │  │ Analysis  │ │Detection │ │   Detection      │ │ │
│  │  │ Engine    │ │ Engine   │ │   Engine         │ │ │
│  │  └───────────┘ └──────────┘ └──────────────────┘ │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │        Repository Layer (sqlc — generated code)     │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │        Background Workers (goroutines)              │ │
│  │  ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │ │
│  │  │API Sync  │ │Metrics   │ │ AI Analysis        │ │ │
│  │  │Workers   │ │Aggregator│ │ Workers            │ │ │
│  │  └──────────┘ └──────────┘ └────────────────────┘ │ │
│  └────────────────────────────────────────────────────┘ │
└────────────────────────┬─────────────────────────────────┘
                         │
          ┌──────────────┼──────────────────┐
          ▼              ▼                  ▼
┌──────────────┐  ┌───────────┐  ┌──────────────────────┐
│ PostgreSQL   │  │   Redis   │  │  External APIs       │
│              │  │           │  │  ┌────────────────┐  │
│ - tables     │  │ - cache   │  │  │ Scopus         │  │
│ - indexes    │  │ - rates   │  │  │ CrossRef       │  │
│ - full-text  │  │           │  │  │ ORCID          │  │
│              │  │           │  │  │ OpenAlex       │  │
└──────────────┘  └───────────┘  │  │ DeepSeek AI    │  │
                                 │  │ Altmetric      │  │
                                 │  └────────────────┘  │
                                 └──────────────────────┘
```

---

## 3. تصميم قاعدة البيانات (Database Schema)

### الجداول الأساسية

#### جدول المؤسسات (institutions)
```
id              uuid (PK)
name_ar         varchar - اسم المؤسسة بالعربي
name_en         varchar - اسم المؤسسة بالإنجليزي
domain          varchar (unique)
logo_url        text
created_at      timestamptz
```

#### جدول الكليات (colleges)
```
id                  uuid (PK)
name_ar             varchar
name_en             varchar
code                varchar (unique)
institution_id      uuid (FK -> institutions)
dean_researcher_id  uuid (FK -> researchers)
created_at          timestamptz
```

#### جدول الأقسام (departments)
```
id                  uuid (PK)
name_ar             varchar
name_en             varchar
code                varchar (unique)
college_id          uuid (FK -> colleges)
head_researcher_id  uuid (FK -> researchers)
created_at          timestamptz
```

#### جدول الباحثين (researchers)
```
id                    uuid (PK)
orcid_id              varchar (unique) - معرف ORCID
scopus_author_id      varchar (unique) - معرف Scopus
google_scholar_id     varchar (unique) - معرف Google Scholar
name_ar               varchar - الاسم بالعربي
name_en               varchar - الاسم بالإنجليزي
email                 varchar (unique)
title_ar              varchar - (أستاذ، أستاذ مشارك، أستاذ مساعد)
title_en              varchar
department_id         uuid (FK -> departments)
h_index               integer
h_index_updated_at    timestamptz
total_citations       integer
total_publications    integer
avatar_url            text
bio_ar                text
bio_en                text
research_interests    jsonb - [{ar, en}]
status                enum('active', 'inactive', 'emeritus', 'visiting')
created_at            timestamptz
updated_at            timestamptz
```

### جداول الأبحاث والتصنيفات

#### جدول المنشورات (publications)
```
id                        uuid (PK)
doi                       varchar (unique)
scopus_id                 varchar (unique)
title_ar                  text
title_en                  text
abstract_ar               text
abstract_en               text
publication_date          date
publication_type          enum('journal_article', 'conference_paper',
                               'book_chapter', 'book', 'preprint',
                               'thesis', 'patent', 'report')
journal_id                uuid (FK -> journals)
conference_name           varchar
volume                    varchar
issue                     varchar
pages                     varchar
url                       text
pdf_url                   text
language                  varchar (default 'ar')
status                    enum('published', 'accepted', 'in_press',
                               'submitted', 'preprint')
citation_count            integer (default 0)
citation_count_updated_at timestamptz
metadata                  jsonb - بيانات خام من APIs
created_at                timestamptz
updated_at                timestamptz
```

#### جدول مؤلفي المنشورات (publication_authors)
```
id                uuid (PK)
publication_id    uuid (FK -> publications)
researcher_id     uuid (FK -> researchers) - null للمؤلفين الخارجيين
author_name       varchar - اسم المؤلف (denormalized)
author_order      integer - ترتيب المؤلف
is_corresponding  boolean - هل هو المؤلف المراسل
affiliation_text  text
```

#### جدول المجلات (journals)
```
id                uuid (PK)
issn              varchar (unique)
eissn             varchar (unique)
name              varchar
publisher         varchar
country           varchar
language          varchar
sjr_ranking       enum('Q1', 'Q2', 'Q3', 'Q4', 'unranked')
impact_factor     decimal(6,3)
impact_factor_year integer
subject_areas     jsonb
is_open_access    boolean
updated_at        timestamptz
```

#### جدول مجالات البحث (research_fields)
```
id          uuid (PK)
name_ar     varchar
name_en     varchar
parent_id   uuid (FK -> research_fields) - للتصنيف الهرمي
asjc_code   varchar - كود Scopus
```

#### جدول ربط المنشورات بالمجالات (publication_fields)
```
publication_id  uuid (FK -> publications)
field_id        uuid (FK -> research_fields)
PRIMARY KEY (publication_id, field_id)
```

### جداول المقاييس والمتابعة

#### مقاييس المنشورات (publication_metrics) - سلسلة زمنية
```
id              uuid (PK)
publication_id  uuid (FK -> publications)
recorded_at     date
citation_count  integer
download_count  integer
view_count      integer
altmetric_score decimal
source          enum('scopus', 'crossref', 'google_scholar', 'internal')
UNIQUE (publication_id, recorded_at, source)
```

#### مقاييس الباحثين (researcher_metrics) - سلسلة زمنية
```
id                  uuid (PK)
researcher_id       uuid (FK -> researchers)
recorded_at         date
h_index             integer
total_citations     integer
total_publications  integer
i10_index           integer
source              enum('scopus', 'google_scholar', 'calculated')
UNIQUE (researcher_id, recorded_at, source)
```

#### مقاييس الأقسام الشهرية (department_metrics_monthly)
```
id                  uuid (PK)
department_id       uuid (FK -> departments)
year                integer
month               integer
total_publications  integer
total_citations     integer
avg_h_index         decimal
q1_publications     integer
q2_publications     integer
open_access_count   integer
UNIQUE (department_id, year, month)
```

### جداول الذكاء الاصطناعي والنظام

#### تحليلات الذكاء الاصطناعي (ai_analyses)
```
id               uuid (PK)
analysis_type    enum('collaboration_rec', 'trend_analysis',
                      'publication_strategy', 'gap_analysis',
                      'funding_match')
target_type      enum('researcher', 'department', 'college', 'institution')
target_id        uuid
input_data       jsonb - البيانات المدخلة لـ DeepSeek
result           jsonb - نتائج التحليل
model_used       varchar
tokens_used      integer
confidence_score decimal
status           enum('pending', 'processing', 'completed', 'failed')
requested_by     uuid (FK -> users)
created_at       timestamptz
expires_at       timestamptz
```

#### المستخدمون (users)
```
id                    uuid (PK)
email                 varchar (unique)
password_hash         varchar
name_ar               varchar
name_en               varchar
role                  enum('admin', 'vp_research', 'dean_research',
                           'scientific_affairs', 'college_dean',
                           'center_director', 'department_head',
                           'researcher', 'viewer')
researcher_id         uuid (FK -> researchers)
institution_id        uuid (FK -> institutions)
college_id            uuid (FK -> colleges)
department_id         uuid (FK -> departments)
can_approve_promotions boolean (default false)
can_approve_budgets   boolean (default false)
preferred_language    enum('ar', 'en') (default 'ar')
last_login_at         timestamptz
created_at            timestamptz
```

**الأدوار التنظيمية:**
| الدور | الوصف | نطاق الرؤية |
|-------|-------|-------------|
| `admin` | مدير النظام | كامل |
| `vp_research` | المساعد العلمي / وكيل الجامعة للبحث | كل المؤسسة — قرارات استراتيجية |
| `dean_research` | عميد البحث العلمي | كل المؤسسة — إدارة تشغيلية |
| `scientific_affairs` | مدير الشؤون العلمية | كل المؤسسة — عمليات يومية |
| `college_dean` | عميد الكلية | كليته وأقسامها |
| `center_director` | مدير مركز بحثي | مركزه وأعضاؤه |
| `department_head` | رئيس القسم | قسمه وأعضاؤه |
| `researcher` | عضو هيئة التدريس / الباحث | ملفه الشخصي فقط |
| `viewer` | مشاهد | قراءة فقط حسب النطاق |

#### وظائف المزامنة (sync_jobs)
```
id              uuid (PK)
job_type        enum('scopus_sync', 'crossref_sync', 'scholar_sync',
                     'orcid_sync', 'metrics_update')
target_type     enum('researcher', 'publication', 'journal')
target_id       uuid
status          enum('queued', 'running', 'completed', 'failed', 'retrying')
attempts        integer (default 0)
max_attempts    integer (default 3)
error_message   text
started_at      timestamptz
completed_at    timestamptz
created_at      timestamptz
```

### جداول الشؤون العلمية وإدارة العمليات

#### طلبات الترقية العلمية (promotion_applications)
```
id                    uuid (PK)
researcher_id         uuid (FK -> researchers)
current_rank          enum('lecturer', 'assistant_prof', 'associate_prof', 'professor')
target_rank           enum('assistant_prof', 'associate_prof', 'professor')
application_date      date
q1_publications       integer - عدد أبحاث Q1
q2_publications       integer - عدد أبحاث Q2
total_publications    integer
h_index_at_apply      integer
supervised_phd        integer - عدد رسائل الدكتوراه المُشرف عليها
supervised_masters    integer
external_funding      decimal - إجمالي التمويل الخارجي
has_book              boolean - هل لديه كتاب منشور
years_in_rank         integer - سنوات في الرتبة الحالية
eligibility_score     decimal - درجة الاستيفاء المحسوبة تلقائياً
status                enum('draft', 'submitted', 'dept_review', 'college_review',
                           'scientific_affairs_review', 'vp_review',
                           'council_review', 'approved', 'rejected', 'returned')
rejection_reason      text
documents_url         text - رابط ملف المستندات
submitted_at          timestamptz
decided_at            timestamptz
created_at            timestamptz
updated_at            timestamptz
```

#### متطلبات الترقية (promotion_requirements)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
from_rank             enum('lecturer', 'assistant_prof', 'associate_prof')
to_rank               enum('assistant_prof', 'associate_prof', 'professor')
field_category        enum('sciences', 'engineering', 'humanities', 'medical', 'all')
min_q1_publications   integer
min_total_publications integer
min_h_index           integer
min_years_in_rank     integer
min_supervised_phd    integer
min_supervised_masters integer
requires_book         boolean
min_external_funding  decimal
additional_criteria   jsonb - معايير إضافية خاصة بالمؤسسة
created_at            timestamptz
updated_at            timestamptz
```

#### طلبات التفرغ العلمي (sabbatical_requests)
```
id                    uuid (PK)
researcher_id         uuid (FK -> researchers)
destination           varchar - الجهة المستضيفة
destination_country   varchar
research_plan         text - خطة البحث خلال التفرغ
start_date            date
end_date              date
status                enum('draft', 'submitted', 'dept_review', 'college_review',
                           'scientific_affairs_review', 'vp_review',
                           'approved', 'rejected', 'active', 'completed')
outputs_report        text - تقرير المخرجات بعد الانتهاء
publications_during   integer - أبحاث منشورة خلال التفرغ
productivity_before   decimal - معدل الإنتاجية قبل التفرغ
productivity_during   decimal - معدل الإنتاجية خلال التفرغ
productivity_after    decimal - معدل الإنتاجية بعد التفرغ
created_at            timestamptz
updated_at            timestamptz
```

#### طلبات المشاركة في المؤتمرات (conference_participations)
```
id                    uuid (PK)
researcher_id         uuid (FK -> researchers)
conference_name       varchar
conference_location   varchar
conference_country    varchar
conference_date_start date
conference_date_end   date
conference_url        text
paper_title           text - عنوان الورقة المقدمة
publication_id        uuid (FK -> publications) - ربط بالبحث إن وجد
participation_type    enum('presenter', 'keynote', 'attendee', 'organizer', 'chair')
is_international      boolean
estimated_cost        decimal - التكلفة التقديرية
approved_budget       decimal - الميزانية المعتمدة
status                enum('draft', 'submitted', 'dept_review', 'college_review',
                           'approved', 'rejected', 'attended', 'report_pending',
                           'completed')
post_report           text - تقرير ما بعد المشاركة
created_at            timestamptz
updated_at            timestamptz
```

#### مكافآت النشر (publication_incentives)
```
id                    uuid (PK)
publication_id        uuid (FK -> publications)
researcher_id         uuid (FK -> researchers)
journal_classification enum('Q1', 'Q2', 'Q3', 'Q4', 'scopus_only', 'other')
incentive_amount      decimal - مبلغ المكافأة
is_corresponding      boolean - هل هو المؤلف المراسل
author_share          decimal - نصيب الباحث (في حالة أبحاث مشتركة)
status                enum('pending_review', 'eligible', 'ineligible',
                           'approved', 'paid', 'rejected')
review_notes          text
reviewed_by           uuid (FK -> users)
approved_by           uuid (FK -> users)
payment_batch         varchar - رقم دفعة الصرف
paid_at               timestamptz
created_at            timestamptz
```

#### قواعد مكافآت النشر (incentive_rules)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
journal_classification enum('Q1', 'Q2', 'Q3', 'Q4', 'scopus_only')
base_amount           decimal - المبلغ الأساسي
corresponding_bonus   decimal - مكافأة إضافية للمؤلف المراسل
international_collab_bonus decimal - مكافأة التعاون الدولي
max_authors_share     integer - الحد الأقصى لتقسيم المكافأة
effective_from        date
effective_to          date
created_at            timestamptz
```

#### التقييم السنوي للأداء البحثي (annual_performance_reviews)
```
id                    uuid (PK)
researcher_id         uuid (FK -> researchers)
review_year           integer
q1_publications       integer
q2_publications       integer
total_publications    integer
citations_this_year   integer
h_index_current       integer
funding_acquired      decimal
students_supervised   integer
conferences_attended  integer
community_service     integer - أنشطة خدمة المجتمع
committees_membership integer - عضوية اللجان
weighted_score        decimal - الدرجة الموزونة
rank_in_department    integer - الترتيب في القسم
rank_in_college       integer - الترتيب في الكلية
reviewer_id           uuid (FK -> users) - رئيس القسم
reviewer_notes        text
dean_notes            text
status                enum('auto_generated', 'dept_reviewed', 'college_reviewed',
                           'finalized')
created_at            timestamptz
updated_at            timestamptz
```

#### أوزان التقييم (performance_criteria_weights)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
criterion             enum('q1_pubs', 'q2_pubs', 'other_pubs', 'citations',
                           'h_index', 'funding', 'supervision', 'conferences',
                           'community', 'committees')
weight_percentage     decimal - الوزن (المجموع = 100%)
effective_year        integer
created_at            timestamptz
```

#### الاتفاقيات والشراكات (agreements)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
partner_name_ar       varchar
partner_name_en       varchar
partner_country       varchar
agreement_type        enum('mou', 'research_collaboration', 'student_exchange',
                           'joint_supervision', 'industrial_partnership',
                           'funding_agreement')
title_ar              varchar
title_en              varchar
description           text
start_date            date
end_date              date
auto_renew            boolean
status                enum('draft', 'legal_review', 'pending_signature',
                           'active', 'expiring_soon', 'expired', 'terminated')
responsible_id        uuid (FK -> researchers) - المسؤول عن الاتفاقية
outputs_count         integer - عدد المخرجات (أبحاث مشتركة، تبادل، إلخ)
document_url          text
renewal_reminder_days integer (default 90)
created_at            timestamptz
updated_at            timestamptz
```

#### الكراسي البحثية (research_chairs)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
name_ar               varchar
name_en               varchar
sponsor_name          varchar - الجهة الراعية
chair_holder_id       uuid (FK -> researchers) - شاغل الكرسي
department_id         uuid (FK -> departments)
annual_budget         decimal
start_date            date
end_date              date
status                enum('active', 'vacant', 'expired', 'suspended')
total_publications    integer
total_patents         integer
total_funding         decimal
created_at            timestamptz
updated_at            timestamptz
```

#### المراكز البحثية (research_centers)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
name_ar               varchar
name_en               varchar
director_id           uuid (FK -> researchers)
focus_areas           jsonb - [{ar, en}]
annual_budget         decimal
status                enum('active', 'under_establishment', 'suspended', 'closed')
established_date      date
created_at            timestamptz
updated_at            timestamptz
```

#### أعضاء المراكز البحثية (research_center_members)
```
center_id             uuid (FK -> research_centers)
researcher_id         uuid (FK -> researchers)
role                  enum('director', 'co_director', 'principal_investigator',
                           'researcher', 'affiliate')
joined_at             date
PRIMARY KEY (center_id, researcher_id)
```

### جداول الاعتماد وضمان الجودة

#### شواهد الاعتماد (accreditation_evidence)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
accreditation_body    enum('ncaaa', 'abet', 'aacsb', 'qs', 'the', 'arwu', 'custom')
standard_code         varchar - رمز المعيار (مثل: 4.2.1)
standard_name_ar      varchar
standard_name_en      varchar
evidence_type         enum('report', 'data', 'screenshot', 'document', 'link', 'auto')
evidence_description  text
evidence_url          text - رابط الشاهد
auto_generated        boolean - هل مُولّد تلقائياً من النظام
college_id            uuid (FK -> colleges) - للاعتماد البرامجي
cycle_year            integer - سنة دورة الاعتماد
status                enum('draft', 'submitted', 'accepted', 'needs_update')
uploaded_by           uuid (FK -> users)
created_at            timestamptz
updated_at            timestamptz
```

#### توصيات الاعتماد السابقة (accreditation_recommendations)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
accreditation_body    enum('ncaaa', 'abet', 'aacsb', 'custom')
college_id            uuid (FK -> colleges)
visit_date            date - تاريخ الزيارة
recommendation_type   enum('observation', 'recommendation', 'weakness', 'strength')
recommendation_text   text
improvement_plan      text - خطة التحسين
responsible_id        uuid (FK -> users) - المسؤول عن التنفيذ
deadline              date
completion_percentage integer (default 0)
evidence_of_action    text - شاهد التنفيذ
status                enum('open', 'in_progress', 'completed', 'overdue', 'closed')
created_at            timestamptz
updated_at            timestamptz
```

#### مؤشرات الأداء الاستراتيجية (strategic_kpis)
```
id                    uuid (PK)
institution_id        uuid (FK -> institutions)
kpi_name_ar           varchar
kpi_name_en           varchar
category              enum('research_output', 'research_quality', 'funding',
                           'collaboration', 'innovation', 'community_impact',
                           'student_research', 'sdg')
target_value          decimal - المستهدف
minimum_value         decimal - الحد الأدنى المقبول
excellence_value      decimal - مستوى التميز
actual_value          decimal - القيمة الفعلية (تُحدّث تلقائياً)
unit                  varchar - الوحدة (عدد، نسبة، مبلغ)
measurement_period    enum('monthly', 'quarterly', 'semester', 'annual')
strategic_plan_ref    varchar - مرجع الخطة الاستراتيجية
data_source           enum('auto_calculated', 'manual', 'mixed')
scope_type            enum('institution', 'college', 'department')
scope_id              uuid - معرف الجهة
year                  integer
created_at            timestamptz
updated_at            timestamptz
```

#### سير عمل الموافقات (approval_workflows)
```
id                    uuid (PK)
workflow_type         enum('promotion', 'sabbatical', 'conference', 'incentive',
                           'agreement', 'research_project')
request_id            uuid - معرف الطلب (polymorphic)
step_order            integer - ترتيب الخطوة
step_name             varchar - اسم الخطوة
approver_role         enum('department_head', 'college_dean', 'scientific_affairs',
                           'dean_research', 'vp_research', 'council')
approver_id           uuid (FK -> users)
decision              enum('pending', 'approved', 'rejected', 'returned')
comments              text
decided_at            timestamptz
created_at            timestamptz
```

---

## 4. مراحل التنفيذ

### المرحلة 1: الأساس (الأسابيع 1-3)

**الهدف**: هيكل المشروعين (frontend + backend)، قاعدة البيانات، المصادقة، CRUD الأساسي

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 1.1 | تهيئة Go Backend | إنشاء مشروع Go مع Chi router, sqlc, migrations | منخفض |
| 1.2 | تهيئة Next.js Frontend | إنشاء مشروع Next.js مع TypeScript, Tailwind, Shadcn/ui | منخفض |
| 1.3 | إعداد RTL والترجمة | تهيئة next-intl مع العربية كلغة افتراضية، خطوط Cairo و Inter | متوسط |
| 1.4 | قاعدة البيانات | كتابة SQL migrations + توليد sqlc code | منخفض |
| 1.5 | المصادقة | JWT auth في Go + RBAC middleware | متوسط |
| 1.6 | هيكل الواجهة | Sidebar, Header, App Shell مع Shadcn/ui | منخفض |
| 1.7 | إدارة الباحثين | Go API + Next.js pages (قائمة، ملف شخصي، إضافة/تعديل) | منخفض |
| 1.8 | إدارة المنشورات | Go API + Next.js pages (قائمة، تفاصيل، إدخال يدوي مع DOI) | منخفض |
| 1.9 | الهيكل التنظيمي | إدارة المؤسسة، الكليات، الأقسام | منخفض |
| 1.10 | Docker Compose | بيئة تطوير كاملة (Go + Next.js + PostgreSQL + Redis) | منخفض |

**المخرجات**: نظام عامل لإدارة الباحثين والأبحاث مع واجهة عربية

---

### المرحلة 2: ربط واجهات برمجة التطبيقات الخارجية (الأسابيع 4-6)

**الهدف**: استيراد تلقائي للبيانات من Scopus, CrossRef, ORCID, OpenAlex

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 2.1 | بنية عملاء API في Go | HTTP client مع retry, rate limiting, circuit breaker | متوسط |
| 2.2 | ربط Scopus | استيراد الأبحاث، الاستشهادات، h-index (goroutine worker) | **عالي** |
| 2.3 | ربط CrossRef | استيراد بيانات DOI، بيانات المنشورات | منخفض |
| 2.4 | ربط ORCID | استيراد ملفات الباحثين، قائمة الأعمال | منخفض |
| 2.5 | ربط OpenAlex | بديل مجاني لـ Scopus — بيانات أكاديمية مفتوحة | منخفض |
| 2.6 | تنسيق المزامنة | إزالة التكرار، دمج البيانات، goroutine scheduler | **عالي** |
| 2.7 | استيراد بيانات المجلات | تصنيفات SJR (Q1-Q4)، Impact Factor | منخفض |

> **ميزة Go هنا**: مزامنة 10+ APIs بالتوازي باستخدام goroutines + semaphore — بدون BullMQ أو Redis queues

#### تفاصيل واجهات برمجة التطبيقات

| API | المصادقة | حد الطلبات | التكلفة |
|-----|----------|-----------|---------|
| Scopus (Elsevier) | مفتاح API مؤسسي | 6 طلب/ثانية | مجاني مع اشتراك مؤسسي |
| CrossRef | بدون (mailto فقط) | 50 طلب/ثانية | مجاني |
| ORCID | بدون (عام) | 24 طلب/ثانية | مجاني |
| OpenAlex | بدون (mailto فقط) | 100 طلب/ثانية | مجاني بالكامل |
| SerpAPI (Scholar) | مفتاح API | 1 طلب/ثانية | $50/شهر لـ 5000 بحث |

**المخرجات**: استيراد تلقائي للبيانات من المصادر الأكاديمية

---

### المرحلة 3: المقاييس ولوحات المتابعة (الأسابيع 7-9)

**الهدف**: مقاييس آنية وتاريخية على جميع المستويات التنظيمية

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 3.1 | خدمة جمع المقاييس | Go goroutine لقطات يومية + تجميعات شهرية | متوسط |
| 3.2 | لوحة الباحث | h-index، استشهادات، أبحاث/سنة، توزيع المجلات، شبكة المؤلفين | متوسط |
| 3.3 | لوحة القسم | إجمالي الأبحاث، ترتيب الباحثين، تغطية المجالات | منخفض |
| 3.4 | لوحة الكلية | مقارنة الأقسام، أفضل الباحثين، كثافة التعاون | منخفض |
| 3.5 | لوحة المؤسسة | KPIs، مقارنة الكليات، اتجاهات 5 سنوات، خريطة التعاون الدولي | متوسط |
| 3.6 | التصدير والتقارير | CSV، PDF (مع دعم العربية)، تقارير بريدية | متوسط |

#### محتويات لوحة الباحث
- اتجاه h-index عبر الزمن
- نمو الاستشهادات
- عدد الأبحاث لكل سنة
- توزيع تصنيف المجلات (Q1/Q2/Q3/Q4)
- شبكة المؤلفين المشاركين (رسم بياني تفاعلي)
- أكثر الأبحاث استشهاداً

#### محتويات لوحة المؤسسة
- إجمالي الباحثين / الأبحاث / الاستشهادات
- مقارنة الكليات
- اتجاه النشر (5 سنوات)
- نسبة أبحاث Q1
- خريطة التعاون الدولي
- أفضل 10 باحثين
- آخر الأبحاث المنشورة

**المخرجات**: لوحات متابعة تفاعلية على كل المستويات

---

### المرحلة 4: ميزات الذكاء الاصطناعي (الأسابيع 10-13)

**الهدف**: ربط DeepSeek API لتحليلات ذكية ومقترحات

| # | المهمة | الوصف | النموذج | المخاطر |
|---|--------|-------|---------|---------|
| 4.1 | بنية خدمة AI | عميل DeepSeek في Go (OpenAI-compatible)، prompt templates | - | متوسط |
| 4.2 | توصيات التعاون | اقتراح باحثين للتعاون بناءً على التقارب البحثي | DeepSeek-V3 | متوسط |
| 4.3 | المواضيع الرائجة | تحليل اتجاهات البحث في المؤسسة | DeepSeek-V3 | منخفض |
| 4.4 | استراتيجية النشر | نصائح مخصصة لكل باحث (مجلات مستهدفة، مواضيع) | DeepSeek-V3 | منخفض |
| 4.5 | تحليل الفجوات | مجالات ناقصة التمثيل، فرص بحثية جديدة | DeepSeek-R1 | متوسط |
| 4.6 | مطابقة التمويل | ربط الباحثين بفرص التمويل المناسبة | DeepSeek-V3 | **عالي** |
| 4.7 | لوحة التحليلات الموحدة | واجهة Next.js موحدة لعرض جميع تحليلات AI | - | منخفض |

#### تقدير تكلفة الذكاء الاصطناعي

**تسعير DeepSeek:**
- **DeepSeek-V3**: $0.27/MTok input, $1.10/MTok output (مع cache: $0.07/MTok input)
- **DeepSeek-R1**: $0.55/MTok input, $2.19/MTok output (مع cache: $0.14/MTok input)

| نوع التحليل | الرموز المدخلة | الرموز المخرجة | النموذج | التكلفة/تشغيل |
|-------------|---------------|---------------|---------|--------------|
| توصيات التعاون | ~4,000 | ~1,500 | DeepSeek-V3 | ~$0.003 |
| المواضيع الرائجة | ~8,000 | ~2,000 | DeepSeek-V3 | ~$0.004 |
| استراتيجية النشر | ~3,000 | ~2,000 | DeepSeek-V3 | ~$0.003 |
| تحليل الفجوات | ~10,000 | ~3,000 | DeepSeek-R1 | ~$0.012 |
| مطابقة التمويل | ~2,000 | ~1,000 | DeepSeek-V3 | ~$0.002 |

> **التوفير**: تكلفة تحليل كامل ~$0.024 (توفير ~93% مقارنة بـ Claude)

**المخرجات**: نظام ذكاء اصطناعي يقدم نصائح ومقترحات بحثية مخصصة

---

### المرحلة 5: التحسين والإنتاج (الأسابيع 14-16)

| # | المهمة | الوصف |
|---|--------|-------|
| 5.1 | البحث والاكتشاف | PostgreSQL full-text search (عربي + إنجليزي) مع pg_trgm |
| 5.2 | نظام الإشعارات | إشعارات داخلية + بريد إلكتروني |
| 5.3 | تحسين الأداء | Redis caching، Go connection pooling، query optimization |
| 5.4 | مراجعة RTL والوصولية | WCAG 2.1 AA، اختبار قارئ الشاشة العربي |

**المخرجات**: نظام جاهز للإنتاج مع بحث وإشعارات وأداء محسّن

---

### المرحلة 6: إدارة الشؤون العلمية (الأسابيع 17-21)

**الهدف**: تحويل النظام من أداة رصد إلى منصة عمليات يومية لإدارة الشؤون العلمية

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 6.1 | إدارة الترقيات العلمية | متطلبات كل رتبة، فحص الأهلية التلقائي، لوحة جاهزية الترقية لكل باحث | **عالي** |
| 6.2 | سير عمل الترقية | تدفق الطلب: باحث → قسم → كلية → شؤون علمية → المساعد العلمي → مجلس الجامعة | **عالي** |
| 6.3 | مكافآت النشر | قواعد مكافآت قابلة للتخصيص حسب تصنيف المجلة، حساب تلقائي، جدولة الصرف | متوسط |
| 6.4 | التقييم السنوي للأداء | بطاقة أداء تلقائية لكل باحث بمعايير موزونة، ترتيب على مستوى القسم والكلية | متوسط |
| 6.5 | إدارة التفرغ العلمي | طلب تفرغ مع خطة بحثية، سير عمل موافقات، تقييم العائد بعد التفرغ | منخفض |
| 6.6 | إدارة المؤتمرات | طلب مشاركة، موافقات، ميزانية، تقرير ما بعد المشاركة | منخفض |
| 6.7 | الكراسي والمراكز البحثية | سجل الكراسي البحثية والمراكز، أعضاء، ميزانيات، مخرجات | منخفض |
| 6.8 | الاتفاقيات والشراكات | سجل اتفاقيات، تتبع حالة، تنبيهات تجديد، مخرجات كل شراكة | منخفض |

#### تفاصيل نظام الترقيات العلمية

```
تدفق طلب الترقية:

  الباحث              رئيس القسم           عميد الكلية
  ┌──────────┐       ┌──────────────┐      ┌─────────────┐
  │ يتحقق من │──────→│ يراجع الملف  │─────→│ يوافق أو    │
  │ الأهلية  │       │ + يوافق/يرفض │      │ يعيد للقسم  │
  │ (تلقائي) │       └──────────────┘      └──────┬──────┘
  └──────────┘                                     │
                                                   ▼
  مجلس الجامعة       المساعد العلمي          الشؤون العلمية
  ┌──────────┐       ┌──────────────┐      ┌──────────────┐
  │ القرار   │←──────│ يعتمد        │←─────│ تتحقق من     │
  │ النهائي  │       │ التوصية      │      │ الشروط + ترفع│
  └──────────┘       └──────────────┘      └──────────────┘
```

#### تفاصيل نظام مكافآت النشر

```
تدفق مكافأة النشر:

  بحث جديد يُنشر         النظام (تلقائي)         الشؤون العلمية
  ┌──────────────┐       ┌──────────────┐        ┌──────────────┐
  │ يُكتشف عبر  │──────→│ يتحقق:       │───────→│ تراجع        │
  │ المزامنة    │       │ - المجلة آمنة?│        │ + تعتمد      │
  └──────────────┘       │ - التصنيف?   │        │ المكافأة     │
                         │ - المبلغ?    │        └──────┬───────┘
                         └──────────────┘               │
                                                  المساعد العلمي
                                                  ┌──────▼───────┐
                                                  │ يعتمد الصرف  │
                                                  │ (دفعة شهرية) │
                                                  └──────────────┘
```

**المخرجات**: نظام إدارة عمليات يومية كامل لقسم الشؤون العلمية

---

### المرحلة 7: الاعتماد وضمان الجودة (الأسابيع 22-25)

**الهدف**: تحويل النظام إلى مصدر أساسي لبيانات وشواهد الاعتماد الأكاديمي

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 7.1 | سجل الأدلة والشواهد | مستودع شواهد مُهيكل، كل معيار اعتماد مربوط بأدلته تلقائياً | متوسط |
| 7.2 | متابعة توصيات الاعتماد | تسجيل توصيات الزيارات السابقة، خطط تحسين، نسب إنجاز | متوسط |
| 7.3 | مؤشرات الأداء الاستراتيجية | KPIs مخصصة مربوطة بالخطة الاستراتيجية، مستهدفات، لوحة الفجوة | متوسط |
| 7.4 | تقرير التقييم الذاتي | قوالب جاهزة لـ NCAAA/ABET/AACSB، تعبئة تلقائية، مراجعة AI | **عالي** |
| 7.5 | ربط البحث بالتدريس | تتبع دمج الأبحاث في المقررات، مشاريع طلاب بحثية | منخفض |
| 7.6 | لوحة الاعتماد | لوحة شاملة لكل جهة اعتماد: نسبة الاستيفاء، الفجوات، المواعيد | منخفض |

#### تفاصيل سجل الأدلة والشواهد

```
سجل الشواهد لمعيار NCAAA 4.2 (البحث العلمي):

  معيار 4.2.1: إنتاجية البحث
  ├── 📊 شاهد تلقائي: تقرير الأبحاث المنشورة (مُولّد من النظام)
  ├── 📊 شاهد تلقائي: إحصائيات h-index و الاستشهادات
  ├── 📄 شاهد يدوي: محاضر لجنة البحث العلمي
  └── 🔗 رابط: سياسة البحث العلمي المعتمدة

  معيار 4.2.2: جودة البحث
  ├── 📊 شاهد تلقائي: توزيع الأبحاث Q1/Q2/Q3/Q4
  ├── 📊 شاهد تلقائي: تقرير النزاهة البحثية
  └── 📄 شاهد يدوي: تقارير المراجعة الخارجية

  الحالة الإجمالية: ████████████████░░░░  82% مكتمل
  فجوات: 3 شواهد ناقصة ⚠️
```

#### تفاصيل لوحة مؤشرات الأداء الاستراتيجية

```
╔══════════════════════════════════════════════════════════════╗
║  مؤشرات الخطة الاستراتيجية — البحث العلمي                   ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  │ المؤشر              │ الحد الأدنى │ المستهدف │ الفعلي │   ║
║  │ أبحاث Q1/Q2 سنوياً  │    200     │   400   │  312  │ 🟡 ║
║  │ تمويل خارجي (م.ر.)  │    10M     │   25M   │  18M  │ 🟡 ║
║  │ نسبة التعاون الدولي  │    30%     │   50%   │  45%  │ 🟢 ║
║  │ براءات اختراع        │     10     │    20   │   12  │ 🟡 ║
║  │ متوسط h-index        │     8      │    13   │ 11.2  │ 🟢 ║
║  │ تغطية SDGs          │   10/17    │  15/17  │ 12/17 │ 🟡 ║
║                                                              ║
║  📈 اتجاه 3 دورات:                                          ║
║  2022: ██████████ 65%                                        ║
║  2023: ████████████ 72%                                      ║
║  2024: ██████████████ 78% ↑ تحسن مستمر ✅                   ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
```

**المخرجات**: نظام اعتماد جاهز يوفر أسابيع من العمل اليدوي في كل دورة اعتماد

---

### المرحلة 8: سير العمل متعدد المستويات والحسابات (الأسابيع 26-29)

**الهدف**: لوحات مخصصة لكل دور + محرك موافقات إلكتروني يربط جميع المستويات

| # | المهمة | الوصف | المخاطر |
|---|--------|-------|---------|
| 8.1 | لوحة الباحث | ملخص أداء، جاهزية الترقية، مهام معلقة، اقتراحات AI، إجراءات سريعة | متوسط |
| 8.2 | لوحة رئيس القسم | مؤشرات القسم، أعضاء يحتاجون انتباه، طلبات معلقة، مقارنة أقسام | متوسط |
| 8.3 | لوحة عميد الكلية | مقارنة الأقسام، طلبات مُصعّدة، معايير الاعتماد، تقارير | متوسط |
| 8.4 | لوحة الشؤون العلمية | ملفات ترقيات، مكافآت نشر، تقييم أداء، اتفاقيات، اعتماد | **عالي** |
| 8.5 | لوحة المساعد العلمي | مؤشرات استراتيجية، ترتيب كليات، تنبيهات قرار، مقارنة جامعات، رؤى AI | **عالي** |
| 8.6 | محرك الموافقات | سير عمل قابل للتخصيص، إشعارات، تصعيد تلقائي عند التأخير، سجل قرارات | **عالي** |
| 8.7 | الإشعارات الذكية | إشعارات مخصصة لكل دور (بريد + داخلي)، تذكيرات مواعيد، تنبيهات أداء | متوسط |

#### محتويات لوحة كل دور

| الدور | يرى | يفعل |
|-------|------|------|
| **الباحث** | أداؤه، جاهزية ترقيته، مهامه | يقدم طلبات، يحدث ملفه، يفحص مجلات |
| **رئيس القسم** | أداء قسمه، باحثون متأخرون/متميزون | يوافق طلبات، يقيّم أداء، يقارن بأقسام |
| **عميد الكلية** | مقارنة أقسام، معايير اعتماد | يوافق مُصعّدات، يولّد تقارير |
| **الشؤون العلمية** | كل العمليات اليومية | يدير ترقيات، مكافآت، تقييم، شراكات |
| **المساعد العلمي** | الصورة الاستراتيجية | يعتمد قرارات، يرى مقارنات، يقرر |

#### محرك الموافقات المتعدد المستويات

```
محرك الموافقات العام:

  ┌─────────────┐     ┌──────────────┐     ┌──────────────┐
  │  طلب جديد   │────→│ الخطوة 1     │────→│ الخطوة 2     │──→ ...
  │  (أي نوع)   │     │ (رئيس القسم) │     │ (عميد الكلية)│
  └─────────────┘     └──────┬───────┘     └──────┬───────┘
                             │                     │
                      ┌──────▼───────┐      ┌──────▼───────┐
                      │ موافقة ✅    │      │ موافقة ✅    │
                      │ رفض ❌       │      │ رفض ❌       │
                      │ إعادة 🔄     │      │ إعادة 🔄     │
                      └──────────────┘      └──────────────┘

  قواعد التصعيد التلقائي:
  ├── إذا لم يُتخذ قرار خلال 7 أيام → تنبيه
  ├── إذا لم يُتخذ قرار خلال 14 يوم → تصعيد للمستوى الأعلى
  └── جميع القرارات مسجلة مع التاريخ والتعليقات
```

**المخرجات**: منصة جامعية متكاملة يدخلها كل تدريسي ومسؤول يومياً

---

## 5. هيكل المجلدات

```
turath-research/
├── frontend/                        # Next.js Frontend
│   ├── src/
│   │   ├── app/
│   │   │   └── [locale]/
│   │   │       ├── layout.tsx
│   │   │       ├── page.tsx
│   │   │       ├── dashboard/page.tsx
│   │   │       ├── researchers/
│   │   │       │   ├── page.tsx
│   │   │       │   ├── new/page.tsx
│   │   │       │   └── [id]/
│   │   │       │       ├── page.tsx
│   │   │       │       ├── dashboard/page.tsx
│   │   │       │       └── insights/
│   │   │       ├── publications/
│   │   │       │   ├── page.tsx
│   │   │       │   └── [id]/page.tsx
│   │   │       ├── departments/[id]/dashboard/
│   │   │       ├── colleges/[id]/dashboard/
│   │   │       ├── insights/
│   │   │       │   ├── trends/page.tsx
│   │   │       │   └── gaps/page.tsx
│   │   │       ├── search/page.tsx
│   │   │       ├── promotions/              # إدارة الترقيات
│   │   │       │   ├── page.tsx             # قائمة الترقيات
│   │   │       │   ├── eligibility/page.tsx # فحص الأهلية
│   │   │       │   └── [id]/page.tsx        # تفاصيل طلب ترقية
│   │   │       ├── incentives/              # مكافآت النشر
│   │   │       │   ├── page.tsx             # قائمة المكافآت
│   │   │       │   └── rules/page.tsx       # قواعد المكافآت
│   │   │       ├── performance/             # التقييم السنوي
│   │   │       │   ├── page.tsx             # لوحة التقييم
│   │   │       │   └── [id]/page.tsx        # تقييم باحث
│   │   │       ├── sabbaticals/             # التفرغ العلمي
│   │   │       │   ├── page.tsx
│   │   │       │   └── [id]/page.tsx
│   │   │       ├── conferences/             # المؤتمرات
│   │   │       │   ├── page.tsx
│   │   │       │   └── [id]/page.tsx
│   │   │       ├── agreements/              # الاتفاقيات والشراكات
│   │   │       │   ├── page.tsx
│   │   │       │   └── [id]/page.tsx
│   │   │       ├── accreditation/           # الاعتماد وضمان الجودة
│   │   │       │   ├── page.tsx             # لوحة الاعتماد
│   │   │       │   ├── evidence/page.tsx    # سجل الشواهد
│   │   │       │   ├── recommendations/page.tsx # توصيات سابقة
│   │   │       │   ├── kpis/page.tsx        # مؤشرات الأداء
│   │   │       │   └── self-study/page.tsx  # تقرير التقييم الذاتي
│   │   │       ├── centers/                 # المراكز والكراسي البحثية
│   │   │       │   ├── page.tsx
│   │   │       │   └── [id]/page.tsx
│   │   │       └── admin/
│   │   │           ├── institutions/
│   │   │           ├── colleges/
│   │   │           ├── departments/
│   │   │           ├── sync/page.tsx
│   │   │           └── workflow-settings/page.tsx  # إعدادات سير العمل
│   │   ├── components/
│   │   │   ├── ui/                  # Shadcn components
│   │   │   ├── layout/              # Sidebar, Header, Shell
│   │   │   ├── charts/              # Recharts/Nivo wrappers
│   │   │   ├── dashboard/           # Dashboard components (per role)
│   │   │   ├── promotions/          # Promotion workflow components
│   │   │   ├── performance/         # Performance review components
│   │   │   ├── accreditation/       # Accreditation components
│   │   │   ├── workflow/            # Approval workflow components
│   │   │   ├── ai/                  # AI insight components
│   │   │   └── notifications/
│   │   ├── lib/
│   │   │   ├── api-client.ts        # Go backend HTTP client
│   │   │   └── auth.ts              # JWT token management
│   │   ├── i18n/
│   │   ├── messages/                # ar.json, en.json
│   │   └── types/                   # TypeScript types (mirror Go types)
│   ├── e2e/                         # Playwright tests
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   └── package.json
│
├── backend/                         # Go Backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # Entry point
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go            # Environment configuration
│   │   ├── server/
│   │   │   ├── server.go            # HTTP server setup
│   │   │   └── routes.go            # Route registration
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT authentication
│   │   │   ├── rbac.go              # Role-based access control
│   │   │   ├── ratelimit.go         # Rate limiting
│   │   │   └── cors.go              # CORS for Next.js
│   │   ├── handler/                 # HTTP handlers (controllers)
│   │   │   ├── researcher.go
│   │   │   ├── publication.go
│   │   │   ├── metrics.go
│   │   │   ├── ai.go
│   │   │   ├── sync.go
│   │   │   ├── auth.go
│   │   │   ├── promotion.go        # طلبات الترقية
│   │   │   ├── incentive.go        # مكافآت النشر
│   │   │   ├── performance.go      # التقييم السنوي
│   │   │   ├── sabbatical.go       # التفرغ العلمي
│   │   │   ├── conference.go       # المؤتمرات
│   │   │   ├── agreement.go        # الاتفاقيات
│   │   │   ├── accreditation.go    # الاعتماد
│   │   │   └── workflow.go         # سير عمل الموافقات
│   │   ├── service/                 # Business logic
│   │   │   ├── researcher.go
│   │   │   ├── publication.go
│   │   │   ├── metrics.go
│   │   │   ├── ai/
│   │   │   │   ├── deepseek.go      # DeepSeek client
│   │   │   │   ├── prompts/         # Prompt templates
│   │   │   │   ├── collaboration.go
│   │   │   │   ├── trends.go
│   │   │   │   ├── strategy.go
│   │   │   │   └── gap_analysis.go
│   │   │   ├── sync/
│   │   │   │   ├── orchestrator.go
│   │   │   │   ├── scopus.go
│   │   │   │   ├── crossref.go
│   │   │   │   ├── orcid.go
│   │   │   │   └── openalex.go
│   │   │   ├── promotion/           # إدارة الترقيات العلمية
│   │   │   │   ├── eligibility.go   # فحص الأهلية التلقائي
│   │   │   │   └── requirements.go  # متطلبات كل رتبة
│   │   │   ├── incentive/           # مكافآت النشر
│   │   │   │   ├── calculator.go    # حساب المكافأة تلقائياً
│   │   │   │   └── rules.go        # قواعد المكافآت
│   │   │   ├── performance/         # التقييم السنوي
│   │   │   │   ├── evaluator.go     # حساب الدرجة الموزونة
│   │   │   │   └── ranking.go      # ترتيب الباحثين
│   │   │   ├── sabbatical/          # التفرغ العلمي
│   │   │   ├── conference/          # المؤتمرات
│   │   │   ├── agreement/           # الاتفاقيات والشراكات
│   │   │   ├── accreditation/       # الاعتماد وضمان الجودة
│   │   │   │   ├── evidence.go      # سجل الشواهد
│   │   │   │   ├── recommendations.go # توصيات سابقة
│   │   │   │   ├── kpi.go          # مؤشرات الأداء الاستراتيجية
│   │   │   │   └── self_study.go   # توليد تقرير التقييم الذاتي
│   │   │   ├── workflow/            # محرك الموافقات
│   │   │   │   ├── engine.go       # محرك سير العمل
│   │   │   │   ├── escalation.go   # التصعيد التلقائي
│   │   │   │   └── notification.go # إشعارات الموافقات
│   │   │   └── integrity/           # Research integrity (Phase 17)
│   │   │       ├── citation_analysis.go
│   │   │       ├── plagiarism.go
│   │   │       └── predatory.go
│   │   ├── repository/              # Database access (sqlc generated + custom)
│   │   │   └── queries/             # SQL query files for sqlc
│   │   ├── worker/                  # Background goroutine workers
│   │   │   ├── scheduler.go
│   │   │   ├── sync_worker.go
│   │   │   ├── metrics_worker.go
│   │   │   └── ai_worker.go
│   │   ├── apiclient/               # External API clients
│   │   │   ├── base.go              # Shared HTTP client with retry
│   │   │   ├── scopus.go
│   │   │   ├── crossref.go
│   │   │   ├── orcid.go
│   │   │   └── openalex.go
│   │   └── model/                   # Domain models
│   │       ├── researcher.go
│   │       ├── publication.go
│   │       └── metrics.go
│   ├── db/
│   │   └── migrations/              # SQL migration files
│   ├── sqlc.yaml                    # sqlc configuration
│   ├── go.mod
│   └── go.sum
│
├── docker-compose.yml               # Coolify: 4 services منفصلة
├── frontend/Dockerfile              # Next.js container
├── backend/Dockerfile               # Go container
├── Makefile                         # Build, test, migrate commands
├── .env.example
└── docs/
    ├── PLAN.md
    └── ENHANCEMENTS.md
```

---

## 6. النشر على Coolify (Deployment)

### البنية: 4 حاويات Docker منفصلة

عند الاستيراد من GitHub في Coolify، ينشئ النظام 4 حاويات منفصلة تتواصل عبر شبكة Docker داخلية:

```
┌─────────────────────────────────────────────────────────┐
│                    Coolify Server                        │
│                                                         │
│  ┌─────────────┐    ┌─────────────┐                    │
│  │  frontend   │    │   backend   │                    │
│  │  (Next.js)  │←──→│    (Go)     │                    │
│  │  Port 3000  │    │  Port 8080  │                    │
│  │             │    │             │                    │
│  │ Dockerfile  │    │ Dockerfile  │                    │
│  └──────┬──────┘    └──────┬──────┘                    │
│         │                  │                            │
│         │    ┌─────────────┼────────────┐               │
│         │    ▼             ▼            │               │
│         │  ┌─────────────┐ ┌─────────┐ │               │
│         │  │ postgresql  │ │  redis  │ │               │
│         │  │  Port 5432  │ │Port 6379│ │               │
│         │  │             │ │         │ │               │
│         │  │  Volume:    │ │ Volume: │ │               │
│         │  │  pgdata     │ │ redisdata│ │              │
│         │  └─────────────┘ └─────────┘ │               │
│         │                              │               │
│  ┌──────▼──────────────────────────────▼──────┐        │
│  │          Docker Network: turath-net          │        │
│  └──────────────────────────────────────────────┘        │
│                                                         │
│  ┌─────────────────────────────────────────────┐        │
│  │  Coolify Proxy (Traefik/Caddy)              │        │
│  │  turath.example.com → frontend:3000          │        │
│  │  turath.example.com/api/* → backend:8080     │        │
│  │  SSL/TLS تلقائي عبر Let's Encrypt           │        │
│  └─────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────┘
```

### docker-compose.yml (Coolify يقرأه تلقائياً)

```yaml
services:
  # ── PostgreSQL ──────────────────────────────────
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-turath}
      POSTGRES_USER: ${POSTGRES_USER:-turath}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?required}
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - turath-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-turath}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ── Redis ───────────────────────────────────────
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redisdata:/data
    networks:
      - turath-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ── Go Backend ──────────────────────────────────
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    restart: unless-stopped
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER:-turath}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB:-turath}?sslmode=disable
      REDIS_URL: redis://redis:6379
      SERVER_PORT: "8080"
      FRONTEND_URL: ${FRONTEND_URL:-http://frontend:3000}
      JWT_SECRET: ${JWT_SECRET:?required}
      JWT_EXPIRY: ${JWT_EXPIRY:-24h}
      DEEPSEEK_API_KEY: ${DEEPSEEK_API_KEY}
      DEEPSEEK_BASE_URL: ${DEEPSEEK_BASE_URL:-https://api.deepseek.com}
      SCOPUS_API_KEY: ${SCOPUS_API_KEY}
      CROSSREF_MAILTO: ${CROSSREF_MAILTO}
      ORCID_CLIENT_ID: ${ORCID_CLIENT_ID}
      ORCID_CLIENT_SECRET: ${ORCID_CLIENT_SECRET}
      SERPAPI_KEY: ${SERPAPI_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - turath-net
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/api/v1/health"]
      interval: 15s
      timeout: 5s
      retries: 3

  # ── Next.js Frontend ────────────────────────────
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL:-http://backend:8080}
    restart: unless-stopped
    environment:
      NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL:-/api}
    depends_on:
      backend:
        condition: service_healthy
    networks:
      - turath-net

volumes:
  pgdata:
    driver: local
  redisdata:
    driver: local

networks:
  turath-net:
    driver: bridge
```

### backend/Dockerfile

```dockerfile
# ── Build Stage ───────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# ── Run Stage ─────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /server /server
COPY --from=builder /app/db/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/server"]
```

### frontend/Dockerfile

```dockerfile
# ── Dependencies ──────────────────────────────────
FROM node:20-alpine AS deps

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts

# ── Build ─────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

RUN npm run build

# ── Run ───────────────────────────────────────────
FROM node:20-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
```

### خطوات النشر على Coolify

```
1. ادفع الكود لـ GitHub
2. في Coolify: New Resource → Docker Compose
3. اربط بـ GitHub repository
4. Coolify يكتشف docker-compose.yml تلقائياً
5. أضف متغيرات البيئة في Coolify:
   - POSTGRES_PASSWORD (مطلوب)
   - JWT_SECRET (مطلوب)
   - DEEPSEEK_API_KEY
   - SCOPUS_API_KEY
   - ... باقي المتغيرات
6. اضبط الدومين في Coolify:
   - frontend → turath.example.com
   - backend → turath.example.com (path: /api/*)
7. Deploy — Coolify يبني 4 حاويات تلقائياً
```

### ميزات Coolify التلقائية
- **SSL/TLS** تلقائي عبر Let's Encrypt
- **Auto-deploy** عند Push لـ GitHub (webhook)
- **Rollback** سهل لأي نسخة سابقة
- **Logs** لكل حاوية منفصلة
- **Monitoring** استهلاك CPU/RAM لكل حاوية
- **Backup** تلقائي لـ PostgreSQL volumes

---

## 7. متغيرات البيئة المطلوبة

```env
# ── مطلوبة (Coolify يحقنها في الحاويات) ──────────
POSTGRES_DB=turath
POSTGRES_USER=turath
POSTGRES_PASSWORD=               # مطلوب — Coolify يولده تلقائياً
JWT_SECRET=                      # مطلوب — Coolify يولده تلقائياً

# ── الخادم ─────────────────────────────────────────
SERVER_PORT=8080
FRONTEND_URL=https://turath.example.com
NEXT_PUBLIC_API_URL=/api

# ── APIs الأكاديمية ────────────────────────────────
SCOPUS_API_KEY=
CROSSREF_MAILTO=admin@university.edu
SERPAPI_KEY=

# ── ORCID ──────────────────────────────────────────
ORCID_CLIENT_ID=
ORCID_CLIENT_SECRET=

# ── الذكاء الاصطناعي (DeepSeek) ───────────────────
DEEPSEEK_API_KEY=
DEEPSEEK_BASE_URL=https://api.deepseek.com

# ── تلقائية (لا تغيرها — Docker network) ──────────
# DATABASE_URL=postgresql://turath:***@postgres:5432/turath?sslmode=disable
# REDIS_URL=redis://redis:6379
```

---

## 7. استراتيجية الاختبار

### Backend (Go)
| النوع | الأداة | التغطية | الأولوية |
|------|--------|---------|---------|
| وحدة (Unit) | `go test` + testify | 80%+ | Services, Handlers, API Clients |
| تكامل (Integration) | testcontainers-go | - | Database queries, API sync |
| Benchmark | `go test -bench` | - | Citation analysis, metrics aggregation |

### Frontend (Next.js)
| النوع | الأداة | التغطية | الأولوية |
|------|--------|---------|---------|
| وحدة (Unit) | Vitest + Testing Library | 80%+ | Components, hooks, utilities |
| شامل (E2E) | Playwright | - | تسجيل الدخول، الملف الشخصي، البحث، اللوحات، RTL |

---

## 8. المخاطر وخطط التخفيف

### مخاطر المراحل الأساسية (1-5)

| المخاطر | الشدة | خطة التخفيف |
|---------|-------|------------|
| تأخر الحصول على مفتاح Scopus | عالي | البدء بـ OpenAlex + CrossRef + ORCID (مجانية بالكامل) |
| جودة البحث العربي | متوسط | pg_trgm للمطابقة الغامضة + Meilisearch كخطة بديلة |
| هلوسة AI في التوصيات | عالي | التحقق من المخرجات مقابل IDs حقيقية، درجات الثقة |
| أخطاء RTL في الواجهة | متوسط | Tailwind logical properties حصراً + اختبار بمحتوى عربي |
| تجاوز حدود API الخارجية | متوسط | Go rate limiter (golang.org/x/time/rate) + circuit breaker |
| تكاليف AI | **منخفض** | DeepSeek أرخص بـ 93% + cache 7 أيام |
| تعقيد إزالة التكرار | عالي | مطابقة DOI أولاً، ثم Levenshtein distance > 0.85 |

### مخاطر المراحل الجديدة (6-8)

| المخاطر | الشدة | خطة التخفيف |
|---------|-------|------------|
| اختلاف متطلبات الترقية بين المؤسسات | **عالي** | جداول متطلبات قابلة للتخصيص (promotion_requirements) لكل مؤسسة وتخصص |
| تعقيد سير العمل متعدد المستويات | **عالي** | محرك موافقات عام (approval_workflows) بدلاً من كتابة منطق مخصص لكل نوع طلب |
| مقاومة التغيير من المستخدمين | **عالي** | تدريب تدريجي، واجهة بسيطة، إلزام باستخدام النظام عبر ربطه بصرف المكافآت |
| حساسية بيانات التقييم والترقيات | عالي | صلاحيات صارمة — كل مستوى يرى فقط ما يخصه، تشفير البيانات الحساسة |
| اختلاف معايير الاعتماد بين الجهات | متوسط | قوالب قابلة للتخصيص لكل جهة (NCAAA, ABET, AACSB) |
| التكامل مع الأنظمة الجامعية القائمة | متوسط | API مفتوح + import/export يدوي كخطة بديلة في المرحلة الأولى |
| دقة الحسابات التلقائية (الأهلية، المكافآت) | عالي | مراجعة بشرية إلزامية قبل الاعتماد النهائي — النظام يقترح والمسؤول يقرر |
| تأخر الموافقات وتراكم الطلبات | متوسط | تصعيد تلقائي بعد 14 يوم + تقارير دورية للإدارة عن الطلبات المتأخرة |

---

## 9. معايير النجاح

### معايير المراحل الأساسية (1-5)
- [ ] يستطيع الباحثون عرض ملفاتهم مع مقاييس دقيقة من Scopus/CrossRef/OpenAlex
- [ ] تصنيف الأبحاث صحيح (Q1-Q4) ومجالات البحث
- [ ] لوحات المتابعة تعرض مقاييس مجمعة دقيقة لكل مستوى
- [ ] اتجاهات الاستشهاد و h-index تعمل بشكل صحيح
- [ ] توصيات التعاون من AI تشير لباحثين حقيقيين
- [ ] تحليل الاتجاهات يكتشف أنماطاً فعلية
- [ ] الواجهة العربية RTL تعمل بشكل كامل
- [ ] التبديل بين العربية والإنجليزية يعمل في كل الصفحات
- [ ] المزامنة التلقائية تعمل دون تجاوز حدود API
- [ ] البحث يعطي نتائج دقيقة بالعربية والإنجليزية
- [ ] Go API يستجيب خلال < 100ms (cached) / < 500ms (uncached)
- [ ] لوحات المتابعة تحمل خلال ثانيتين
- [ ] تغطية اختبارية 80%+ (Go + Next.js)
- [ ] الصلاحيات تعمل بشكل صحيح لكل دور
- [ ] Docker Compose يشغل البيئة الكاملة بأمر واحد

### معايير الشؤون العلمية (المرحلة 6)
- [ ] فحص أهلية الترقية يحسب تلقائياً بناءً على المتطلبات المُعرّفة
- [ ] سير عمل الترقية يتدفق بين المستويات (قسم → كلية → شؤون علمية → مساعد علمي)
- [ ] مكافآت النشر تُحسب تلقائياً حسب تصنيف المجلة وقواعد المؤسسة
- [ ] التقييم السنوي يُولّد تلقائياً مع درجة موزونة وترتيب
- [ ] طلبات التفرغ والمؤتمرات تمر عبر سير عمل موافقات كامل
- [ ] تقارير أداء القسم والكلية تُولّد بضغطة زر
- [ ] سجل الاتفاقيات يرسل تنبيهات تجديد قبل 90 يوم

### معايير الاعتماد (المرحلة 7)
- [ ] كل معيار اعتماد مربوط بشواهد (تلقائية + يدوية)
- [ ] توصيات الاعتماد السابقة مُتتبعة مع نسب إنجاز
- [ ] مؤشرات الأداء الاستراتيجية تعرض الفعلي مقابل المستهدف
- [ ] تقرير التقييم الذاتي يُولّد بتعبئة تلقائية من بيانات النظام
- [ ] لوحة الاعتماد تعرض نسبة الاستيفاء لكل جهة (NCAAA/ABET)

### معايير سير العمل والحسابات (المرحلة 8)
- [ ] كل دور يرى لوحة مخصصة عند تسجيل الدخول
- [ ] الباحث يرى جاهزيته للترقية ومهامه واقتراحات AI
- [ ] رئيس القسم يرى أداء قسمه وطلبات معلقة ومقارنات
- [ ] عميد الكلية يرى مقارنة الأقسام ومعايير الاعتماد
- [ ] الشؤون العلمية ترى كل العمليات اليومية (ترقيات، مكافآت، تقييم)
- [ ] المساعد العلمي يرى المؤشرات الاستراتيجية ومقارنة الجامعات
- [ ] محرك الموافقات يصعّد تلقائياً عند تجاوز 14 يوم بدون قرار
- [ ] الإشعارات تصل لكل مستوى حسب الدور (بريد + داخلي)

---

## 10. مراجع GitHub ومشاريع مرجعية

### هيكل Go Backend (الأساس)

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [bxcodec/go-clean-arch](https://github.com/bxcodec/go-clean-arch) | 10,061 | **الهيكل الأساسي** — Clean Architecture في Go (المعيار الذهبي) |
| [AleksK1NG/Go-Clean-Architecture-REST-API](https://github.com/AleksK1NG/Go-Clean-Architecture-REST-API) | 954 | REST API كامل مع clean architecture وأنماط عملية |
| [naeemaei/golang-clean-web-api](https://github.com/naeemaei/golang-clean-web-api) | 397 | قالب إنتاجي شامل: Docker, Redis, PostgreSQL, Prometheus, JWT |
| [high-horse/go-chi-pg-sqlc-goose-template](https://github.com/high-horse/go-chi-pg-sqlc-goose-template) | 23 | **الأقرب لتقنياتنا** — Chi + SQLC + Goose + PostgreSQL |

### عملاء API الأكاديمية

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [caltechlibrary/crossrefapi](https://github.com/caltechlibrary/crossrefapi) | 8 | **عميل Go جاهز لـ CrossRef** — من مكتبة Caltech |
| [diverged/openalex-openapi](https://github.com/diverged/openalex-openapi) | 6 | OpenAPI spec لـ OpenAlex — نولد منه Go client بـ oapi-codegen |
| [ourresearch/openalex-api-tutorials](https://github.com/ourresearch/openalex-api-tutorials) | 134 | **مرجع أساسي** — أنماط الاستعلام والبيانات في OpenAlex |
| [ricardogomes/elsevier-data-gatherer](https://github.com/ricardogomes/elsevier-data-gatherer) | 0 | عميل Go وحيد لـ Scopus API — نقطة بداية |

> **ملاحظة**: لا يوجد عميل Go لـ ORCID أو Scopus بجودة إنتاجية — سنبنيهم من الصفر

### شبكات الاستشهاد (Citation Networks)

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [LocalCitationNetwork](https://github.com/LocalCitationNetwork/LocalCitationNetwork.github.io) | 135 | **مرجع مباشر** — يستخدم OpenAlex + CrossRef لبناء شبكات استشهاد |
| [IllDepence/unarXive](https://github.com/IllDepence/unarXive) | 298 | بنية بيانات شبكات الاستشهاد + NLP |
| [edsu/etudier](https://github.com/edsu/etudier) | 162 | استخراج شبكات الاستشهاد من Google Scholar |
| [CitationGecko/citation-network-explorer](https://github.com/CitationGecko/citation-network-explorer) | 57 | مرجع UI/UX لتصور شبكات الاستشهاد |

> **فجوة**: لا يوجد أي مشروع مفتوح المصدر لكشف الاستشهادات الذاتية أو حلقات الاستشهاد — ميزتنا ستكون **فريدة**

### كشف المجلات المفترسة (Predatory Detection)

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [stop-predatory-journals](https://github.com/stop-predatory-journals/stop-predatory-journals.github.io) | 88 | **مصدر بيانات** — قائمة مجلات مفترسة قابلة للقراءة آلياً |
| [sustainet-guardian/aletheia-probe](https://github.com/sustainet-guardian/aletheia-probe) | 38 | **خوارزميات الكشف** — نحول المنطق لـ Go |
| [apifyforge/scientific-fraud-detection-mcp](https://github.com/apifyforge/scientific-fraud-detection-mcp) | 0 | مفهوم كشف التلاعب العلمي — مرجع للمنهجية |

### كشف الانتحال وتشابه النصوص

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [malteos/awesome-document-similarity](https://github.com/malteos/awesome-document-similarity) | 256 | **مرجع شامل** — قائمة خوارزميات تشابه المستندات |
| [allenai/aspire](https://github.com/allenai/aspire) | 55 | تشابه المستندات العلمية من Allen AI |
| [NETkiddy/simhash_similarity](https://github.com/NETkiddy/simhash_similarity) | 26 | **Go SimHash** — كشف النصوص المتشابهة على نطاق واسع |
| [cvcio/go-plagiarism](https://github.com/cvcio/go-plagiarism) | 7 | مكتبة Go وحيدة لكشف الانتحال — نقطة بداية |

### تصنيف أهداف التنمية المستدامة (SDG)

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [sadickam/sdg-classification-bert](https://github.com/sadickam/sdg-classification-bert) | 12 | مصنف BERT لـ SDGs — يمكن تغليفه كـ microservice |
| [SDGClassification/llm-sdg-classifier](https://github.com/SDGClassification/llm-sdg-classifier) | 6 | تصنيف SDG بـ LLMs — النهج الأحدث (يناسب DeepSeek) |
| [SDGClassification/benchmark](https://github.com/SDGClassification/benchmark) | 16 | بيانات مرجعية لتقييم دقة التصنيف |

### مقاييس بديلة (Altmetrics)

| المشروع | النجوم | الاستخدام |
|---------|--------|----------|
| [CenterForOpenScience/PyAltmetric](https://github.com/CenterForOpenScience/PyAltmetric) | 20 | أنماط تكامل Altmetric API |
| [ropensci/rAltmetric](https://github.com/ropensci/rAltmetric) | 50 | استعلام وتصور بيانات altmetric.com |
| [ropensci/roadoi](https://github.com/ropensci/roadoi) | 66 | تكامل Unpaywall (كشف الوصول المفتوح) |

### ملخص: ما نبنيه vs ما نستخدمه

| المكون | الحالة |
|--------|--------|
| هيكل Go backend | نستخدم go-clean-arch كقالب |
| عميل CrossRef | نستخدم caltechlibrary/crossrefapi |
| عميل OpenAlex | نولده من openapi spec |
| عميل Scopus | **نبنيه من الصفر** |
| عميل ORCID | **نبنيه من الصفر** |
| تحليل شبكات الاستشهاد | **نبنيه من الصفر** (gonum للحسابات) |
| كشف الاستشهاد الذاتي | **نبنيه من الصفر** (لا يوجد مثيل مفتوح المصدر) |
| كشف المجلات المفترسة | نستخدم بيانات stop-predatory-journals + خوارزميات aletheia-probe |
| كشف تشابه النصوص | نستخدم simhash_similarity + نبني محرك إضافي |
| تصنيف SDG | نستخدم DeepSeek مع منهجية llm-sdg-classifier |
| لوحة Next.js عربية RTL | **نبنيها من الصفر** (لا يوجد مثيل) |

---

## ملخص الجدول الزمني الكامل

```
المراحل الأساسية (نظام تتبع بحثي):
──────────────────────────────────────────────────
المرحلة 1: الأساس                    ← أسابيع 1-3    (3 أسابيع)
المرحلة 2: ربط APIs الخارجية         ← أسابيع 4-6    (3 أسابيع)
المرحلة 3: المقاييس ولوحات المتابعة   ← أسابيع 7-9    (3 أسابيع)
المرحلة 4: ميزات الذكاء الاصطناعي    ← أسابيع 10-13  (4 أسابيع)
المرحلة 5: التحسين والإنتاج          ← أسابيع 14-16  (3 أسابيع)
                                       ─────────────
                                       المجموع: 16 أسبوع

المراحل التشغيلية (منصة إدارة جامعية):
──────────────────────────────────────────────────
المرحلة 6: إدارة الشؤون العلمية      ← أسابيع 17-21  (5 أسابيع)
  ├── ترقيات، مكافآت، تقييم أداء
  ├── تفرغ، مؤتمرات
  └── اتفاقيات، كراسي ومراكز بحثية

المرحلة 7: الاعتماد وضمان الجودة     ← أسابيع 22-25  (4 أسابيع)
  ├── سجل شواهد، توصيات سابقة
  ├── مؤشرات أداء استراتيجية
  └── تقرير تقييم ذاتي تلقائي

المرحلة 8: سير العمل والحسابات       ← أسابيع 26-29  (4 أسابيع)
  ├── لوحات مخصصة لكل دور
  ├── محرك موافقات متعدد المستويات
  └── إشعارات ذكية حسب الدور
                                       ─────────────
                                       المجموع: 13 أسبوع

═══════════════════════════════════════════════════
المجموع الكلي: 29 أسبوع (~7 أشهر)
═══════════════════════════════════════════════════

  الأسبوع 16: نظام تتبع بحثي جاهز للإنتاج
  الأسبوع 29: منصة إدارة بحث علمي جامعية متكاملة (RIMS)
```

> **ملاحظة**: المراحل 6-8 تحوّل النظام من أداة رصد بيانات إلى منصة عمليات يومية يدخلها كل تدريسي ومسؤول في الجامعة.

---

## هل تريد البدء بالتنفيذ؟

بعد مراجعة الخطة، يمكننا البدء بالمرحلة 1 (الأساس) فوراً:
1. `backend/` — Go project مع Chi + sqlc + migrations
2. `frontend/` — Next.js مع Shadcn/ui + RTL
3. `docker-compose.yml` — PostgreSQL + Redis
