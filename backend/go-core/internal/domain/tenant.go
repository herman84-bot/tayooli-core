package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID
	Name      string
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
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
