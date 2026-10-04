package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

// freshApprovalTestDB returns a *sql.DB whose session has NOT had
// app.current_tenant_id set. This reproduces the real API scenario: pooled
// connections that never ran SET on the session, where RLS would hide rows
// unless each query sets the tenant var (via setTenantLocally).
func freshApprovalTestDB(t *testing.T) *sql.DB {
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
	return db
}

// seedApprovalRequestRows inserts a workflow + request row. RLS requires the
// tenant session var for writes, so this uses a dedicated connection that sets
// it, then closes — leaving the connection pool (and any other connection)
// without the var, exactly like production.
func seedApprovalRequestRows(t *testing.T, db *sql.DB, tenantID uuid.UUID, requestedBy uuid.UUID) (workflowID string, approvalID string, targetID string) {
	t.Helper()
	workflowID = uuid.New().String()
	approvalID = uuid.New().String()
	targetID = uuid.New().String()

	seedDB, err := sql.Open("postgres", mustDSN())
	require.NoError(t, err)
	defer seedDB.Close()

	_, err = seedDB.ExecContext(context.Background(), `SET app.current_tenant_id = '`+tenantID.String()+`'`)
	require.NoError(t, err)

	_, err = seedDB.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, plan) VALUES ($1, 'Approval Test Tenant', 'basic') ON CONFLICT (id) DO NOTHING`,
		tenantID)
	require.NoError(t, err)

	_, err = seedDB.ExecContext(context.Background(),
		`INSERT INTO approval_workflows (id, tenant_id, name, steps, created_at, updated_at)
		 VALUES ($1, $2, 'test workflow', '["approver"]', NOW(), NOW())`,
		workflowID, tenantID.String())
	require.NoError(t, err)

	_, err = seedDB.ExecContext(context.Background(),
		`INSERT INTO approval_requests (id, tenant_id, workflow_id, target_type, target_id, status, current_step_index, requested_by, created_at, updated_at)
		 VALUES ($1, $2, $3, 'invoice', $4, 'pending', 0, $5, NOW(), NOW())`,
		approvalID, tenantID.String(), workflowID, targetID, requestedBy.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupDB, err := sql.Open("postgres", mustDSN())
		if err != nil {
			return
		}
		defer cleanupDB.Close()
		_, _ = cleanupDB.ExecContext(context.Background(), `SET app.current_tenant_id = '`+tenantID.String()+`'`)
		_, _ = cleanupDB.ExecContext(context.Background(), `DELETE FROM approval_requests WHERE id = $1`, approvalID)
		_, _ = cleanupDB.ExecContext(context.Background(), `DELETE FROM approval_workflows WHERE id = $1`, workflowID)
		_, _ = cleanupDB.ExecContext(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
	})
	return workflowID, approvalID, targetID
}

func mustDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		return dsn
	}
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
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, sslmode)
}

// Regression test: GetRequestByID must see the row on a connection whose
// session never ran SET app.current_tenant_id. Before the fix the repo queried
// r.db directly and RLS filtered the row out → ErrNotFound (API returned 404).
func TestApprovalRepo_GetRequestByID_RespectsRLS(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	repo := postgres.NewApprovalRepository(db)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	_, approvalID, _ := seedApprovalRequestRows(t, db, tenantID, requestedBy)

	// No SET app.current_tenant_id happened on db's sessions.
	ar, err := repo.GetRequestByID(context.Background(), uuid.MustParse(approvalID), tenantID)
	require.NoError(t, err, "GetRequestByID must not be RLS-filtered")
	assert.Equal(t, approvalID, ar.ID)
	assert.Equal(t, "pending", ar.Status)

	// Wrong tenant must still be hidden (tenant isolation preserved).
	_, err = repo.GetRequestByID(context.Background(), uuid.MustParse(approvalID), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Regression test: UpdateRequestStatus must UPDATE on a connection whose
// session never ran SET app.current_tenant_id. Before the fix the repo called
// r.db.ExecContext directly → 0 rows affected → ErrNotFound (API returned 404).
func TestApprovalRepo_UpdateRequestStatus_RespectsRLS(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	repo := postgres.NewApprovalRepository(db)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	_, approvalID, _ := seedApprovalRequestRows(t, db, tenantID, requestedBy)

	now := time.Now()
	approvedBy := requestedBy.String()
	ar := &domain.ApprovalRequest{
		ID:               approvalID,
		TenantID:         tenantID.String(),
		Status:           "approved",
		ApprovedBy:       &approvedBy,
		ApprovedAt:       &now,
		RejectionReason:  "",
		CurrentStepIndex: 0,
	}

	err := repo.UpdateRequestStatus(context.Background(), ar)
	require.NoError(t, err, "UpdateRequestStatus must not be RLS-filtered")

	// Verify the update actually landed.
	got, err := repo.GetRequestByID(context.Background(), uuid.MustParse(approvalID), tenantID)
	require.NoError(t, err)
	assert.Equal(t, "approved", got.Status)
	assert.Equal(t, approvedBy, *got.ApprovedBy)
}

// End-to-end repo check of the full approve path the usecase performs:
// GetRequestByID → UpdateRequestStatus, on fresh RLS-restricted connections.
func TestApprovalRepo_ApproveFlow_RLS(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	repo := postgres.NewApprovalRepository(db)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	_, approvalID, _ := seedApprovalRequestRows(t, db, tenantID, requestedBy)

	// Re-insert as pending (seed used 'pending').
	ar, err := repo.GetRequestByID(context.Background(), uuid.MustParse(approvalID), tenantID)
	require.NoError(t, err)

	now := time.Now()
	approvedBy := requestedBy.String()
	ar.Status = "approved"
	ar.ApprovedBy = &approvedBy
	ar.ApprovedAt = &now

	err = repo.UpdateRequestStatus(context.Background(), ar)
	require.NoError(t, err)

	got, err := repo.GetRequestByID(context.Background(), uuid.MustParse(approvalID), tenantID)
	require.NoError(t, err)
	assert.Equal(t, "approved", got.Status)
}
