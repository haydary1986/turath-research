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
call POST "/portal/requests/$AR/issue-letter" "$SA"; expect "letter issued" 201
LETTER=$(echo "$BODY" | jq_ "d['data']['id']")
echo "$BODY" | jq_ "d['data']['letter_no']" | grep -Eq '^[0-9]+/[0-9]{4}$' && ok "letter number format" || bad "letter no" "$BODY"
call POST "/portal/requests/$AR/issue-letter" "$SA"; expect "letter cannot be issued twice" 409
call GET "/portal/letters/$LETTER" "$RES"; expect "applicant reads own letter" 200
call GET "/portal/letters/$LETTER" "$OTHER"; expect "other researcher cannot read letter" 404
call GET "/verify/letter/$LETTER" ""; expect "public verification" 200

echo "== visibility, templates, admin =="
call GET "/portal/requests/$PLAN" "$DEAN"; expect "dean views his college's request" 200
call GET "/portal/requests/$PLAN" "$DEAN2"; expect "other dean cannot view" 404
call GET /portal/templates "$VP"; expect "VP may manage templates" 200
call GET /portal/templates "$DEAN"; expect "dean may not manage templates" 403
upload "/portal/template-assets?role=background" "$SA" "$TMP/doc.pdf"; expect "template asset must be an image" 415
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

echo
echo "Passed: $PASS  Failed: $FAIL"
[ "$FAIL" -eq 0 ]
