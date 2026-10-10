package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// authUsecase abstracts the operations the auth handler needs for testability.
type authUsecase interface {
	LoginByEmail(ctx context.Context, email, password string) (string, *domain.User, error)
	GetMe(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error)
	Register(ctx context.Context, fullName, email, password string) (*domain.User, error)
	IssueToken(user *domain.User) (string, error)
	UpdateWorkspaceName(ctx context.Context, tenantID uuid.UUID, name string) error
	VerifyEmail(ctx context.Context, token string) (*domain.User, error)
	ResendVerification(ctx context.Context, email string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
}

// tokenRevoker persists a logged-out token so it is rejected server-side.
type tokenRevoker interface {
	Revoke(ctx context.Context, tokenHash string, userID *uuid.UUID, expiresAt time.Time) error
}

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	uc      authUsecase
	revoker tokenRevoker
}

// WithTokenRevoker enables server-side revocation on logout.
func (h *AuthHandler) WithTokenRevoker(rv tokenRevoker) *AuthHandler {
	h.revoker = rv
	return h
}

// NewAuthHandler returns a new AuthHandler.
func NewAuthHandler(uc authUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// isSecureCookie returns true when the cookie Secure flag should be set.
// In production (APP_ENV=production) the flag is true; in development it is false
// so that cookies work over plain HTTP on localhost.
func isSecureCookie() bool {
	return os.Getenv("APP_ENV") == "production"
}

// normalizeEmail trims whitespace, lowercases and validates an email address.
// All accounts created via the API are stored with this canonical form so the
// same address cannot be registered twice with different casing/spacing.
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", errors.New("email is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || parsed.Name != "" {
		return "", errors.New("format email tidak valid")
	}
	// Require a real domain (contains a dot), mirroring HTML/zod email rules.
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || !strings.Contains(email[at+1:], ".") {
		return "", errors.New("format email tidak valid")
	}
	return email, nil
}

// loginRequest is the JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login.
// Authenticates the user via email/password, sets an HttpOnly cookie with the JWT token.
// This endpoint is registered BEFORE TenantMiddleware, so it cannot rely on tenant_id
// being in the context. Instead, the tenant_id is resolved from the DB via LoginByEmail.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if req.Password == "" {
		respondError(w, r, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, r, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	token, user, err := h.uc.LoginByEmail(r.Context(), email, req.Password)
	// Backward compatibility: accounts created before email normalization may
	// be stored with the original casing. Retry once with the exact typed
	// (trimmed) address before giving up.
	if err != nil {
		original := strings.TrimSpace(req.Email)
		if original != email {
			token, user, err = h.uc.LoginByEmail(r.Context(), original, req.Password)
		}
	}
	if err != nil {
		respondError(w, r, http.StatusUnauthorized, "invalid credentials")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tayooli_auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour.Seconds()),
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":    user.ID.String(),
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// Me handles GET /api/v1/auth/me.
// Returns the current authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := appMiddleware.GetUserID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.uc.GetMe(r.Context(), userID, tenantID)
	if err != nil {
		respondError(w, r, http.StatusNotFound, "user not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":    user.ID.String(),
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// Logout handles POST /api/v1/auth/logout.
// Registered OUTSIDE TenantMiddleware so an expired/revoked/garbage cookie can
// always be cleared (otherwise the middleware 401s first and the dead httpOnly
// cookie sticks). The cookie is cleared on EVERY outcome. A still-valid token
// is revoked server-side; if that revoke fails the response is 500 so the
// client never believes a live token was invalidated.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "tayooli_auth",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // emits Max-Age=0: delete now
	})

	if h.revoker != nil {
		if tok := appMiddleware.ExtractToken(r); tok != "" {
			// Only a token that verifies (signature + exp) is worth revoking;
			// invalid/expired ones are already unusable.
			if claims, ok := appMiddleware.VerifyToken(tok); ok {
				var uidPtr *uuid.UUID
				if uid, err := uuid.Parse(claims.Subject); err == nil {
					uidPtr = &uid
				}
				if err := h.revoker.Revoke(r.Context(), appMiddleware.TokenHash(tok), uidPtr, claims.ExpiresAt); err != nil {
					log.Error().Err(err).Msg("logout: revoke token failed")
					respondError(w, r, http.StatusInternalServerError, "logout failed")
					return
				}
			}
		}
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// registerRequest is the JSON body for POST /api/v1/auth/register.
type registerRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register.
// Creates a new user account and sends verification email.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	fullName := strings.TrimSpace(req.FullName)
	email, err := normalizeEmail(req.Email)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if fullName == "" || email == "" || req.Password == "" {
		respondError(w, r, http.StatusBadRequest, "full_name, email, and password are required")
		return
	}
	if len(fullName) < 2 {
		respondError(w, r, http.StatusBadRequest, "full name must be at least 2 characters")
		return
	}
	if len(fullName) > 255 {
		respondError(w, r, http.StatusBadRequest, "full name must not exceed 255 characters")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, r, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if len(req.Password) > 72 {
		respondError(w, r, http.StatusBadRequest, "password must not exceed 72 characters")
		return
	}

	// Check if email already exists (normalized — prevents duplicate accounts
	// differing only by case or surrounding whitespace)
	existing, _ := h.uc.GetUserByEmail(r.Context(), email)
	if existing != nil {
		respondError(w, r, http.StatusConflict, "email sudah terdaftar")
		return
	}

	user, err := h.uc.Register(r.Context(), fullName, email, req.Password)
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("failed to register user")
		respondError(w, r, http.StatusInternalServerError, "gagal membuat akun")
		return
	}

	token, err := h.uc.IssueToken(user)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "gagal membuat token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tayooli_auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour.Seconds()),
	})

	respondJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user": map[string]string{
			"id":        user.ID.String(),
			"tenant_id": user.TenantID.String(),
			"email":     user.Email,
			"role":      user.Role,
		},
	})
}

