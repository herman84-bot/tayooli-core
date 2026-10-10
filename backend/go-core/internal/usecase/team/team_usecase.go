package team

import (
	"context"
	"fmt"
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

// Usecase handles team management operations.
type Usecase struct {
	repo UserReader
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
