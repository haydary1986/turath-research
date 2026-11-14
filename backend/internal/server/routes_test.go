package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/middleware"
)

const testSecret = "route-test-secret"

func testRouter() http.Handler {
	cfg := &config.Config{}
	cfg.JWT.Secret = testSecret
	cfg.Server.RateLimitPerMin = 100000
	cfg.Server.AllowedOrigins = "http://localhost"
	return NewRouter(cfg, nil, nil, nil)
}

func tokenFor(t *testing.T, role string) string {
	t.Helper()
	claims := middleware.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		UserID:           "00000000-0000-0000-0000-000000000001",
		Role:             role,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func status(t *testing.T, router http.Handler, method, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader("not json"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

type route struct{ method, path string }

// Writes to shared registers and anything that spends upstream quota for someone else.
var managerRoutes = []route{
	{"POST", "/researchers"}, {"PUT", "/researchers/x"}, {"DELETE", "/researchers/x"},
	{"POST", "/publications"}, {"DELETE", "/publications/x"},
	{"POST", "/institutions"}, {"POST", "/colleges"}, {"POST", "/departments"},
	{"GET", "/export/researchers"},
	{"POST", "/sync/researcher/x"}, {"POST", "/sync/doi"}, {"GET", "/sync/jobs"},
	{"POST", "/performance/generate/x"},
	{"POST", "/accreditation/evidence"}, {"PUT", "/accreditation/recommendations/x"},
	{"POST", "/workflows/initiate"},
	{"PUT", "/site"}, {"GET", "/announcements"}, {"POST", "/announcements"},
	{"GET", "/announcements/x"}, {"PUT", "/announcements/x"}, {"DELETE", "/announcements/x"},
	{"POST", "/announcements/x/publish"}, {"POST", "/announcements/x/pin"},
	{"POST", "/announcements/x/attachment"}, {"DELETE", "/announcements/x/attachment"},
	{"POST", "/activities/x/announce"},
	{"GET", "/site/stats"}, {"POST", "/site/director-photo"}, {"DELETE", "/site/director-photo"},
}

var staffRoutes = []route{
	{"GET", "/ai/collaboration/x"}, {"GET", "/ai/trends"}, {"GET", "/ai/strategy/x"}, {"GET", "/ai/gaps"},
	{"GET", "/promotions"}, {"GET", "/sabbaticals"}, {"GET", "/agreements"},
	{"GET", "/accreditation/dashboard"}, {"GET", "/accreditation/evidence"},
	{"GET", "/accreditation/recommendations"}, {"GET", "/accreditation/kpis"},
	{"GET", "/workflows/history"},
}

func TestRestrictedRoutesRefuseOtherRoles(t *testing.T) {
	router := testRouter()
	outsiders := []string{
		middleware.RoleResearcher, middleware.RoleViewer, middleware.RoleDepartmentHead,
		middleware.RoleCollegeDean, middleware.RoleCenterDirector, middleware.RoleDeanResearch,
	}
	for _, role := range outsiders {
		token := tokenFor(t, role)
		for _, r := range append(append([]route{}, managerRoutes...), staffRoutes...) {
			if got := status(t, router, r.method, r.path, token); got != http.StatusForbidden {
				t.Errorf("%s %s as %s = %d, want 403", r.method, r.path, role, got)
			}
		}
	}
	vp := tokenFor(t, middleware.RoleVPResearch)
	for _, r := range managerRoutes {
		if got := status(t, router, r.method, r.path, vp); got != http.StatusForbidden {
			t.Errorf("%s %s as vp_research = %d, want 403", r.method, r.path, got)
		}
	}
}

func TestRestrictedRoutesNeedAToken(t *testing.T) {
	router := testRouter()
	for _, r := range append(append([]route{}, managerRoutes...), staffRoutes...) {
		if got := status(t, router, r.method, r.path, ""); got != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", r.method, r.path, got)
		}
	}
}

// These stop at request validation, so they prove the role passed without a database.
func TestManagersPassTheRoleCheck(t *testing.T) {
	router := testRouter()
	cases := []struct {
		method, path string
		want         int
	}{
		{"POST", "/researchers", http.StatusBadRequest},
		{"PUT", "/researchers/not-a-uuid", http.StatusNotFound},
		{"POST", "/sync/doi", http.StatusBadRequest},
		{"POST", "/sync/researcher/not-a-uuid", http.StatusNotFound},
		{"PUT", "/site", http.StatusBadRequest},
		{"POST", "/announcements", http.StatusBadRequest},
		{"PUT", "/announcements/not-a-uuid", http.StatusNotFound},
		{"DELETE", "/announcements/not-a-uuid", http.StatusNotFound},
		{"POST", "/announcements/not-a-uuid/publish", http.StatusNotFound},
	}
	for _, role := range []string{middleware.RoleAdmin, middleware.RoleScientificAffairs} {
		token := tokenFor(t, role)
		for _, tc := range cases {
			if got := status(t, router, tc.method, tc.path, token); got != tc.want {
				t.Errorf("%s %s as %s = %d, want %d", tc.method, tc.path, role, got, tc.want)
			}
		}
	}
}

func TestBulkSyncAndImportAreAdminOnly(t *testing.T) {
	router := testRouter()
	refused := []string{
		middleware.RoleResearcher, middleware.RoleCollegeDean, middleware.RoleScientificAffairs, middleware.RoleVPResearch,
	}
	routes := []route{
		{"POST", "/admin/import/researchers"}, {"GET", "/admin/sync/orcid"},
		{"POST", "/admin/sync/orcid"}, {"POST", "/admin/sync/orcid/stop"},
	}
	for _, role := range refused {
		token := tokenFor(t, role)
		for _, rt := range routes {
			if code := status(t, router, rt.method, rt.path, token); code != http.StatusForbidden {
				t.Errorf("%s %s as %s: got %d", rt.method, rt.path, role, code)
			}
		}
	}
}

// Public site reads need no token and reject malformed ids before touching the database.
func TestPublicSiteRoutesNeedNoToken(t *testing.T) {
	router := testRouter()
	for _, path := range []string{
		"/public/announcements/not-a-uuid", "/public/files/not-a-uuid",
		"/public/site/director-photo/not-a-uuid", "/public/journals/entries/Not_A_Slug",
	} {
		if got := status(t, router, "GET", path, ""); got != http.StatusNotFound {
			t.Errorf("GET %s without a token = %d, want 404", path, got)
		}
	}
}

// The login log, account activity and responsibility management are the admin's alone.
func TestAuditAndResponsibilityRoutesAreAdminOnly(t *testing.T) {
	router := testRouter()
	refused := []string{
		middleware.RoleResearcher, middleware.RoleCollegeDean, middleware.RoleDepartmentHead,
		middleware.RoleScientificAffairs, middleware.RoleVPResearch,
	}
	routes := []route{
		{"GET", "/admin/login-events"}, {"GET", "/admin/accounts"}, {"GET", "/admin/accounts/summary"},
		{"GET", "/admin/users/x/responsibilities"}, {"PUT", "/admin/users/x/responsibilities"},
	}
	for _, role := range refused {
		token := tokenFor(t, role)
		for _, rt := range routes {
			if code := status(t, router, rt.method, rt.path, token); code != http.StatusForbidden {
				t.Errorf("%s %s as %s: got %d", rt.method, rt.path, role, code)
			}
		}
	}
	admin := tokenFor(t, middleware.RoleAdmin)
	if code := status(t, router, "PUT", "/admin/users/not-a-uuid/responsibilities", admin); code != http.StatusNotFound {
		t.Errorf("admin reaches the responsibility handler: got %d", code)
	}
	for _, rt := range routes {
		if code := status(t, router, rt.method, rt.path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: got %d", rt.method, rt.path, code)
		}
	}
}

// The public journal-list and journal-check endpoints were retired with their pages:
// the public site now links to the source lists instead.
func TestPublicJournalEndpointsAreRetired(t *testing.T) {
	router := testRouter()
	for _, path := range []string{"/public/journals/entries?type=hijacked", "/public/journals/check?name=x", "/public/journals/status"} {
		if got := status(t, router, "GET", path, ""); got != http.StatusNotFound && got != http.StatusMethodNotAllowed {
			t.Errorf("GET %s = %d, want 404", path, got)
		}
	}
}
