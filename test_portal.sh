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
PLAN_DETAILS='{"work_type":"joint","co_researcher_1":"د. علي حسن","expected_completion":"2027-06-30"}'

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

echo "== research plan: co-researchers =="
co_plan() { call POST /portal/requests "$RES" "{\"kind\":\"research_plan\",\"title_ar\":\"مشترك\",\"title_en\":\"Joint\",\"college_id\":\"$COLLEGE\",\"details\":$1}"
  CP=$(echo "$BODY" | jq_ "d['data']['id']"); upload "/portal/requests/$CP/files?role=research_form" "$RES" "$TMP/doc.pdf" >/dev/null
  call POST "/portal/requests/$CP/submit" "$RES"; }
co_plan '{"work_type":"joint","expected_completion":"2027-06-30"}'; expect "joint plan needs a co-researcher" 422
echo "$BODY" | grep -q "co_researchers:required" && ok "missing co-researchers named" || bad "co required msg" "$BODY"
co_plan '{"work_type":"individual","co_researcher_1":"أ","expected_completion":"2027-06-30"}'; expect "individual plan cannot list partners" 422
co_plan '{"work_type":"joint","co_researcher_1":"أ","co_researcher_2":"ب","co_researcher_3":"ج","co_researcher_4":"د","expected_completion":"2027-06-30"}'; expect "a fourth co-researcher is refused" 422
echo "$BODY" | grep -q "co_researchers:too_many" && ok "over three partners named" || bad "too many msg" "$BODY"
co_plan '{"work_type":"joint","co_researcher_1":"أ","co_researcher_2":"ب","co_researcher_3":"ج","expected_completion":"2027-06-30"}'; expect "joint plan with three partners submitted" 200

echo "== research title change: SA decides alone and the plan is renamed =="
call POST /portal/requests "$OTHER" "{\"kind\":\"title_change\",\"title_ar\":\"x\",\"related_request_id\":\"$PLAN\",\"details\":{\"old_title\":\"x\",\"reason\":\"y\"}}"; expect "cannot target someone else's plan" 400
call POST /portal/requests "$RES" "{\"kind\":\"title_change\",\"title_ar\":\"عنوان جديد\",\"title_en\":\"New title\",\"doi\":\"10.1/x\",\"college_id\":\"$COLLEGE\",\"related_request_id\":\"$PLAN\",\"details\":{\"old_title\":\"بحث\"}}"; expect "title change draft saved" 201
TC=$(echo "$BODY" | jq_ "d['data']['id']")
[ "$(echo "$BODY" | jq_ "d['data']['doi']")" = "None" ] && ok "title change stores no DOI" || bad "tc doi" "$BODY"
call POST "/portal/requests/$TC/submit" "$RES"; expect "title change needs a reason" 422
echo "$BODY" | grep -q "reason:required" && ok "missing reason named" || bad "reason msg" "$BODY"
call PUT "/portal/requests/$TC" "$RES" "{\"title_ar\":\"عنوان جديد\",\"title_en\":\"New title\",\"college_id\":\"$COLLEGE\",\"related_request_id\":\"$PLAN\",\"details\":{\"old_title\":\"بحث\",\"reason\":\"توصية لجنة السمنار\"}}"; expect "reason added" 200
call POST "/portal/requests/$TC/submit" "$RES"; expect "title change submitted" 200
[ "$(echo "$BODY" | jq_ "d['data']['status']")" = "pending" ] && ok "title change goes straight to scientific affairs" || bad "tc status" "$BODY"
echo "$BODY" | jq_ "d['data']['request_no']" | grep -q "^TC-" && ok "title change numbered TC-" || bad "tc number" "$BODY"
call POST "/portal/requests/$TC/transition" "$DEAN" '{"to":"approved"}'; expect "dean cannot decide a title change" 403
call POST "/portal/requests/$TC/transition" "$VP" '{"to":"approved"}'; expect "VP is not in the title-change chain" 403
call POST "/portal/requests/$TC/transition" "$SA" '{"to":"pending_vp","decision":"approved"}'; expect "no VP stage for a title change" 409
call POST "/portal/requests/$TC/transition" "$SA" '{"to":"approved"}'; expect "SA approves the title change" 200
call GET "/portal/requests/$PLAN" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['request']['title_ar']")" = "عنوان جديد" ] && [ "$(echo "$BODY" | jq_ "d['data']['request']['title_en']")" = "New title" ] && ok "approved plan renamed" || bad "rename" "$BODY"
echo "$BODY" | jq_ "[e for e in d['data']['events'] if e['to_status']=='title_changed'][0]['note']" | grep -q "العنوان السابق: بحث / Research" && ok "plan history keeps the previous title" || bad "rename history" "$BODY"
call POST /portal/requests "$RES" "{\"kind\":\"title_change\",\"title_ar\":\"عنوان آخر\",\"college_id\":\"$COLLEGE\",\"details\":{\"old_title\":\"بحث قبل البوابة\",\"reason\":\"تصحيح\"}}"; TC2=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/portal/requests/$TC2/submit" "$RES"; expect "title change without a linked plan submitted" 200
call POST "/portal/requests/$TC2/transition" "$SA" '{"to":"rejected"}'; expect "rejecting a title change needs a note" 422
call POST "/portal/requests/$TC2/transition" "$SA" '{"to":"rejected","note":"العنوان مطابق"}'; expect "SA rejects a title change" 200

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
call POST /admin/users "$ADM" "{\"email\":\"New.Dean$RUN@uoturath.edu.iq\",\"name_ar\":\"عميد جديد\",\"role\":\"college_dean\",\"college_id\":\"$COLLEGE\"}"; expect "admin pre-registers a university account" 201
[ "$(echo "$BODY" | jq_ "(d['data']['email'], d['data']['role'])")" = "('new.dean$RUN@uoturath.edu.iq', 'college_dean')" ] && ok "new account stored lower-case with its role" || bad "created user" "$BODY"
call POST /admin/users "$ADM" "{\"email\":\"new.dean$RUN@uoturath.edu.iq\",\"name_ar\":\"مكرر\",\"role\":\"researcher\"}"; expect "duplicate email rejected" 409
call POST /admin/users "$ADM" "{\"email\":\"someone$RUN@gmail.com\",\"name_ar\":\"خارجي\",\"role\":\"researcher\"}"; expect "non-university email rejected" 422
call POST /admin/users "$RES" "{\"email\":\"x$RUN@uoturath.edu.iq\",\"name_ar\":\"س\",\"role\":\"admin\"}"; expect "researcher cannot add users" 403
[ "$(sqlv "SELECT password_hash FROM users WHERE email='new.dean$RUN@uoturath.edu.iq'")" = "!google" ] && ok "pre-registered account cannot use password login" || bad "password hash" "set"
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

