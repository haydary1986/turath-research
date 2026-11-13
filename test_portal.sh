#!/bin/bash
# End-to-end test of the scientific affairs portal API with the 4-stage approval chain:
# researcher submits -> college dean (initial approval) -> scientific affairs (recommendation)
# -> VP (confirmation) -> clerical steps (signatures / letter issue).
# Needs ALLOW_PASSWORD_REGISTER=true on the backend (docker compose up -d with that env) to create test users.
set -u
API="${API:-http://localhost:8091/api/v1}"
RUN="$(date +%s)"
PASS=0; FAIL=0
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

ok()   { PASS=$((PASS+1)); echo "  PASS  $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  FAIL  $1 -> $2"; }
jq_() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)" 2>/dev/null; }
sqlv() { docker compose exec -T postgres psql -U turath -d turath -tAc "$1" | head -1 | tr -d '[:space:]'; }

call() { # METHOD PATH TOKEN [JSON] -> sets CODE and BODY
  local m="$1" p="$2" t="$3" b="${4:-}"
  local args=(-s -o "$TMP/body" -w "%{http_code}" -X "$m" "$API$p")
  [ -n "$t" ] && args+=(-H "Authorization: Bearer $t")
  [ -n "$b" ] && args+=(-H "Content-Type: application/json" -d "$b")
  CODE=$(curl "${args[@]}"); BODY=$(cat "$TMP/body")
}
upload() { # PATH TOKEN FILE
  CODE=$(curl -s -o "$TMP/body" -w "%{http_code}" -X POST "$API$1" -H "Authorization: Bearer $2" -F "file=@$3"); BODY=$(cat "$TMP/body")
}
expect() { # NAME WANT_CODE
  if [ "$CODE" = "$2" ]; then ok "$1"; else bad "$1" "HTTP $CODE (want $2): ${BODY:0:200}"; fi
}

register() { curl -s -X POST "$API/auth/register" -H "Content-Type: application/json" \
  -d "{\"email\":\"$1\",\"password\":\"Test1234!\",\"name_ar\":\"$2\"}" | jq_ "d['data']['token']"; }
login() { curl -s -X POST "$API/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"$1\",\"password\":\"Test1234!\"}" | jq_ "d['data']['token']"; }

echo "== setup: users, college, roles =="
RES_EMAIL="res$RUN@test.iq"; DEAN_EMAIL="dean$RUN@test.iq"; SA_EMAIL="sa$RUN@test.iq"
VP_EMAIL="vp$RUN@test.iq"; OTHER_EMAIL="oth$RUN@test.iq"; DEAN2_EMAIL="dean2$RUN@test.iq"
for u in "$RES_EMAIL:د. أحمد" "$DEAN_EMAIL:عميد الهندسة" "$SA_EMAIL:مسؤول الشؤون" "$VP_EMAIL:المساعد العلمي" "$OTHER_EMAIL:باحث آخر" "$DEAN2_EMAIL:عميد آخر"; do
  register "${u%%:*}" "${u#*:}" >/dev/null
done
COLLEGE=$(sqlv "INSERT INTO colleges (name_ar, institution_id) SELECT 'كلية اختبار $RUN', id FROM institutions LIMIT 1 RETURNING id")
COLLEGE2=$(sqlv "INSERT INTO colleges (name_ar, institution_id) SELECT 'كلية أخرى $RUN', id FROM institutions LIMIT 1 RETURNING id")
sqlv "UPDATE users SET role='college_dean', college_id='$COLLEGE' WHERE email='$DEAN_EMAIL'" >/dev/null
sqlv "UPDATE users SET role='college_dean', college_id='$COLLEGE2' WHERE email='$DEAN2_EMAIL'" >/dev/null
sqlv "UPDATE users SET role='scientific_affairs' WHERE email='$SA_EMAIL'" >/dev/null
sqlv "UPDATE users SET role='vp_research' WHERE email='$VP_EMAIL'" >/dev/null
RES=$(login "$RES_EMAIL"); DEAN=$(login "$DEAN_EMAIL"); DEAN2=$(login "$DEAN2_EMAIL")
SA=$(login "$SA_EMAIL"); VP=$(login "$VP_EMAIL"); OTHER=$(login "$OTHER_EMAIL")
[ -n "$RES" ] && [ -n "$DEAN" ] && [ -n "$SA" ] && [ -n "$VP" ] && ok "users ready" || { bad "setup" "login failed"; exit 1; }

printf '%%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%%%EOF\n' > "$TMP/doc.pdf"
printf '<html><script>alert(1)</script></html>' > "$TMP/evil.pdf"
python3 - "$TMP/bg.png" <<'EOF'
import sys,base64
open(sys.argv[1],'wb').write(base64.b64decode("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="))
EOF

DOI="10.1016/j.test.$RUN"
DETAILS='{"journal_name":"Energy and Buildings","publisher":"Elsevier","index_db":"scopus","quartile":"Q1","publication_date":"2024-03-15","issn":"0378-7788","article_url":"https://example.org/a"}'
PLAN_DETAILS='{"work_type":"joint","expected_completion":"2027-06-30"}'

echo "== research plan: full chain =="
call POST /portal/requests "$RES" "{\"kind\":\"research_plan\",\"title_ar\":\"بحث\",\"title_en\":\"Research\",\"details\":$PLAN_DETAILS}"; expect "plan draft saved" 201
PLAN=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$PLAN/submit" "$RES"; expect "submit without college rejected" 422
call PUT "/portal/requests/$PLAN" "$RES" "{\"title_ar\":\"بحث\",\"title_en\":\"Research\",\"college_id\":\"$COLLEGE\",\"details\":$PLAN_DETAILS}"; expect "college set" 200
call POST "/portal/requests/$PLAN/submit" "$RES"; expect "plan needs the committee-signed form" 422
echo "$BODY" | grep -q "research_form" && ok "missing signed form is named" || bad "missing form message" "$BODY"
call PUT "/portal/requests/$PLAN" "$RES" "{\"title_ar\":\"بحث\",\"title_en\":\"Research\",\"doi\":\"10.1234/plan.$RUN\",\"college_id\":\"$COLLEGE\",\"details\":$PLAN_DETAILS}"; expect "plan saved with a DOI typed in" 200
[ "$(echo "$BODY" | jq_ "d['data']['doi']")" = "None" ] && ok "plan never stores a DOI" || bad "plan doi" "$BODY"
upload "/portal/requests/$PLAN/files?role=research_form" "$RES" "$TMP/doc.pdf"; expect "signed plan form uploaded" 201
call POST "/portal/requests/$PLAN/submit" "$RES"; expect "plan submitted" 200
[ "$(echo "$BODY" | jq_ "d['data']['status']")" = "pending_dean" ] && ok "submission lands with the dean" || bad "status" "$BODY"
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"pending"}'; expect "SA cannot act at dean stage" 403
call POST "/portal/requests/$PLAN/transition" "$DEAN2" '{"to":"pending"}'; expect "dean of another college blocked" 404
call POST "/portal/requests/$PLAN/transition" "$DEAN" '{"to":"approved"}'; expect "dean cannot decide outcome" 409
call POST "/portal/requests/$PLAN/transition" "$DEAN" '{"to":"pending"}'; expect "dean initial approval" 200
call POST "/portal/requests/$PLAN/transition" "$DEAN" '{"to":"pending_vp","decision":"approved"}'; expect "dean cannot act at SA stage" 403
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"approved"}'; expect "SA cannot finalize alone" 409
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"pending_vp"}'; expect "recommendation needs a decision" 422
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"pending_vp","decision":"verified"}'; expect "plan cannot be 'verified'" 422
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"pending_vp","decision":"approved"}'; expect "SA raises recommendation" 200
call POST "/portal/requests/$PLAN/transition" "$SA" '{"to":"approved"}'; expect "SA cannot confirm own recommendation" 403
call POST "/portal/requests/$PLAN/transition" "$VP" '{"to":"rejected","note":"x"}'; expect "VP cannot override recommendation" 409
call POST "/portal/requests/$PLAN/transition" "$RES" '{"to":"approved"}'; expect "researcher cannot act" 403
call POST "/portal/requests/$PLAN/transition" "$VP" '{"to":"approved"}'; expect "VP confirms approval" 200
call GET "/portal/requests/$PLAN" "$RES"
[ "$(echo "$BODY" | jq_ "len(d['data']['events'])")" = "4" ] && ok "audit trail: submit→dean→SA→VP" || bad "events" "$BODY"

echo "== journal verification: chain + VP send-back + publication link =="
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"AI paper\",\"doi\":\"https://doi.org/$DOI\",\"college_id\":\"$COLLEGE\",\"details\":$DETAILS}"; expect "journal draft saved" 201
JV=$(echo "$BODY" | jq_ "d['data']['id']")
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"dup\",\"doi\":\"$DOI\",\"college_id\":\"$COLLEGE\",\"details\":$DETAILS}"; expect "duplicate DOI blocked" 409
call POST "/portal/requests/$JV/submit" "$RES"; expect "journal submitted" 200
call POST "/portal/requests/$JV/transition" "$DEAN" '{"to":"returned"}'; expect "dean return needs note" 422
call POST "/portal/requests/$JV/transition" "$DEAN" '{"to":"pending"}'; expect "dean forwards journal" 200
call POST "/portal/requests/$JV/transition" "$SA" '{"to":"pending_vp","decision":"verified"}'; expect "SA recommends verified" 200
call POST "/portal/requests/$JV/transition" "$VP" '{"to":"pending"}'; expect "VP send-back needs note" 422
call POST "/portal/requests/$JV/transition" "$VP" '{"to":"pending","note":"دقق المستوعب"}'; expect "VP sends back to SA" 200
call POST "/portal/requests/$JV/transition" "$SA" '{"to":"pending_vp","decision":"verified"}'; expect "SA re-raises" 200
call POST "/portal/requests/$JV/transition" "$VP" '{"to":"verified"}'; expect "VP confirms verified" 200
call POST "/portal/requests/$JV/transition" "$VP" '{"to":"awaiting_signatures"}'; expect "signatures step is SA's, not VP's" 403
call POST "/portal/requests/$JV/transition" "$SA" '{"to":"awaiting_signatures"}'; expect "to committee signatures" 200
call POST "/portal/requests/$JV/transition" "$SA" '{"to":"completed"}'; expect "completion needs signed form" 422
upload "/portal/requests/$JV/files?role=signed_form" "$RES" "$TMP/doc.pdf"; expect "applicant cannot upload signed form" 403
upload "/portal/requests/$JV/files?role=signed_form" "$SA" "$TMP/evil.pdf"; expect "fake PDF (html) rejected" 415
upload "/portal/requests/$JV/files?role=signed_form" "$SA" "$TMP/doc.pdf"; expect "SA uploads signed form" 201
FILE=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$JV/transition" "$SA" '{"to":"completed"}'; expect "journal completed" 200
[ "$(sqlv "SELECT COUNT(*) FROM publications WHERE doi='$DOI'")" = "1" ] && ok "publication materialized" || bad "publication link" "doi missing"
[ "$(sqlv "SELECT COUNT(*) FROM users u JOIN researchers r ON r.id=u.researcher_id WHERE u.email='$RES_EMAIL'")" = "1" ] && ok "user linked to researcher" || bad "researcher link" "missing"
[ "$(sqlv "SELECT COUNT(*) FROM publication_authors pa JOIN publications p ON p.id=pa.publication_id WHERE p.doi='$DOI'")" = "1" ] && ok "authorship recorded" || bad "authorship" "missing"
CODE=$(curl -s -o "$TMP/dl" -w "%{http_code}" "$API/portal/files/$FILE" -H "Authorization: Bearer $RES"); [ "$CODE" = "200" ] && head -c4 "$TMP/dl" | grep -q '%PDF' && ok "applicant downloads file" || bad "download" "$CODE"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/portal/files/$FILE" -H "Authorization: Bearer $OTHER"); [ "$CODE" = "404" ] && ok "other user cannot download" || bad "download isolation" "$CODE"

