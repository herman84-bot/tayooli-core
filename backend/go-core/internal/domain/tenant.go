package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID
	Name      string
	Plan      string
	Address   string
	Phone     string
	Email     string
	TaxID     string
	Website   string
	LogoURL   string
	Tagline   string
	Division  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TenantProfile struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Phone    string    `json:"phone"`
	Email    string    `json:"email"`
	TaxID    string    `json:"tax_id"`
	Website  string    `json:"website"`
	LogoURL  string    `json:"logo_url"`
	Tagline  string    `json:"tagline"`
	Division string    `json:"division"`
}

// Enterprise RBAC Roles
const (
	RoleOwner            = "owner"
	RoleAdmin            = "admin"
	RoleWarehouseManager = "warehouse_manager"
	RoleWarehouse        = "warehouse"
	RoleCashier          = "cashier"
	RoleAuditor          = "auditor"
	RoleMember           = "member"
)

// AssignedWarehouse represents a summary of a warehouse assigned to a user.
type AssignedWarehouse struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type User struct {
	ID                      uuid.UUID
	TenantID                uuid.UUID
	Email                   string
	FullName                string
	PasswordHash            string
	Role                    string
	AssignedWarehouses      []AssignedWarehouse `json:"assigned_warehouses,omitempty"`
	EmailVerifiedAt         *time.Time
	VerificationToken       *string
	VerificationExpiresAt   *time.Time
	PasswordResetToken      *string
	PasswordResetExpiresAt  *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
