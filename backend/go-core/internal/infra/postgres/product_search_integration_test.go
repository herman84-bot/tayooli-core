package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

func TestProductRepoSearch_PagingFilterAndIsolation(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	ctx := context.Background()
	tenant, other := uuid.New(), uuid.New()

	seed := func(tid uuid.UUID, q string, args ...any) {
		t.Helper()
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback() //nolint:errcheck
		_, err = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tid.String()+`'`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, q, args...)
		require.NoError(t, err, q)
		require.NoError(t, tx.Commit())
	}
	seed(tenant, `INSERT INTO tenants (id, name) VALUES ($1, 'Search T')`, tenant)
	seed(other, `INSERT INTO tenants (id, name) VALUES ($1, 'Search O')`, other)
	t.Cleanup(func() {
		for _, tid := range []uuid.UUID{tenant, other} {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return
			}
			_, _ = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tid.String()+`'`)
			_, _ = tx.ExecContext(ctx, `DELETE FROM products WHERE tenant_id = $1`, tid)
			_, _ = tx.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tid)
			_ = tx.Commit()
		}
	})

	for i := 0; i < 25; i++ {
		seed(tenant, `INSERT INTO products (tenant_id, name, sku, price) VALUES ($1, $2, $3, 1000)`,
			tenant, fmt.Sprintf("Kopi Bubuk %02d", i), fmt.Sprintf("KOPI-%02d", i))
	}
	seed(tenant, `INSERT INTO products (tenant_id, name, sku, price) VALUES ($1, 'Teh 100%_asli', 'TEH-1', 1000)`, tenant)
	seed(tenant, `INSERT INTO products (tenant_id, name, sku, price, deleted_at) VALUES ($1, 'Kopi Terhapus', 'KOPI-DEL', 1000, NOW())`, tenant)
	seed(other, `INSERT INTO products (tenant_id, name, sku, price) VALUES ($1, 'Kopi Tenant Lain', 'KOPI-X', 1000)`, other)

	repo := postgres.NewProductRepo(db)

	// Paging: 25 matches for "kopi" (case-insensitive), excluding deleted + other tenant.
	page1, total, err := repo.Search(ctx, tenant, domain.ProductFilter{Query: "kopi", Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, 25, total)
	require.Len(t, page1, 10)
	page3, total, err := repo.Search(ctx, tenant, domain.ProductFilter{Query: "kopi", Limit: 10, Offset: 20})
	require.NoError(t, err)
	require.Equal(t, 25, total)
	require.Len(t, page3, 5)

	// No overlap between pages (stable order).
	seen := map[uuid.UUID]bool{}
	for _, p := range append(page1, page3...) {
		require.False(t, seen[p.ID], "duplicate across pages")
		seen[p.ID] = true
	}

	// SKU match.
	_, total, err = repo.Search(ctx, tenant, domain.ProductFilter{Query: "kopi-07", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)

	// LIKE metacharacters are literal: "%" alone must not match everything.
	_, total, err = repo.Search(ctx, tenant, domain.ProductFilter{Query: "%", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total, "only the product whose name contains a literal %%")
	_, total, err = repo.Search(ctx, tenant, domain.ProductFilter{Query: "_", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total, "only the product whose name contains a literal _")

	// Empty query = all active products of the tenant.
	_, total, err = repo.Search(ctx, tenant, domain.ProductFilter{Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 26, total)

	// Past the end: empty page but total still reported.
	empty, total, err := repo.Search(ctx, tenant, domain.ProductFilter{Query: "kopi", Limit: 10, Offset: 100})
	require.NoError(t, err)
	require.Empty(t, empty)
	require.Equal(t, 25, total)
}