echo "== appreciation letter: chain + issue =="
call POST /portal/requests "$RES" "{\"kind\":\"appreciation_letter\",\"title_en\":\"AI paper\",\"doi\":\"$DOI\",\"college_id\":\"$COLLEGE\",\"related_request_id\":\"$JV\",\"details\":$DETAILS}"; expect "letter draft saved" 201
AR=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$AR/submit" "$RES"; expect "letter needs committee approval" 422
upload "/portal/requests/$AR/files?role=committee_approval" "$RES" "$TMP/doc.pdf"; expect "committee approval uploaded" 201
call POST "/portal/requests/$AR/submit" "$RES"; expect "letter submitted" 200
call POST "/portal/requests/$AR/transition" "$DEAN" '{"to":"pending"}'; expect "dean forwards letter" 200
call POST "/portal/requests/$AR/issue-letter" "$SA"; expect "cannot issue before verification" 409
call POST "/portal/requests/$AR/transition" "$SA" '{"to":"pending_vp","decision":"verified"}'; expect "SA recommends letter" 200
call POST "/portal/requests/$AR/transition" "$VP" '{"to":"verified"}'; expect "VP confirms letter" 200
call POST "/portal/requests/$AR/transition" "$SA" '{"to":"issued"}'; expect "issued only via issue-letter" 409
call POST "/portal/requests/$AR/issue-letter" "$RES"; expect "researcher cannot issue" 403
call POST "/portal/requests/$AR/issue-letter" "$VP"; expect "issuing is SA's clerical step" 403
call POST "/portal/requests/$AR/issue-letter" "$SA" '{"diwan_no":"DW-1234"}'; expect "letter issued" 201
[ "$(echo "$BODY" | jq_ "d['data']['diwan_no']")" = "DW-1234" ] && ok "diwan number stored on letter" || bad "diwan no" "$BODY"
LETTER=$(echo "$BODY" | jq_ "d['data']['id']")
echo "$BODY" | jq_ "d['data']['letter_no']" | grep -Eq '^[0-9]+/[0-9]{4}$' && ok "letter number format" || bad "letter no" "$BODY"
call POST "/portal/requests/$AR/issue-letter" "$SA"; expect "letter cannot be issued twice" 409
call GET "/portal/letters/$LETTER" "$RES"; expect "applicant reads own letter" 200
call GET "/portal/letters/$LETTER" "$OTHER"; expect "other researcher cannot read letter" 404
call GET "/verify/letter/$LETTER" ""; expect "public verification" 200


echo "== scientific activities =="
call POST /activities "$RES" '{"activity_type":"party","title_ar":"x","activity_date":"2026-11-02"}'; expect "unknown activity type rejected" 422
call POST /activities "$RES" "{\"activity_type\":\"workshop\",\"title_ar\":\"ورشة الذكاء الاصطناعي\",\"presenter\":\"د. أحمد\",\"activity_date\":\"2026-11-02\",\"location\":\"قاعة المؤتمرات\",\"attendees_count\":40,\"college_id\":\"$COLLEGE\"}"; expect "activity created" 201
ACT=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/activities/$ACT/decide" "$RES" '{"status":"approved","approval_no":"77/2026"}'; expect "researcher cannot decide activity" 403
call POST "/activities/$ACT/decide" "$SA" '{"status":"approved"}'; expect "approval needs letter number" 422
call POST "/activities/$ACT/decide" "$SA" '{"status":"approved","approval_no":"77/2026"}'; expect "activity approved with letter no" 200
call POST "/activities/$ACT/decide" "$SA" '{"status":"rejected","note":"x"}'; expect "decided activity is final" 409
echo "$BODY" | grep -q "activity_already_decided" && ok "decided activity reports a mapped error code" || bad "decided activity code" "$BODY"
call GET "/activities?year=2026&type=workshop" "$SA"
[ "$(echo "$BODY" | jq_ "[a['approval_no'] for a in d['data'] if a['id']=='$ACT'][0]")" = "77/2026" ] && ok "approval number stored" || bad "approval no" "$BODY"
call GET /activities "$OTHER"
[ "$(echo "$BODY" | jq_ "sum(1 for a in d['data'] if a['id']=='$ACT')")" = "0" ] && ok "activities scoped per user" || bad "activity scope" "$BODY"

echo "== journal watchlist =="
PRED_ISSN="2049-3630"
sqlv "DELETE FROM journal_screening WHERE issn='$PRED_ISSN'" >/dev/null
call POST /screening "$RES" "{\"name\":\"x\",\"classification\":\"predatory\"}"; expect "researcher cannot manage watchlist" 403
call POST /screening "$SA" "{\"name\":\"Predatory Journal of Everything\",\"issn\":\"$PRED_ISSN\",\"classification\":\"fake\"}"; expect "bad classification rejected" 422
call POST /screening "$SA" "{\"name\":\"Predatory Journal of Everything\",\"issn\":\"$PRED_ISSN\",\"classification\":\"predatory\",\"note\":\"قائمة الوزارة\"}"; expect "predatory journal added" 200
call POST /screening "$SA" "{\"name\":\"Another Name\",\"issn\":\"2049 3630\",\"classification\":\"suspended\"}"; expect "duplicate ISSN blocked (normalized)" 409
call GET "/screening/check?issn=20493630" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['classification']")" = "predatory" ] && ok "live check matches normalized ISSN" || bad "check" "$BODY"
call GET "/screening/check?name=predatory  journal OF everything" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['matched']")" = "True" ] && ok "live check matches folded name" || bad "name check" "$BODY"
PRED_DETAILS='{"journal_name":"Predatory Journal of Everything","publisher":"X","index_db":"scopus","publication_date":"2024-03-15","issn":"2049-3630"}'
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Flagged paper\",\"doi\":\"10.9999/flag.$RUN\",\"college_id\":\"$COLLEGE\",\"details\":$PRED_DETAILS}"; expect "flagged journal draft saved" 201
FLAGREQ=$(echo "$BODY" | jq_ "d['data']['id']")
call GET "/portal/requests/$FLAGREQ" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['journal_flag']['classification']")" = "predatory" ] && ok "request detail carries predatory flag" || bad "flag on detail" "$BODY"

echo "== annual report =="
CODE=$(curl -s -o "$TMP/report.xlsx" -w "%{http_code}" "$API/reports/annual.xlsx?year=2026" -H "Authorization: Bearer $RES"); [ "$CODE" = "403" ] && ok "researcher cannot download report" || bad "report rbac" "$CODE"
CODE=$(curl -s -o "$TMP/report.xlsx" -w "%{http_code}" "$API/reports/annual.xlsx?year=2026" -H "Authorization: Bearer $SA")
[ "$CODE" = "200" ] && head -c2 "$TMP/report.xlsx" | grep -q "PK" && ok "annual xlsx downloads" || bad "report" "HTTP $CODE"
REPORT_CHECK=$(python3 -X utf8 -c "
import zipfile,sys
z=zipfile.ZipFile(sys.argv[1]); s=''.join(z.read(n).decode('utf8') for n in z.namelist() if n.endswith('.xml'))
codes=['pending_vp','pending_dean','not_eligible','awaiting_signatures','>completed<','>verified<','>approved<','>seminar<']
print('ok' if 'الملخص الإحصائي' in s and not any(c in s for c in codes) else 'bad')" "$TMP/report.xlsx")
[ "$REPORT_CHECK" = "ok" ] && ok "report has summary sheet and no raw English codes" || bad "report content" "$REPORT_CHECK"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/reports/annual.xlsx?from=2026-01-01&to=2026-03-31" -H "Authorization: Bearer $SA"); [ "$CODE" = "200" ] && ok "quarterly report downloads" || bad "quarter report" "$CODE"

echo "== watchlist import =="
IMP_A=$(printf '%04d-%04d' $((RUN % 10000)) 1111); IMP_B=$(printf '%04d-%04d' $((RUN % 10000)) 2222)
make_xlsx() { # FILE ROWS_JSON
python3 -X utf8 -c "
import zipfile,sys,json
rows=json.loads(sys.argv[2])
def c(col,row,v): return f'<c r=\"{col}{row}\" t=\"inlineStr\"><is><t>{v}</t></is></c>' if v else ''
sd=''.join('<row r=\"%d\">%s</row>'%(i+1,''.join(c('ABCDEFGHIJKLMNOPQRSTUVWXYZ'[j],i+1,v) for j,v in enumerate(r))) for i,r in enumerate(rows))
z=zipfile.ZipFile(sys.argv[1],'w')
z.writestr('[Content_Types].xml','<?xml version=\"1.0\"?><Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\"><Default Extension=\"rels\" ContentType=\"application/vnd.openxmlformats-package.relationships+xml\"/><Default Extension=\"xml\" ContentType=\"application/xml\"/><Override PartName=\"/xl/workbook.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml\"/><Override PartName=\"/xl/worksheets/sheet1.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml\"/></Types>')
z.writestr('_rels/.rels','<?xml version=\"1.0\"?><Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\"><Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument\" Target=\"xl/workbook.xml\"/></Relationships>')
z.writestr('xl/workbook.xml','<?xml version=\"1.0\"?><workbook xmlns=\"http://schemas.openxmlformats.org/spreadsheetml/2006/main\" xmlns:r=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships\"><sheets><sheet name=\"S\" sheetId=\"1\" r:id=\"rId1\"/></sheets></workbook>')
z.writestr('xl/_rels/workbook.xml.rels','<?xml version=\"1.0\"?><Relationships xmlns=\"http://schemas.openxmlformats.org/package/2006/relationships\"><Relationship Id=\"rId1\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet\" Target=\"worksheets/sheet1.xml\"/></Relationships>')
z.writestr('xl/worksheets/sheet1.xml','<?xml version=\"1.0\"?><worksheet xmlns=\"http://schemas.openxmlformats.org/spreadsheetml/2006/main\"><sheetData>'+sd+'</sheetData></worksheet>')
z.close()" "$1" "$2"
}
make_xlsx "$TMP/watch.xlsx" "[[\"اسم المجلة\",\"ISSN\",\"eISSN\",\"التصنيف\"],[\"Imported Predatory $RUN\",\"$IMP_A\",\"$IMP_B\",\"مفترسة\"],[\"Bad Row\",\"\",\"\",\"غير معروف\"]]"
upload /screening/import "$RES" "$TMP/watch.xlsx"; expect "researcher cannot import watchlist" 403
upload /screening/import "$SA" "$TMP/watch.xlsx"; expect "watchlist imports" 200
[ "$(echo "$BODY" | jq_ "(d['data']['created'], len(d['data']['problems']))")" = "(2, 1)" ] && ok "both ISSNs stored, bad row reported" || bad "import report" "$BODY"
call GET "/screening/check?issn=$IMP_B" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['classification']")" = "predatory" ] && ok "imported online ISSN is matched" || bad "import check" "$BODY"

