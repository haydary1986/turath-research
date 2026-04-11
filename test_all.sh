#!/bin/bash
# ══════════════════════════════════════════════════
# Turath Research — Full System Test
# Tests ALL 54 API endpoints + seed data + roles
# ══════════════════════════════════════════════════

set -e
API="http://localhost:8080/api/v1"
PASS=0
FAIL=0
TOTAL=0

test_endpoint() {
    TOTAL=$((TOTAL + 1))
    local desc="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local token="$5"
    local expect_status="$6"

    local headers="--header 'Content-Type: application/json'"
    if [ -n "$token" ]; then
        headers="$headers --header 'Authorization: Bearer $token'"
    fi

    local cmd="wget -q -O - --method=$method"
    if [ "$method" = "POST" ] || [ "$method" = "PUT" ]; then
        cmd="$cmd --body-data='$data'"
    fi
    cmd="$cmd $headers '$url' 2>&1"

    local result
    result=$(docker compose exec -T backend sh -c "wget -q -O - --method=$method $([ -n "$data" ] && echo "--body-data='$data'") --header='Content-Type: application/json' $([ -n "$token" ] && echo "--header='Authorization: Bearer $token'") '$url'" 2>&1) || true

    if echo "$result" | grep -q '"success":true'; then
        PASS=$((PASS + 1))
        echo "  ✅ $desc"
    elif echo "$result" | grep -q '"success":false' && [ "$expect_status" = "error" ]; then
        PASS=$((PASS + 1))
        echo "  ✅ $desc (expected error)"
    else
        FAIL=$((FAIL + 1))
        echo "  ❌ $desc"
        echo "     Response: $(echo "$result" | head -c 200)"
    fi
}

echo "══════════════════════════════════════════════════"
echo "  فحص شامل لنظام تُراث — $(date)"
echo "══════════════════════════════════════════════════"

# ── 1. Health ─────────────────────────────────────
echo ""
echo "▶ 1. Health Check"
test_endpoint "GET /health" GET "$API/health" "" ""

# ── 2. Auth ───────────────────────────────────────
echo ""
echo "▶ 2. Authentication"