echo "== public site: content and announcements =="
pub() { # PATH -> sets CODE and BODY without any token
  CODE=$(curl -s -o "$TMP/body" -w "%{http_code}" "$API$1"); BODY=$(cat "$TMP/body")
}
SA_ID=$(sqlv "SELECT id FROM users WHERE email='$SA_EMAIL'")
pub /public/site; expect "public site readable without a token" 200
[ -n "$(echo "$BODY" | jq_ "d['data']['hero_title']")" ] && ok "site content has a hero title" || bad "site content" "$BODY"
ORIG_SITE=$(echo "$BODY" | jq_ "json.dumps(d['data'], ensure_ascii=False)")
SITE_JSON="{\"hero_title\":\"قسم الشؤون العلمية $RUN\",\"hero_subtitle\":\"x\",\"about\":\"نبذة\",\"services\":[{\"title\":\"خدمة\",\"description\":\"وصف\"}],\"contact\":{\"email\":\"\",\"phone\":\"\",\"location\":\"\",\"hours\":\"\"},\"links\":[{\"label\":\"الجامعة\",\"url\":\"https://uoturath.edu.iq\"}]}"
call PUT /site "$RES" "$SITE_JSON"; expect "researcher cannot edit the public site" 403
call PUT /site "$VP" "$SITE_JSON"; expect "VP cannot edit the public site" 403
call PUT /site "$SA" "${SITE_JSON/https:\/\/uoturath/javascript:\/\/uoturath}"; expect "non-https quick link rejected" 422
echo "$BODY" | grep -q "link_url:https_required" && ok "bad link reported by field and code" || bad "link code" "$BODY"
call PUT /site "$SA" "${SITE_JSON/قسم الشؤون العلمية $RUN/}"; expect "empty hero title rejected" 422
call PUT /site "$SA" "$SITE_JSON"; expect "scientific affairs edits the public site" 200
pub /public/site
[ "$(echo "$BODY" | jq_ "d['data']['hero_title']")" = "قسم الشؤون العلمية $RUN" ] && ok "public site shows the edit" || bad "site edit" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['hero_title_en']")" = "" ] && ok "english hero title is optional" || bad "hero en" "$BODY"
call PUT /site "$SA" "${SITE_JSON/\"hero_subtitle\":\"x\"/\"hero_title_en\":\"Scientific Affairs $RUN\",\"hero_subtitle\":\"x\"}"; expect "english hero title saved" 200
pub /public/site
[ "$(echo "$BODY" | jq_ "d['data']['hero_title_en']")" = "Scientific Affairs $RUN" ] && ok "public site returns the english text" || bad "site en" "$BODY"
echo "$BODY" | grep -q "$SA_ID" && bad "site leaks editor" "$BODY" || ok "public site names no editor"
call PUT /site "$SA" "$ORIG_SITE"; expect "original site content restored" 200