echo "== visibility, templates, admin =="
call GET "/portal/requests/$PLAN" "$DEAN"; expect "dean views his college's request" 200
call GET "/portal/requests/$PLAN" "$DEAN2"; expect "other dean cannot view" 404
call GET /portal/templates "$VP"; expect "VP may manage templates" 200
call GET /portal/templates "$DEAN"; expect "dean may not manage templates" 403
upload "/portal/template-assets?role=background" "$SA" "$TMP/doc.pdf"; expect "template asset must be an image" 415
upload "/portal/form-templates/research_plan" "$RES" "$TMP/doc.pdf"; expect "researcher cannot upload official forms" 403
printf '\x89PNG\r\n\x1a\n0000000000000000' > "$TMP/img.png"
upload "/portal/form-templates/research_plan" "$SA" "$TMP/img.png"; expect "official form must be a PDF" 415
upload "/portal/form-templates/research_plan" "$SA" "$TMP/doc.pdf"; expect "SA uploads official plan form" 200
FORM_FILE=$(echo "$BODY" | jq_ "d['data']['file_id']")
upload "/portal/form-templates/research_plan" "$SA" "$TMP/doc.pdf"; expect "SA replaces official plan form" 200
call GET /portal/form-templates "$RES"
[ "$(echo "$BODY" | jq_ "sum(1 for f in d['data'] if f['kind']=='research_plan')")" = "1" ] && ok "researcher sees one official plan form" || bad "form list" "$BODY"
NEW_FILE=$(echo "$BODY" | jq_ "[f['file_id'] for f in d['data'] if f['kind']=='research_plan'][0]")
CODE=$(curl -s -o "$TMP/form.pdf" -w "%{http_code}" "$API/portal/files/$NEW_FILE" -H "Authorization: Bearer $RES"); [ "$CODE" = "200" ] && head -c4 "$TMP/form.pdf" | grep -q "%PDF" && ok "researcher downloads official form" || bad "form download" "$CODE"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/portal/files/$FORM_FILE" -H "Authorization: Bearer $RES"); [ "$CODE" = "404" ] && ok "replaced form file is gone" || bad "old form file" "$CODE"
call DELETE /portal/form-templates/research_plan "$RES"; expect "researcher cannot delete official form" 403
call GET "/screening/reference-list?type=predatory_journal&limit=5" "$RES"; expect "researcher browses predatory list" 200
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "5" ] && ok "predatory list returns a page" || bad "list page" "${BODY:0:200}"
call GET "/screening/reference-list?type=hijacked&q=Acta" "$RES"; expect "hijacked list is searchable" 200
call GET "/screening/reference-list?type=other" "$RES"; expect "unknown list type rejected" 400
CODE=$(curl -s -o "$TMP/body" -w "%{http_code}" -X POST "$API/portal/template-assets?role=background" -H "Authorization: Bearer $SA" -F "file=@$TMP/bg.png"); BODY=$(cat "$TMP/body"); expect "background image uploaded" 201
call GET /portal/requests "$OTHER"
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "0" ] && ok "unrelated researcher sees nothing" || bad "isolation" "$BODY"
call GET /portal/stats "$SA"; expect "stats" 200
ADMIN_EMAIL="adm$RUN@test.iq"; register "$ADMIN_EMAIL" "مدير" >/dev/null
sqlv "UPDATE users SET role='admin' WHERE email='$ADMIN_EMAIL'" >/dev/null; ADM=$(login "$ADMIN_EMAIL")
call GET "/admin/users?q=$RUN" "$ADM"; expect "admin lists users" 200
UID_=$(echo "$BODY" | jq_ "[u['id'] for u in d['data'] if u['email']=='$OTHER_EMAIL'][0]")
call PUT "/admin/users/$UID_" "$ADM" '{"role":"department_head"}'; expect "admin assigns department head" 200
call PUT "/admin/users/$UID_" "$ADM" '{"role":"viewer"}'; expect "legacy role no longer assignable" 400
call PUT "/admin/users/$UID_" "$ADM" '{"role":"superuser"}'; expect "unknown role rejected" 400

echo "== stage actions on editable requests =="
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Returned\",\"doi\":\"10.9999/ret.$RUN\",\"college_id\":\"$COLLEGE\",\"details\":$DETAILS}"; RET=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$RET/submit" "$RES"; call POST "/portal/requests/$RET/transition" "$DEAN" '{"to":"returned","note":"صحح"}'; expect "dean returns request" 200
call GET "/portal/requests/$RET" "$ADM"
[ "$(echo "$BODY" | jq_ "d['data']['allowed_transitions']")" = "[]" ] && ok "admin is offered nothing on a returned request" || bad "admin offered" "$BODY"
call POST "/portal/requests/$RET/transition" "$ADM" '{"to":"pending_dean"}'; expect "admin cannot bypass submit validation" 409

echo "== review queue =="
call POST /portal/requests "$RES" "{\"kind\":\"research_plan\",\"title_ar\":\"صف\",\"title_en\":\"Queue\",\"college_id\":\"$COLLEGE\",\"details\":$PLAN_DETAILS}"; QP=$(echo "$BODY" | jq_ "d['data']['id']")
upload "/portal/requests/$QP/files?role=research_form" "$RES" "$TMP/doc.pdf"
call POST "/portal/requests/$QP/submit" "$RES"; expect "queue plan submitted" 200
inq() { call GET "/portal/requests?queue=true&limit=50" "$1"; echo "$BODY" | jq_ "any(r['id']=='$2' for r in d['data'])"; }
[ "$(inq "$DEAN" "$QP")" = "True" ] && ok "dean queue holds own college" || bad "dean queue" "$BODY"
[ "$(inq "$DEAN2" "$QP")" = "False" ] && ok "other dean's queue excludes it" || bad "dean2 queue" "$BODY"
[ "$(inq "$SA" "$QP")" = "False" ] && ok "SA queue excludes dean stage" || bad "sa queue" "$BODY"
[ "$(inq "$ADM" "$QP")" = "True" ] && ok "admin queue includes dean stage" || bad "admin queue" "$BODY"
call GET "/portal/requests?queue=true" "$RES"
[ "$(echo "$BODY" | jq_ "d['meta'].get('total', 0)")" = "0" ] && ok "researcher queue is empty" || bad "res queue" "$BODY"
call POST "/portal/requests/$QP/transition" "$DEAN" '{"to":"pending"}'
[ "$(inq "$SA" "$QP")" = "True" ] && ok "SA queue holds pending" || bad "sa queue" "$BODY"
call POST "/portal/requests/$QP/transition" "$SA" '{"to":"pending_vp","decision":"approved"}'
[ "$(inq "$VP" "$QP")" = "True" ] && ok "VP queue holds pending_vp" || bad "vp queue" "$BODY"
[ "$(sqlv "SELECT COUNT(*) FROM notifications n JOIN users u ON u.id=n.user_id WHERE u.email='$SA_EMAIL' AND n.title_en='Verified request awaiting scientific affairs'")" -ge 1 ] && ok "SA notified on verified" || bad "verified notice" "none"
[ "$(sqlv "SELECT COUNT(*) FROM notifications n JOIN users u ON u.id=n.user_id WHERE u.email='$SA_EMAIL' AND n.title_en='Request awaiting committee signatures'")" -ge 1 ] && ok "SA notified on awaiting signatures" || bad "signatures notice" "none"
sqlv "UPDATE users SET college_id='$COLLEGE' WHERE email='$DEAN_EMAIL'" >/dev/null
call POST /portal/requests "$DEAN" "{\"kind\":\"research_plan\",\"title_ar\":\"للعميد\",\"title_en\":\"Dean own\",\"college_id\":\"$COLLEGE\",\"details\":$PLAN_DETAILS}"; DP=$(echo "$BODY" | jq_ "d['data']['id']")
upload "/portal/requests/$DP/files?role=research_form" "$DEAN" "$TMP/doc.pdf"
call POST "/portal/requests/$DP/submit" "$DEAN"; expect "dean submits own request" 200
[ "$(inq "$DEAN" "$DP")" = "False" ] && ok "own request stays out of the queue" || bad "own in queue" "$BODY"
DPNO=$(sqlv "SELECT request_no FROM portal_requests WHERE id='$DP'")
[ "$(sqlv "SELECT COUNT(*) FROM notifications n JOIN users u ON u.id=n.user_id WHERE u.email='$ADMIN_EMAIL' AND n.title_en LIKE 'Request needs an admin%' AND n.body LIKE '%$DPNO%'")" = "1" ] && ok "admin notified when applicant holds the stage" || bad "admin notice" "none"

echo "== evidence on finished requests =="
call DELETE "/portal/files/$FILE" "$SA"; expect "SA cannot delete evidence on a completed request" 403
upload "/portal/requests/$JV/files?role=other" "$SA" "$TMP/doc.pdf"; expect "SA cannot add files to a completed request" 403
upload "/portal/requests/$PLAN/files?role=other" "$ADM" "$TMP/doc.pdf"; expect "admin cannot add files to an approved request" 403

