package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// --- Mocks ---

type mockUserRepo struct {
	users   map[string]*domain.User // key: tenantID/email
	byID    map[uuid.UUID]*domain.User
	tenants map[uuid.UUID]*domain.Tenant
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
		tenants: make(map[uuid.UUID]*domain.Tenant),
	}
}

func (m *mockUserRepo) addUser(u *domain.User) {
	key := u.TenantID.String() + "/" + u.Email
	m.users[key] = u
	m.byID[u.ID] = u
}

func (m *mockUserRepo) GetByEmail(_ context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	key := tenantID.String() + "/" + email
	u, ok := m.users[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, userID, tenantID uuid.UUID) (*domain.User, error) {
	u, ok := m.byID[userID]
	if !ok || u.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmailGlobal(_ context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) CreateUser(_ context.Context, u *domain.User) error {
	m.addUser(u)
	return nil
}

func (m *mockUserRepo) CreateTenantAndUser(_ context.Context, tenant *domain.Tenant, user *domain.User) error {
	m.tenants[tenant.ID] = tenant
	m.addUser(user)
	return nil
}

func (m *mockUserRepo) UpdateTenantName(_ context.Context, tenantID uuid.UUID, name string) error {
	t, ok := m.tenants[tenantID]
	if !ok {
		return domain.ErrNotFound
	}
	t.Name = name
	return nil
}

func (m *mockUserRepo) GetUserByVerificationToken(_ context.Context, token string) (*domain.User, error) {
	for _, u := range m.users {
		if u.VerificationToken != nil && *u.VerificationToken == token {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) UpdateUserVerification(_ context.Context, userID, _ uuid.UUID, _ time.Time) error {
	if u, ok := m.byID[userID]; ok {
		now := time.Now()
		u.EmailVerifiedAt = &now
	}
	return nil
}

func (m *mockUserRepo) ClearVerificationToken(_ context.Context, userID, _ uuid.UUID) error {
	if u, ok := m.byID[userID]; ok {
		u.VerificationToken = nil
		u.VerificationExpiresAt = nil
	}
	return nil
}

func (m *mockUserRepo) UpdateVerificationToken(_ context.Context, userID, _ uuid.UUID, token string, expiresAt time.Time) error {
	if u, ok := m.byID[userID]; ok {
		u.VerificationToken = &token
		u.VerificationExpiresAt = &expiresAt
	}
	return nil
}

func (m *mockUserRepo) UpdatePasswordResetToken(_ context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	if u, ok := m.byID[userID]; ok {
		u.PasswordResetToken = &token
		u.PasswordResetExpiresAt = &expiresAt
	}
	return nil
}

func (m *mockUserRepo) GetUserByResetToken(_ context.Context, token string) (*domain.User, error) {
	for _, u := range m.users {
		if u.PasswordResetToken != nil && *u.PasswordResetToken == token {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string) error {
	if u, ok := m.byID[userID]; ok {
		u.PasswordHash = passwordHash
		u.PasswordResetToken = nil
		u.PasswordResetExpiresAt = nil
	}
	return nil
}

type mockPasswordVerifier struct {
	result bool
}

func (m *mockPasswordVerifier) Verify(_, _ string) bool {
	return m.result
}

// --- Helpers ---

func seedTestUser(t *testing.T) (*domain.User, string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	u := &domain.User{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Email:        "admin@test.com",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	return u, string(hash)
}

// --- Tests ---

func TestAuthUsecase_Login_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil) // nil pv uses DefaultPasswordVerifier, nil mailer skips email

	token, result, err := uc.Login(context.Background(), user.TenantID, user.Email, "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Email, result.Email)
	assert.Equal(t, user.Role, result.Role)

	// Verify token is valid JWT
	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID.String(), claims["sub"])
	assert.Equal(t, user.TenantID.String(), claims["tenant_id"])
	assert.Equal(t, "admin", claims["role"])
}

func TestAuthUsecase_Login_WrongPassword(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	_, _, err := uc.Login(context.Background(), user.TenantID, user.Email, "wrongpassword")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_Login_NonExistentUser(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, _, err := uc.Login(context.Background(), tenantID, "nobody@test.com", "password123")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_Login_WrongTenant(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	otherTenant := uuid.New()
	_, _, err := uc.Login(context.Background(), otherTenant, user.Email, "password123")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_Login_RepoError(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, _, err := uc.Login(context.Background(), tenantID, "any@test.com", "pass")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_GetMe_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	result, err := uc.GetMe(context.Background(), user.ID, user.TenantID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Email, result.Email)
	assert.Equal(t, user.TenantID, result.TenantID)
	assert.Equal(t, user.Role, result.Role)
}

func TestAuthUsecase_GetMe_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, err := uc.GetMe(context.Background(), uuid.New(), tenantID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAuthUsecase_GetMe_WrongTenant(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	otherTenant := uuid.New()
	_, err := uc.GetMe(context.Background(), user.ID, otherTenant)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAuthUsecase_Login_MockPasswordVerifier(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	// Use a mock password verifier that always returns true
	pv := &mockPasswordVerifier{result: true}
	uc := New(repo, "test-secret", time.Hour, pv, nil)

	token, result, err := uc.Login(context.Background(), user.TenantID, user.Email, "anything")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, user.ID, result.ID)
}

func TestAuthUsecase_Login_MockPasswordVerifier_Fail(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	// Use a mock password verifier that always returns false
	pv := &mockPasswordVerifier{result: false}
	uc := New(repo, "test-secret", time.Hour, pv, nil)

	_, _, err := uc.Login(context.Background(), user.TenantID, user.Email, "password123")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_Login_TokenExpiry(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", 30*time.Minute, nil, nil)

	token, _, err := uc.Login(context.Background(), user.TenantID, user.Email, "password123")
	require.NoError(t, err)

	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	claims := parsed.Claims.(jwt.MapClaims)
	exp := int64(claims["exp"].(float64))
	iat := int64(claims["iat"].(float64))
	// Token TTL should be approximately 30 minutes
	assert.InDelta(t, 30*60, exp-iat, 5) // within 5 seconds tolerance
}

func TestAuthUsecase_Login_InvalidSecret(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	token, _, err := uc.Login(context.Background(), user.TenantID, user.Email, "password123")
	require.NoError(t, err)

	// Try to parse with wrong secret
	_, err = jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	assert.Error(t, err)
}

type alwaysErrorRepo struct{}

func (alwaysErrorRepo) GetByEmail(_ context.Context, _ uuid.UUID, _ string) (*domain.User, error) {
	return nil, errors.New("db connection failed")
}

func (alwaysErrorRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.User, error) {
	return nil, errors.New("db connection failed")
}

func (alwaysErrorRepo) GetByEmailGlobal(_ context.Context, _ string) (*domain.User, error) {
	return nil, errors.New("db connection failed")
}

func (alwaysErrorRepo) CreateUser(_ context.Context, _ *domain.User) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) CreateTenantAndUser(_ context.Context, _ *domain.Tenant, _ *domain.User) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) UpdateTenantName(_ context.Context, _ uuid.UUID, _ string) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) GetUserByVerificationToken(_ context.Context, _ string) (*domain.User, error) {
	return nil, errors.New("db connection failed")
}

func (alwaysErrorRepo) UpdateUserVerification(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) ClearVerificationToken(_ context.Context, _, _ uuid.UUID) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) UpdateVerificationToken(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) UpdatePasswordResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return errors.New("db connection failed")
}

func (alwaysErrorRepo) GetUserByResetToken(_ context.Context, _ string) (*domain.User, error) {
	return nil, errors.New("db connection failed")
}

func (alwaysErrorRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return errors.New("db connection failed")
}

func TestAuthUsecase_GetMe_RepoError(t *testing.T) {
	uc := New(alwaysErrorRepo{}, "test-secret", time.Hour, nil, nil)

	_, err := uc.GetMe(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestAuthUsecase_LoginByEmail_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	token, result, err := uc.LoginByEmail(context.Background(), user.Email, "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Email, result.Email)
	assert.Equal(t, user.TenantID, result.TenantID)

	// Verify token contains correct claims
	parsed, err := jwt.Parse(token, func(tok *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID.String(), claims["sub"])
	assert.Equal(t, user.TenantID.String(), claims["tenant_id"])
}

func TestAuthUsecase_LoginByEmail_WrongPassword(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	_, _, err := uc.LoginByEmail(context.Background(), user.Email, "wrongpassword")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_LoginByEmail_NonExistentUser(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	_, _, err := uc.LoginByEmail(context.Background(), "nobody@test.com", "password123")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_LoginByEmail_RepoError(t *testing.T) {
	uc := New(alwaysErrorRepo{}, "test-secret", time.Hour, nil, nil)

	_, _, err := uc.LoginByEmail(context.Background(), "any@test.com", "pass")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_VerifyEmail_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	token := "valid-token"
	user.VerificationToken = &token
	exp := time.Now().Add(time.Hour)
	user.VerificationExpiresAt = &exp

	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)
	verifiedUser, err := uc.VerifyEmail(context.Background(), token)
	assert.NoError(t, err)
	assert.NotNil(t, verifiedUser)
	assert.NotNil(t, verifiedUser.EmailVerifiedAt)
}

func TestAuthUsecase_ResendVerification_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	uc := New(repo, "test-secret", time.Hour, nil, nil)
	err := uc.ResendVerification(context.Background(), user.Email)
	assert.NoError(t, err)
}

func TestAuthUsecase_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	user, err := uc.Register(context.Background(), "John Doe", "john@example.com", "secret12345")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, "John Doe", user.FullName)
	assert.Equal(t, "owner", user.Role)
	assert.NotEqual(t, uuid.Nil, user.TenantID)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.NotNil(t, user.VerificationToken)

	// Check tenant created
	tenant, ok := repo.tenants[user.TenantID]
	require.True(t, ok)
	assert.Equal(t, "John Doe", tenant.Name)
	assert.Equal(t, "starter", tenant.Plan)
}

func TestAuthUsecase_Register_RepoError(t *testing.T) {
	uc := New(alwaysErrorRepo{}, "test-secret", time.Hour, nil, nil)

	_, err := uc.Register(context.Background(), "John Doe", "john@example.com", "secret12345")
	assert.Error(t, err)
}

func TestAuthUsecase_IssueToken(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	user := &domain.User{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Role:     "owner",
	}

	token, err := uc.IssueToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user.ID.String(), claims["sub"])
	assert.Equal(t, user.TenantID.String(), claims["tenant_id"])
	assert.Equal(t, "owner", claims["role"])
}

func TestAuthUsecase_UpdateWorkspaceName(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	tenantID := uuid.New()
	repo.tenants[tenantID] = &domain.Tenant{
		ID:   tenantID,
		Name: "Old Name",
	}

	err := uc.UpdateWorkspaceName(context.Background(), tenantID, "New Company Name")
	require.NoError(t, err)
	assert.Equal(t, "New Company Name", repo.tenants[tenantID].Name)

	// Blank name
	err = uc.UpdateWorkspaceName(context.Background(), tenantID, "   ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	// Non-existent tenant
	err = uc.UpdateWorkspaceName(context.Background(), uuid.New(), "Valid Name")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

type mockMailer struct {
	resetEmailSent    bool
	verifEmailSent    bool
	lastToEmail       string
	lastToken         string
}

func (m *mockMailer) SendPasswordResetEmail(toEmail, resetToken string) error {
	m.resetEmailSent = true
	m.lastToEmail = toEmail
	m.lastToken = resetToken
	return nil
}

func (m *mockMailer) SendVerificationEmail(toEmail, verificationToken string) error {
	m.verifEmailSent = true
	m.lastToEmail = toEmail
	m.lastToken = verificationToken
	return nil
}

func TestAuthUsecase_ForgotPassword_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	mailer := &mockMailer{}
	uc := New(repo, "test-secret", time.Hour, nil, mailer)

	err := uc.ForgotPassword(context.Background(), user.Email)
	require.NoError(t, err)

	// Sleep slightly to allow the goroutine to execute
	time.Sleep(50 * time.Millisecond)

	assert.True(t, mailer.resetEmailSent)
	assert.Equal(t, user.Email, mailer.lastToEmail)
	assert.NotEmpty(t, mailer.lastToken)

	// Check token was stored in repo
	assert.NotNil(t, user.PasswordResetToken)
	assert.Equal(t, mailer.lastToken, *user.PasswordResetToken)
	assert.NotNil(t, user.PasswordResetExpiresAt)
}

func TestAuthUsecase_ForgotPassword_NonExistentEmail(t *testing.T) {
	repo := newMockUserRepo()
	mailer := &mockMailer{}
	uc := New(repo, "test-secret", time.Hour, nil, mailer)

	// Should not error (anti-enumeration) but shouldn't send email
	err := uc.ForgotPassword(context.Background(), "nobody@example.com")
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.False(t, mailer.resetEmailSent)
}

func TestAuthUsecase_ResetPassword_Success(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	token := "valid-reset-token"
	expiresAt := time.Now().Add(time.Hour)
	user.PasswordResetToken = &token
	user.PasswordResetExpiresAt = &expiresAt

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	err := uc.ResetPassword(context.Background(), token, "newpassword123")
	require.NoError(t, err)

	// Token should be cleared
	assert.Nil(t, user.PasswordResetToken)
	assert.Nil(t, user.PasswordResetExpiresAt)

	// New password should be verifiable
	verifier := DefaultPasswordVerifier{}
	assert.True(t, verifier.Verify(user.PasswordHash, "newpassword123"))
}

func TestAuthUsecase_ResetPassword_ExpiredToken(t *testing.T) {
	user, _ := seedTestUser(t)
	repo := newMockUserRepo()
	repo.addUser(user)

	token := "expired-reset-token"
	expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	user.PasswordResetToken = &token
	user.PasswordResetExpiresAt = &expiresAt

	uc := New(repo, "test-secret", time.Hour, nil, nil)

	err := uc.ResetPassword(context.Background(), token, "newpassword123")
	assert.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestAuthUsecase_ResetPassword_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()
	uc := New(repo, "test-secret", time.Hour, nil, nil)

	err := uc.ResetPassword(context.Background(), "invalid-token", "newpassword123")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}

