package postgres_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

func freshRevokedTokenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "tayooli"
		}
		pass := os.Getenv("DB_PASSWORD")
		if pass == "" {
			pass = "tayooli"
		}
		name := os.Getenv("DB_NAME")
		if name == "" {
			name = "tayooli"
		}
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, sslmode)
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	err = db.Ping()
	if err != nil {
		t.Skipf("PostgreSQL not available at %s: %v", dsn, err)
	}

	// Verify revoked_tokens table exists (migrations applied)
	var exists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables 
		WHERE table_name = 'revoked_tokens'
	)`).Scan(&exists)
	if err != nil || !exists {
		t.Skip("revoked_tokens table does not exist, skipping integration test")
	}

	return db
}

func testTokenHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestRevokedTokenRepo_Integration(t *testing.T) {
	db := freshRevokedTokenTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewRevokedTokenRepo(db)

	userID := uuid.New()
	hash1 := testTokenHash(uuid.NewString() + "-tok1")
	hash2 := testTokenHash(uuid.NewString() + "-tok2")
	expiredHash := testTokenHash(uuid.NewString() + "-expired")

	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM revoked_tokens WHERE token_hash IN ($1, $2, $3)", hash1, hash2, expiredHash)
	})

	t.Run("unrevoked token reports false", func(t *testing.T) {
		revoked, err := repo.IsRevoked(ctx, hash1)
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("revoking an active token makes IsRevoked return true", func(t *testing.T) {
		exp := time.Now().Add(1 * time.Hour)
		err := repo.Revoke(ctx, hash1, &userID, exp)
		require.NoError(t, err)

		revoked, err := repo.IsRevoked(ctx, hash1)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("revoking the same token hash twice is idempotent", func(t *testing.T) {
		exp := time.Now().Add(1 * time.Hour)
		err := repo.Revoke(ctx, hash1, &userID, exp)
		require.NoError(t, err)

		revoked, err := repo.IsRevoked(ctx, hash1)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("expired token reports false and gets pruned on next Revoke", func(t *testing.T) {
		// Insert an already expired token directly
		pastExp := time.Now().Add(-5 * time.Minute)
		_, err := db.ExecContext(ctx, `
INSERT INTO revoked_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (token_hash) DO NOTHING`, expiredHash, userID, pastExp)
		require.NoError(t, err)

		// IsRevoked must report false because expires_at > NOW() is false
		revoked, err := repo.IsRevoked(ctx, expiredHash)
		require.NoError(t, err)
		assert.False(t, revoked)

		// Revoking another token must prune expired rows
		err = repo.Revoke(ctx, hash2, nil, time.Now().Add(30*time.Minute))
		require.NoError(t, err)

		// Confirm expiredHash was deleted from DB
		var rowExists bool
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_hash = $1)`, expiredHash).Scan(&rowExists)
		require.NoError(t, err)
		assert.False(t, rowExists, "expired token should have been pruned")
	})
}