echo "== authorship position =="
finish_jv() { # ID
  call POST "/portal/requests/$1/transition" "$DEAN" '{"to":"pending"}'
  call POST "/portal/requests/$1/transition" "$SA" '{"to":"pending_vp","decision":"verified"}'
  call POST "/portal/requests/$1/transition" "$VP" '{"to":"verified"}'
  call POST "/portal/requests/$1/transition" "$SA" '{"to":"awaiting_signatures"}'
  upload "/portal/requests/$1/files?role=signed_form" "$SA" "$TMP/doc.pdf"
  call POST "/portal/requests/$1/transition" "$SA" '{"to":"completed"}'
}
AO_DETAILS='{"journal_name":"Energy and Buildings","publisher":"Elsevier","index_db":"scopus","publication_date":"2024-03-15","authors_count":"3","author_order":"2"}'
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Second author\",\"doi\":\"10.9999/ao.$RUN\",\"college_id\":\"$COLLEGE\",\"details\":$AO_DETAILS}"; AO=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$AO/submit" "$RES"; finish_jv "$AO"; expect "completes with author_order 2" 200
[ "$(sqlv "SELECT pa.author_order||'/'||pa.is_corresponding FROM publication_authors pa JOIN publications p ON p.id=pa.publication_id WHERE p.doi='10.9999/ao.$RUN'")" = "2/false" ] && ok "author_order honoured, not corresponding" || bad "author order" "wrong"
COLLIDE="10.9999/co.$RUN"
PUB=$(sqlv "INSERT INTO publications (doi, title_en, publication_type, status) VALUES ('$COLLIDE', 'Existing', 'journal_article', 'published') RETURNING id")
sqlv "INSERT INTO publication_authors (publication_id, author_name, author_order, is_corresponding) VALUES ('$PUB', 'First', 1, TRUE), ('$PUB', 'Second', 2, FALSE)" >/dev/null
call POST /portal/requests "$OTHER" "{\"kind\":\"journal_verification\",\"title_en\":\"Collision\",\"doi\":\"$COLLIDE\",\"college_id\":\"$COLLEGE\",\"details\":$AO_DETAILS}"; CO=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$CO/submit" "$OTHER"; finish_jv "$CO"; expect "occupied position does not fail completion" 200
[ "$(sqlv "SELECT MAX(author_order)||'/'||COUNT(*) FILTER (WHERE is_corresponding) FROM publication_authors WHERE publication_id='$PUB'")" = "3/1" ] && ok "next free position, slot 1 keeps corresponding" || bad "collision" "wrong"
UPGRADE_DOI="10.9999/up.$RUN"
RES_RID=$(sqlv "SELECT researcher_id FROM users WHERE email='$RES_EMAIL'")
UP_PUB=$(sqlv "INSERT INTO publications (doi, title_en, publication_type, status) VALUES ('$UPGRADE_DOI', 'Synced first', 'journal_article', 'published') RETURNING id")
sqlv "INSERT INTO publication_authors (publication_id, researcher_id, author_name, author_order, source) VALUES ('$UP_PUB', '$RES_RID', 'x', 1, 'sync')" >/dev/null
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Upgrade\",\"doi\":\"$UPGRADE_DOI\",\"college_id\":\"$COLLEGE\",\"details\":$AO_DETAILS}"; UP=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$UP/submit" "$RES"; finish_jv "$UP"; expect "verification completes over a synced link" 200
[ "$(sqlv "SELECT source FROM publication_authors WHERE publication_id='$UP_PUB' AND researcher_id='$RES_RID'")" = "portal" ] && ok "verified authorship upgraded from sync to portal" || bad "upgrade" "still sync"

echo "== journal field validation =="
submit_details() { # JSON -> sets CODE, BODY
  call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"V\",\"college_id\":\"$COLLEGE\",\"details\":$1}"
  call POST "/portal/requests/$(echo "$BODY" | jq_ "d['data']['id']")/submit" "$RES"
}
BASE='"journal_name":"J","publisher":"P","publication_date":"2024-03-15"'
submit_details "{$BASE,\"index_db\":\"local\",\"quartile\":\"Q1\"}"; expect "quartile rejected on a local journal" 422; echo "$BODY" | grep -q "quartile:invalid" && ok "quartile code" || bad "quartile code" "$BODY"
submit_details "{$BASE,\"index_db\":\"scopus\",\"plagiarism_pct\":\"101\"}"; expect "plagiarism above 100 rejected" 422
submit_details "{$BASE,\"index_db\":\"scopus\",\"authors_count\":\"2\",\"author_order\":\"3\"}"; expect "author_order above authors_count rejected" 422
submit_details "{\"journal_name\":\"J\",\"publisher\":\"P\",\"index_db\":\"scopus\",\"publication_date\":\"1850-01-01\"}"; expect "date before 1900 rejected" 422

echo "== registry number after issue =="
call POST "/portal/requests/$AR/issue-letter" "$SA" 'not json'; expect "malformed issue body rejected" 400
call PUT "/portal/letters/$LETTER/diwan-no" "$RES" '{"diwan_no":"X"}'; expect "researcher cannot edit registry number" 403
call PUT "/portal/letters/$LETTER/diwan-no" "$VP" '{"diwan_no":"X"}'; expect "VP cannot edit registry number" 403
call PUT "/portal/letters/$LETTER/diwan-no" "$SA" '{"diwan_no":'; expect "malformed registry body rejected" 400
call PUT "/portal/letters/$LETTER/diwan-no" "$SA" "{\"diwan_no\":\"$(printf 'ع%.0s' $(seq 1 101))\"}"; expect "101 characters rejected" 422
call PUT "/portal/letters/$LETTER/diwan-no" "$SA" "{\"diwan_no\":\"$(printf 'ع%.0s' $(seq 1 100))\"}"; expect "100 multibyte characters accepted" 200
call PUT "/portal/letters/$LETTER/diwan-no" "$SA" '{"diwan_no":"DW-9"}'; expect "SA edits registry number" 200
call PUT "/portal/letters/$LETTER/diwan-no" "$SA"; expect "empty body is harmless" 200
[ "$(echo "$BODY" | jq_ "d['data']['diwan_no']")" = "DW-9" ] && ok "empty body keeps the number" || bad "empty body" "$BODY"
call GET "/portal/requests/$AR" "$SA"
[ "$(echo "$BODY" | jq_ "sum(1 for e in d['data']['events'] if 'DW-9' in (e['note'] or ''))")" = "1" ] && ok "history records the change" || bad "history" "$BODY"
call PUT "/portal/letters/not-a-uuid/diwan-no" "$SA" '{"diwan_no":"1"}'; expect "unknown letter" 404

echo "== activity ownership =="
ACT_BODY="{\"activity_type\":\"seminar\",\"title_ar\":\"ندوة\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\"}"
call POST /activities "$RES" "{\"activity_type\":\"seminar\",\"title_ar\":\"x\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\",\"attendance_mode\":\"  \"}"; expect "whitespace attendance mode rejected" 422
call POST /activities "$SA" "$ACT_BODY"; OWN=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/activities/$OWN/decide" "$SA" '{"status":"approved","approval_no":"1/2026"}'; expect "SA cannot decide own activity" 403
echo "== activity by a scientific-affairs user reaches another decider =="
SA2_EMAIL="sa2$RUN@test.iq"; register "$SA2_EMAIL" "مسؤول ثان" >/dev/null
sqlv "UPDATE users SET role='scientific_affairs' WHERE email='$SA2_EMAIL'" >/dev/null; SA2=$(login "$SA2_EMAIL")
SA_TITLE="نشاطالشؤون$RUN"
call POST /activities "$SA" "{\"activity_type\":\"seminar\",\"title_ar\":\"$SA_TITLE\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\"}"; SAACT=$(echo "$BODY" | jq_ "d['data']['id']")
notified() { sqlv "SELECT COUNT(*) FROM notifications n JOIN users u ON u.id=n.user_id WHERE u.email='$1' AND n.body='$SA_TITLE'"; }
[ "$(notified "$ADMIN_EMAIL")" = "1" ] && ok "admin told of SA's activity" || bad "admin notice" "$(notified "$ADMIN_EMAIL")"
[ "$(notified "$SA2_EMAIL")" = "1" ] && ok "other SA told of SA's activity" || bad "other SA notice" "$(notified "$SA2_EMAIL")"
[ "$(notified "$SA_EMAIL")" = "0" ] && ok "author not asked to review own activity" || bad "self notice" "$(notified "$SA_EMAIL")"
call POST "/activities/$SAACT/decide" "$SA2" '{"status":"approved","approval_no":"9/2026"}'; expect "another SA decides it" 200

echo "== activity department follows its college =="
DEPT1=$(sqlv "INSERT INTO departments (name_ar, college_id) VALUES ('قسم أول $RUN', '$COLLEGE') RETURNING id")
ORG_EMAIL="org$RUN@test.iq"; register "$ORG_EMAIL" "منظم" >/dev/null
sqlv "UPDATE users SET college_id='$COLLEGE', department_id='$DEPT1' WHERE email='$ORG_EMAIL'" >/dev/null; ORG=$(login "$ORG_EMAIL")
org_body() { echo "{\"activity_type\":\"seminar\",\"title_ar\":\"قسم\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$1\",\"department_id\":\"$2\"}"; }
call POST /activities "$ORG" "$(org_body "$COLLEGE2" "")"; expect "activity in another college" 201
[ "$(echo "$BODY" | jq_ "d['data']['department_id']")" = "None" ] && ok "other college does not inherit the department" || bad "inherited dept" "$BODY"
ORGACT=$(echo "$BODY" | jq_ "d['data']['id']")
call POST /activities "$ORG" "$(org_body "$COLLEGE" "")"; expect "activity in own college" 201
[ "$(echo "$BODY" | jq_ "d['data']['department_id']")" = "$DEPT1" ] && ok "own college inherits the department" || bad "own dept" "$BODY"
call POST /activities "$ORG" "$(org_body "$COLLEGE2" "$DEPT1")"; expect "department outside the college rejected" 422
echo "$BODY" | grep -q "department:wrong_college" && ok "wrong_college code" || bad "wrong_college code" "$BODY"
call PUT "/activities/$ORGACT" "$ORG" "$(org_body "$COLLEGE2" "$DEPT1")"; expect "update also checks the department" 422
LONGLOC=$(python3 -c "print('x'*256)")
call POST /activities "$ORG" "{\"activity_type\":\"seminar\",\"title_ar\":\"x\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\",\"location\":\"$LONGLOC\"}"; expect "over-long location rejected" 422
echo "$BODY" | grep -q "location:too_long" && ok "location code" || bad "location code" "$BODY"

call POST /activities "$ADM" "$ACT_BODY"; OWNA=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/activities/$OWNA/decide" "$ADM" '{"status":"approved","approval_no":"2/2026"}'; expect "admin may decide own activity" 200
call POST /activities "$RES" "{\"activity_type\":\"seminar\",\"title_ar\":\"معادة $RUN\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\"}"; RACT=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/activities/$RACT/decide" "$SA" '{"status":"returned","note":"عدل"}'; expect "activity returned" 200
call PUT "/activities/$RACT" "$RES" "{\"activity_type\":\"seminar\",\"title_ar\":\"معادة $RUN\",\"activity_date\":\"2026-11-04\",\"college_id\":\"$COLLEGE\"}"; expect "organizer resubmits" 200
[ "$(sqlv "SELECT COUNT(*) FROM notifications n JOIN users u ON u.id=n.user_id WHERE u.email='$SA_EMAIL' AND n.title_en='Returned scientific activity resubmitted'")" = "1" ] && ok "SA notified of resubmission" || bad "resubmit notice" "none"

