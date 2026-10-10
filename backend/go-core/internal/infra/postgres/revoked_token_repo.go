package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RevokedTokenRepo persists logged-out JWTs (by SHA-256 hash) so
// TenantMiddleware can reject them before their natural expiry.
type RevokedTokenRepo struct {
	db *sql.DB
}

func NewRevokedTokenRepo(db *sql.DB) *RevokedTokenRepo {
	return &RevokedTokenRepo{db: db}
}

// Revoke records tokenHash as revoked until expiresAt. Idempotent: revoking the
// same token twice is a no-op. Expired rows are pruned in the same transaction.
func (r *RevokedTokenRepo) Revoke(ctx context.Context, tokenHash string, userID *uuid.UUID, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("RevokedTokenRepo.Revoke: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM revoked_tokens WHERE expires_at < NOW()`); err != nil {
		return fmt.Errorf("RevokedTokenRepo.Revoke: prune: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO revoked_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (token_hash) DO NOTHING`, tokenHash, ptrToNullUUID(userID), expiresAt); err != nil {
		return fmt.Errorf("RevokedTokenRepo.Revoke: insert: %w", err)
	}
	return tx.Commit()
}

// IsRevoked reports whether tokenHash was revoked and has not yet expired.
func (r *RevokedTokenRepo) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	var revoked bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_hash = $1 AND expires_at > NOW())`,
		tokenHash).Scan(&revoked)
	if err != nil {
		return false, fmt.Errorf("RevokedTokenRepo.IsRevoked: %w", err)
	}
	return revoked, nil
}
