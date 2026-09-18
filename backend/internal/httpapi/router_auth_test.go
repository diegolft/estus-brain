package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafael/estus-vault/backend/internal/service"
)

// TestRouterSessionGate pins which routes are reachable without a session:\n// a regression here means an endpoint silently lost its gate.\n//\n// A token-less request never reaches the repos: UserFromToken short-circuits
// on the empty string, so nil repos are safe here.
func TestRouterSessionGate(t *testing.T) {
	auth := service.NewAuthService(nil, nil)
	mcpHit := false
	r := NewRouter(&Handlers{}, Modules{
		Auth: NewAuthHandlers(auth),
		MCP: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mcpHit = true
			w.WriteHeader(http.StatusOK)
		}),
	})

	cases := []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/health", http.StatusOK},
		{"GET", "/api/categories", http.StatusUnauthorized},
		{"GET", "/api/auth/me", http.StatusUnauthorized},
		{"POST", "/api/auth/logout", http.StatusUnauthorized},
		{"POST", "/api/transactions", http.StatusUnauthorized},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}

	// MCP must bypass the session gate entirely.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("POST", "/mcp", nil))
	if !mcpHit || rec.Code != http.StatusOK {
		t.Errorf("/mcp gated: hit=%v code=%d", mcpHit, rec.Code)
	}

	// Login must be reachable without a session (400/422 from an empty body
	// is fine; 401 would mean the gate swallowed it).
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/login", nil))
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("POST /api/auth/login was gated by requireSession")
	} else {
		t.Logf("login reachable, code=%d", rec.Code)
	}
}