echo "== report period handling =="
report_code() { curl -s -o /dev/null -w "%{http_code}" "$API/reports/annual.xlsx?$1" -H "Authorization: Bearer $SA"; }
[ "$(report_code 'year=abc')" = "400" ] && ok "invalid year rejected" || bad "year" "accepted"
[ "$(report_code 'from=2026-01-01')" = "400" ] && ok "from without to rejected" || bad "from" "accepted"
[ "$(report_code 'from=2026-03-01&to=2026-01-01')" = "400" ] && ok "reversed range rejected" || bad "range" "accepted"
curl -s -o /dev/null -D "$TMP/headers" "$API/reports/annual.xlsx?from=2026-01-01&to=2026-03-31" -H "Authorization: Bearer $SA"
grep -qi 'turath-report-2026-01-01_2026-03-31.xlsx' "$TMP/headers" && ok "file name identifies the quarter" || bad "file name" "$(cat "$TMP/headers")"
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Bad date draft\",\"college_id\":\"$COLLEGE\",\"details\":{\"publication_date\":\"garbage\"}}"; expect "draft keeps a partial date" 201
[ "$(report_code 'year=2024')" = "200" ] && ok "unvalidated draft date cannot break the report" || bad "report cast" "failed"

echo "== watchlist import failures =="
make_xlsx "$TMP/nocols.xlsx" '[["ISSN"],["2049-3630"]]'
upload /screening/import "$SA" "$TMP/nocols.xlsx"; expect "missing columns answer 200" 200
[ "$(echo "$BODY" | jq_ "sorted(d['data']['missing_columns'])")" = "['classification', 'name']" ] && ok "missing columns named" || bad "missing columns" "$BODY"
LONG=$(python3 -c "print('x'*600)")
make_xlsx "$TMP/rollback.xlsx" "[[\"اسم المجلة\",\"التصنيف\"],[\"Rollback First $RUN\",\"مفترسة\"],[\"$LONG\",\"مفترسة\"]]"
upload /screening/import "$SA" "$TMP/rollback.xlsx"; expect "unsavable row fails the import" 422
echo "$BODY" | grep -q "import_failed_line:3" && ok "error names the failing line" || bad "failing line" "$BODY"
[ "$(sqlv "SELECT COUNT(*) FROM journal_screening WHERE name='Rollback First $RUN'")" = "0" ] && ok "import is all or nothing" || bad "rollback" "row kept"

echo "== automatic journal check =="
CHK_RUN="e2e$RUN"
sqlv "DELETE FROM journal_reference_list WHERE source='e2e'" >/dev/null
sqlv "INSERT INTO journal_reference_list (name_key,name,url,url_host,list_type,source,fetched_at) VALUES ('$CHK_RUN listed journal','$CHK_RUN Listed Journal','http://$CHK_RUN-bad.org/','$CHK_RUN-bad.org','predatory_journal','e2e',NOW())" >/dev/null
sqlv "INSERT INTO journal_reference_list (name_key,name,url_host,list_type,source,fetched_at) VALUES ('$CHK_RUN press','$CHK_RUN Press','$CHK_RUN-press.org','predatory_publisher','e2e',NOW())" >/dev/null
sqlv "INSERT INTO journal_reference_list (name_key,name,url,url_host,list_type,authentic_name,authentic_url,source,fetched_at) VALUES ('$CHK_RUN acta','$CHK_RUN Acta','http://$CHK_RUN-clone.org/','$CHK_RUN-clone.org','hijacked','Acta','https://$CHK_RUN-real.org/','e2e',NOW())" >/dev/null
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/screening/check?name=x"); [ "$CODE" = "401" ] && ok "check needs auth" || bad "check auth" "$CODE"
call GET "/screening/check?issn=20493630" "$RES"; expect "check answers" 200
[ "$(echo "$BODY" | jq_ "d['data']['verdict']")" = "danger" ] && ok "manual predatory is danger" || bad "manual verdict" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['advisory']")" = "True" ] && ok "result labelled advisory" || bad "advisory" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['evidence']['scopus']['status']")" = "unknown" ] && ok "scopus stays unknown" || bad "scopus" "$BODY"
call GET "/screening/check?name=$CHK_RUN%20Listed%20Journal" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['reference_hits'][0]['list']")" = "predatory_journal" ] && ok "name hits predatory list" || bad "list hit" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['verdict']")" = "danger" ] && ok "predatory name is danger" || bad "list verdict" "$BODY"
call GET "/screening/check?publisher=$CHK_RUN%20Press" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['reference_hits'][0]['matched_by']")" = "publisher" ] && ok "publisher name checked" || bad "publisher hit" "$BODY"
call GET "/screening/check?url=https://ojs.$CHK_RUN-bad.org/x" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['reference_hits'][0]['matched_by']")" = "url" ] && ok "subdomain matches listed host" || bad "host hit" "$BODY"
call GET "/screening/check?name=$CHK_RUN%20Acta" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['verdict']")" = "caution" ] && ok "hijacked name alone is caution" || bad "hijacked name" "$BODY"
call GET "/screening/check?name=$CHK_RUN%20Acta&url=http://$CHK_RUN-clone.org/" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['verdict']")" = "danger" ] && ok "hijacked site is danger" || bad "hijacked url" "$BODY"
ARABIC_ISSN="%D9%A2%D9%A0%D9%A4%D9%A9-%D9%A3%D9%A6%D9%A3%D9%A0"
call GET "/screening/check?issn=$ARABIC_ISSN" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['verdict']")" = "danger" ] && ok "Arabic-Indic ISSN normalised" || bad "arabic issn" "${BODY:0:200}"
call POST /portal/requests "$RES" "{\"kind\":\"journal_verification\",\"title_en\":\"Arabic ISSN\",\"doi\":\"10.9999/arissn.$RUN\",\"college_id\":\"$COLLEGE\",\"details\":{\"journal_name\":\"J\",\"issn\":\"٢٠٤٩-٣٦٣٠\"}}"; expect "Arabic-Indic ISSN draft saved" 201
[ "$(echo "$BODY" | jq_ "d['data']['details']['issn']")" = "2049-3630" ] && ok "request stores ASCII ISSN" || bad "stored issn" "${BODY:0:200}"
call GET "/screening/check?url=javascript:alert(1)" "$RES"
echo "$BODY" | grep -q "url_rejected" && ok "rejected URL gets a reason" || bad "url_rejected" "${BODY:0:200}"
call GET "/screening/check?name=e2e%20no%20such%20journal%20$RUN" "$RES"
[ "$(echo "$BODY" | jq_ "'no_evidence' in d['data']['reasons'] and 'not_checked' not in d['data']['reasons'] or 'openalex' in d['data']['sources_failed']")" = "True" ] && ok "unknown full name is no evidence, not incomplete input" || bad "no_evidence" "${BODY:0:200}"
call GET /screening "$RES"; expect "watchlist stays staff-only" 403
LONGQ=$(python3 -c "print('x'*600)")
call GET "/screening/check?name=$LONGQ" "$RES"; expect "over-long name rejected" 422
echo "$BODY" | grep -q "journal_name:too_long" && ok "too-long code" || bad "too-long code" "$BODY"
call GET "/screening/check" "$RES"; expect "empty check still answers" 200
[ "$(echo "$BODY" | jq_ "d['data']['matched']")" = "False" ] && ok "empty check is unmatched" || bad "empty check" "$BODY"
call GET "/portal/requests/$FLAGREQ" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['journal_flag']['verdict']")" = "danger" ] && ok "request detail flag enriched" || bad "enriched flag" "$BODY"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/screening/status"); [ "$CODE" = "401" ] && ok "status needs auth" || bad "status auth" "$CODE"
call GET /screening/status "$RES"; expect "list status answers" 200
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "3" ] && ok "status covers three lists" || bad "status" "$BODY"
sqlv "DELETE FROM journal_reference_list WHERE source='e2e'" >/dev/null