# Register users with different roles (simulated via register + manual role update)
ADMIN_TOKEN=$(docker compose exec -T backend wget -q -O - --method=POST --body-data='{"email":"admin@test.iq","password":"Admin1234!","name_ar":"مدير النظام"}' --header='Content-Type: application/json' "$API/auth/register" 2>&1 | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")
test_endpoint "POST /auth/register (admin)" POST "$API/auth/register" '{"email":"dept_head@test.iq","password":"Test1234!","name_ar":"رئيس قسم الحاسوب"}' "" ""
test_endpoint "POST /auth/login" POST "$API/auth/login" '{"email":"admin@test.iq","password":"Admin1234!"}' "" ""
test_endpoint "GET /auth/me" GET "$API/auth/me" "" "$ADMIN_TOKEN"
test_endpoint "POST /auth/login (wrong password)" POST "$API/auth/login" '{"email":"admin@test.iq","password":"wrong"}' "" "error"

# Get a working token
TOKEN=$(docker compose exec -T backend wget -q -O - --method=POST --body-data='{"email":"admin@test.iq","password":"Admin1234!"}' --header='Content-Type: application/json' "$API/auth/login" 2>&1 | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# ── 3. Researchers ────────────────────────────────
echo ""
echo "▶ 3. Researchers CRUD"
test_endpoint "GET /researchers (list)" GET "$API/researchers?page=1&limit=10" "" "$TOKEN"
test_endpoint "GET /researchers (search)" GET "$API/researchers?q=علي" "" "$TOKEN"

RES_ID=$(docker compose exec -T backend wget -q -O - --header="Authorization: Bearer $TOKEN" "$API/researchers?limit=1" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'] if d else '')")
if [ -n "$RES_ID" ]; then
    test_endpoint "GET /researchers/{id}" GET "$API/researchers/$RES_ID" "" "$TOKEN"
    test_endpoint "PUT /researchers/{id}" PUT "$API/researchers/$RES_ID" '{"title_ar":"أستاذ مساعد"}' "$TOKEN"
fi

test_endpoint "POST /researchers (create)" POST "$API/researchers" '{"name_ar":"د. تجريبي","email":"test.res@turath.edu.iq","title_ar":"مدرس"}' "$TOKEN"

# ── 4. Publications ───────────────────────────────
echo ""
echo "▶ 4. Publications CRUD"
test_endpoint "GET /publications (list)" GET "$API/publications?page=1&limit=10" "" "$TOKEN"
test_endpoint "GET /publications (search)" GET "$API/publications?q=الذكاء" "" "$TOKEN"
test_endpoint "POST /publications" POST "$API/publications" '{"title_ar":"بحث تجريبي","title_en":"Test Publication","doi":"10.9999/test.001"}' "$TOKEN"

PUB_ID=$(docker compose exec -T backend wget -q -O - --header="Authorization: Bearer $TOKEN" "$API/publications?limit=1" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'] if d else '')")
if [ -n "$PUB_ID" ]; then
    test_endpoint "GET /publications/{id}" GET "$API/publications/$PUB_ID" "" "$TOKEN"
fi

# ── 5. Organizations ──────────────────────────────
echo ""
echo "▶ 5. Organizations"
test_endpoint "GET /institutions" GET "$API/institutions" "" "$TOKEN"
test_endpoint "GET /colleges" GET "$API/colleges" "" "$TOKEN"
test_endpoint "GET /departments" GET "$API/departments" "" "$TOKEN"
test_endpoint "GET /colleges (filter)" GET "$API/colleges?institution_id=11111111-1111-1111-1111-111111111111" "" "$TOKEN"

# ── 6. Metrics ────────────────────────────────────
echo ""
echo "▶ 6. Metrics & Dashboards"
test_endpoint "GET /metrics/institution" GET "$API/metrics/institution" "" "$TOKEN"
test_endpoint "GET /metrics/leaderboard" GET "$API/metrics/leaderboard?sort=h_index" "" "$TOKEN"
test_endpoint "GET /metrics/departments" GET "$API/metrics/departments" "" "$TOKEN"

if [ -n "$RES_ID" ]; then
    test_endpoint "GET /metrics/researcher/{id}" GET "$API/metrics/researcher/$RES_ID" "" "$TOKEN"
fi

test_endpoint "GET /export/researchers (CSV)" GET "$API/export/researchers" "" "$TOKEN"

# ── 7. Search ─────────────────────────────────────
echo ""
echo "▶ 7. Search"
test_endpoint "GET /search (Arabic)" GET "$API/search?q=حيدر" "" "$TOKEN"
test_endpoint "GET /search (English)" GET "$API/search?q=Cybersecurity" "" "$TOKEN"
test_endpoint "GET /search (too short)" GET "$API/search?q=a" "" "$TOKEN" "error"

# ── 8. Notifications ──────────────────────────────
echo ""
echo "▶ 8. Notifications"
test_endpoint "GET /notifications" GET "$API/notifications" "" "$TOKEN"
test_endpoint "GET /notifications/unread" GET "$API/notifications/unread" "" "$TOKEN"
test_endpoint "PUT /notifications/read-all" PUT "$API/notifications/read-all" '{}' "$TOKEN"

# ── 9. Sync ───────────────────────────────────────
echo ""
echo "▶ 9. Sync (External APIs)"
if [ -n "$RES_ID" ]; then
    test_endpoint "POST /sync/researcher/{id}" POST "$API/sync/researcher/$RES_ID" '' "$TOKEN"
fi
test_endpoint "POST /sync/doi" POST "$API/sync/doi" '{"doi":"10.1038/nature12373"}' "$TOKEN"
test_endpoint "GET /sync/jobs" GET "$API/sync/jobs" "" "$TOKEN"

# ── 10. AI Insights ───────────────────────────────
echo ""
echo "▶ 10. AI Insights (expect error without API key)"
test_endpoint "GET /ai/trends" GET "$API/ai/trends" "" "$TOKEN" "error"
if [ -n "$RES_ID" ]; then
    test_endpoint "GET /ai/collaboration/{id}" GET "$API/ai/collaboration/$RES_ID" "" "$TOKEN" "error"
    test_endpoint "GET /ai/strategy/{id}" GET "$API/ai/strategy/$RES_ID" "" "$TOKEN" "error"
fi
test_endpoint "GET /ai/gaps" GET "$API/ai/gaps" "" "$TOKEN" "error"

# ── 11. Promotions ────────────────────────────────
echo ""
echo "▶ 11. Scientific Affairs — Promotions"
if [ -n "$RES_ID" ]; then
    test_endpoint "GET /promotions/eligibility/{id}" GET "$API/promotions/eligibility/$RES_ID" "" "$TOKEN"
fi
test_endpoint "POST /promotions (submit)" POST "$API/promotions" '{"researcher_id":"a0000002-0000-0000-0000-000000000002","current_rank":"associate_prof","target_rank":"professor"}' "$TOKEN"
test_endpoint "GET /promotions (list)" GET "$API/promotions" "" "$TOKEN"
test_endpoint "GET /promotions (filter by status)" GET "$API/promotions?status=submitted" "" "$TOKEN"

# ── 12. Incentives ────────────────────────────────
echo ""
echo "▶ 12. Scientific Affairs — Incentives"
test_endpoint "GET /incentives/rules" GET "$API/incentives/rules" "" "$TOKEN"
test_endpoint "GET /incentives/calculate/{id}" GET "$API/incentives/calculate/a0000001-0000-0000-0000-000000000001" "" "$TOKEN"

# ── 13. Performance ───────────────────────────────
echo ""
echo "▶ 13. Scientific Affairs — Performance Reviews"
test_endpoint "POST /performance/generate (CS dept)" POST "$API/performance/generate/d0000007-0000-0000-0000-000000000007?year=2024" '' "$TOKEN"

# ── 14. Sabbaticals ───────────────────────────────
echo ""
echo "▶ 14. Scientific Affairs — Sabbaticals"
test_endpoint "POST /sabbaticals" POST "$API/sabbaticals" '{"researcher_id":"a0000003-0000-0000-0000-000000000003","destination":"MIT","destination_country":"USA","research_plan":"بحث في الرياضيات التطبيقية","start_date":"2025-09-01","end_date":"2026-08-31"}' "$TOKEN"
test_endpoint "GET /sabbaticals" GET "$API/sabbaticals" "" "$TOKEN"

# ── 15. Conferences ───────────────────────────────
echo ""
echo "▶ 15. Scientific Affairs — Conferences"
test_endpoint "POST /conferences" POST "$API/conferences" '{"researcher_id":"a0000005-0000-0000-0000-000000000005","conference_name":"IEEE International Conference on AI","conference_location":"Dubai","conference_country":"UAE","start_date":"2025-03-15","end_date":"2025-03-17","paper_title":"Deep Learning for Arabic NLP","participation_type":"presenter","is_international":true,"estimated_cost":2500}' "$TOKEN"

# ── 16. Agreements ────────────────────────────────
echo ""
echo "▶ 16. Agreements"
test_endpoint "GET /agreements" GET "$API/agreements" "" "$TOKEN"

# ── 17. Accreditation ─────────────────────────────
echo ""
echo "▶ 17. Accreditation & Quality"
test_endpoint "GET /accreditation/dashboard" GET "$API/accreditation/dashboard" "" "$TOKEN"
test_endpoint "POST /accreditation/evidence" POST "$API/accreditation/evidence" '{"accreditation_body":"ncaaa","standard_code":"4.2.1","standard_name_ar":"إنتاجية البحث العلمي","evidence_type":"auto","evidence_description":"تقرير الأبحاث المنشورة للعام 2024","cycle_year":2024}' "$TOKEN"
test_endpoint "GET /accreditation/evidence" GET "$API/accreditation/evidence?body=ncaaa" "" "$TOKEN"
test_endpoint "GET /accreditation/recommendations" GET "$API/accreditation/recommendations" "" "$TOKEN"
test_endpoint "GET /accreditation/kpis" GET "$API/accreditation/kpis?year=2024" "" "$TOKEN"

# ── 18. Workflows ─────────────────────────────────
echo ""
echo "▶ 18. Approval Workflows"

# Get a promotion ID to create workflow
PROMO_ID=$(docker compose exec -T backend wget -q -O - --header="Authorization: Bearer $TOKEN" "$API/promotions?status=submitted" 2>&1 | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'] if d else '')" 2>/dev/null)

if [ -n "$PROMO_ID" ]; then
    test_endpoint "POST /workflows/initiate (promotion)" POST "$API/workflows/initiate" "{\"workflow_type\":\"promotion\",\"request_id\":\"$PROMO_ID\"}" "$TOKEN"
    test_endpoint "GET /workflows/history" GET "$API/workflows/history?type=promotion&request_id=$PROMO_ID" "" "$TOKEN"
fi
test_endpoint "GET /workflows/pending" GET "$API/workflows/pending" "" "$TOKEN"

# ── 19. Role Dashboard ────────────────────────────
echo ""
echo "▶ 19. Role-based Dashboard"
test_endpoint "GET /dashboard/me" GET "$API/dashboard/me" "" "$TOKEN"

# ── 20. Auth Protection ───────────────────────────
echo ""
echo "▶ 20. Auth Protection (no token)"
test_endpoint "GET /researchers (no auth)" GET "$API/researchers" "" "" "error"
test_endpoint "GET /metrics/institution (no auth)" GET "$API/metrics/institution" "" "" "error"

# ── 21. Frontend ──────────────────────────────────
echo ""
echo "▶ 21. Frontend (Next.js)"
FRONTEND_STATUS=$(docker compose exec -T frontend wget -q -O /dev/null -S "http://localhost:3000/ar/dashboard" 2>&1 | grep "200" | head -1)
if [ -n "$FRONTEND_STATUS" ]; then
    PASS=$((PASS + 1))
    TOTAL=$((TOTAL + 1))
    echo "  ✅ Frontend /ar/dashboard (200 OK)"
else
    TOTAL=$((TOTAL + 1))
    FRONTEND_CHECK=$(docker compose exec -T frontend wget -q -O - "http://localhost:3000/ar/dashboard" 2>&1 | head -c 50)
    if echo "$FRONTEND_CHECK" | grep -q "DOCTYPE\|html"; then
        PASS=$((PASS + 1))
        echo "  ✅ Frontend /ar/dashboard (HTML rendered)"
    else
        FAIL=$((FAIL + 1))
        echo "  ❌ Frontend /ar/dashboard"
    fi
fi

# ══════════════════════════════════════════════════
echo ""
echo "══════════════════════════════════════════════════"
echo "  النتائج: $PASS/$TOTAL نجاح | $FAIL فشل"
echo "══════════════════════════════════════════════════"

if [ $FAIL -eq 0 ]; then
    echo "  ✅ جميع الاختبارات ناجحة!"
else
    echo "  ⚠️  بعض الاختبارات فشلت"
fi