call POST /announcements "$RES" '{"type":"circular","title":"x"}'; expect "researcher cannot publish announcements" 403
call POST /announcements "$VP" '{"type":"circular","title":"x"}'; expect "VP cannot publish announcements" 403
call POST /announcements "$SA" '{"type":"news","title":""}'; expect "invalid announcement rejected" 422
echo "$BODY" | grep -q "type:invalid" && echo "$BODY" | grep -q "title:required" && ok "announcement problems reported by field" || bad "ann codes" "$BODY"
call POST /announcements "$SA" "{\"type\":\"circular\",\"title\":\"تعميم اختبار $RUN\",\"body\":\"نص التعميم\\n\\n- بند\",\"reference_no\":\"$RUN/ع\",\"event_date\":\"2026-01-01\"}"; expect "SA drafts a circular" 201
ANN=$(echo "$BODY" | jq_ "d['data']['id']")
[ "$(echo "$BODY" | jq_ "d['data']['published']")" = "False" ] && ok "new announcement starts unpublished" || bad "draft state" "$BODY"
[ "$(echo "$BODY" | jq_ "d['data']['event_date']")" = "None" ] && ok "circular drops an event date" || bad "circular date" "$BODY"
upload "/announcements/$ANN/attachment" "$RES" "$TMP/doc.pdf"; expect "researcher cannot attach to an announcement" 403
upload "/announcements/$ANN/attachment" "$SA" "$TMP/evil.pdf"; expect "fake PDF attachment rejected" 415
upload "/announcements/$ANN/attachment" "$SA" "$TMP/doc.pdf"; expect "SA attaches a PDF" 201
ANNFILE=$(echo "$BODY" | jq_ "d['data']['attachment']['id']")
pub "/public/announcements/$ANN"; expect "unpublished announcement hidden publicly" 404
pub "/public/announcements?type=circular&limit=50"
[ "$(echo "$BODY" | jq_ "sum(1 for a in d['data'] if a['id']=='$ANN')")" = "0" ] && ok "unpublished announcement not listed" || bad "public list" "$BODY"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/public/files/$ANNFILE"); [ "$CODE" = "404" ] && ok "draft attachment 404 publicly" || bad "draft file" "$CODE"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/portal/files/$ANNFILE" -H "Authorization: Bearer $RES"); [ "$CODE" = "404" ] && ok "researcher cannot fetch a draft attachment" || bad "draft file authed" "$CODE"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/portal/files/$ANNFILE" -H "Authorization: Bearer $SA"); [ "$CODE" = "200" ] && ok "staff preview the draft attachment" || bad "staff preview" "$CODE"
call POST "/announcements/$ANN/publish" "$RES" '{"published":true}'; expect "researcher cannot publish" 403
call POST "/announcements/$ANN/publish" "$SA" '{}'; expect "publish needs a state" 422
call POST "/announcements/$ANN/publish" "$SA" '{"published":true}'; expect "SA publishes" 200
call POST "/announcements/$ANN/pin" "$SA" '{"pinned":true}'; expect "SA pins" 200
pub "/public/announcements/$ANN"; expect "published announcement visible without a token" 200
KEYS=$(echo "$BODY" | jq_ "','.join(sorted(d['data'].keys()))")
[ "$KEYS" = "attachment,body,event_date,id,lang,pinned,published_at,reference_no,title,type" ] && ok "public announcement carries no personal or editorial fields" || bad "public keys" "$KEYS"
[ "$(echo "$BODY" | jq_ "d['data']['lang']")" = "ar" ] && ok "announcement language defaults to Arabic" || bad "announcement lang" "$BODY"
echo "$BODY" | grep -q -e "$SA_ID" -e "$SA_EMAIL" -e "uploaded_by" -e "created_by" && bad "public leak" "$BODY" || ok "no author id or email in public JSON"
[ "$(echo "$BODY" | jq_ "','.join(sorted(d['data']['attachment'].keys()))")" = "id,mime_type,name,size_bytes" ] && ok "public attachment has no uploader" || bad "attachment keys" "$BODY"
pub "/public/announcements?type=circular"
[ "$(echo "$BODY" | jq_ "d['data'][0]['id']")" = "$ANN" ] && ok "pinned circular listed first" || bad "pinned order" "$BODY"
echo "$BODY" | grep -q -e "$SA_ID" -e "$SA_EMAIL" && bad "list leak" "$BODY" || ok "no personal data in the public list"
[ "$(echo "$BODY" | jq_ "all(a['type']=='circular' for a in d['data'])")" = "True" ] && ok "public list filters by type" || bad "type filter" "$BODY"
CODE=$(curl -s -o "$TMP/pubdl" -w "%{http_code}" "$API/public/files/$ANNFILE"); [ "$CODE" = "200" ] && head -c4 "$TMP/pubdl" | grep -q '%PDF' && ok "published attachment downloads without a token" || bad "public download" "$CODE"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/public/files/$FILE"); [ "$CODE" = "404" ] && ok "request files never served publicly" || bad "request file public" "$CODE"
call PUT "/announcements/$ANN" "$SA" "{\"type\":\"circular\",\"title\":\"تعميم معدل $RUN\",\"body\":\"\",\"reference_no\":\"1\"}"; expect "SA edits the announcement" 200
call PUT "/announcements/$ANN" "$SA" "{\"type\":\"circular\",\"lang\":\"fr\",\"title\":\"Circular $RUN\"}"; expect "unknown announcement language rejected" 422
echo "$BODY" | grep -q "lang:invalid" && ok "announcement language reported by field" || bad "lang code" "$BODY"
call PUT "/announcements/$ANN" "$SA" "{\"type\":\"circular\",\"lang\":\"en\",\"title\":\"Circular $RUN\",\"body\":\"\",\"reference_no\":\"1\"}"; expect "SA writes an English announcement" 200
[ "$(echo "$BODY" | jq_ "d['data']['lang']")" = "en" ] && ok "announcement language stored" || bad "lang stored" "$BODY"
call POST "/announcements/$ANN/publish" "$SA" '{"published":false}'; expect "SA unpublishes" 200
pub "/public/announcements/$ANN"; expect "withdrawn announcement hidden again" 404
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/public/files/$ANNFILE"); [ "$CODE" = "404" ] && ok "withdrawn attachment 404 publicly" || bad "withdrawn file" "$CODE"

