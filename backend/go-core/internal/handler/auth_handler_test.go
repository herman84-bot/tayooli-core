package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ---------------------------------------------------------------------------
// Mock usecase -- satisfies handler's private authUsecase interface structurally.
// ---------------------------------------------------------------------------

type mockAuthUsecase struct {
	loginByEmailFn        func(ctx context.Context, email, password string) (string, *domain.User, error)
	getMeFn               func(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error)
	registerFn            func(ctx context.Context, fullName, email, password string) (*domain.User, error)
	issueTokenFn          func(user *domain.User) (string, error)
	updateWorkspaceNameFn func(ctx context.Context, tenantID uuid.UUID, name string) error
	getCompanyProfileFn    func(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error)
	updateCompanyProfileFn func(ctx context.Context, tenantID uuid.UUID, p domain.TenantProfile) error
	verifyEmailFn         func(ctx context.Context, token string) (*domain.User, error)
	getUserByEmailFn      func(ctx context.Context, email string) (*domain.User, error)
}

func (m *mockAuthUsecase) LoginByEmail(ctx context.Context, email, password string) (string, *domain.User, error) {
	return m.loginByEmailFn(ctx, email, password)
}

func (m *mockAuthUsecase) GetMe(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error) {
	return m.getMeFn(ctx, userID, tenantID)
}

func (m *mockAuthUsecase) Register(ctx context.Context, fullName, email, password string) (*domain.User, error) {
	if m.registerFn != nil {
		return m.registerFn(ctx, fullName, email, password)
	}
	return nil, nil
}

func (m *mockAuthUsecase) IssueToken(user *domain.User) (string, error) {
	if m.issueTokenFn != nil {
		return m.issueTokenFn(user)
	}
	return "mock-jwt-token", nil
}

func (m *mockAuthUsecase) UpdateWorkspaceName(ctx context.Context, tenantID uuid.UUID, name string) error {
	if m.updateWorkspaceNameFn != nil {
		return m.updateWorkspaceNameFn(ctx, tenantID, name)
	}
	return nil
}

func (m *mockAuthUsecase) GetCompanyProfile(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error) {
	if m.getCompanyProfileFn != nil {
		return m.getCompanyProfileFn(ctx, tenantID)
	}
	return &domain.TenantProfile{TenantID: tenantID, Name: "PT Tayooli Indonesia"}, nil
}

func (m *mockAuthUsecase) UpdateCompanyProfile(ctx context.Context, tenantID uuid.UUID, p domain.TenantProfile) error {
	if m.updateCompanyProfileFn != nil {
		return m.updateCompanyProfileFn(ctx, tenantID, p)
	}
	return nil
}

func (m *mockAuthUsecase) VerifyEmail(ctx context.Context, token string) (*domain.User, error) {
	if m.verifyEmailFn != nil {
		return m.verifyEmailFn(ctx, token)
	}
	return nil, nil
}

func (m *mockAuthUsecase) ResendVerification(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthUsecase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockAuthUsecase) ForgotPassword(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthUsecase) ResetPassword(_ context.Context, _, _ string) error {
	return nil
}

