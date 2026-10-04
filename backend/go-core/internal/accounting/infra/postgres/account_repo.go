package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
)

type accountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) domain.AccountRepository {
	return &accountRepository{db: db}
}

// setTenantLocally injects tenant UUID into the current transaction,
// activating RLS policies that check current_setting('app.current_tenant_id').
func setTenantLocally(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) error {
	q := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID.String())
	_, err := tx.ExecContext(ctx, q)
	return err
}

func (r *accountRepository) Create(ctx context.Context, account *domain.Account) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("accountRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, account.TenantID); err != nil {
		return fmt.Errorf("accountRepo.Create: set tenant: %w", err)
	}

	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}
	query := `INSERT INTO accounts (id, tenant_id, code, name, type, balance) VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.ExecContext(ctx, query, account.ID, account.TenantID, account.Code, account.Name, account.Type, account.Balance); err != nil {
		return fmt.Errorf("accountRepo.Create: exec: %w", err)
	}
	return tx.Commit()
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*domain.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("accountRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("accountRepo.GetByID: set tenant: %w", err)
	}

	query := `SELECT id, tenant_id, code, name, type, balance, created_at, updated_at FROM accounts WHERE id = $1 AND tenant_id = $2`
	row := tx.QueryRowContext(ctx, query, id, tenantID)

	var a domain.Account
	if err := row.Scan(&a.ID, &a.TenantID, &a.Code, &a.Name, &a.Type, &a.Balance, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("accountRepo.GetByID: scan: %w", err)
	}
	return &a, tx.Commit()
}

func (r *accountRepository) GetByCode(ctx context.Context, code string, tenantID uuid.UUID) (*domain.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("accountRepo.GetByCode: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("accountRepo.GetByCode: set tenant: %w", err)
	}

	query := `SELECT id, tenant_id, code, name, type, balance, created_at, updated_at FROM accounts WHERE code = $1 AND tenant_id = $2`
	row := tx.QueryRowContext(ctx, query, code, tenantID)

	var a domain.Account
	if err := row.Scan(&a.ID, &a.TenantID, &a.Code, &a.Name, &a.Type, &a.Balance, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("accountRepo.GetByCode: scan: %w", err)
	}
	return &a, tx.Commit()
}

func (r *accountRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("accountRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("accountRepo.List: set tenant: %w", err)
	}

	query := `SELECT id, tenant_id, code, name, type, balance, created_at, updated_at FROM accounts WHERE tenant_id = $1`
	rows, err := tx.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("accountRepo.List: query: %w", err)
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		var a domain.Account
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Code, &a.Name, &a.Type, &a.Balance, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("accountRepo.List: scan: %w", err)
		}
		accounts = append(accounts, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accountRepo.List: rows err: %w", err)
	}
	return accounts, tx.Commit()
}
