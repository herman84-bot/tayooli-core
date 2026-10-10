package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeRevocation struct {
	revoked map[string]bool
	err     error
}

func (f *fakeRevocation) IsRevoked(_ context.Context, h string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.revoked[h], nil
}

func revocationToken(t *testing.T) string {
	return createTestToken(t, jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"sub":       uuid.New().String(),
		"role":      "admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
}

func runWithChecker(t *testing.T, c RevocationChecker, tok string) (int, bool) {
	t.Helper()
	setupTestEnv(t)
	SetRevocationChecker(c)
	t.Cleanup(func() { SetRevocationChecker(nil) })
	called := false
	h := TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, ok := GetTokenExpiry(r.Context())
		assert.True(t, ok, "token expiry must be propagated for logout")
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: tok})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, called
}

func TestTenantMiddleware_RevokedTokenRejected(t *testing.T) {
	tok := revocationToken(t)
	code, called := runWithChecker(t, &fakeRevocation{revoked: map[string]bool{TokenHash(tok): true}}, tok)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.False(t, called)
}

func TestTenantMiddleware_NotRevokedPasses(t *testing.T) {
	tok := revocationToken(t)
	code, called := runWithChecker(t, &fakeRevocation{revoked: map[string]bool{}}, tok)
	assert.Equal(t, http.StatusOK, code)
	assert.True(t, called)
}

func TestTenantMiddleware_RevocationStoreErrorFailsClosed(t *testing.T) {
	tok := revocationToken(t)
	code, called := runWithChecker(t, &fakeRevocation{err: errors.New("db down")}, tok)
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.False(t, called)
}

func TestTokenHash_StableAndHex(t *testing.T) {
	assert.Equal(t, TokenHash("abc"), TokenHash("abc"))
	assert.Len(t, TokenHash("abc"), 64)
	assert.NotEqual(t, TokenHash("abc"), TokenHash("abd"))
}
