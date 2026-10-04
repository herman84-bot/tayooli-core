package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type AIPermissionsRepo struct {
	db *sql.DB
}

func NewAIPermissionsRepo(db *sql.DB) *AIPermissionsRepo {
	return &AIPermissionsRepo{db: db}
}

const selectAIPermissions = `
SELECT id, tenant_id, autonomy_level, allowed_scopes, emergency_stop, updated_by, created_at, updated_at
FROM tenant_ai_permissions
WHERE tenant_id = $1`

const upsertDefaultAIPermissions = `
INSERT INTO tenant_ai_permissions (id, tenant_id, autonomy_level, allowed_scopes, emergency_stop, created_at, updated_at)
VALUES ($1, $2, 'assisted', '["workspace.read", "workspace.profile_write", "workspace.master_write"]'::jsonb, false, NOW(), NOW())
ON CONFLICT (tenant_id) DO UPDATE SET updated_at = NOW()
RETURNING id, tenant_id, autonomy_level, allowed_scopes, emergency_stop, updated_by, created_at, updated_at`

func (r *AIPermissionsRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.AIPermissions, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.GetByTenantID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.GetByTenantID: set tenant: %w", err)
	}

	var p domain.AIPermissions
	var rawScopes []byte
	var updatedBy sql.NullString

	err = tx.QueryRowContext(ctx, selectAIPermissions, tenantID).Scan(
		&p.ID,
		&p.TenantID,
		&p.AutonomyLevel,
		&rawScopes,
		&p.EmergencyStop,
		&updatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		// Insert default record if none exists for this tenant
		newID := uuid.New()
		err = tx.QueryRowContext(ctx, upsertDefaultAIPermissions, newID, tenantID).Scan(
			&p.ID,
			&p.TenantID,
			&p.AutonomyLevel,
			&rawScopes,
			&p.EmergencyStop,
			&updatedBy,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.GetByTenantID: query/upsert: %w", err)
	}

	if len(rawScopes) > 0 {
		_ = json.Unmarshal(rawScopes, &p.AllowedScopes)
	} else {
		p.AllowedScopes = []string{"workspace.read", "workspace.profile_write", "workspace.master_write"}
	}

	if updatedBy.Valid {
		if uid, parseErr := uuid.Parse(updatedBy.String); parseErr == nil {
			p.UpdatedBy = &uid
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.GetByTenantID: commit: %w", err)
	}

	return &p, nil
}

func (r *AIPermissionsRepo) Update(ctx context.Context, tenantID uuid.UUID, input domain.UpdateAIPermissionsInput, updatedBy *uuid.UUID) (*domain.AIPermissions, error) {
	current, err := r.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if input.AutonomyLevel != nil {
		current.AutonomyLevel = *input.AutonomyLevel
	}
	if input.AllowedScopes != nil {
		current.AllowedScopes = input.AllowedScopes
	}
	if input.EmergencyStop != nil {
		current.EmergencyStop = *input.EmergencyStop
	}
	current.UpdatedBy = updatedBy
	current.UpdatedAt = time.Now()

	scopesJSON, err := json.Marshal(current.AllowedScopes)
	if err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.Update: marshal scopes: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.Update: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.Update: set tenant: %w", err)
	}

	const updateQuery = `
	UPDATE tenant_ai_permissions
	SET autonomy_level = $1, allowed_scopes = $2, emergency_stop = $3, updated_by = $4, updated_at = $5
	WHERE tenant_id = $6
	RETURNING id, tenant_id, autonomy_level, allowed_scopes, emergency_stop, updated_by, created_at, updated_at`

	var rawScopes []byte
	var updatedByScan sql.NullString

	err = tx.QueryRowContext(ctx, updateQuery,
		current.AutonomyLevel,
		scopesJSON,
		current.EmergencyStop,
		current.UpdatedBy,
		current.UpdatedAt,
		tenantID,
	).Scan(
		&current.ID,
		&current.TenantID,
		&current.AutonomyLevel,
		&rawScopes,
		&current.EmergencyStop,
		&updatedByScan,
		&current.CreatedAt,
		&current.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.Update: exec update: %w", err)
	}

	if len(rawScopes) > 0 {
		_ = json.Unmarshal(rawScopes, &current.AllowedScopes)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("AIPermissionsRepo.Update: commit: %w", err)
	}

	return current, nil
}