echo "== research profile: links, ORCID sync, attribution =="
SAMPLE_ORCID="0000-0002-1825-0097"
sqlv "UPDATE researchers SET orcid_id=NULL WHERE orcid_id='$SAMPLE_ORCID'" >/dev/null
PROF_EMAIL="prof$RUN@test.iq"; PROF2_EMAIL="prof2$RUN@test.iq"
# Josiah Carberry is ORCID's public sample person, so this researcher passes the name check.
register "$PROF_EMAIL" "Josiah Carberry" >/dev/null; register "$PROF2_EMAIL" "باحث ثان" >/dev/null
PROF=$(login "$PROF_EMAIL"); PROF2=$(login "$PROF2_EMAIL")
links() { echo "{\"google_scholar_url\":\"$1\",\"scopus_url\":\"$2\",\"researchgate_url\":\"$3\",\"linkedin_url\":\"$4\",\"orcid\":\"$5\"}"; }
wait_sync() { # TOKEN: polls until the run ends, leaves the profile in BODY
  for _ in $(seq 1 45); do
    call GET /me/research-profile "$1"
    [ "$(echo "$BODY" | jq_ "d['data']['syncing']")" = "False" ] && return 0
    sleep 2
  done
  return 1
}
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/me/research-profile"); [ "$CODE" = "401" ] && ok "profile needs auth" || bad "profile auth" "$CODE"
call GET /me/research-profile "$PROF"; expect "empty profile answers" 200
[ "$(echo "$BODY" | jq_ "d['data']['publications']")" = "[]" ] && ok "empty profile has no publications" || bad "empty profile" "$BODY"
[ "$(sqlv "SELECT COUNT(*) FROM researchers WHERE email='$PROF_EMAIL'")" = "0" ] && ok "reading creates no researcher" || bad "read side effect" "row created"
call POST /me/research-profile/sync "$PROF"; expect "sync without ORCID refused" 422
echo "$BODY" | grep -q "orcid:orcid_required" && ok "orcid_required code" || bad "orcid_required" "$BODY"
call PUT /me/research-profile "$PROF" "$(links "https://scholar.google.evil.com/citations?user=AbC-12_xyzAAJ" "" "" "" "0000-0002-1825-0098")"
expect "bad links rejected" 422
echo "$BODY" | grep -q "google_scholar_url:wrong_host" && echo "$BODY" | grep -q "orcid:invalid_orcid" && ok "every bad field reported" || bad "link codes" "$BODY"
SCHOLAR="AbC$RUN"; SCOPUS="57$RUN"
call PUT /me/research-profile "$PROF" "$(links "https://scholar.google.com/citations?hl=ar&user=$SCHOLAR" "https://www.scopus.com/authid/detail.uri?authorId=$SCOPUS" "https://www.researchgate.net/profile/Prof-$RUN/stats" "https://www.linkedin.com/in/prof-$RUN/" "https://orcid.org/$SAMPLE_ORCID")"
expect "links saved" 200
[ "$(echo "$BODY" | jq_ "d['data']['orcid']")" = "$SAMPLE_ORCID" ] && ok "ORCID extracted from URL" || bad "orcid" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['researchgate_url']")" = "https://www.researchgate.net/profile/Prof-$RUN" ] && ok "link canonicalized" || bad "canonical" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['syncing']")" = "True" ] && ok "new ORCID starts a sync" || bad "auto sync" "$BODY"
[ "$(sqlv "SELECT google_scholar_id FROM researchers WHERE email='$PROF_EMAIL'")" = "$SCHOLAR" ] && ok "scholar id stored" || bad "scholar id" "missing"
call POST /me/research-profile/sync "$PROF"; expect "second sync refused by cooldown" 429
echo "$BODY" | grep -q "sync_cooldown" && ok "cooldown code" || bad "cooldown code" "$BODY"
call PUT /me/research-profile "$PROF" "$(links "" "" "" "" "0000-0002-1694-233X")"; expect "ORCID change refused while a sync runs" 429
echo "$BODY" | grep -q "orcid_change_cooldown" && ok "ORCID change code" || bad "ORCID change code" "$BODY"
[ "$(sqlv "SELECT orcid_id FROM researchers WHERE email='$PROF_EMAIL'")" = "$SAMPLE_ORCID" ] && ok "refused change saved nothing" || bad "refused change" "stored"
call PUT /me/research-profile "$PROF2" "$(links "" "" "" "" "$SAMPLE_ORCID")"; expect "duplicate ORCID refused" 409
echo "$BODY" | grep -q "orcid_taken" && ok "orcid_taken code" || bad "orcid_taken" "$BODY"
call PUT /me/research-profile "$PROF2" "$(links "https://scholar.google.com/citations?user=$SCHOLAR" "" "" "" "")"; expect "duplicate scholar refused" 409
call PUT /me/research-profile "$PROF2" "$(links "" "https://www.scopus.com/authid/detail.uri?authorId=$SCOPUS" "" "" "")"; expect "duplicate scopus refused" 409
wait_sync "$PROF" && ok "sync finishes" || bad "sync finishes" "$BODY"
SYNC_ERR=$(echo "$BODY" | jq_ "d['data']['last_sync_error']")
# OpenAlex files the sample ORCID's top author under a different person, so it must not be trusted.
[ "$SYNC_ERR" = "openalex_name_mismatch" ] && ok "OpenAlex author refused by name" || bad "OpenAlex refusal" "$SYNC_ERR"
[ "$(echo "$BODY" | jq_ "d['data']['openalex_id']")" = "None" ] && [ "$(echo "$BODY" | jq_ "d['data']['h_index']")" = "0" ] && [ "$(echo "$BODY" | jq_ "d['data']['total_citations']")" = "0" ] && ok "no OpenAlex metrics from the other person" || bad "OpenAlex metrics" "$BODY"
FIRST_COUNT=$(echo "$BODY" | jq_ "d['data']['total_publications']")
[ "${FIRST_COUNT:-0}" -gt 0 ] && ok "publications imported ($FIRST_COUNT)" || bad "publications imported" "$BODY"
[ "$(echo "$BODY" | jq_ "len(d['data']['publications']) > 0")" = "True" ] && ok "publications listed" || bad "list" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['last_synced_at'] is not None")" = "True" ] && ok "last sync time set" || bad "last sync" "$BODY"
PROF_RES=$(sqlv "SELECT id FROM researchers WHERE email='$PROF_EMAIL'")
NAMESAKE=$(sqlv "INSERT INTO publications (title_en, citation_count_updated_at) VALUES ('Namesake $RUN', NOW()) RETURNING id")
sqlv "INSERT INTO publication_authors (publication_id, researcher_id, author_name, author_order, source) VALUES ('$NAMESAKE','$PROF_RES','x',1,'sync')" >/dev/null
PORTAL_PUB=$(sqlv "INSERT INTO publications (title_en) VALUES ('Portal paper $RUN') RETURNING id")
sqlv "INSERT INTO publication_authors (publication_id, researcher_id, author_name, author_order) VALUES ('$PORTAL_PUB','$PROF_RES','x',1)" >/dev/null
sqlv "UPDATE researchers SET last_sync_started_at = NOW() - interval '10 minutes' WHERE id='$PROF_RES'" >/dev/null
call POST /me/research-profile/sync "$PROF"; expect "refresh accepted after cooldown" 202
wait_sync "$PROF" && ok "refresh finishes" || bad "refresh finishes" "$BODY"
[ "$(sqlv "SELECT COUNT(*) FROM publications WHERE id='$NAMESAKE'")" = "0" ] && ok "misattributed work withdrawn" || bad "namesake" "still linked"
[ "$(echo "$BODY" | jq_ "d['data']['total_publications']")" = "$((FIRST_COUNT+1))" ] && ok "refresh adds no duplicates, keeps portal paper" || bad "idempotent" "$BODY"
call GET "/researchers/$PROF_RES/links" "$SA"; expect "staff read links" 200
[ "$(echo "$BODY" | jq_ "d['data']['orcid_url']")" = "https://orcid.org/$SAMPLE_ORCID" ] && ok "staff see ORCID link" || bad "staff links" "$BODY"
call GET "/researchers/$PROF_RES/links" "$PROF2"; expect "researcher cannot read staff links" 403
call PUT /me/research-profile "$PROF" "$(links "" "" "" "" "")"; expect "links cleared" 200
[ "$(echo "$BODY" | jq_ "d['data']['total_publications']")" = "1" ] && ok "clearing ORCID drops imported data" || bad "clear orcid" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['h_index']")" = "0" ] && ok "clearing ORCID resets h-index" || bad "clear h" "$BODY"
call PUT /me/research-profile "$PROF" "$(links "" "" "" "" "$SAMPLE_ORCID")"; expect "clearing then re-entering does not dodge the cooldown" 429
call PUT /me/research-profile "$PROF2" "$(links "" "" "" "" "$SAMPLE_ORCID")"; expect "released ORCID can be claimed" 200
wait_sync "$PROF2" >/dev/null
[ "$(echo "$BODY" | jq_ "d['data']['last_sync_error']")" = "orcid_name_mismatch" ] && ok "another person's ORCID flagged" || bad "orcid mismatch" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['total_publications']")" = "0" ] && [ "$(echo "$BODY" | jq_ "d['data']['openalex_id']")" = "None" ] && ok "nothing imported from another person's ORCID" || bad "mismatch import" "$BODY"
PROF2_RES=$(sqlv "SELECT id FROM researchers WHERE email='$PROF2_EMAIL'")
call GET "/researchers/$PROF2_RES/links" "$SA"
[ "$(echo "$BODY" | jq_ "d['data']['last_sync_error']")" = "orcid_name_mismatch" ] && ok "staff see the review flag" || bad "review flag" "$BODY"
call GET "/researchers/$PROF2_RES" "$SA"
[ "$(echo "$BODY" | jq_ "d['data']['needs_identity_review']")" = "True" ] && ok "researcher record marked for review" || bad "review mark" "flag missing"
call PUT /me/research-profile "$PROF2" "$(links "" "" "" "" "")"; expect "second researcher clears ORCID" 200
sqlv "UPDATE researchers SET orcid_id='https://orcid.org/0000-0002-1694-233X' WHERE id='$PROF2_RES'" >/dev/null
call PUT /me/research-profile "$PROF" "$(links "" "" "" "" "0000-0002-1694-233X")"; expect "legacy URL spelling still counts as taken" 409
sqlv "UPDATE researchers SET orcid_id=NULL WHERE id='$PROF2_RES'" >/dev/null

