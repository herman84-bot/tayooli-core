package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

func setupTestDB(t *testing.T) *sql.DB {
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

	// Ensure the tenant exists for FK constraint.
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, _ = db.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, plan) VALUES ($1, 'Test Tenant', 'basic') ON CONFLICT (id) DO NOTHING`,
		tenantID)

	// Set RLS tenant context for this connection so INSERT/UPDATE/DELETE work.
	_, _ = db.ExecContext(context.Background(),
		`SET app.current_tenant_id = '`+tenantID.String()+`'`)

	t.Cleanup(func() {
		// Clean up test data.
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM users WHERE tenant_id = $1 AND email LIKE '%@example.com'`, tenantID)
		db.Close()
	})
	return db
}

func TestUserRepo_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepo(db)
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Seed user
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (id, tenant_id, email, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO UPDATE SET password_hash = $4`,
		uuid.New(), tenantID, "test@example.com",
		"$2a$10$N9qo8uLOickgx2ZMRZoMye", "admin")
	require.NoError(t, err)

	// Test GetByEmail - found
	u, err := repo.GetByEmail(context.Background(), tenantID, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", u.Email)
	assert.Equal(t, tenantID, u.TenantID)
	assert.Equal(t, "admin", u.Role)
	assert.False(t, u.ID == uuid.Nil, "user ID should not be nil")

	// Test GetByEmail - not found
	_, err = repo.GetByEmail(context.Background(), tenantID, "none@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Test GetByEmail - wrong tenant
	otherTenant := uuid.New()
	_, err = repo.GetByEmail(context.Background(), otherTenant, "test@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestVerifyPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	assert.True(t, postgres.VerifyPassword(string(hash), "secret123"))
	assert.False(t, postgres.VerifyPassword(string(hash), "wrong"))
	assert.False(t, postgres.VerifyPassword("invalid-hash", "secret123"))
}