// withAuthContext injects tenant_id and user_id into the request context,
// simulating what TenantMiddleware does for real requests.
func withAuthContext(r *http.Request, tenantID, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestAuthHandler_Login_Success(t *testing.T) {
	// Secure cookie flag is only set in production; force it for this test.
	t.Setenv("APP_ENV", "production")

	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		loginByEmailFn: func(_ context.Context, email, password string) (string, *domain.User, error) {
			return "mock-jwt-token", &domain.User{
				ID:       userID,
				TenantID: tenantID,
				Email:    email,
				Role:     "admin",
			}, nil
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"email":"admin@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify cookie
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header, got none")
	}
	cookie := cookies[0]
	if cookie.Name != "tayooli_auth" {
		t.Errorf("expected cookie name tayooli_auth, got %s", cookie.Name)
	}
	if cookie.Value != "mock-jwt-token" {
		t.Errorf("expected cookie value mock-jwt-token, got %s", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly=true")
	}
	if !cookie.Secure {
		t.Error("expected Secure=true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite=Strict, got %v", cookie.SameSite)
	}

	// Verify response body
	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	user, ok := respBody["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user object in response, got %T", respBody["user"])
	}
	if user["email"] != "admin@test.com" {
		t.Errorf("expected email admin@test.com, got %v", user["email"])
	}
	if user["role"] != "admin" {
		t.Errorf("expected role admin, got %v", user["role"])
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	uc := &mockAuthUsecase{
		loginByEmailFn: func(_ context.Context, _, _ string) (string, *domain.User, error) {
			return "", nil, domain.ErrInvalidCredentials
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"email":"admin@test.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuthHandler_Login_MalformedJSON(t *testing.T) {
	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("not-json{{{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAuthHandler_Login_MissingEmail(t *testing.T) {
	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	body := `{"password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAuthHandler_Login_ShortPassword(t *testing.T) {
	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	body := `{"email":"admin@test.com","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Me tests
// ---------------------------------------------------------------------------

func TestAuthHandler_Me_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		getMeFn: func(_ context.Context, uid, tid uuid.UUID) (*domain.User, error) {
			return &domain.User{
				ID:       uid,
				TenantID: tid,
				Email:    "admin@test.com",
				Role:     "admin",
			}, nil
		},
	}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	user, ok := respBody["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user object, got %T", respBody["user"])
	}
	if user["email"] != "admin@test.com" {
		t.Errorf("expected email admin@test.com, got %v", user["email"])
	}
}

func TestAuthHandler_Me_UserNotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		getMeFn: func(_ context.Context, _, _ uuid.UUID) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAuthHandler_Me_Unauthorized_NoUserInContext(t *testing.T) {
	tenantID := uuid.New()

	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	// Only inject tenant, no user_id
	ctx := context.WithValue(req.Context(), appMiddleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthHandler_Me_Unauthorized_NoTenantInContext(t *testing.T) {
	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	// No context values at all
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Logout tests
// ---------------------------------------------------------------------------

func TestAuthHandler_Logout_Success(t *testing.T) {
	// Secure cookie flag is only set in production; force it for this test.
	t.Setenv("APP_ENV", "production")

	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify cookie is cleared (MaxAge=0, empty value)
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header, got none")
	}
	cookie := cookies[0]
	if cookie.Name != "tayooli_auth" {
		t.Errorf("expected cookie name tayooli_auth, got %s", cookie.Name)
	}
	if cookie.Value != "" {
		t.Errorf("expected empty cookie value, got %s", cookie.Value)
	}
	// MaxAge:-1 is serialized as "Max-Age=0" (delete now); parsed back as -1.
	if cookie.MaxAge >= 0 {
		t.Errorf("expected cookie deletion (MaxAge<0), got %d", cookie.MaxAge)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly=true")
	}
	if !cookie.Secure {
		t.Error("expected Secure=true")
	}

	// Verify response
	var respBody map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if respBody["message"] != "logged out" {
		t.Errorf("expected message 'logged out', got %v", respBody["message"])
	}
}

type fakeRevoker struct {
	hash   string
	userID *uuid.UUID
	exp    time.Time
	err    error
}

func (f *fakeRevoker) Revoke(_ context.Context, h string, uid *uuid.UUID, exp time.Time) error {
	f.hash, f.userID, f.exp = h, uid, exp
	return f.err
}

const logoutTestSecret = "test-secret-key-32-chars-long!!!"

func signLogoutToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": sub, "exp": exp.Unix()}).
		SignedString([]byte(logoutTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertCookieCleared(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == "tayooli_auth" && c.Value == "" && c.MaxAge < 0 {
			return
		}
	}
	t.Fatalf("expected tayooli_auth to be cleared, got %v", rr.Result().Cookies())
}

func TestAuthHandler_Logout_RevokesTokenServerSide(t *testing.T) {
	t.Setenv("JWT_SECRET", logoutTestSecret)
	rv := &fakeRevoker{}
	h := handler.NewAuthHandler(&mockAuthUsecase{}).WithTokenRevoker(rv)
	userID := uuid.New()
	exp := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	tok := signLogoutToken(t, userID.String(), exp)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: tok})
	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	assertCookieCleared(t, rr)
	if rv.hash != appMiddleware.TokenHash(tok) {
		t.Errorf("revoked wrong hash: %q", rv.hash)
	}
	if rv.userID == nil || *rv.userID != userID {
		t.Errorf("expected user id %s, got %v", userID, rv.userID)
	}
	if !rv.exp.Equal(exp) {
		t.Errorf("expected exp %v, got %v", exp, rv.exp)
	}
}

func TestAuthHandler_Logout_RevokeFailureReturns500(t *testing.T) {
	t.Setenv("JWT_SECRET", logoutTestSecret)
	h := handler.NewAuthHandler(&mockAuthUsecase{}).WithTokenRevoker(&fakeRevoker{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+signLogoutToken(t, uuid.NewString(), time.Now().Add(time.Hour)))
	rr := httptest.NewRecorder()
	h.Logout(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when revocation fails, got %d", rr.Code)
	}
	assertCookieCleared(t, rr)
}

// A dead (expired/garbage) cookie must still be cleared, without a revoke call,
// so the browser is never stuck with an unusable httpOnly session.
func TestAuthHandler_Logout_DeadCookieStillCleared(t *testing.T) {
	t.Setenv("JWT_SECRET", logoutTestSecret)
	for name, tok := range map[string]string{
		"expired": signLogoutToken(t, uuid.NewString(), time.Now().Add(-time.Minute)),
		"garbage": "not-a-jwt",
	} {
		t.Run(name, func(t *testing.T) {
			rv := &fakeRevoker{}
			h := handler.NewAuthHandler(&mockAuthUsecase{}).WithTokenRevoker(rv)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "tayooli_auth", Value: tok})
			rr := httptest.NewRecorder()
			h.Logout(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
			assertCookieCleared(t, rr)
			if rv.hash != "" {
				t.Errorf("dead token must not be written to revocation store")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Register tests
// ---------------------------------------------------------------------------

func TestAuthHandler_Register_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		getUserByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
		registerFn: func(_ context.Context, fullName, email, password string) (*domain.User, error) {
			return &domain.User{
				ID:       userID,
				TenantID: tenantID,
				Email:    email,
				FullName: fullName,
				Role:     "owner",
			}, nil
		},
		issueTokenFn: func(_ *domain.User) (string, error) {
			return "reg-jwt-token", nil
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"full_name":"Budi Santoso","email":"budi@perusahaan.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify cookie
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header, got none")
	}
	if cookies[0].Name != "tayooli_auth" || cookies[0].Value != "reg-jwt-token" {
		t.Errorf("cookie mismatch: %v", cookies[0])
	}

	// Verify response body
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["token"] != "reg-jwt-token" {
		t.Errorf("expected token reg-jwt-token, got %v", resp["token"])
	}
	user, ok := resp["user"].(map[string]any)
	if !ok || user["email"] != "budi@perusahaan.com" || user["role"] != "owner" {
		t.Errorf("unexpected user object: %v", user)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	uc := &mockAuthUsecase{
		getUserByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:    uuid.New(),
				Email: email,
			}, nil
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"full_name":"Budi Santoso","email":"budi@perusahaan.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["message"] != "email sudah terdaftar" {
		t.Errorf("expected error message 'email sudah terdaftar', got %v", resp)
	}
}

func TestAuthHandler_Register_WeakPassword(t *testing.T) {
	uc := &mockAuthUsecase{
		getUserByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"full_name":"Budi Santoso","email":"budi@perusahaan.com","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["message"] != "password must be at least 8 characters" {
		t.Errorf("expected error message 'password must be at least 8 characters', got %v", resp)
	}
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	uc := &mockAuthUsecase{
		getUserByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(uc)

	tests := []struct {
		name string
		body string
	}{
		{"empty json", `{}`},
		{"missing password", `{"full_name":"Budi","email":"budi@perusahaan.com"}`},
		{"missing email", `{"full_name":"Budi","password":"password123"}`},
		{"missing full name", `{"email":"budi@perusahaan.com","password":"password123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.Register(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d: %s", tt.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestAuthHandler_Register_ShortFullName(t *testing.T) {
	uc := &mockAuthUsecase{
		getUserByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"full_name":"A","email":"budi@perusahaan.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["message"] != "full name must be at least 2 characters" {
		t.Errorf("expected error message 'full name must be at least 2 characters', got %v", resp)
	}
}

// ---------------------------------------------------------------------------
// VerifyEmail tests
// ---------------------------------------------------------------------------

func TestAuthHandler_VerifyEmail_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		verifyEmailFn: func(_ context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:       userID,
				TenantID: tenantID,
				Email:    "budi@perusahaan.com",
				Role:     "owner",
			}, nil
		},
		issueTokenFn: func(_ *domain.User) (string, error) {
			return "verified-token", nil
		},
	}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=valid-token", nil)
	rr := httptest.NewRecorder()

	h.VerifyEmail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value != "verified-token" {
		t.Fatalf("expected tayooli_auth cookie with verified-token, got %v", cookies)
	}
}

// ---------------------------------------------------------------------------
// CreateWorkspace tests
// ---------------------------------------------------------------------------

func TestAuthHandler_CreateWorkspace_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	var updatedTenantID uuid.UUID
	var updatedName string
	uc := &mockAuthUsecase{
		updateWorkspaceNameFn: func(_ context.Context, tid uuid.UUID, name string) error {
			updatedTenantID = tid
			updatedName = name
			return nil
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"company_name":"PT Maju Jaya"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.CreateWorkspace(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if updatedTenantID != tenantID {
		t.Errorf("expected tenantID %v, got %v", tenantID, updatedTenantID)
	}
	if updatedName != "PT Maju Jaya" {
		t.Errorf("expected PT Maju Jaya, got %s", updatedName)
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["success"] != true || resp["company_name"] != "PT Maju Jaya" {
		t.Errorf("unexpected response body: %v", resp)
	}
}

func TestAuthHandler_CreateWorkspace_Unauthorized(t *testing.T) {
	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	body := `{"company_name":"PT Maju Jaya"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateWorkspace(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthHandler_CreateWorkspace_InvalidBody(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{}
	h := handler.NewAuthHandler(uc)

	body := `{"company_name":" "}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.CreateWorkspace(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAuthHandler_GetCompanyProfile_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	uc := &mockAuthUsecase{
		getCompanyProfileFn: func(ctx context.Context, tid uuid.UUID) (*domain.TenantProfile, error) {
			return &domain.TenantProfile{
				TenantID: tid,
				Name:     "PT Sumber Rezeki",
				Address:  "Jl. Hayam Wuruk 10",
				Phone:    "08123456789",
				Email:    "halo@sumberrezeki.id",
			}, nil
		},
	}
	h := handler.NewAuthHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/profile", nil)
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.GetCompanyProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok || data["name"] != "PT Sumber Rezeki" {
		t.Errorf("unexpected profile data: %v", resp)
	}
}

func TestAuthHandler_UpdateCompanyProfile_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	var updatedProfile domain.TenantProfile
	uc := &mockAuthUsecase{
		updateCompanyProfileFn: func(ctx context.Context, tid uuid.UUID, p domain.TenantProfile) error {
			updatedProfile = p
			return nil
		},
	}
	h := handler.NewAuthHandler(uc)

	body := `{"company_name":"PT Makmur Abadi","address":"Jl. Thamrin 88","phone":"021-999888","email":"support@makmur.com","tax_id":"09.876.543.2-100.000","division":"Logistik Sentral"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuthContext(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.UpdateCompanyProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if updatedProfile.Name != "PT Makmur Abadi" || updatedProfile.Address != "Jl. Thamrin 88" {
		t.Errorf("profile not passed correctly to usecase: %+v", updatedProfile)
	}
}
