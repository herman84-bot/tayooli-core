package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry records an immutable business action for compliance and
// traceability. Each entry captures WHO performed WHAT action on WHICH entity,
// with a snapshot of relevant details as JSONB.
type AuditLogEntry struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	EntityType string          `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Action     string          `json:"action"`
	Details    json.RawMessage `json:"details"`
	CreatedAt  time.Time       `json:"created_at"`
}

// AuditLogRepository is the persistence port for audit log entries.
type AuditLogRepository interface {
	Create(ctx context.Context, entry AuditLogEntry) error
}
