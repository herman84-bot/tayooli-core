package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// VendorStatus represents the lifecycle status of a vendor.
type VendorStatus string

const (
	VendorStatusActive    VendorStatus = "active"
	VendorStatusInactive  VendorStatus = "inactive"
	VendorStatusSuspended VendorStatus = "suspended"
)

// Vendor is a per-tenant vendor record with cached rating aggregates.
type Vendor struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Email       *string         `json:"email,omitempty"`
	Phone       *string         `json:"phone,omitempty"`
	Address     *string         `json:"address,omitempty"`
	BankAccount *string         `json:"bank_account,omitempty"`
	BankName    *string         `json:"bank_name,omitempty"`
	TaxID       *string         `json:"tax_id,omitempty"`
	Status      VendorStatus    `json:"status"`
	AvgRating   decimal.Decimal `json:"avg_rating"`
	RatingCount int             `json:"rating_count"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateVendorParams are the inputs for creating a new vendor.
type CreateVendorParams struct {
	TenantID    uuid.UUID
	Name        string
	Email       *string
	Phone       *string
	Address     *string
	BankAccount *string
	BankName    *string
	TaxID       *string
}

// UpdateVendorParams are the inputs for updating an existing vendor.
type UpdateVendorParams struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Email       *string
	Phone       *string
	Address     *string
	BankAccount *string
	BankName    *string
	TaxID       *string
}

// VendorRating holds a single user rating for a vendor.
type VendorRating struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	VendorID  uuid.UUID  `json:"vendor_id"`
	RatedBy   *uuid.UUID `json:"rated_by,omitempty"`
	Rating    int        `json:"rating"`
	Comment   *string    `json:"comment,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// AddVendorRatingParams are the inputs for rating a vendor.
type AddVendorRatingParams struct {
	TenantID uuid.UUID
	VendorID uuid.UUID
	RatedBy  *uuid.UUID
	Rating   int
	Comment  *string
}

// VendorListPage holds a paginated result set plus total count.
type VendorListPage struct {
	Data    []Vendor `json:"data"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	PerPage int      `json:"per_page"`
}

// VendorRepository is the persistence port for vendors and vendor ratings.
type VendorRepository interface {
	Create(ctx context.Context, params CreateVendorParams) (*Vendor, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*VendorListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Vendor, error)
	Update(ctx context.Context, params UpdateVendorParams) (*Vendor, error)
	SoftDelete(ctx context.Context, id, tenantID uuid.UUID) error
	AddRating(ctx context.Context, params AddVendorRatingParams) (*VendorRating, error)
	ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
}
