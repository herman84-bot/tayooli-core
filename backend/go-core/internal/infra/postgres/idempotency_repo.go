package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// IdempotencyRepo implements domain.IdempotencyStore on idempotency_keys.
type IdempotencyRepo struct {
	db *sql.DB
	// StaleAfter is how long an unfinished reservation blocks retries. A row
	// stays unfinished only when the process died mid-request (crash, deploy,
	// power loss); after this window a retry may take the key over.
	StaleAfter string
}

func NewIdempotencyRepo(db *sql.DB) *IdempotencyRepo {
	return &IdempotencyRepo{db: db, StaleAfter: "2 minutes"}
}

func (r *IdempotencyRepo) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *IdempotencyRepo) Reserve(ctx context.Context, tenantID uuid.UUID, key, requestHash string) (*domain.IdempotencyRecord, bool, error) {
	var existing *domain.IdempotencyRecord
	fresh := false
	err := r.withTenant(ctx, tenantID, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO idempotency_keys (tenant_id, key, request_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, key) DO NOTHING`, tenantID, key, requestHash)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			fresh = true
			return nil
		}
		// Take over an abandoned reservation (same body only).
		res, err = tx.ExecContext(ctx, `
			UPDATE idempotency_keys SET created_at = NOW()
			WHERE tenant_id = $1 AND key = $2 AND request_hash = $3
			  AND completed_at IS NULL AND created_at < NOW() - $4::interval`,
			tenantID, key, requestHash, r.StaleAfter)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			fresh = true
			return nil
		}
		rec := &domain.IdempotencyRecord{Key: key}
		var status sql.NullInt64
		var completedAt sql.NullTime
		if err := tx.QueryRowContext(ctx, `
			SELECT request_hash, status_code, response_body, completed_at
			FROM idempotency_keys WHERE tenant_id = $1 AND key = $2`, tenantID, key).
			Scan(&rec.RequestHash, &status, &rec.ResponseBody, &completedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Released between our INSERT and SELECT; caller may retry.
				return domain.ErrConflict
			}
			return err
		}
		rec.Completed = completedAt.Valid
		rec.StatusCode = int(status.Int64)
		existing = rec
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("IdempotencyRepo.Reserve: %w", err)
	}
	return existing, fresh, nil
}

func (r *IdempotencyRepo) Complete(ctx context.Context, tenantID uuid.UUID, key string, statusCode int, body []byte) error {
	err := r.withTenant(ctx, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE idempotency_keys SET status_code = $3, response_body = $4, completed_at = NOW()
			WHERE tenant_id = $1 AND key = $2`, tenantID, key, statusCode, body)
		return err
	})
	if err != nil {
		return fmt.Errorf("IdempotencyRepo.Complete: %w", err)
	}
	return nil
}

func (r *IdempotencyRepo) Release(ctx context.Context, tenantID uuid.UUID, key string) error {
	err := r.withTenant(ctx, tenantID, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			DELETE FROM idempotency_keys
			WHERE tenant_id = $1 AND key = $2 AND completed_at IS NULL`, tenantID, key)
		return err
	})
	if err != nil {
		return fmt.Errorf("IdempotencyRepo.Release: %w", err)
	}
	return nil
}
