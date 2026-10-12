package postgres_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

// Regression: concurrent first audit writes for a brand-new tenant must not
// fork the hash chain. With row-level FOR UPDATE, an empty audit_logs locked
// nothing, so several writers each saw prev_hash = "" and chained off nothing.
func TestWriteAuditTx_ConcurrentFirstWritesKeepSingleChain(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	ctx := context.Background()
	tenantID := uuid.New()

	seed, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = seed.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenantID.String()+`'`)
	require.NoError(t, err)
	_, err = seed.ExecContext(ctx, `INSERT INTO tenants (id, name, plan) VALUES ($1, 'Audit Chain Tenant', 'basic')`, tenantID)
	require.NoError(t, err)
	require.NoError(t, seed.Commit())
	t.Cleanup(func() {
		c, err := sql.Open("postgres", mustDSN())
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = c.ExecContext(ctx, `SET app.current_tenant_id = '`+tenantID.String()+`'`)
		_, _ = c.ExecContext(ctx, `DELETE FROM audit_logs WHERE tenant_id = $1`, tenantID)
		_, _ = c.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	repo := postgres.NewWMSRepo(db)
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				errs <- err
				return
			}
			defer tx.Rollback() //nolint:errcheck
			if _, err := tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenantID.String()+`'`); err != nil {
				errs <- err
				return
			}
			if err := repo.WriteAuditTx(ctx, tx, tenantID, nil, "TEST", uuid.New(), "CREATE", json.RawMessage(`{}`)); err != nil {
				errs <- err
				return
			}
			errs <- tx.Commit()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck
	_, err = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenantID.String()+`'`)
	require.NoError(t, err)

	var total, roots, distinctPrev int
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE prev_hash IS NULL OR prev_hash = ''),
		       COUNT(DISTINCT COALESCE(prev_hash, ''))
		FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&total, &roots, &distinctPrev))
	require.Equal(t, writers, total)
	require.Equal(t, 1, roots, "exactly one chain root")
	require.Equal(t, writers, distinctPrev, "no two entries share a predecessor (no fork)")
}