// VerifyEmail handles GET /api/v1/auth/verify-email?token=xxx.
// Verifies the user's email address and sets auth cookie.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	user, err := h.uc.VerifyEmail(r.Context(), token)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "token tidak valid atau sudah kedaluwarsa")
		return
	}

	// Generate JWT and set cookie
	tokenStr, err := h.uc.IssueToken(user)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]string{"message": "email terverifikasi"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tayooli_auth",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour.Seconds()),
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "email terverifikasi"})
}

// ResendVerification handles POST /api/v1/auth/resend-verification.
// Resends the verification email.
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = h.uc.ResendVerification(r.Context(), email)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "gagal mengirim ulang")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "link verifikasi dikirim ulang"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
// Sends a password reset email if the email exists.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = h.uc.ForgotPassword(r.Context(), email)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "gagal mengirim email reset password")
		return
	}

	// Always return success to prevent email enumeration
	respondJSON(w, http.StatusOK, map[string]string{"message": "link reset password telah dikirim"})
}

// ResetPassword handles POST /api/v1/auth/reset-password.
// Resets the user's password using the reset token.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		respondError(w, r, http.StatusBadRequest, "token and password are required")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, r, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	err := h.uc.ResetPassword(r.Context(), req.Token, req.Password)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "token tidak valid atau sudah kedaluwarsa")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "password berhasil direset"})
}

type createWorkspaceRequest struct {
	CompanyName string `json:"company_name"`
}

// CreateWorkspace handles POST /api/v1/workspaces.
// Updates the tenant's company name and completes workspace setup.
func (h *AuthHandler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req createWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	companyName := strings.TrimSpace(req.CompanyName)
	if len(companyName) < 2 || len(companyName) > 255 {
		respondError(w, r, http.StatusBadRequest, "company name must be between 2 and 255 characters")
		return
	}

	if err := h.uc.UpdateWorkspaceName(r.Context(), tenantID, companyName); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "workspace tidak ditemukan")
			return
		}
		respondError(w, r, http.StatusInternalServerError, "gagal memperbarui workspace")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"company_name": companyName,
	})
}

// UpdateCompanyProfile handles PATCH /api/v1/settings/profile.
func (h *AuthHandler) UpdateCompanyProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		CompanyName string `json:"company_name"`
		Address     string `json:"address"`
		TaxID       string `json:"tax_id"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	companyName := strings.TrimSpace(req.CompanyName)
	if len(companyName) < 2 || len(companyName) > 255 {
		respondError(w, r, http.StatusBadRequest, "company name must be between 2 and 255 characters")
		return
	}

	low := strings.ToLower(companyName)
	if low == "perusahaan" || low == "pt" || low == "baru" {
		respondError(w, r, http.StatusBadRequest, "invalid generic company name")
		return
	}

	if err := h.uc.UpdateWorkspaceName(r.Context(), tenantID, companyName); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "workspace tidak ditemukan")
			return
		}
		respondError(w, r, http.StatusInternalServerError, "gagal memperbarui profil perusahaan")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"company_name": companyName,
		"address":      req.Address,
		"tax_id":       req.TaxID,
		"currency":     req.Currency,
	})
}

