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

// UserReader provides the user operations needed by team management.
type UserReader interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error)
	GetByID(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUserRole(ctx context.Context, userID, tenantID uuid.UUID, role string) error
	DeleteUser(ctx context.Context, userID, tenantID uuid.UUID) error
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
func (u *Usecase) InviteMember(ctx context.Context, tenantID uuid.UUID, email, role string, requesterID uuid.UUID) (*domain.User, error) {
	allowedRoles := map[string]bool{"admin": true, "member": true, "accountant": true, "approver": true}
	if !allowedRoles[role] {
		return nil, fmt.Errorf("invalid role: %s", role)
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

	newUser := &domain.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        email,
		FullName:     strings.Split(email, "@")[0],
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := u.repo.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return newUser, nil
}

// ChangeRole updates a user's role. Validates the role value and prevents changing the owner's role.
func (u *Usecase) ChangeRole(ctx context.Context, tenantID, userID uuid.UUID, newRole string, requesterID uuid.UUID) error {
	allowedRoles := map[string]bool{"admin": true, "member": true, "accountant": true, "approver": true}
	if !allowedRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
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
	if target.Role == "owner" {
		return fmt.Errorf("cannot change the owner's role")
	}

	return u.repo.UpdateUserRole(ctx, userID, tenantID, newRole)
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
	if target.Role == "owner" {
		return fmt.Errorf("cannot remove the owner")
	}

	return u.repo.DeleteUser(ctx, userID, tenantID)
}
