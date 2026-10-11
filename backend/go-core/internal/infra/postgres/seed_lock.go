package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

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

// LockSeedAccounts disables login for the public test accounts. Called at
// startup in production only: the seed migrations run everywhere (they are
// needed by CI/E2E), so production must neutralise them after migrating.
// Uses the SECURITY DEFINER update_user_password() (migration 019) because a
// plain UPDATE on users is filtered by RLS outside a tenant context.
// Idempotent; IDs that do not exist are a no-op.
func LockSeedAccounts(ctx context.Context, db *sql.DB) error {
	for _, id := range SeedAccountIDs {
		if _, err := db.ExecContext(ctx, `SELECT update_user_password($1::uuid, $2)`, id, LockedPasswordHash); err != nil {
			return fmt.Errorf("lock seed account %s: %w", id, err)
		}
	}
	return nil
}