call POST "/activities/$ACT/announce" "$RES" '{}'; expect "researcher cannot announce an activity" 403
call POST "/activities/$ACT/announce" "$SA" '{}'; expect "approved activity drafted as announcement" 201
FROMACT=$(echo "$BODY" | jq_ "d['data']['id']")
[ "$(echo "$BODY" | jq_ "d['data']['type']")" = "activity" ] && [ "$(echo "$BODY" | jq_ "d['data']['event_date']")" = "2026-11-02" ] && [ "$(echo "$BODY" | jq_ "d['data']['published']")" = "False" ] && ok "activity prefill: type, date, unpublished" || bad "prefill" "$BODY"
call POST "/activities/$ACT/announce" "$SA" '{}'; expect "announcing again returns the same draft" 200
[ "$(echo "$BODY" | jq_ "d['data']['id']")" = "$FROMACT" ] && ok "one announcement per activity" || bad "dup announce" "$BODY"
call POST /activities "$RES" "{\"activity_type\":\"seminar\",\"title_ar\":\"ندوة معلقة\",\"activity_date\":\"2026-11-03\",\"college_id\":\"$COLLEGE\"}"; PENDACT=$(echo "$BODY" | jq_ "d['data']['id']")
call POST "/activities/$PENDACT/announce" "$SA" '{}'; expect "pending activity cannot be announced" 409

call DELETE "/announcements/$ANN" "$RES"; expect "researcher cannot delete announcements" 403
call DELETE "/announcements/$ANN" "$SA"; expect "SA deletes the announcement" 200
call GET "/announcements/$ANN" "$SA"; expect "deleted announcement gone" 404
[ "$(sqlv "SELECT COUNT(*) FROM portal_files WHERE id='$ANNFILE'")" = "0" ] && ok "attachment removed with its announcement" || bad "orphan file" "left behind"
call DELETE "/announcements/$FROMACT" "$SA"; expect "activity draft cleaned up" 200

echo "== FS 003 categories and data-entry responsibilities =="
HOLD_EMAIL="hold$RUN@test.iq"; HOLD2_EMAIL="hold2$RUN@test.iq"
register "$HOLD_EMAIL" "مسؤول الاتفاقيات" >/dev/null; register "$HOLD2_EMAIL" "مسؤول كلية أخرى" >/dev/null
HOLD=$(login "$HOLD_EMAIL"); HOLD2=$(login "$HOLD2_EMAIL")
HOLD_ID=$(sqlv "SELECT id FROM users WHERE email='$HOLD_EMAIL'"); HOLD2_ID=$(sqlv "SELECT id FROM users WHERE email='$HOLD2_EMAIL'")
agreement() { echo "{\"category\":\"$1\",\"title_ar\":\"$2\",\"category_detail\":\"جامعة شريكة\",\"activity_date\":\"2026-11-05\",\"college_id\":\"$3\"}"; }
call POST /activities "$RES" "{\"category\":\"infrastructure\",\"title_ar\":\"مختبر\",\"item_count\":2,\"activity_date\":\"2026-11-05\",\"college_id\":\"$COLLEGE\"}"; expect "researcher cannot add infrastructure" 403
echo "$BODY" | grep -q "activity_category_forbidden" && ok "forbidden category has a code" || bad "forbidden code" "$BODY"
call POST /activities "$RES" "$(agreement international_agreements "اتفاقية باحث" "$COLLEGE")"; expect "researcher cannot add an international agreement" 403
call POST /activities "$RES" "$(agreement local_agreements "اتفاقية محلية" "$COLLEGE")"; expect "researcher cannot add a local agreement" 403
call POST /activities "$RES" "{\"category\":\"workshops\",\"activity_type\":\"panel\",\"title_ar\":\"حلقة نقاشية $RUN\",\"category_detail\":\"القاعة الكبرى\",\"activity_date\":\"2026-11-06\",\"college_id\":\"$COLLEGE\"}"; expect "researcher adds own workshop" 201
[ "$(echo "$BODY" | jq_ "(d['data']['category'], d['data']['activity_type'], d['data']['location'], d['data']['can_edit'])")" = "('workshops', 'panel', 'القاعة الكبرى', True)" ] && ok "workshop keeps sub-type, venue mirrors location" || bad "workshop fields" "$BODY"
call POST /activities "$RES" "{\"category\":\"books\",\"title_ar\":\"كتاب\",\"activity_date\":\"2026-11-06\",\"college_id\":\"$COLLEGE\"}"; expect "book without publisher rejected" 422
echo "$BODY" | grep -q "category_detail:required" && ok "publisher required code" || bad "detail code" "$BODY"
[ "$(sqlv "SELECT category FROM activities WHERE id='$ACT'")" = "workshops" ] && ok "legacy workshop has the workshops category" || bad "legacy category" "$(sqlv "SELECT category FROM activities WHERE id='$ACT'")"

call PUT "/admin/users/$HOLD_ID/responsibilities" "$SA" '{"items":[]}'; expect "only the admin assigns responsibilities" 403
call PUT "/admin/users/$HOLD_ID/responsibilities" "$ADM" '{"items":[{"category":"sheet8","college_id":""}]}'; expect "unknown responsibility category rejected" 422
call PUT "/admin/users/$HOLD_ID/responsibilities" "$ADM" '{"items":[{"category":"international_agreements","college_id":""},{"category":"international_agreements","college_id":""}]}'; expect "admin grants a university-wide responsibility" 200
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "1" ] && ok "duplicate responsibility collapsed" || bad "dedupe" "$BODY"
call PUT "/admin/users/$HOLD2_ID/responsibilities" "$ADM" "{\"items\":[{\"category\":\"local_agreements\",\"college_id\":\"$COLLEGE2\"}]}"; expect "admin grants a college-scoped responsibility" 200
call GET /me/responsibilities "$HOLD"
[ "$(echo "$BODY" | jq_ "[(r['category'], r['college_id']) for r in d['data']]")" = "[('international_agreements', None)]" ] && ok "holder reads own responsibilities (same token, no re-login)" || bad "me responsibilities" "$BODY"

