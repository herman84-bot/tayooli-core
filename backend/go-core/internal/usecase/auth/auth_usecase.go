package auth

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// UserReader abstracts the user repository for testability.
type UserReader interface {
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	GetByID(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error)
	// GetByEmailGlobal looks up a user by email across all tenants.
	// Used only by the login flow where no tenant context exists yet.
	GetByEmailGlobal(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	CreateTenantAndUser(ctx context.Context, tenant *domain.Tenant, user *domain.User) error
	UpdateTenantName(ctx context.Context, tenantID uuid.UUID, name string) error
	GetUserByVerificationToken(ctx context.Context, token string) (*domain.User, error)
	UpdateUserVerification(ctx context.Context, userID, tenantID uuid.UUID, verifiedAt time.Time) error
	ClearVerificationToken(ctx context.Context, userID, tenantID uuid.UUID) error
	UpdateVerificationToken(ctx context.Context, userID, tenantID uuid.UUID, token string, expiresAt time.Time) error
	GetUserByResetToken(ctx context.Context, token string) (*domain.User, error)
	UpdatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// PasswordVerifier checks if a plaintext password matches a hash.
type PasswordVerifier interface {
	Verify(hash, password string) bool
}

// EmailSender sends transactional emails (verification, password reset).
// Implemented by infra/mailer.Mailer. Nil means no email sending (test mode).
type EmailSender interface {
	SendPasswordResetEmail(toEmail, resetToken string) error
	SendVerificationEmail(toEmail, verificationToken string) error
}

// DefaultPasswordVerifier uses bcrypt via the postgres package.
type DefaultPasswordVerifier struct{}

func (DefaultPasswordVerifier) Verify(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Usecase handles authentication operations.
type Usecase struct {
	userRepo         UserReader
	passwordVerifier PasswordVerifier
	mailer           EmailSender
	jwtSecret        string
	tokenTTL         time.Duration
}

// New creates a new AuthUsecase. The mailer parameter is optional — pass nil
// to disable email sending (useful for tests and environments without SMTP).
func New(userRepo UserReader, jwtSecret string, tokenTTL time.Duration, pv PasswordVerifier, mailer EmailSender) *Usecase {
	if pv == nil {
		pv = DefaultPasswordVerifier{}
	}
	return &Usecase{
		userRepo:         userRepo,
		passwordVerifier: pv,
		mailer:           mailer,
		jwtSecret:        jwtSecret,
		tokenTTL:         tokenTTL,
	}
}

// Login authenticates a user by email/password and returns a JWT token.
func (u *Usecase) Login(ctx context.Context, tenantID uuid.UUID, email, password string) (string, *domain.User, error) {
	user, err := u.userRepo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}
	if !u.passwordVerifier.Verify(user.PasswordHash, password) {
		return "", nil, domain.ErrInvalidCredentials
	}
	token, err := u.generateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// LoginByEmail authenticates a user by email/password without requiring
// a tenant_id from context. Used by the login endpoint which runs before
// TenantMiddleware. The tenant_id is extracted from the DB record.
func (u *Usecase) LoginByEmail(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := u.userRepo.GetByEmailGlobal(ctx, email)
	if err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}
	if !u.passwordVerifier.Verify(user.PasswordHash, password) {
		return "", nil, domain.ErrInvalidCredentials
	}
	token, err := u.generateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// GetMe returns the current user by ID and tenant.
func (u *Usecase) GetMe(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, userID, tenantID)
}

func (u *Usecase) generateToken(userID, tenantID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"tenant_id": tenantID.String(),
		"role":      role,
		"exp":       time.Now().Add(u.tokenTTL).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(u.jwtSecret))
}

// IssueToken generates a JWT token for the given user.
func (u *Usecase) IssueToken(user *domain.User) (string, error) {
	return u.generateToken(user.ID, user.TenantID, user.Role)
}

// Register creates a new user account with unverified status.
func (u *Usecase) Register(ctx context.Context, fullName, email, password string) (*domain.User, error) {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	// Generate verification token
	verifToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)
	now := time.Now()

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:        tenantID,
		Name:      fullName,
		Plan:      "starter",
		CreatedAt: now,
		UpdatedAt: now,
	}

	user := &domain.User{
		ID:                    uuid.New(),
		TenantID:              tenantID,
		Email:                 email,
		FullName:              fullName,
		PasswordHash:          string(hash),
		Role:                  "owner",
		VerificationToken:     &verifToken,
		VerificationExpiresAt: &expiresAt,
		EmailVerifiedAt:       &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	// Provision tenant and user together
	if err := u.userRepo.CreateTenantAndUser(ctx, tenant, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateWorkspaceName updates the tenant's company name.
func (u *Usecase) UpdateWorkspaceName(ctx context.Context, tenantID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidInput
	}
	return u.userRepo.UpdateTenantName(ctx, tenantID, name)
}

// VerifyEmail verifies a user's email address using the verification token.
func (u *Usecase) VerifyEmail(ctx context.Context, token string) (*domain.User, error) {
	user, err := u.userRepo.GetUserByVerificationToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Update verified_at
	now := time.Now()
	if err := u.userRepo.UpdateUserVerification(ctx, user.ID, user.TenantID, now); err != nil {
		return nil, err
	}

	// Clear verification token
	_ = u.userRepo.ClearVerificationToken(ctx, user.ID, user.TenantID)

	user.EmailVerifiedAt = &now
	return user, nil
}

// ResendVerification regenerates a verification token and sends a new email.
func (u *Usecase) ResendVerification(ctx context.Context, email string) error {
	user, err := u.userRepo.GetByEmailGlobal(ctx, email)
	if err != nil {
		log.Printf("[AUTH] resend-verification requested for non-existent email: %s", email)
		// Don't reveal if email exists
		return nil
	}

	// Generate new token
	verifToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	if err := u.userRepo.UpdateVerificationToken(ctx, user.ID, user.TenantID, verifToken, expiresAt); err != nil {
		return err
	}

	// Send verification email asynchronously (non-blocking).
	if u.mailer != nil {
		go func() {
			if err := u.mailer.SendVerificationEmail(user.Email, verifToken); err != nil {
				log.Printf("[AUTH] failed to send verification email to %s: %v", user.Email, err)
			} else {
				log.Printf("[AUTH] verification email successfully dispatched to %s", user.Email)
			}
		}()
	}
	return nil
}

// GetUserByEmail returns a user by email (for duplicate check).
func (u *Usecase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return u.userRepo.GetByEmailGlobal(ctx, email)
}

// ForgotPassword generates a reset token and sends an email.
func (u *Usecase) ForgotPassword(ctx context.Context, email string) error {
	user, err := u.userRepo.GetByEmailGlobal(ctx, email)
	if err != nil {
		log.Printf("[AUTH] forgot-password requested for non-existent email: %s", email)
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token
	resetToken := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour) // Token expires in 1 hour

	if err := u.userRepo.UpdatePasswordResetToken(ctx, user.ID, resetToken, expiresAt); err != nil {
		log.Printf("[AUTH] failed to store password reset token for %s: %v", user.Email, err)
		return err
	}

	// Send reset email asynchronously (non-blocking).
	if u.mailer != nil {
		go func() {
			if err := u.mailer.SendPasswordResetEmail(user.Email, resetToken); err != nil {
				log.Printf("[AUTH] failed to send password reset email to %s: %v", user.Email, err)
			} else {
				log.Printf("[AUTH] password reset email successfully dispatched to %s", user.Email)
			}
		}()
	}
	return nil
}

// ResetPassword resets the user's password using the reset token.
func (u *Usecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	user, err := u.userRepo.GetUserByResetToken(ctx, token)
	if err != nil {
		return domain.ErrInvalidToken
	}

	// Check if token is expired
	if user.PasswordResetExpiresAt != nil && user.PasswordResetExpiresAt.Before(time.Now()) {
		return domain.ErrTokenExpired
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// Update password and clear reset token
	if err := u.userRepo.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
		return err
	}

	return nil
}
