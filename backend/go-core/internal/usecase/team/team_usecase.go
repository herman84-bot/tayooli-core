package team

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// AllowedInviteRoles lists valid roles that can be invited or assigned.
var AllowedInviteRoles = map[string]bool{
	domain.RoleAdmin:            true,
	domain.RoleWarehouseManager: true,
	"regional_manager":          true, // alias backward compatibility
	domain.RoleWarehouse:        true,
	domain.RoleCashier:          true,
	domain.RoleAuditor:          true,
	domain.RoleMember:           true,
	"accountant":                true, // backward compatibility
	"approver":                  true, // backward compatibility
}

// UserReader provides the user operations needed by team management.
type UserReader interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error)
	GetByID(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUserRole(ctx context.Context, userID, tenantID uuid.UUID, role string) error
	DeleteUser(ctx context.Context, userID, tenantID uuid.UUID) error
	CreateUserWithWarehouses(ctx context.Context, user *domain.User, warehouseIDs []uuid.UUID) error
	UpdateUserRoleAndWarehouses(ctx context.Context, userID, tenantID uuid.UUID, role string, warehouseIDs []uuid.UUID) error
	GetUserWarehouses(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.AssignedWarehouse, error)
}

// InviteTokenStore persists a one-time token that lets an invited user set
// their own password (reuses the password-reset token mechanism).
type InviteTokenStore interface {
	UpdatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
}

// InvitationSender delivers the invitation email.
type InvitationSender interface {
	SendInvitationEmail(toEmail, token string) error
	IsConfigured() bool
}

// InviteTokenTTL is how long an invitation link stays valid.
const InviteTokenTTL = 72 * time.Hour

// InviteResult reports the created member and whether the email was delivered.
type InviteResult struct {
	User       *domain.User
	EmailSent  bool
	EmailError string
}

// Usecase handles team management operations.
type Usecase struct {
	repo   UserReader
	tokens InviteTokenStore
	mailer InvitationSender
}

// WithInviter enables invitation emails. Without it, members are created but
// no email is sent (EmailSent=false is reported to the caller).
func (u *Usecase) WithInviter(tokens InviteTokenStore, mailer InvitationSender) *Usecase {
	u.tokens = tokens
	u.mailer = mailer
	return u
}

// Invite creates the member and sends an invitation email containing a link
// to set their password. Email delivery failure does not roll back the member;
// it is reported so the admin can resend instead of failing silently.
func (u *Usecase) Invite(ctx context.Context, tenantID uuid.UUID, email, role string, warehouseIDs []uuid.UUID, requesterID uuid.UUID) (*InviteResult, error) {
	user, err := u.InviteMember(ctx, tenantID, email, role, warehouseIDs, requesterID)
	if err != nil {
		return nil, err
	}
	res := &InviteResult{User: user}
	res.EmailSent, res.EmailError = u.sendInvitation(ctx, user)
	return res, nil
}

// ResendInvitation issues a fresh token and resends the email for an existing member.
func (u *Usecase) ResendInvitation(ctx context.Context, tenantID, userID uuid.UUID) (*InviteResult, error) {
	user, err := u.repo.GetByID(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if user.Role == domain.RoleOwner {
		return nil, fmt.Errorf("owner tidak perlu diundang")
	}
	res := &InviteResult{User: user}
	res.EmailSent, res.EmailError = u.sendInvitation(ctx, user)
	return res, nil
}

func (u *Usecase) sendInvitation(ctx context.Context, user *domain.User) (bool, string) {
	if u.tokens == nil || u.mailer == nil {
		return false, "layanan email belum dikonfigurasi"
	}
	if !u.mailer.IsConfigured() {
		return false, "layanan email (Brevo/SMTP) belum dikonfigurasi di server"
	}
	token := uuid.New().String()
	if err := u.tokens.UpdatePasswordResetToken(ctx, user.ID, token, time.Now().Add(InviteTokenTTL)); err != nil {
		log.Printf("[TEAM] store invite token for %s: %v", user.Email, err)
		return false, "gagal membuat tautan undangan"
	}
	if err := u.mailer.SendInvitationEmail(user.Email, token); err != nil {
		log.Printf("[TEAM] send invitation to %s: %v", user.Email, err)
		return false, "gagal mengirim email undangan"
	}
	log.Printf("[TEAM] invitation email dispatched to %s", user.Email)
	return true, ""
}

func New(repo UserReader) *Usecase {
	return &Usecase{repo: repo}
}

// ListMembers returns all users in the tenant.
func (u *Usecase) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	return u.repo.ListByTenant(ctx, tenantID)
}