call POST /activities "$HOLD" "$(agreement international_agreements "اتفاقية دولية $RUN" "")"; expect "university holder adds a university-level agreement" 201
HAGR=$(echo "$BODY" | jq_ "d['data']['id']")
[ "$(echo "$BODY" | jq_ "d['data']['college_id']")" = "None" ] && ok "institutional record does not inherit a college" || bad "university level" "$BODY"
call POST /activities "$SA" "$(agreement international_agreements "اتفاقية الشؤون $RUN" "$COLLEGE")"; expect "scientific affairs adds an agreement" 201
SAAGR=$(echo "$BODY" | jq_ "d['data']['id']")
call GET "/activities?category=international_agreements&limit=50" "$HOLD"
[ "$(echo "$BODY" | jq_ "[a['can_edit'] for a in d['data'] if a['id']=='$SAAGR']")" = "[True]" ] && ok "holder lists and may edit others' agreements" || bad "holder list" "$BODY"
call PUT "/activities/$SAAGR" "$HOLD" "$(agreement international_agreements "اتفاقية الشؤون معدلة $RUN" "$COLLEGE")"; expect "holder edits another member's agreement" 200
call PUT "/activities/$SAAGR" "$HOLD" "$(agreement local_agreements "نقل" "$COLLEGE")"; expect "holder cannot move a record out of their categories" 403
call POST "/activities/$HAGR/decide" "$HOLD" '{"status":"approved","approval_no":"5/2026"}'; expect "holder cannot approve" 403
call POST "/activities/$HAGR/decide" "$SA" '{"status":"approved","approval_no":"55/2026"}'; expect "SA approves the holder's agreement" 200

call POST /activities "$HOLD2" "$(agreement local_agreements "محلية خارج النطاق" "$COLLEGE")"; expect "college holder cannot add for another college" 403
call POST /activities "$HOLD2" "$(agreement local_agreements "محلية $RUN" "$COLLEGE2")"; expect "college holder adds for own college" 201
call POST /activities "$SA" "$(agreement local_agreements "محلية الكلية الأولى $RUN" "$COLLEGE")"; SALOC=$(echo "$BODY" | jq_ "d['data']['id']")
call PUT "/activities/$SALOC" "$HOLD2" "$(agreement local_agreements "تعديل" "$COLLEGE2")"; expect "college holder cannot touch another college's record" 404
call GET "/activities?category=local_agreements&limit=50" "$HOLD2"
[ "$(echo "$BODY" | jq_ "sum(1 for a in d['data'] if a['id']=='$SALOC')")" = "0" ] && ok "college holder does not see another college" || bad "scope leak" "$BODY"
call PUT "/admin/users/$HOLD2_ID/responsibilities" "$ADM" '{"items":[]}'; expect "admin withdraws a responsibility" 200
call POST /activities "$HOLD2" "$(agreement local_agreements "بعد السحب" "$COLLEGE2")"; expect "withdrawal applies to the same token at once" 403

