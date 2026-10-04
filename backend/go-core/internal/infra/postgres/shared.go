package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// setTenantLocally injects the tenant UUID into the current transaction as a
// PostgreSQL local variable, activating RLS policies that check
// current_setting('app.current_tenant_id', true).
// Must be called at the start of every transaction, before any query.
// Uses SET LOCAL (transaction-scoped) so the variable is cleared automatically
// when the transaction ends — no accidental cross-tenant leakage via pool reuse.
// Note: SET LOCAL does not support parameterized queries ($1), so we use
// fmt.Sprintf. This is safe because uuid.UUID.String() only produces hex digits and dashes.
func setTenantLocally(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) error {
	q := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID.String())
	_, err := tx.ExecContext(ctx, q)
	return err
}
