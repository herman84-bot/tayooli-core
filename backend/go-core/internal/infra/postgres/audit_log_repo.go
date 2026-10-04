package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

const insertAuditLog = `
INSERT INTO audit_logs (id, tenant_id, user_id, entity_type, entity_id, action, details)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

// AuditLogRepo implements domain.AuditLogRepository using database/sql.
// Every write runs inside a transaction that sets app.current_tenant_id so the
// audit_logs RLS policy enforces tenant isolation at the database level.
type AuditLogRepo struct {
	db *sql.DB
}

func NewAuditLogRepo(db *sql.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, entry domain.AuditLogEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AuditLogRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, entry.TenantID); err != nil {
		return fmt.Errorf("AuditLogRepo.Create: set tenant: %w", err)
	}

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	_, err = tx.ExecContext(ctx, insertAuditLog,
		entry.ID,
		entry.TenantID,
		entry.UserID,
		entry.EntityType,
		entry.EntityID,
		entry.Action,
		entry.Details,
	)
	if err != nil {
		return fmt.Errorf("AuditLogRepo.Create: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("AuditLogRepo.Create: commit: %w", err)
	}
	return nil
}