echo "== FS 003 export =="
fs003() { curl -s -o "$TMP/fs003.xlsx" -w "%{http_code}" "$API/activities/fs003.xlsx?$1" -H "Authorization: Bearer $2"; }
[ "$(fs003 "from=2024-01-01&to=2026-12-31" "$RES")" = "403" ] && ok "plain researcher cannot export FS 003" || bad "fs003 rbac" "researcher allowed"
[ "$(fs003 "from=2026-12-31&to=2026-01-01" "$SA")" = "400" ] && ok "reversed FS 003 period rejected" || bad "fs003 period" "accepted"
xlsx_text() { python3 -X utf8 -c "
import zipfile,sys
z=zipfile.ZipFile(sys.argv[1]); print(''.join(z.read(n).decode('utf8') for n in z.namelist() if n.endswith('.xml')))" "$TMP/fs003.xlsx"; }
CODE=$(fs003 "from=2024-01-01&to=2026-12-31&coordinator=منسق%20الاختبار" "$SA")
[ "$CODE" = "200" ] && head -c2 "$TMP/fs003.xlsx" | grep -q PK && ok "FS 003 downloads as xlsx" || bad "fs003 download" "HTTP $CODE"
TEXT=$(xlsx_text)
[ "$(python3 -c "import sys; t=sys.stdin.read(); print(sum(s in t for s in ['ورشة الذكاء الاصطناعي','اتفاقية دولية $RUN','AI paper','Scopus','اسم المنسق : منسق الاختبار']))" <<<"$TEXT")" = "5" ] && ok "FS 003 holds approved records, verified research and the coordinator" || bad "fs003 content" "${TEXT:0:200}"
echo "$TEXT" | grep -q "اتفاقية الشؤون معدلة $RUN" && bad "fs003 approval filter" "pending record exported" || ok "FS 003 leaves out unapproved records"
CODE=$(fs003 "from=2026-01-01&to=2026-12-31" "$HOLD"); TEXT=$(xlsx_text)
[ "$CODE" = "200" ] && echo "$TEXT" | grep -q "اتفاقية دولية $RUN" && ok "holder exports their category" || bad "holder export" "HTTP $CODE"
echo "$TEXT" | grep -q "ورشة الذكاء الاصطناعي" && bad "holder export scope" "workshop leaked" || ok "holder export limited to their scope"

echo "== login audit and account activity =="
curl -s -o /dev/null -X POST "$API/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"$HOLD_EMAIL\",\"password\":\"wrong-pass\"}"
curl -s -o /dev/null -X POST "$API/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"nobody$RUN@test.iq\",\"password\":\"x\"}"
login "$HOLD_EMAIL" >/dev/null
[ "$(sqlv "SELECT COUNT(*) FROM login_events WHERE email='$HOLD_EMAIL' AND NOT success AND reason='wrong_password' AND user_id='$HOLD_ID'")" = "1" ] && ok "failed password login recorded" || bad "failed login event" "missing"
[ "$(sqlv "SELECT COUNT(*) FROM login_events WHERE email='nobody$RUN@test.iq' AND reason='unknown_user' AND user_id IS NULL")" = "1" ] && ok "unknown user recorded without an id" || bad "unknown user event" "missing"
[ "$(sqlv "SELECT COUNT(*) FROM login_events WHERE email='$HOLD_EMAIL' AND success AND reason IS NULL AND ip <> ''")" -ge 1 ] && ok "successful login recorded with IP" || bad "success event" "missing"
[ "$(sqlv "SELECT COUNT(*) FROM login_events WHERE user_agent LIKE '%wrong-pass%' OR email LIKE '%wrong-pass%'")" = "0" ] && ok "no password stored in the log" || bad "password leak" "found"
call GET "/admin/login-events?q=$HOLD_EMAIL&result=failure" "$ADM"; expect "admin reads the login log" 200
[ "$(echo "$BODY" | jq_ "[(e['result'], e['reason'], e['method']) for e in d['data']]")" = "[('failure', 'wrong_password', 'password')]" ] && ok "login log filters by user and result" || bad "log filter" "$BODY"
for p in /admin/login-events /admin/accounts /admin/accounts/summary; do
  call GET "$p" "$SA"; expect "scientific affairs refused $p" 403
  call GET "$p" "$RES"; expect "researcher refused $p" 403
done
SEEN_EMAIL="seen$RUN@test.iq"; SEEN=$(register "$SEEN_EMAIL" "حساب جديد")
[ -z "$(sqlv "SELECT last_seen_at FROM users WHERE email='$SEEN_EMAIL'")" ] && ok "new account has no last_seen_at" || bad "last seen before" "set"
call GET "/admin/accounts?status=never&q=$SEEN_EMAIL" "$ADM"
[ "$(echo "$BODY" | jq_ "len(d['data'])")" = "1" ] && ok "new account listed as never signed in" || bad "never list" "$BODY"
call GET /me/responsibilities "$SEEN"; sleep 1
[ -n "$(sqlv "SELECT last_seen_at FROM users WHERE email='$SEEN_EMAIL' AND last_seen_at > NOW() - INTERVAL '1 minute'")" ] && ok "last_seen_at updated by an authenticated request" || bad "last seen" "not updated"
call GET "/admin/accounts?status=online&q=$SEEN_EMAIL" "$ADM"
[ "$(echo "$BODY" | jq_ "[a['online'] for a in d['data']]")" = "[True]" ] && ok "account shown online" || bad "online list" "$BODY"
call GET /admin/accounts/summary "$ADM"
[ "$(echo "$BODY" | jq_ "d['data']['online'] >= 1 and d['data']['total'] >= d['data']['last_30_days']")" = "True" ] && ok "account summary counts" || bad "summary" "$BODY"

echo "== current role from the database, not the token =="
PROMO_EMAIL="promo$RUN@test.iq"; register "$PROMO_EMAIL" "مرقّى" >/dev/null; PROMO=$(login "$PROMO_EMAIL")
PROMO_ID=$(sqlv "SELECT id FROM users WHERE email='$PROMO_EMAIL'")
call GET /screening "$PROMO"; expect "researcher token refused a staff screen" 403
call PUT "/admin/users/$PROMO_ID" "$ADM" '{"role":"scientific_affairs"}'; expect "admin promotes via the API" 200
call GET /auth/me "$PROMO"
[ "$(echo "$BODY" | jq_ "d['data']['role']")" = "scientific_affairs" ] && ok "/auth/me returns the new role with the old token" || bad "me role" "$BODY"
call GET /screening "$PROMO"; expect "promotion applies at once to the old token" 200
call PUT "/admin/users/$PROMO_ID" "$ADM" '{"role":"researcher"}'; expect "admin demotes" 200
call GET /screening "$PROMO"; expect "demotion applies at once to the old token" 403
GHOST_EMAIL="ghost$RUN@test.iq"; GHOST=$(register "$GHOST_EMAIL" "محذوف")
sqlv "DELETE FROM users WHERE email='$GHOST_EMAIL'" >/dev/null
call GET /auth/me "$GHOST"; expect "token of a deleted account rejected" 401

echo "== researcher ownership needs a verified email =="
OWN_EMAIL="own$RUN@test.iq"
OWN_RID=$(sqlv "INSERT INTO researchers (name_ar, email, status) VALUES ('باحث مستورد $RUN', '$OWN_EMAIL', 'active') RETURNING id")
register "$OWN_EMAIL" "منتحل" >/dev/null; IMPOSTOR=$(login "$OWN_EMAIL")
call GET /me/research-profile "$IMPOSTOR"
[ "$(echo "$BODY" | jq_ "d['data'].get('linked', False)")" = "False" ] && ok "password account does not own a record by email" || bad "email ownership" "$BODY"
call GET "/researchers/$OWN_RID/contact" "$IMPOSTOR"; expect "password account cannot read the record's contact data" 403
sqlv "UPDATE users SET google_sub='e2e-$RUN' WHERE email='$OWN_EMAIL'" >/dev/null
call GET /me/research-profile "$IMPOSTOR"
[ "$(echo "$BODY" | jq_ "d['data'].get('linked', False)")" = "True" ] && ok "Google-verified email owns the matching record" || bad "verified ownership" "$BODY"
call GET "/researchers/$OWN_RID/contact" "$IMPOSTOR"; expect "verified owner reads own contact data" 200

echo "== public site v2: statistics, publications, director photo, blocks =="
pub /public/stats; expect "public statistics readable without a token" 200
[ "$(echo "$BODY" | jq_ "all(sorted(s.keys())==['key','label','label_en','number','value'] for s in d['data'])")" = "True" ] && ok "statistics are aggregates only" || bad "stats shape" "$BODY"
call GET /site/stats "$RES"; expect "researcher cannot read the stats editor" 403
call GET /site/stats "$SA"; expect "SA reads computed statistics" 200
[ "$(echo "$BODY" | jq_ "len(d['data']['items'])")" = "5" ] && ok "editor lists all five statistics" || bad "staff stats" "$BODY"
PUBS_LOCAL=$(sqlv "SELECT COUNT(*) FROM publications p WHERE status='published' AND EXISTS (SELECT 1 FROM publication_authors pa WHERE pa.publication_id=p.id)")
PUBS_SHOWN=$(echo "$BODY" | jq_ "[i['computed'] for i in d['data']['items'] if i['key']=='publications'][0]")
[ "$PUBS_SHOWN" -ge 1000 ] && [ "$PUBS_SHOWN" -ge "$PUBS_LOCAL" ] && ok "published research uses the university's OpenAlex total ($PUBS_SHOWN)" || bad "publications total" "$PUBS_SHOWN vs local $PUBS_LOCAL"
pub /public/site; BASE_SITE=$(echo "$BODY" | jq_ "json.dumps(d['data'], ensure_ascii=False)")
OVERRIDE=$(echo "$BASE_SITE" | python3 -c "
import sys,json; d=json.load(sys.stdin)
for s in d['stats']:
    if s['key']=='academics': s['value']='999+'; s['label']='E2E label'
    if s['key']=='activities': s['hidden']=True
d['director']['photo_id']='00000000-0000-0000-0000-000000000000'
print(json.dumps(d, ensure_ascii=False))")
call PUT /site "$SA" "$OVERRIDE"; expect "SA overrides statistics" 200
pub /public/stats
[ "$(echo "$BODY" | jq_ "[s for s in d['data'] if s['key']=='academics'][0]['value']")" = "999+" ] && ok "manual value shown instead of the computed one" || bad "override value" "$BODY"
[ "$(echo "$BODY" | jq_ "[s for s in d['data'] if s['key']=='academics'][0]['number']")" = "None" ] && ok "overridden statistic carries no number" || bad "override number" "$BODY"
[ "$(echo "$BODY" | jq_ "any(s['key']=='activities' for s in d['data'])")" = "False" ] && ok "hidden statistic left out" || bad "hidden stat" "$BODY"
pub /public/site
[ "$(echo "$BODY" | jq_ "d['data']['director']['photo_id']")" = "" ] && ok "editor cannot set the photo id by hand" || bad "photo id forged" "$BODY"
RESET=$(echo "$OVERRIDE" | python3 -c "
import sys,json; d=json.load(sys.stdin)
for s in d['stats']: s['value']=''; s['hidden']=False
print(json.dumps(d, ensure_ascii=False))")
call PUT /site "$SA" "$RESET"; expect "statistics reset to automatic" 200
pub /public/stats
[ "$(echo "$BODY" | jq_ "[s for s in d['data'] if s['key']=='academics'][0]['number'] is not None")" = "True" ] && ok "reset statistic shows the computed number" || bad "reset" "$BODY"
BAD_SITE=$(echo "$BASE_SITE" | python3 -c "
import sys,json; d=json.load(sys.stdin)
d['services']=[{'title':'x','description':'','icon':'rocket','link':'https://evil.example'}]
d['socials']=[{'platform':'x','url':'http://x.com/u'}]; d['faq']=[{'question':'q','answer':''}]
print(json.dumps(d, ensure_ascii=False))")
call PUT /site "$SA" "$BAD_SITE"; expect "invalid blocks rejected" 422
for code in service_icon:invalid service_link:invalid social_url:https_required faq_answer:required; do
  echo "$BODY" | grep -q "$code" && ok "block problem reported: $code" || bad "block code $code" "$BODY"
done
pub /public/site
[ "$(echo "$BODY" | jq_ "len(d['data']['colleges'])")" -ge 14 ] 2>/dev/null && ok "college sites seeded" || bad "colleges" "$BODY"
[ "$(echo "$BODY" | jq_ "len(d['data']['faq'])")" -ge 1 ] 2>/dev/null && ok "FAQ seeded" || bad "faq" "$BODY"

pub /public/publications/latest; expect "latest publications readable without a token" 200
echo "$BODY" | grep -q -e '"email"' -e '"orcid' -e '"h_index"' && bad "publication leak" "$BODY" || ok "publications expose no researcher contact data"

upload /site/director-photo "$RES" "$TMP/bg.png"; expect "researcher cannot upload the director photo" 403
upload /site/director-photo "$SA" "$TMP/doc.pdf"; expect "a PDF is not accepted as the photo" 415
upload /site/director-photo "$SA" "$TMP/bg.png"; expect "SA uploads the director photo" 201
PHOTO=$(echo "$BODY" | jq_ "d['data']['director']['photo_id']")
CT=$(curl -s -o /dev/null -w "%{http_code} %{content_type}" "$API/public/site/director-photo/$PHOTO"); [ "$CT" = "200 image/png" ] && ok "director photo served publicly" || bad "photo serve" "$CT"
call PUT /site "$SA" "$RESET"; expect "saving the page keeps the photo" 200
[ "$(echo "$BODY" | jq_ "d['data']['director']['photo_id']")" = "$PHOTO" ] && ok "photo id survives a content save" || bad "photo kept" "$BODY"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/public/site/director-photo/$FILE"); [ "$CODE" = "404" ] && ok "other files never served as the photo" || bad "photo route leak" "$CODE"
call DELETE /site/director-photo "$SA"; expect "SA removes the director photo" 200
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/public/site/director-photo/$PHOTO"); [ "$CODE" = "404" ] && ok "removed photo no longer served" || bad "removed photo" "$CODE"
call PUT /site "$SA" "$BASE_SITE"; expect "site content restored after v2 checks" 200

echo "== predatory journals page (explanation and links only) =="
# ISSN 2049-3630 is on the staff watchlist (see the screening section above).
call GET "/screening/check?issn=20493630" "$RES"
[ "$(echo "$BODY" | jq_ "d['data']['classification']")" = "predatory" ] && ok "request forms keep the automatic journal check" || bad "portal check" "$BODY"
for path in "/public/journals/check?issn=20493630" "/public/journals/entries?type=hijacked" "/public/journals/status"; do
  pub "$path"; [ "$CODE" = "404" ] && ok "retired public endpoint ${path%%\?*} is gone" || bad "retired endpoint $path" "$CODE"
done
call GET /metrics/institution "$SA"
[ "$(echo "$BODY" | jq_ "d['data']['total_publications'] >= max(1000, d['data']['portal_publications'])")" = "True" ] && ok "dashboard publications use the university's OpenAlex total" || bad "dashboard total" "$BODY"

FRONT="${FRONT:-http://localhost:3001}"
CODE=$(curl -s -o "$TMP/robots" -w "%{http_code}" "$FRONT/robots.txt"); [ "$CODE" = "200" ] && grep -q "Disallow: /\*/portal" "$TMP/robots" && grep -q "Sitemap: .*/sitemap.xml" "$TMP/robots" && ok "robots.txt served with portal disallowed and sitemaps" || bad "robots" "$CODE"
CODE=$(curl -s -o "$TMP/sitemap" -w "%{http_code}" "$FRONT/sitemap.xml"); [ "$CODE" = "200" ] && grep -q "/ar/journals/predatory</loc>" "$TMP/sitemap" && ok "sitemap.xml lists the predatory journals page" || bad "sitemap" "$CODE"
grep -q "journals/check\|journals/guide" "$TMP/sitemap" && bad "sitemap" "still lists retired pages" || ok "sitemap drops the retired journal pages"
for path in /ar/journals /ar/journals/check /en/journals/guide/warning-signs /ar/journals/predatory/list /ar/journals/predatory/some-entry; do
  CODE=$(curl -s -o /dev/null -w "%{http_code} %{redirect_url}" "$FRONT$path")
  [ "${CODE%% *}" = "308" ] && echo "$CODE" | grep -q "/journals/predatory$" && ok "$path redirects to the predatory page" || bad "redirect $path" "$CODE"
done
CODE=$(curl -s -o /dev/null -w "%{http_code} %{redirect_url}" "$FRONT/ar/portal/journal-check"); [ "${CODE%% *}" = "308" ] && echo "$CODE" | grep -q "/ar/portal/predatory-journals$" && ok "portal journal check redirects to the predatory page" || bad "portal check redirect" "$CODE"
CODE=$(curl -s -o "$TMP/guide.html" -w "%{http_code}" "$FRONT/ar/journals/predatory"); [ "$CODE" = "200" ] && grep -q 'id="explain-what"' "$TMP/guide.html" && grep -q 'id="explain-signs"' "$TMP/guide.html" && ok "predatory page explains predatory and hijacked journals" || bad "explanation" "$CODE"
grep -q 'href="https://beallslist.net/"' "$TMP/guide.html" && grep -q 'retractionwatch.com/the-retraction-watch-hijacked-journal-checker' "$TMP/guide.html" && grep -q 'href="https://jor.rdd.edu.iq/"' "$TMP/guide.html" && ok "predatory page links the source lists" || bad "guide links" "missing"
grep -q 'href="/ar/journals/check"' "$TMP/guide.html" && bad "guide" "links the retired check" || ok "predatory page no longer offers a journal check"
CODE=$(curl -s -o "$TMP/home.html" -w "%{http_code}" "$FRONT/ar"); [ "$CODE" = "200" ] && grep -q 'id="stats-title"' "$TMP/home.html" && grep -q 'href="/ar/journals/predatory"' "$TMP/home.html" && ok "homepage renders statistics and links the predatory page" || bad "home" "$CODE"
grep -q 'href="/ar/journals/check"\|href="/ar/portal/new/journal_verification"' "$TMP/home.html" && bad "home" "still shows journal check or verification" || ok "homepage keeps only the predatory journals card"
grep -q 'id="links' "$TMP/home.html" && grep -q 'href="https://nadwati.rdd.edu.iq/"' "$TMP/home.html" && ok "homepage shows the Nadwati link among useful links" || bad "home links" "nadwati missing"

echo
echo "Passed: $PASS  Failed: $FAIL"
[ "$FAIL" -eq 0 ]
