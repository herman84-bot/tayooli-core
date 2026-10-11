package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestMetricsAuth(t *testing.T) {
	cases := []struct {
		name, token, header string
		want                int
	}{
		{"token unset is fail-closed", "", "", http.StatusNotFound},
		{"token unset ignores any header", "", "Bearer ", http.StatusNotFound},
		{"missing header", "s3cret", "", http.StatusNotFound},
		{"wrong token", "s3cret", "Bearer nope", http.StatusNotFound},
		{"token without Bearer prefix", "s3cret", "s3cret", http.StatusNotFound},
		{"correct token", "s3cret", "Bearer s3cret", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			MetricsAuth(c.token)(okHandler()).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("want %d got %d", c.want, rec.Code)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	want := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s: want %q got %q", k, v, got)
		}
	}
}

// Mirrors the accounting gates wired in cmd/api/main.go.
func TestAccountingRoleGates(t *testing.T) {
	read := RequireRole("owner", "admin", "accountant", "auditor")
	write := RequireRole("owner", "admin", "accountant")
	cases := []struct {
		role                string
		wantRead, wantWrite int
	}{
		{"owner", 200, 200},
		{"admin", 200, 200},
		{"accountant", 200, 200},
		{"auditor", 200, 403},
		{"cashier", 403, 403},
		{"warehouse", 403, 403},
		{"warehouse_manager", 403, 403},
		{"member", 403, 403},
		{"", 403, 403},
	}
	for _, c := range cases {
		for _, g := range []struct {
			mw   func(http.Handler) http.Handler
			want int
		}{{read, c.wantRead}, {write, c.wantWrite}} {
			req := httptest.NewRequest(http.MethodPost, "/accounting/journal-entries", nil)
			req = req.WithContext(context.WithValue(req.Context(), RoleKey, c.role))
			rec := httptest.NewRecorder()
			g.mw(okHandler()).ServeHTTP(rec, req)
			if rec.Code != g.want {
				t.Fatalf("role %q: want %d got %d", c.role, g.want, rec.Code)
			}
		}
	}
}