// InviteMember adds a new team member to the tenant.
func (u *Usecase) InviteMember(ctx context.Context, tenantID uuid.UUID, email, role string, warehouseIDs []uuid.UUID, requesterID uuid.UUID) (*domain.User, error) {
	normRole := strings.ToLower(strings.TrimSpace(role))
	if !AllowedInviteRoles[normRole] {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Invariant: Staf gudang dan Kepala gudang wajib memiliki minimal 1 penugasan gudang
	if needsWarehouse(normRole) && len(warehouseIDs) == 0 {
		return nil, fmt.Errorf("staf atau kepala gudang wajib ditugaskan ke minimal 1 gudang")
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Check if user with this email already exists in tenant
	existing, err := u.repo.GetByEmail(ctx, tenantID, email)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("user with this email already exists in workspace")
	}

	// Generate temporary password hash for the invited user
	randomPass := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(randomPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generate password hash: %w", err)
	}

	now := time.Now().UTC()
	newUser := &domain.User{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Email:              email,
		FullName:           strings.Split(email, "@")[0],
		PasswordHash:       string(hash),
		Role:               normRole,
		AssignedWarehouses: []domain.AssignedWarehouse{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Hanya role gudang yang menyimpan penugasan gudang.
	if !needsWarehouse(normRole) {
		warehouseIDs = nil
	}
	if err := u.repo.CreateUserWithWarehouses(ctx, newUser, warehouseIDs); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if len(warehouseIDs) > 0 {
		whs, err := u.repo.GetUserWarehouses(ctx, tenantID, newUser.ID)
		if err == nil {
			newUser.AssignedWarehouses = whs
		}
	}

	return newUser, nil
}

// ChangeRole updates a user's role and assigned warehouses. Validates the role value and prevents changing the owner's role.
func (u *Usecase) ChangeRole(ctx context.Context, tenantID, userID uuid.UUID, newRole string, warehouseIDs []uuid.UUID, requesterID uuid.UUID) error {
	normRole := strings.ToLower(strings.TrimSpace(newRole))
	if !AllowedInviteRoles[normRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Invariant: Staf gudang dan Kepala gudang wajib memiliki minimal 1 penugasan gudang
	if needsWarehouse(normRole) && len(warehouseIDs) == 0 {
		return fmt.Errorf("staf atau kepala gudang wajib ditugaskan ke minimal 1 gudang")
	}

	// Cannot change your own role
	if userID == requesterID {
		return fmt.Errorf("cannot change your own role")
	}

	// Fetch target user to check if they're the owner
	target, err := u.repo.GetByID(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if target.Role == domain.RoleOwner {
		return fmt.Errorf("cannot change the owner's role")
	}

	if !needsWarehouse(normRole) {
		warehouseIDs = nil // role non-gudang: cabut semua penugasan gudang
	}
	return u.repo.UpdateUserRoleAndWarehouses(ctx, userID, tenantID, normRole, warehouseIDs)
}

// RemoveMember deletes a user from the tenant. Prevents removing the owner.
func (u *Usecase) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID, requesterID uuid.UUID) error {
	// Cannot remove yourself
	if userID == requesterID {
		return fmt.Errorf("cannot remove yourself")
	}

	target, err := u.repo.GetByID(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if target.Role == domain.RoleOwner {
		return fmt.Errorf("cannot remove the owner")
	}

	return u.repo.DeleteUser(ctx, userID, tenantID)
}

// needsWarehouse reports whether a role is scoped to assigned warehouses.
func needsWarehouse(role string) bool {
	return role == domain.RoleWarehouse || role == domain.RoleWarehouseManager || role == "regional_manager"
}
