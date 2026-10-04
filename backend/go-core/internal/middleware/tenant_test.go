package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-32-chars-long!!!")
}

func createTestToken(t *testing.T, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret-key-32-chars-long!!!"))
	require.NoError(t, err)
	return tokenStr
}

func TestTenantMiddleware_ValidToken(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// Verify tenant ID and role are in context
		tid, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, tenantID, tid)

		role := GetRole(r.Context())
		assert.Equal(t, "admin", role)
	})

	middleware := TenantMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestTenantMiddleware_CookieToken(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "admin",
		"sub":   uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		tid, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, tenantID, tid)

		role := GetRole(r.Context())
		assert.Equal(t, "admin", role)
	})

	middleware := TenantMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: tokenStr})
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestTenantMiddleware_CookieTakesPriority(t *testing.T) {
	setupTestEnv(t)

	// Cookie token with valid claims
	cookieTenantID := uuid.New()
	cookieClaims := jwt.MapClaims{
		"tenant_id": cookieTenantID.String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	cookieToken := createTestToken(t, cookieClaims)

	// Header token with different tenant
	headerTenantID := uuid.New()
	headerClaims := jwt.MapClaims{
		"tenant_id": headerTenantID.String(),
		"role":      "viewer",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	headerToken := createTestToken(t, headerClaims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// Should use cookie tenant, not header tenant
		tid, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, cookieTenantID, tid)
		assert.NotEqual(t, headerTenantID, tid)
	})

	middleware := TenantMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: cookieToken})
	req.Header.Set("Authorization", "Bearer "+headerToken)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestTenantMiddleware_FallbackToHeader(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "approver",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		tid, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, tenantID, tid)
		role := GetRole(r.Context())
		assert.Equal(t, "approver", role)
	})

	middleware := TenantMiddleware(next)
	// No cookie, only Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestTenantMiddleware_EmptyCookieIgnored(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		tid, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, tenantID, tid)
	})

	middleware := TenantMiddleware(next)
	// Empty cookie value should be skipped, header used instead
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: ""})
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestTenantMiddleware_NoAuthSources(t *testing.T) {
	setupTestEnv(t)

	middleware := TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "missing authentication", resp["error"])
}

func TestTenantMiddleware_InvalidToken(t *testing.T) {
	setupTestEnv(t)

	middleware := TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	// Test missing auth header (no cookie, no header)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Test invalid bearer format
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Invalid token")
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Test expired token
	claims := jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"exp":       time.Now().Add(-time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Test wrong signing method
	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	wrongTokenStr, _ := wrongToken.SignedString([]byte("wrong-secret"))
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+wrongTokenStr)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Test invalid tenant_id format
	claims = jwt.MapClaims{
		"tenant_id": "not-a-uuid",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr = createTestToken(t, claims)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// Test missing tenant_id claim
	claims = jwt.MapClaims{
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	tokenStr = createTestToken(t, claims)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestTenantMiddleware_ShortJWTSecret(t *testing.T) {
	// Below the 32-char minimum enforced by config.Load — the middleware must
	// reject it too instead of silently signing/verifying with a weak key.
	t.Setenv("JWT_SECRET", "short-secret")

	middleware := TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	claims := jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "JWT_SECRET must be at least 32 characters")
}

func TestTenantMiddleware_MissingJWTSecret(t *testing.T) {
	// Don't set JWT_SECRET
	os.Unsetenv("JWT_SECRET")

	middleware := TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	claims := jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "server misconfiguration", resp["error"])
}

func TestRequireRole_AllowedRole(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	// Chain: TenantMiddleware -> RequireRole
	chained := TenantMiddleware(RequireRole("admin", "approver")(next))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequireRole_DeniedRole(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"role":      "viewer",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	chained := TenantMiddleware(RequireRole("admin", "approver")(next))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "forbidden", resp["error"])
}

func TestRequireRole_NoRoleInToken(t *testing.T) {
	setupTestEnv(t)
	tenantID := uuid.New()
	claims := jwt.MapClaims{
		"tenant_id": tenantID.String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := createTestToken(t, claims)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	chained := TenantMiddleware(RequireRole("admin")(next))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequestBodyLimit_WithinLimit(t *testing.T) {
	body := `{"key": "value"}`
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		bodyBytes, _ := io.ReadAll(r.Body)
		assert.Equal(t, body, string(bodyBytes))
	})

	limited := RequestBodyLimit(64 * 1024)(next)

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	limited.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled)
}

func TestRequestBodyLimit_ExceedsLimit(t *testing.T) {
	largeBody := string(bytes.Repeat([]byte("x"), 65*1024)) // 65 KiB
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	limited := RequestBodyLimit(64 * 1024)(next)

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	limited.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

func TestGetTenantIDFromHeader(t *testing.T) {
	tenantID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())

	id, ok := GetTenantIDFromHeader(req)
	assert.True(t, ok)
	assert.Equal(t, tenantID, id)
}

func TestGetTenantIDFromHeader_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	id, ok := GetTenantIDFromHeader(req)
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, id)
}

func TestGetTenantIDFromHeader_InvalidUUID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	id, ok := GetTenantIDFromHeader(req)
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, id)
}

func TestSetTenantIDContext(t *testing.T) {
	tenantID := uuid.New()
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		id, ok := GetTenantID(r.Context())
		assert.True(t, ok)
		assert.Equal(t, tenantID, id)
	})

	wrapped := SetTenantIDContext(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSetTenantIDContext_NoHeader(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, ok := GetTenantID(r.Context())
		assert.False(t, ok)
	})

	wrapped := SetTenantIDContext(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}