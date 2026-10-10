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

type User struct {
	ID                      uuid.UUID
	TenantID                uuid.UUID
	Email                   string
	FullName                string
	PasswordHash            string
	Role                    string
	EmailVerifiedAt         *time.Time
	VerificationToken       *string
	VerificationExpiresAt   *time.Time
	PasswordResetToken      *string
	PasswordResetExpiresAt  *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
