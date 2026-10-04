package domain

import (
	"time"

	"github.com/google/uuid"
)

type AIPermissions struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	AutonomyLevel string     `json:"autonomy_level"` // "advisory", "assisted", "autopilot"
	AllowedScopes []string   `json:"allowed_scopes"`
	EmergencyStop bool       `json:"emergency_stop"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type UpdateAIPermissionsInput struct {
	AutonomyLevel *string  `json:"autonomy_level,omitempty"`
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
	EmergencyStop *bool    `json:"emergency_stop,omitempty"`
}
