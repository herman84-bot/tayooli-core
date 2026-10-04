package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/config"
)

type contextKey string

const TenantIDKey contextKey = "tenant_id"
const RoleKey contextKey = "role"
const UserIDKey contextKey = "user_id"

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = w.Write(b)
}

// extractToken reads the JWT from either:
//  1. the "tayooli_auth" HttpOnly cookie (preferred), or
//  2. the "Authorization: Bearer <token>" header (backward compat).
//
// Returns the raw token string or empty string if neither source is present.
func extractToken(r *http.Request) string {
	// Cookie takes priority.
	if cookie, err := r.Cookie("tayooli_auth"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	// Fallback: Authorization header.
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// TenantMiddleware validates JWT (strict HS256), extracts tenant_id and role,
// and sets them in request context. Must be applied before any handler that
// requires tenant isolation.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing authentication")
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			writeJSONError(w, http.StatusInternalServerError, "server misconfiguration")
			return
		}
		// Enforce the same minimum-length gate as config.Load: a short secret
		// means the CHANGE_ME placeholder (or an equivalent) is still in use.
		if len(secret) < config.MinJWTSecretLength {
			writeJSONError(w, http.StatusInternalServerError,
				fmt.Sprintf("server misconfiguration: JWT_SECRET must be at least %d characters", config.MinJWTSecretLength))
			return
		}

		token, err := jwt.NewParser(
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithExpirationRequired(),
		).Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		tenantIDStr, ok := claims["tenant_id"].(string)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "missing tenant_id in token")
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid tenant_id format")
			return
		}

		role, _ := claims["role"].(string)

		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, RoleKey, role)

		// Extract user_id from "sub" claim (set by auth service on login).
		if userIDStr, ok := claims["sub"].(string); ok {
			if uid, err := uuid.Parse(userIDStr); err == nil {
				ctx = context.WithValue(ctx, UserIDKey, uid)
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenantID extracts tenant_id from context.
func GetTenantID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(TenantIDKey).(uuid.UUID)
	return id, ok
}

// GetRole extracts role from context.
func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(RoleKey).(string)
	return role
}

// GetUserID extracts user_id from context (optional; not all JWTs carry it).
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return id, ok
}

// RequireRole returns middleware that allows only requests whose JWT role
// claim matches one of the provided roles. Must be used after TenantMiddleware.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if !allowed[role] {
				writeJSONError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestBodyLimit returns middleware that limits request body size to maxBytes.
func RequestBodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fast path: reject known-length requests that exceed the limit upfront.
			// For chunked transfer encoding (ContentLength == -1), MaxBytesReader below is the primary defense.
			if r.ContentLength > maxBytes {
				writeJSONError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			// Wrap body reader to enforce limit during reads (handles chunked/unknown length).
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// GetTenantIDFromHeader extracts tenant_id from X-Tenant-ID header.
// Used by internal services that don't use JWT but pass tenant via header.
func GetTenantIDFromHeader(r *http.Request) (uuid.UUID, bool) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		return uuid.Nil, false
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return tenantID, true
}

// SetTenantIDContext sets tenant_id in context from X-Tenant-ID header.
// Use for internal service-to-service calls.
func SetTenantIDContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tenantID, ok := GetTenantIDFromHeader(r); ok {
			ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CORS returns middleware that sets permissive CORS headers for the allowed origins.
// In production, restrict allowedOrigins to the actual frontend domain.
func CORS(allowedOrigins ...string) func(http.Handler) http.Handler {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if originSet[origin] || originSet["*"] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimiter returns middleware that limits requests per IP address.
// maxRequests is the number of allowed requests within the window duration.
// Uses an in-memory map with periodic cleanup — suitable for single-instance
// deployments. For multi-instance, use Redis-backed rate limiting.
//
// Client IPs are resolved via resolveClientIP (trusted-proxy aware), so the
// X-Forwarded-For header is only honored from proxies listed in TRUSTED_PROXY_IPS.
func IPRateLimiter(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	type client struct {
		count    int
		resetAt  time.Time
	}
	trusted := parseTrustedProxies(os.Getenv(trustedProxyEnv))
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Background cleanup goroutine.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.After(c.resetAt) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trusted)

			mu.Lock()
			c, exists := clients[ip]
			now := time.Now()
			if !exists || now.After(c.resetAt) {
				clients[ip] = &client{count: 1, resetAt: now.Add(window)}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}
			c.count++
			if c.count > maxRequests {
				mu.Unlock()
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}