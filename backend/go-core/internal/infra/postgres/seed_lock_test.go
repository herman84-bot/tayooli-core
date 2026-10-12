package postgres

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestLockedPasswordHashRejectsEveryPassword(t *testing.T) {
	for _, pw := range []string{"password123", "", LockedPasswordHash, "!locked-seed-account"} {
		if err := bcrypt.CompareHashAndPassword([]byte(LockedPasswordHash), []byte(pw)); err == nil {
			t.Fatalf("locked hash accepted password %q", pw)
		}
	}
}

func TestSeedAdminPasswordHashMatchesPassword123(t *testing.T) {
	if err := bcrypt.CompareHashAndPassword([]byte(SeedAdminPasswordHash), []byte("password123")); err != nil {
		t.Fatalf("seed admin hash does not match password123: %v", err)
	}
	if SeedAdminID != SeedAccountIDs[0] {
		t.Fatalf("SeedAdminID must be admin@test.com")
	}
}

func TestSeedAccountIDsCoverSeedMigrations(t *testing.T) {
	want := map[string]bool{
		"550e8400-e29b-41d4-a716-446655440001": true,
		"550e8400-e29b-41d4-a716-446655440101": true,
		"550e8400-e29b-41d4-a716-446655440102": true,
		"550e8400-e29b-41d4-a716-446655440003": true,
		"550e8400-e29b-41d4-a716-446655440004": true,
	}
	for _, id := range SeedAccountIDs {
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("seed accounts not locked: %v", want)
	}
}
