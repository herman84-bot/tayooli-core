package middleware

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// TenantRateLimiter applies a token bucket per tenant (GitHub-style per-account
// limits: one noisy tenant cannot starve the shared DB pool for everyone).
// Keyed on tenant, not IP, because every browser request reaches the backend
// through the same Next.js proxy IP. Must run after TenantMiddleware.
type TenantRateLimiter struct {
	buckets *TokenBucketLimiter
}

// NewTenantRateLimiter creates a per-tenant limiter: rps sustained, burst peak.
func NewTenantRateLimiter(rps float64, burst int) *TenantRateLimiter {
	return &TenantRateLimiter{buckets: NewTokenBucketLimiter(rps, burst)}
}

// Stop terminates the background eviction goroutine.
func (l *TenantRateLimiter) Stop() { l.buckets.Stop() }

// Middleware returns 429 with Retry-After (seconds) when the tenant's bucket
// is empty. Requests without a tenant in context pass through.
func (l *TenantRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := GetTenantID(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		res := l.buckets.GetLimiter("tenant:" + tenantID.String()).Reserve()
		if delay := res.Delay(); delay > 0 {
			res.Cancel() // do not consume a future token for a rejected request
			if delay == time.Duration(math.MaxInt64) {
				delay = time.Second
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			writeJSONError(w, http.StatusTooManyRequests, "Terlalu banyak permintaan. Coba lagi sebentar.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
