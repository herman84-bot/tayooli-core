package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

func tenantReq(id uuid.UUID) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	return r.WithContext(context.WithValue(r.Context(), TenantIDKey, id))
}

func TestTenantRateLimit_429WithRetryAfterWhenBurstExhausted(t *testing.T) {
	l := NewTenantRateLimiter(1, 3) // 1 rps, burst 3
	defer l.Stop()
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	tenant := uuid.New()

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, tenantReq(tenant))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d within burst: got %d", i, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, tenantReq(tenant))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after burst, got %d", rec.Code)
	}
	ra, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || ra < 1 {
		t.Fatalf("expected Retry-After >= 1 second, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestTenantRateLimit_TenantsAreIsolated(t *testing.T) {
	l := NewTenantRateLimiter(1, 1)
	defer l.Stop()
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	a, b := uuid.New(), uuid.New()

	h.ServeHTTP(httptest.NewRecorder(), tenantReq(a)) // a uses its only token
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, tenantReq(b))
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant b must not be throttled by tenant a, got %d", rec.Code)
	}
}

func TestTenantRateLimit_NoTenantPassesThrough(t *testing.T) {
	l := NewTenantRateLimiter(1, 1)
	defer l.Stop()
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("unauthenticated paths are not tenant-limited (auth limiter covers them), got %d", rec.Code)
		}
	}
}
