package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// SeedTenantID is the tenant that migrations 012/013 seed ("PT Test Indonesia").
var SeedTenantID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

// SeedAccountIDs are the fixed user IDs that migrations 012/013 seed with the
// public password "password123" (admin/accountant/approver @test.com).
var SeedAccountIDs = []string{
	"550e8400-e29b-41d4-a716-446655440001", // admin@test.com
	"550e8400-e29b-41d4-a716-446655440101", // accountant@test.com (012)
	"550e8400-e29b-41d4-a716-446655440102", // approver@test.com (012)
	"550e8400-e29b-41d4-a716-446655440003", // accountant@test.com (013)
	"550e8400-e29b-41d4-a716-446655440004", // approver@test.com (013)
}

// LockedPasswordHash is not a valid bcrypt hash, so
// bcrypt.CompareHashAndPassword fails for every input and login is impossible.
const LockedPasswordHash = "!locked-seed-account"

// SeedAdminID is admin@test.com in the seed tenant.
const SeedAdminID = "550e8400-e29b-41d4-a716-446655440001"

// SeedAdminPasswordHash is the bcrypt hash of "password123" from migration 012.
const SeedAdminPasswordHash = "$2b$10$QwKrWvf4JTGvRjoqoR993ecO1bHhLPGlC3DA.r7iE3joi/ovc.rW2"

// UnlockSeedAdmin restores login for admin@test.com only (opt-in via
// ALLOW_SEED_LOGIN=true). The other seed accounts stay locked. The account
// lives in the isolated seed tenant, so it cannot see real tenant data.
func UnlockSeedAdmin(ctx context.Context, db *sql.DB) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("unlock seed admin: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := setTenantLocally(ctx, tx, SeedTenantID); err != nil {
		return 0, fmt.Errorf("unlock seed admin: set tenant: %w", err)
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW()
		  WHERE id = $1::uuid AND tenant_id = $3 AND password_hash <> $2`,
		SeedAdminID, SeedAdminPasswordHash, SeedTenantID)
	if err != nil {
		return 0, fmt.Errorf("unlock seed admin: %w", err)
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("unlock seed admin: commit: %w", err)
	}
	return n, nil
}

// LockSeedAccounts disables login for the public test accounts. Called at
// startup in production only: the seed migrations run everywhere (they are
// needed by CI/E2E), so production must neutralise them after migrating.
//
// users has FORCE ROW LEVEL SECURITY (migration 024), which also binds
// SECURITY DEFINER functions, so an update without a tenant context matches
// zero rows silently. The update therefore runs in a transaction scoped to the
// seed tenant. Idempotent; returns how many rows were changed.
func LockSeedAccounts(ctx context.Context, db *sql.DB) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("lock seed accounts: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := setTenantLocally(ctx, tx, SeedTenantID); err != nil {
		return 0, fmt.Errorf("lock seed accounts: set tenant: %w", err)
	}

	var changed int64
	for _, id := range SeedAccountIDs {
		res, err := tx.ExecContext(ctx,
			`UPDATE users
			    SET password_hash = $2,
			        password_reset_token = NULL,
			        password_reset_expires_at = NULL,
			        updated_at = NOW()
			  WHERE id = $1::uuid AND tenant_id = $3 AND password_hash <> $2`,
			id, LockedPasswordHash, SeedTenantID)
		if err != nil {
			return 0, fmt.Errorf("lock seed account %s: %w", id, err)
		}
		n, _ := res.RowsAffected()
		changed += n
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("lock seed accounts: commit: %w", err)
	}
	return changed, nil
}