echo "== access control on shared registers =="
[ "$(sqlv "SELECT version FROM schema_migrations")" -ge 13 ] && ok "canonical ORCID migration applied" || bad "migration" "not applied"
[ "$(sqlv "SELECT COUNT(*) FROM researchers WHERE orcid_id !~ '^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{3}[0-9X]\$'")" = "0" ] && ok "stored ORCIDs are canonical" || bad "canonical rows" "some rows differ"
call PUT "/researchers/$PROF2_RES" "$PROF" "{\"email\":\"stolen$RUN@test.iq\"}"; expect "researcher cannot change another's email" 403
call PUT "/researchers/$PROF2_RES" "$DEAN" "{\"email\":\"stolen$RUN@test.iq\"}"; expect "dean cannot change a researcher's email" 403
call PUT "/researchers/$PROF2_RES" "$VP" "{\"title_ar\":\"x\"}"; expect "VP cannot edit the register" 403
call POST /researchers "$RES" "{\"name_ar\":\"x\",\"email\":\"new$RUN@test.iq\"}"; expect "researcher cannot create a researcher" 403
call DELETE "/researchers/$PROF2_RES" "$RES"; expect "researcher cannot delete a researcher" 403
call POST /publications "$RES" "{\"title_en\":\"x\"}"; expect "researcher cannot create a publication" 403
call DELETE "/publications/$UP_PUB" "$RES"; expect "researcher cannot delete a publication" 403
call POST /colleges "$RES" "{\"name_ar\":\"x\"}"; expect "researcher cannot create a college" 403
call GET /export/researchers "$RES"; expect "researcher cannot export the register" 403
call GET /export/researchers "$VP"; expect "VP cannot export the register" 403
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/export/researchers" -H "Authorization: Bearer $SA"); [ "$CODE" = "200" ] && ok "scientific affairs export" || bad "export" "$CODE"
call POST "/sync/researcher/$PROF2_RES" "$RES"; expect "researcher cannot sync another" 403
call POST /sync/doi "$RES" "{\"doi\":\"10.1038/nature12373\"}"; expect "researcher cannot enrich a DOI" 403
call GET /sync/jobs "$DEAN"; expect "dean cannot read sync jobs" 403
call POST /sync/doi "$SA" "{\"doi\":\"not-a-doi\"}"; expect "malformed DOI refused" 400
call POST /sync/doi "$SA" "{\"doi\":\"https://doi.org/10.9999/none.$RUN\"}"; expect "scientific affairs enrich a DOI" 202
call POST "/sync/researcher/$PROF2_RES" "$SA"; expect "cooldown still binds staff" 429
call GET /ai/trends "$RES"; expect "researcher cannot spend AI budget" 403
call GET "/ai/strategy/$PROF2_RES" "$DEAN"; expect "dean cannot spend AI budget" 403
call GET /promotions "$RES"; expect "researcher cannot list promotions" 403
call GET "/promotions/eligibility/$PROF2_RES" "$RES"; expect "researcher cannot read another's eligibility" 403
call GET "/incentives/calculate/$PROF2_RES" "$RES"; expect "researcher cannot read another's incentives" 403
call POST /promotions "$RES" "{\"researcher_id\":\"$PROF2_RES\",\"current_rank\":\"lecturer\",\"target_rank\":\"professor\"}"; expect "researcher cannot apply for another" 403
call POST /sabbaticals "$RES" "{\"researcher_id\":\"$PROF2_RES\"}"; expect "researcher cannot request leave for another" 403
call POST /conferences "$RES" "{\"researcher_id\":\"$PROF2_RES\"}"; expect "researcher cannot file a conference for another" 403
call GET /accreditation/dashboard "$RES"; expect "researcher cannot read accreditation" 403
call POST /accreditation/evidence "$RES" "{}"; expect "researcher cannot add evidence" 403
call POST /workflows/initiate "$RES" "{}"; expect "researcher cannot start a workflow" 403
WF_REQ=$(sqlv "SELECT gen_random_uuid()")
call POST /workflows/initiate "$SA" "{\"workflow_type\":\"promotion\",\"request_id\":\"$WF_REQ\"}"; expect "scientific affairs start a workflow" 201
WF_STEP=$(sqlv "SELECT id FROM approval_workflows WHERE request_id='$WF_REQ' ORDER BY step_order LIMIT 1")
WF_ROLE=$(sqlv "SELECT approver_role FROM approval_workflows WHERE id='$WF_STEP'")
call PUT "/workflows/$WF_STEP/decide" "$RES" '{"decision":"approved"}'; expect "researcher cannot decide a workflow step" 404
[ "$(sqlv "SELECT decision FROM approval_workflows WHERE id='$WF_STEP'")" = "pending" ] && ok "refused decision changed nothing" || bad "workflow decision" "changed"
sqlv "DELETE FROM approval_workflows WHERE request_id='$WF_REQ'" >/dev/null
call GET /workflows/pending "$RES"; expect "pending list answers a role with no approver step" 200
call GET "/researchers?limit=1" "$RES"; expect "researchers list still answers" 200
[ "$(echo "$BODY" | jq_ "all(not r['email'] for r in d['data'])")" = "True" ] && ok "email hidden from non-staff" || bad "email redaction" "email present"
call GET "/researchers?limit=1" "$SA"
[ "$(echo "$BODY" | jq_ "all(r['email'] for r in d['data'])")" = "True" ] && ok "email shown to staff" || bad "email for staff" "email missing"
call GET "/researchers?q=$PROF2_EMAIL" "$RES"
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "0" ] && ok "non-staff cannot search by email" || bad "email search" "matched"
call GET "/researchers?q=$PROF2_EMAIL" "$SA"
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "1" ] && ok "staff can search by email" || bad "staff email search" "no match"
call POST /researchers "$SA" "{\"name_ar\":\"x\",\"email\":\"reg$RUN@test.iq\",\"orcid_id\":\"not-an-orcid\"}"; expect "register refuses a malformed ORCID" 422
call POST /researchers "$SA" "{\"name_ar\":\"x\",\"email\":\"reg$RUN@test.iq\",\"orcid_id\":\"https://orcid.org/0000-0002-1694-233x\"}"; expect "scientific affairs create a researcher" 201
REG=$(echo "$BODY" | jq_ "d['data']['id']")
[ "$(sqlv "SELECT orcid_id FROM researchers WHERE id='$REG'")" = "0000-0002-1694-233X" ] && ok "create canonicalizes ORCID" || bad "create canonical" "raw stored"
call POST /researchers "$SA" "{\"name_ar\":\"x\",\"email\":\"reg2$RUN@test.iq\",\"orcid_id\":\"0000-0002-1694-233X\"}"; expect "create refuses a taken ORCID" 409
call PUT "/researchers/$PROF2_RES" "$SA" "{\"orcid_id\":\"0000-0002-1694-233x\"}"; expect "update refuses a taken ORCID" 409
call PUT "/researchers/$REG" "$SA" "{\"orcid_id\":\"\"}"; expect "update clears an ORCID" 200
[ -z "$(sqlv "SELECT orcid_id FROM researchers WHERE id='$REG'")" ] && ok "cleared ORCID is NULL" || bad "clear via update" "still set"
call DELETE "/researchers/$REG" "$SA"; expect "scientific affairs delete a researcher" 200
sqlv "DELETE FROM publications WHERE id='$PORTAL_PUB'" >/dev/null

echo "== researcher contact data: owner, staff, nobody else =="
gen_orcid() { python3 -c "
import sys
d='%015d'%int(sys.argv[1]); t=0
for c in d: t=(t+int(c))*2
r=(12-t%11)%11; s=d+('X' if r==10 else str(r))
print('-'.join(s[i:i+4] for i in range(0,16,4)))" "$1"; }
absent() { # NAME NEEDLE: the last BODY must not hold NEEDLE
  case "$BODY" in *"$2"*) bad "$1" "contact data present";; *) ok "$1";; esac; }
CPHONE="07$(printf %09d $((RUN % 1000000000)))"; CPHONE2="07$(printf %09d $(((RUN + 1) % 1000000000)))"; CREC="rec$RUN@backup.example"
COWN_EMAIL="cown$RUN@test.iq"; COTH_EMAIL="coth$RUN@test.iq"
register "$COWN_EMAIL" "باحث تواصل" >/dev/null; register "$COTH_EMAIL" "باحث غريب" >/dev/null
COWN=$(login "$COWN_EMAIL"); COTH=$(login "$COTH_EMAIL")
contact() { echo "{\"phone\":\"$1\",\"phone_secondary\":\"$2\",\"recovery_email\":\"$3\"}"; }
call PUT /me/research-profile "$COWN" "$(contact 0770123 "" "")"; expect "short phone refused" 422
[ "${BODY#*phone:invalid_phone}" != "$BODY" ] && ok "phone problem named" || bad "phone code" "$BODY"
call PUT /me/research-profile "$COWN" "$(contact 08801234567 "" "")"; expect "non-07 phone refused" 422
call PUT /me/research-profile "$COWN" "$(contact "" 123 "")"; expect "bad second phone refused" 422
call PUT /me/research-profile "$COWN" "$(contact "" "" not-an-email)"; expect "invalid recovery email refused" 422
[ "${BODY#*recovery_email:invalid_email}" != "$BODY" ] && ok "recovery email problem named" || bad "email code" "$BODY"
call PUT /me/research-profile "$COWN" "$(contact "$CPHONE" "+964${CPHONE2#0}" "$CREC")"; expect "owner saves contact data" 200
[ "$(echo "$BODY" | jq_ "d['data']['phone_secondary']")" = "$CPHONE2" ] && ok "+964 number stored as 07" || bad "phone canonical" "not canonical"
call GET /me/research-profile "$COWN"
[ "$(echo "$BODY" | jq_ "d['data']['recovery_email']")" = "$CREC" ] && ok "owner reads own recovery email" || bad "owner read" "missing"
CRES=$(sqlv "SELECT id FROM researchers WHERE email='$COWN_EMAIL'")
call PUT /me/research-profile "$COWN" "$(links "" "" "" "" "")"; expect "links form without contact fields" 200
[ "$(sqlv "SELECT phone FROM researchers WHERE id='$CRES'")" = "$CPHONE" ] && ok "omitted contact fields are kept" || bad "omitted" "phone changed"
call GET "/researchers/$CRES/contact" "$COWN"; expect "owner reads contact" 200
call GET "/researchers/$CRES/contact" "$COTH"; expect "another researcher cannot read contact" 403
absent "refusal carries no phone" "$CPHONE"
call GET "/researchers/$CRES/contact" "$DEAN"; expect "dean cannot read contact" 403
call GET "/researchers/$CRES/contact" "$SA"; expect "scientific affairs read contact" 200
[ "$(echo "$BODY" | jq_ "d['data']['phone']")" = "$CPHONE" ] && ok "staff receive the phone" || bad "staff phone" "missing"
call GET "/researchers/$CRES/contact" "$VP"; expect "VP reads contact" 200
call GET "/researchers/$CRES/contact" "$ADM"; expect "admin reads contact" 200
call GET "/researchers/not-a-uuid/contact" "$SA"; expect "malformed id is not found" 404
for who in COTH DEAN SA; do
  tok="${!who}"
  for path in "/researchers/$CRES" "/researchers?q=$(python3 -c "import urllib.parse;print(urllib.parse.quote('باحث تواصل'))")" "/researchers?limit=100" "/metrics/researcher/$CRES" "/search?q=$CPHONE" "/metrics/leaderboard"; do
    call GET "$path" "$tok"
    for needle in "$CPHONE" "$CPHONE2" "$CREC"; do absent "$who: ${path%%\?*} hides contact" "$needle"; done
  done
done
call GET "/researchers?q=$CPHONE" "$SA"
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "0" ] && ok "register cannot be searched by phone" || bad "phone search" "matched"
call GET "/researchers/$CRES/links" "$SA"; absent "staff links carry no contact" "$CPHONE"
curl -s -o "$TMP/export.csv" "$API/export/researchers" -H "Authorization: Bearer $SA"; BODY=$(cat "$TMP/export.csv")
absent "export omits the phone" "$CPHONE"; absent "export omits the second phone" "$CPHONE2"; absent "export omits the recovery email" "$CREC"
[ "$(sqlv "SELECT COUNT(*) FROM notifications WHERE body LIKE '%$CREC%'")" = "0" ] && ok "notifications never carry the recovery email" || bad "notification" "present"
call PUT /me/research-profile "$COWN" "$(contact "" "" "")"; expect "owner clears contact data" 200
[ -z "$(sqlv "SELECT phone FROM researchers WHERE id='$CRES'")" ] && ok "cleared phone is NULL" || bad "clear phone" "still set"

echo "== Web of Science link =="
wos() { echo "{\"web_of_science_url\":\"$1\"}"; }
call PUT /me/research-profile "$COWN" "$(wos "https://webofscience.com.evil.org/wos/author/record/123456")"; expect "foreign host refused" 422
[ "${BODY#*web_of_science_url:wrong_host}" != "$BODY" ] && ok "host problem named" || bad "wos code" "$BODY"
call PUT /me/research-profile "$COWN" "$(wos "https://www.webofscience.com/wos/woscc/summary/x")"; expect "non-profile link refused" 422
call PUT /me/research-profile "$COWN" "$(wos "webofscience.com/wos/author/record/$RUN?state=x")"; expect "profile link saved" 200
[ "$(echo "$BODY" | jq_ "d['data']['web_of_science_url']")" = "https://www.webofscience.com/wos/author/record/$RUN" ] && ok "link canonicalized" || bad "wos canonical" "$BODY"
call GET "/researchers/$CRES/links" "$SA"
[ "$(echo "$BODY" | jq_ "d['data']['web_of_science_url']")" = "https://www.webofscience.com/wos/author/record/$RUN" ] && ok "staff see the link" || bad "wos staff" "$BODY"

