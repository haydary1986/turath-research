# النشر على خادم الجامعة (Coolify)

## الخطوات
1. في Coolify: **+ New → Docker Compose → استيراد من GitHub** واختر هذا المستودع. يقرأ `docker-compose.yml` تلقائيًا (4 حاويات: postgres، redis، backend، frontend).
2. املأ متغيرات البيئة أدناه، ثم Deploy.
3. وجّه النطاق (مثل `research.uoturath.edu.iq`) إلى خدمة **frontend** (منفذ 3000 داخليًا). Coolify يصدر شهادة HTTPS تلقائيًا.
4. لا تعرّض منفذ backend للعامة؛ الواجهة تمرر `/api/*` داخليًا.

## متغيرات البيئة

| المتغير | إلزامي | ملاحظة |
|---|---|---|
| `POSTGRES_PASSWORD` | ✅ | `openssl rand -hex 24` |
| `JWT_SECRET` | ✅ | `openssl rand -hex 32` |
| `GOOGLE_CLIENT_ID` | ✅ للدخول الجامعي | OAuth **Web** client من مشروع `gws-dashboard-461817`، أضف نطاق الموقع في Authorized JavaScript origins |
| `GOOGLE_ALLOWED_DOMAIN` | | الافتراضي `uoturath.edu.iq` |
| `FRONTEND_URL` | ✅ | العنوان العام، مثل `https://research.uoturath.edu.iq` — يُستخدم في روابط البريد |
| `ALLOWED_ORIGINS` | ✅ | نفس العنوان العام |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | للبريد | عبر Google Workspace: `smtp.gmail.com:587` بحساب `noreply@uoturath.edu.iq` وApp Password |
| `DEEPSEEK_API_KEY` | لميزات AI | |
| `ALLOW_PASSWORD_REGISTER` | | اتركه `false` في الإنتاج |

## بعد أول نشر
```bash
# اجعل حسابك مديرًا (بعد أول دخول Google)
docker compose exec postgres psql -U turath -d turath \
  -c "UPDATE users SET role='admin' WHERE email='بريدك@uoturath.edu.iq'"
```
ثم من **البوابة → المستخدمون والصلاحيات** عيّن بقية الأدوار، ومن **استيراد التدريسيين** ارفع ملف الملاك.

## النسخ الاحتياطي
```bash
./scripts/backup.sh /path/to/backups   # قاعدة البيانات + ملفات الرفع، يحتفظ بآخر 14 نسخة
```
أضفه إلى cron يوميًا على الخادم:
`0 2 * * * cd /path/to/app && ./scripts/backup.sh /var/backups/turath >> /var/log/turath-backup.log 2>&1`

## الاختبارات
```bash
ALLOW_PASSWORD_REGISTER=true docker compose up -d   # سكربتات الاختبار تنشئ مستخدمين
./test_portal.sh    # بوابة الشؤون العلمية (الصلاحيات، الحالات، الملفات، الكتب، الربط)
./test_all.sh       # واجهات النظام الأقدم (API_PORT=8091 افتراضيًا)
```
