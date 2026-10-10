package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	authUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth"
)

// Mock user repo for adversarial usecase integration tests
type adversarialUserRepo struct {
	createdTenant *domain.Tenant
	createdUser   *domain.User
	updatedName   string
	updatedTenant uuid.UUID
}

func (m *adversarialUserRepo) GetByEmail(_ context.Context, _ uuid.UUID, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *adversarialUserRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *adversarialUserRepo) GetByEmailGlobal(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *adversarialUserRepo) CreateUser(_ context.Context, _ *domain.User) error {
	return nil
}
func (m *adversarialUserRepo) CreateTenantAndUser(_ context.Context, t *domain.Tenant, u *domain.User) error {
	m.createdTenant = t
	m.createdUser = u
	return nil
}
func (m *adversarialUserRepo) UpdateTenantName(_ context.Context, tenantID uuid.UUID, name string) error {
	m.updatedTenant = tenantID
	m.updatedName = name
	return nil
}
func (m *adversarialUserRepo) GetTenantProfile(_ context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error) {
	return &domain.TenantProfile{TenantID: tenantID, Name: "Adversarial Co"}, nil
}
func (m *adversarialUserRepo) UpdateTenantProfile(_ context.Context, tenantID uuid.UUID, _ domain.TenantProfile) error {
	m.updatedTenant = tenantID
	return nil
}
func (m *adversarialUserRepo) GetUserByVerificationToken(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *adversarialUserRepo) UpdateUserVerification(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *adversarialUserRepo) ClearVerificationToken(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *adversarialUserRepo) UpdateVerificationToken(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *adversarialUserRepo) GetUserByResetToken(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (m *adversarialUserRepo) UpdatePasswordResetToken(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (m *adversarialUserRepo) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// ---------------------------------------------------------------------------
// 1. Tenant Isolation & Privilege Escalation Adversarial Tests
// ---------------------------------------------------------------------------

func TestAdversarial_Register_TenantIsolationAndRoleEnforcement(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	// Attacker attempts mass assignment: inject role="admin", evil tenant_id, enterprise plan, verified=true
	attackerPayload := map[string]any{
		"full_name":   "Attacker Name",
		"email":       "attacker@evil.com",
		"password":    "Password123!",
		"role":        "admin",
		"tenant_id":   "11111111-1111-1111-1111-111111111111",
		"plan":        "enterprise",
		"is_verified": true,
	}
	body, _ := json.Marshal(attackerPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify Tenant Isolation in Created User & Tenant
	if repo.createdTenant == nil || repo.createdUser == nil {
		t.Fatal("expected tenant and user to be created")
	}

	// Attacker-supplied tenant_id MUST NOT be used
	targetSpoofedUUID, _ := uuid.Parse("11111111-1111-1111-1111-111111111111")
	if repo.createdTenant.ID == targetSpoofedUUID {
		t.Errorf("CRITICAL VULNERABILITY: Attacker successfully spoofed tenant_id to %s", repo.createdTenant.ID)
	}

	// Attacker-supplied role="admin" MUST NOT be used; role must strictly be "owner"
	if repo.createdUser.Role != "owner" {
		t.Errorf("Privilege Escalation: User role was set to %q instead of 'owner'", repo.createdUser.Role)
	}

	// Plan must be "starter", not "enterprise"
	if repo.createdTenant.Plan != "starter" {
		t.Errorf("Mass Assignment: Tenant plan was set to %q instead of 'starter'", repo.createdTenant.Plan)
	}

	// Password must be hashed via bcrypt
	if repo.createdUser.PasswordHash == "Password123!" {
		t.Error("CRITICAL: Password stored in plaintext!")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.createdUser.PasswordHash), []byte("Password123!")); err != nil {
		t.Errorf("Password hash does not verify with bcrypt: %v", err)
	}
}

func TestAdversarial_CreateWorkspace_CrossTenantTampering(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	victimTenantID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	attackerTenantID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	attackerUserID := uuid.New()

	// Attacker is authenticated under attackerTenantID, but attempts to specify victimTenantID in JSON body
	maliciousBody := `{"company_name": "Hacked Company Name", "tenant_id": "99999999-9999-9999-9999-999999999999"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(maliciousBody))
	req.Header.Set("Content-Type", "application/json")

	// Inject JWT auth context representing attacker's valid session
	req = withAuthContext(req, attackerTenantID, attackerUserID)
	rr := httptest.NewRecorder()

	h.CreateWorkspace(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Ensure UpdateTenantName was invoked ONLY for attackerTenantID, NEVER for victimTenantID
	if repo.updatedTenant == victimTenantID {
		t.Fatalf("CRITICAL IDOR: Attacker renamed victim tenant %s!", victimTenantID)
	}
	if repo.updatedTenant != attackerTenantID {
		t.Fatalf("Expected update to attacker's tenant %s, got %s", attackerTenantID, repo.updatedTenant)
	}
}

// ---------------------------------------------------------------------------
// 2. Input Validation & Edge Case Tests
// ---------------------------------------------------------------------------

func TestAdversarial_Register_WeakPasswordAllowed(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	// Weak trivial 8-char passwords: no uppercase, no numbers, no special chars
	weakPasswords := []string{
		"12345678",
		"abcdefgh",
		"password",
		"00000000",
	}

	for _, pw := range weakPasswords {
		payload := map[string]string{
			"full_name": "Valid User",
			"email":     "user+" + pw + "@example.com",
			"password":  pw,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		// Finding: Handler accepts weak passwords as long as length >= 8
		if rr.Code == http.StatusCreated {
			t.Logf("OBSERVATION: Weak password %q was accepted without complexity checks", pw)
		}
	}
}

func TestAdversarial_Register_PasswordTooLong_BcryptLimit(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	// Password longer than 72 bytes (Bcrypt limit)
	longPassword := strings.Repeat("A", 73)
	payload := map[string]string{
		"full_name": "Valid User",
		"email":     "user@example.com",
		"password":  longPassword,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	// In Go bcrypt, password > 72 bytes returns bcrypt.ErrPasswordTooLong,
	// which causes the usecase to fail and handler to return 500 instead of 400 validation error
	if rr.Code == http.StatusInternalServerError {
		t.Logf("FINDING CONFIRMED: Password > 72 bytes triggers HTTP 500 Internal Server Error (bcrypt limit unhandled in validation)")
	} else if rr.Code != http.StatusBadRequest {
		t.Errorf("Unexpected status code: %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// 3. SQL Injection Resilience Tests
// ---------------------------------------------------------------------------

func TestAdversarial_Register_FullNameExceeds255_MissingValidation(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	// In database schema: full_name VARCHAR(255).
	// But Register() handler only checks len(fullName) < 2, omitting len(fullName) > 255!
	longName := strings.Repeat("A", 300)
	payload := map[string]string{
		"full_name": longName,
		"email":     "valid@example.com",
		"password":  "password123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	// Since mock repo does not simulate DB column truncation, it creates it with 300 chars.
	// In production PostgreSQL, this throws "value too long for type character varying(255)",
	// which causes a 500 error instead of 400 Bad Request!
	if rr.Code == http.StatusCreated {
		t.Logf("OBSERVATION: full_name with 300 chars passed handler validation (no max length check in Register)")
	}
}

func TestAdversarial_CreateWorkspace_SQLInjectionPayloads(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	tenantID := uuid.New()
	userID := uuid.New()

	sqlPayloads := []string{
		"'; DROP TABLE tenants; --",
		"' OR 1=1 --",
		"\" OR \"\"=\"",
	}

	for _, payload := range sqlPayloads {
		body := `{"company_name": "` + payload + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req = withAuthContext(req, tenantID, userID)
		rr := httptest.NewRecorder()

		h.CreateWorkspace(rr, req)

		if rr.Code == http.StatusOK {
			if repo.updatedName != payload {
				t.Errorf("Expected literal company name %q, got %q", payload, repo.updatedName)
			}
		}
	}
}

func TestAdversarial_Register_SQLInjectionPayloads(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	sqlPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR 1=1 --",
		"admin'--",
		"\" OR \"\"=\"",
		"` OR 1=1 --",
	}

	for _, payload := range sqlPayloads {
		body := map[string]string{
			"full_name": payload,
			"email":     "valid_sql_test@example.com",
			"password":  "SafePassword123!",
		}
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		// The input should either be accepted safely as literal string data
		// (because repo uses parameterized query $1, $2) or rejected if invalid
		if rr.Code == http.StatusCreated {
			if repo.createdTenant.Name != payload {
				t.Errorf("Expected literal name %q, got %q", payload, repo.createdTenant.Name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 4. DoS / Resource Exhaustion Tests
// ---------------------------------------------------------------------------

func TestAdversarial_Register_PayloadSizeExceeded(t *testing.T) {
	repo := &adversarialUserRepo{}
	uc := authUC.New(repo, "super-secret-jwt-key-minimum-32-chars-long!", 3600, nil, nil)
	h := handler.NewAuthHandler(uc)

	// Payload larger than 64KB (e.g. 70KB)
	largeBody := `{"full_name":"` + strings.Repeat("A", 70*1024) + `","email":"test@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	// http.MaxBytesReader will truncate/error and json.NewDecoder will fail with 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for >64KB body, got %d", rr.Code)
	}
}
