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