echo "== researcher import: optional columns =="
IMP_HEAD='"name_ar","email","phone","phone_secondary","recovery_email","google_scholar_url","scopus_url","researchgate_url","linkedin_url","web_of_science_url","orcid","employee_no","title_ar","name_en","college","department","account_status"'
IMP1="imp$RUN@test.iq"; IMP2="imp2$RUN@test.iq"; IMP_BAD="impbad$RUN@test.iq"; IMP_LINK="implink$RUN@test.iq"
IMP_ORCID=$(gen_orcid $((RUN * 10 + 1))); IMP_ORCID2=$(gen_orcid $((RUN * 10 + 2)))
IMP_SCHOLAR="Sch${RUN}AAJ"; IMP_COLLEGE="كلية استيراد $RUN"
register "$IMP_LINK" "حساب سابق" >/dev/null
make_xlsx "$TMP/imp.xlsx" "[[$IMP_HEAD],
 [\"باحث مستورد\",\"$IMP1\",\"${CPHONE#0}\",\"$CPHONE2\",\"$CREC\",\"https://scholar.google.com/citations?user=$IMP_SCHOLAR\",\"https://www.scopus.com/authid/detail.uri?authorId=$RUN\",\"https://www.researchgate.net/profile/Imp-$RUN\",\"https://www.linkedin.com/in/imp-$RUN\",\"https://www.webofscience.com/wos/author/record/$RUN\",\"$IMP_ORCID\",\"E-$RUN\",\"مدرس\",\"Imported Person\",\"$IMP_COLLEGE\",\"قسم $RUN\",\"needs_review\"],
 [\"صف معطوب\",\"$IMP_BAD\",\"12345\",\"\",\"bad-email\",\"\",\"https://evil.example/x\",\"\",\"\",\"https://evil.example/y\",\"\",\"\",\"\",\"\",\"\",\"\",\"unknown\"],
 [\"حساب مرتبط\",\"$IMP_LINK\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"linked\"]]"
upload /admin/import/researchers "$RES" "$TMP/imp.xlsx"; expect "researcher cannot import" 403
upload /admin/import/researchers "$SA" "$TMP/imp.xlsx"; expect "scientific affairs cannot import" 403
upload "/admin/import/researchers?dry_run=true" "$ADM" "$TMP/imp.xlsx"; expect "dry run answers" 200
absent "preview carries no phone" "$CPHONE2"; absent "preview carries no recovery email" "$CREC"
[ "$(sqlv "SELECT COUNT(*) FROM researchers WHERE email IN ('$IMP1','$IMP_LINK')")" = "0" ] && ok "dry run writes nothing" || bad "dry run" "rows written"
[ "$(echo "$BODY" | jq_ "sorted(p['code'] for p in d['data']['problems'])")" = "['invalid_account_status', 'invalid_phone', 'invalid_recovery_email', 'invalid_scopus_url', 'invalid_web_of_science_url']" ] && ok "every bad cell reported" || bad "dry problems" "$BODY"
JOBS_BEFORE=$(sqlv "SELECT COUNT(*) FROM sync_jobs")
upload /admin/import/researchers "$ADM" "$TMP/imp.xlsx"; expect "import runs" 200
[ "$(echo "$BODY" | jq_ "d['data']['created']")" = "2" ] && ok "good rows saved around a bad one" || bad "created" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['users_linked']")" = "1" ] && ok "existing account linked by email" || bad "linked" "$BODY"
[ "$(sqlv "SELECT COUNT(*) FROM researchers WHERE email='$IMP_BAD'")" = "0" ] && ok "bad row stored nothing" || bad "bad row" "stored"
imported() { sqlv "SELECT $1 FROM researchers WHERE email='$IMP1'"; }
[ "$(imported phone)" = "$CPHONE" ] && ok "dropped leading zero restored" || bad "import phone" "wrong"
[ "$(imported phone_secondary)" = "$CPHONE2" ] && [ "$(imported recovery_email)" = "$CREC" ] && [ "$(imported employee_no)" = "E-$RUN" ] && [ "$(imported account_status)" = "needs_review" ] && ok "contact columns stored" || bad "import contact" "wrong"
[ "$(imported google_scholar_id)" = "$IMP_SCHOLAR" ] && [ "$(imported scopus_author_id)" = "$RUN" ] && [ "$(imported orcid_id)" = "$IMP_ORCID" ] && [ "$(imported web_of_science_url)" = "https://www.webofscience.com/wos/author/record/$RUN" ] && ok "link columns stored with their ids" || bad "import links" "wrong"
[ -z "$(imported last_sync_started_at)" ] && [ "$(sqlv "SELECT COUNT(*) FROM sync_jobs")" = "$JOBS_BEFORE" ] && ok "import starts no sync" || bad "import sync" "started"
make_xlsx "$TMP/imp2.xlsx" "[[$IMP_HEAD],
 [\"باحث مستورد\",\"$IMP1\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\"],
 [\"ينتحل\",\"$IMP2\",\"\",\"\",\"\",\"https://scholar.google.com/citations?user=$IMP_SCHOLAR\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"\"]]"
upload /admin/import/researchers "$ADM" "$TMP/imp2.xlsx"; expect "second import runs" 200
[ "$(echo "$BODY" | jq_ "[p['code'] for p in d['data']['problems']]")" = "['google_scholar_taken']" ] && ok "taken identifier reported by code" || bad "taken" "$BODY"
[ "$(imported phone)" = "$CPHONE" ] && [ "$(imported recovery_email)" = "$CREC" ] && [ "$(imported google_scholar_id)" = "$IMP_SCHOLAR" ] && [ "$(imported department_id | wc -c | tr -d ' ')" -gt 1 ] && [ "$(imported title_ar)" = "مدرس" ] && ok "empty cells erase nothing" || bad "empty cells" "value erased"
make_xlsx "$TMP/imp3.xlsx" "[[$IMP_HEAD],
 [\"باحث مستورد\",\"$(echo "$IMP1" | tr a-z A-Z)\",\"$CPHONE2\",\"\",\"\",\"\",\"\",\"\",\"\",\"\",\"$IMP_ORCID2\",\"\",\"\",\"\",\"\",\"\",\"\"]]"
upload /admin/import/researchers "$ADM" "$TMP/imp3.xlsx"; expect "third import runs" 200
[ "$(echo "$BODY" | jq_ "d['data']['updated']")" = "1" ] && ok "email matches whatever its case" || bad "case" "$BODY"
[ "$(echo "$BODY" | jq_ "[p['code'] for p in d['data']['problems']]")" = "['orcid_kept']" ] && ok "differing ORCID reported" || bad "orcid kept" "$BODY"
[ "$(imported orcid_id)" = "$IMP_ORCID" ] && [ "$(imported phone)" = "$CPHONE2" ] && ok "stored ORCID kept, the rest applied" || bad "orcid stays" "changed"
IMP_ME=$(login "$IMP_LINK"); call GET /me/research-profile "$IMP_ME"
[ "$(echo "$BODY" | jq_ "d['data']['linked']")" = "True" ] && ok "imported researcher reaches the account by email" || bad "link profile" "$BODY"

echo "== bulk ORCID sync =="
call GET /admin/sync/orcid "$RES"; expect "researcher cannot read bulk progress" 403
call POST /admin/sync/orcid "$RES" '{}'; expect "researcher cannot start a bulk sync" 403
call POST /admin/sync/orcid "$SA" '{}'; expect "scientific affairs cannot start a bulk sync" 403
call POST /admin/sync/orcid "$DEAN" '{}'; expect "dean cannot start a bulk sync" 403
call POST /admin/sync/orcid/stop "$SA" '{}'; expect "scientific affairs cannot stop a bulk sync" 403
if [ "$(sqlv "SELECT COUNT(*) FROM bulk_sync_runs WHERE status IN ('running','waiting')")" != "0" ]; then
  bad "bulk preconditions" "a real run is active; skipped the rest"
else
  call GET /admin/sync/orcid "$ADM"; expect "admin reads bulk progress" 200
  [ "$(echo "$BODY" | jq_ "d['data']['eligible'] > 0")" = "True" ] && ok "progress counts eligible researchers" || bad "eligible" "$BODY"
  call POST /admin/sync/orcid/stop "$ADM" '{}'; expect "nothing to stop" 409
  # The cursor sits past every real id, so this pass can reach only the probe researcher.
  CUT="ffffffff-ffff-ffff-ffff-000000000000"; PROBE=$(printf 'ffffffff-ffff-ffff-ffff-%012d' $((RUN % 1000000000000)))
  PROBE_ORCID=$(gen_orcid $((RUN * 10 + 3)))
  sqlv "INSERT INTO researchers (id, name_ar, email, orcid_id) VALUES ('$PROBE', 'باحث فحص', 'probe$RUN@test.iq', '$PROBE_ORCID')" >/dev/null
  WAITRUN=$(sqlv "INSERT INTO bulk_sync_runs (status, cursor_id, wait_reason, resume_at) VALUES ('waiting', '$CUT', 'daily_limit', NOW() + interval '1 hour') RETURNING id")
  call POST /admin/sync/orcid/stop "$ADM" '{}'; expect "admin stops a waiting pass" 200
  [ "$(echo "$BODY" | jq_ "d['data']['run']['status']")" = "stopped" ] && [ "$(echo "$BODY" | jq_ "d['data']['remaining']")" = "1" ] && ok "stopped pass reports what is left" || bad "stop" "$BODY"
  [ -z "$(sqlv "SELECT last_sync_started_at FROM researchers WHERE id='$PROBE'")" ] && ok "a waiting pass synced nothing" || bad "waiting" "synced"
  call POST /admin/sync/orcid "$ADM" '{}'; expect "stopped pass resumes" 202
  [ "$(echo "$BODY" | jq_ "d['data']['run']['id']")" = "$WAITRUN" ] && ok "resume continues the same pass" || bad "resume id" "$BODY"
  for _ in $(seq 1 40); do
    call GET /admin/sync/orcid "$ADM"
    [ "$(echo "$BODY" | jq_ "d['data']['run']['status']")" = "completed" ] && break
    sleep 2
  done
  [ "$(echo "$BODY" | jq_ "d['data']['run']['status']")" = "completed" ] && ok "pass completes" || bad "pass" "$BODY"
  [ "$(echo "$BODY" | jq_ "d['data']['run']['processed']")" = "1" ] && [ "$(echo "$BODY" | jq_ "d['data']['remaining']")" = "0" ] && ok "pass settled only what followed the cursor" || bad "processed" "$BODY"
  [ -n "$(sqlv "SELECT last_sync_started_at FROM researchers WHERE id='$PROBE'")" ] && ok "probe researcher was synced" || bad "probe" "not synced"
  sqlv "DELETE FROM bulk_sync_runs WHERE id='$WAITRUN'" >/dev/null
  sqlv "DELETE FROM sync_jobs WHERE target_id='$PROBE'" >/dev/null
  sqlv "DELETE FROM researchers WHERE id='$PROBE'" >/dev/null
fi

echo
echo "Passed: $PASS  Failed: $FAIL"
[ "$FAIL" -eq 0 ]
