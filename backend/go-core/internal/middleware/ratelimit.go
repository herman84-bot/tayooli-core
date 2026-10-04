package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// trustedProxyEnv lists reverse proxies the limiter may trust, as a
// comma-separated list of IPs or CIDRs (e.g. "127.0.0.1,10.0.0.0/8").
// Only requests whose immediate TCP peer is in this list get their client IP
// resolved from X-Forwarded-For. When unset or empty, the header is ignored
// entirely and the limiter keys on the connecting peer.
const trustedProxyEnv = "TRUSTED_PROXY_IPS"

// limiterEntry pairs a rate limiter with the time of its last access.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// TokenBucketLimiter implements per-IP rate limiting using a token bucket algorithm.
// Each IP gets its own rate.Limiter. Stale entries (no request in 5 minutes)
// are evicted by a background cleanup goroutine.
type TokenBucketLimiter struct {
	mu             sync.Mutex
	limiters       map[string]*limiterEntry
	rps            rate.Limit
	burst          int
	stopCh         chan struct{}
	trustedProxies []netip.Prefix
}

const staleThreshold = 5 * time.Minute

// NewTokenBucketLimiter creates a new rate limiter. rps is requests per second,
// burst is the maximum burst size (initial token count).
// A background goroutine evicts stale entries every minute.
//
// Trusted reverse proxies are read once from TRUSTED_PROXY_IPS at construction
// (mirroring how CORSEnv reads FRONTEND_ORIGIN), so a restart is required after
// changing the proxy topology.
func NewTokenBucketLimiter(rps float64, burst int) *TokenBucketLimiter {
	t := &TokenBucketLimiter{
		limiters:       make(map[string]*limiterEntry),
		rps:            rate.Limit(rps),
		burst:          burst,
		stopCh:         make(chan struct{}),
		trustedProxies: parseTrustedProxies(os.Getenv(trustedProxyEnv)),
	}
	go t.cleanup()
	return t
}

// Stop terminates the background cleanup goroutine.
func (t *TokenBucketLimiter) Stop() {
	close(t.stopCh)
}

// cleanup periodically removes stale limiter entries.
func (t *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			for ip, entry := range t.limiters {
				if time.Since(entry.lastAccess) > staleThreshold {
					delete(t.limiters, ip)
				}
			}
			t.mu.Unlock()
		case <-t.stopCh:
			return
		}
	}
}

// GetLimiter returns (or creates) a rate.Limiter for the given IP.
func (t *TokenBucketLimiter) GetLimiter(ip string) *rate.Limiter {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, exists := t.limiters[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:    rate.NewLimiter(t.rps, t.burst),
			lastAccess: time.Now(),
		}
		t.limiters[ip] = entry
	}
	entry.lastAccess = time.Now()
	return entry.limiter
}

// clientIP resolves the address the rate limit is keyed on, honoring the
// trusted-proxy configuration. Never trusts a spoofable header from an
// untrusted peer.
func (t *TokenBucketLimiter) clientIP(r *http.Request) string {
	return resolveClientIP(r, t.trustedProxies)
}

// Middleware returns an HTTP middleware that applies per-IP token-bucket rate limiting.
// Returns 429 Too Many Requests when the limit is exceeded.
func (t *TokenBucketLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := t.GetLimiter(t.clientIP(r))
		if !limiter.Allow() {
			writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostOnly strips the port from an address (e.g. "1.2.3.4:5678" → "1.2.3.4",
// "[::1]:5678" → "::1"). Returns the input trimmed of brackets when there is
// no port to split.
func hostOnly(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return strings.Trim(addr, "[]")
	}
	return host
}

// parseTrustedProxies parses a comma-separated list of IPs and/or CIDRs into
// prefixes. Malformed entries are skipped silently — a misconfiguration must
// fail closed (fewer trusted proxies), never open.
func parseTrustedProxies(raw string) []netip.Prefix {
	var trusted []netip.Prefix
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if p, err := netip.ParsePrefix(part); err == nil {
			trusted = append(trusted, p)
			continue
		}
		if a, err := netip.ParseAddr(part); err == nil {
			trusted = append(trusted, netip.PrefixFrom(a, a.BitLen()))
		}
	}
	return trusted
}

// isTrustedIP reports whether ip is covered by any trusted prefix.
func isTrustedIP(ip string, trusted []netip.Prefix) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, p := range trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// resolveClientIP determines the real client IP to rate-limit on.
//
// Rate limiting must be keyed on the connecting peer unless that peer is a
// reverse proxy we control. Trusting the X-Forwarded-For header unconditionally
// lets any caller bypass the limiter simply by spoofing the header — critical
// when the API is reachable directly (e.g. a firewalled port), not only through
// nginx.
//
// Resolution rules:
//  1. Start from the immediate TCP peer (r.RemoteAddr).
//  2. If the peer is NOT a trusted proxy (TRUSTED_PROXY_IPS), return it as-is —
//     the header is ignored entirely.
//  3. If the peer IS trusted, walk the X-Forwarded-For chain right-to-left and
//     return the first hop that is not itself a trusted proxy (the original
//     client). If the header is absent or every hop is trusted, fall back to
//     the peer — never a spoofable middle value.
func resolveClientIP(r *http.Request, trusted []netip.Prefix) string {
	peer := hostOnly(r.RemoteAddr)
	if len(trusted) == 0 || !isTrustedIP(peer, trusted) {
		return peer
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return peer
	}
	hops := strings.Split(xff, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		candidate := hostOnly(strings.TrimSpace(hops[i]))
		if candidate == "" || isTrustedIP(candidate, trusted) {
			continue
		}
		return candidate
	}
	return peer
}
