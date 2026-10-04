package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type PaymentGatewayRepo struct {
	db *sql.DB
}

func NewPaymentGatewayRepo(db *sql.DB) *PaymentGatewayRepo {
	return &PaymentGatewayRepo{db: db}
}

func (r *PaymentGatewayRepo) GetConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.TenantPaymentConfig, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetConfig: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetConfig: set tenant: %w", err)
	}

	const q = `
		SELECT id, tenant_id, provider, slug, api_key, server_key, client_key,
		       is_production, is_active, settlement_bank_name, settlement_bank_account,
		       settlement_holder_name, gateway_fee_percent, created_at, updated_at
		FROM tenant_payment_configs
		WHERE tenant_id = $1 AND provider = $2`

	var cfg domain.TenantPaymentConfig
	err = tx.QueryRowContext(ctx, q, tenantID, provider).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Provider, &cfg.Slug, &cfg.APIKey, &cfg.ServerKey, &cfg.ClientKey,
		&cfg.IsProduction, &cfg.IsActive, &cfg.SettlementBankName, &cfg.SettlementBankAccount,
		&cfg.SettlementHolderName, &cfg.GatewayFeePercent, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetConfig: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetConfig: commit: %w", err)
	}
	return &cfg, nil
}

func (r *PaymentGatewayRepo) UpsertConfig(ctx context.Context, cfg *domain.TenantPaymentConfig) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpsertConfig: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, cfg.TenantID); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpsertConfig: set tenant: %w", err)
	}

	const q = `
		INSERT INTO tenant_payment_configs (
			tenant_id, provider, slug, api_key, server_key, client_key,
			is_production, is_active, settlement_bank_name, settlement_bank_account,
			settlement_holder_name, gateway_fee_percent, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, NOW()
		)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET
			slug = COALESCE(EXCLUDED.slug, tenant_payment_configs.slug),
			api_key = COALESCE(NULLIF(EXCLUDED.api_key, ''), tenant_payment_configs.api_key),
			server_key = COALESCE(NULLIF(EXCLUDED.server_key, ''), tenant_payment_configs.server_key),
			client_key = COALESCE(NULLIF(EXCLUDED.client_key, ''), tenant_payment_configs.client_key),
			is_production = EXCLUDED.is_production,
			is_active = EXCLUDED.is_active,
			settlement_bank_name = COALESCE(EXCLUDED.settlement_bank_name, tenant_payment_configs.settlement_bank_name),
			settlement_bank_account = COALESCE(EXCLUDED.settlement_bank_account, tenant_payment_configs.settlement_bank_account),
			settlement_holder_name = COALESCE(EXCLUDED.settlement_holder_name, tenant_payment_configs.settlement_holder_name),
			gateway_fee_percent = EXCLUDED.gateway_fee_percent,
			updated_at = NOW()`

	_, err = tx.ExecContext(ctx, q,
		cfg.TenantID, cfg.Provider, cfg.Slug, cfg.APIKey, cfg.ServerKey, cfg.ClientKey,
		cfg.IsProduction, cfg.IsActive, cfg.SettlementBankName, cfg.SettlementBankAccount,
		cfg.SettlementHolderName, cfg.GatewayFeePercent,
	)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpsertConfig: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpsertConfig: commit: %w", err)
	}
	return nil
}

func (r *PaymentGatewayRepo) CreateTransaction(ctx context.Context, txModel *domain.PaymentTransaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.CreateTransaction: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, txModel.TenantID); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.CreateTransaction: set tenant: %w", err)
	}

	const q = `
		INSERT INTO payment_transactions (
			tenant_id, provider, invoice_id, order_id, amount, method, payment_link, status, is_demo
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, q,
		txModel.TenantID, txModel.Provider, txModel.InvoiceID, txModel.OrderID, txModel.Amount, txModel.Method, txModel.PaymentLink, txModel.Status, txModel.IsDemo,
	).Scan(&txModel.ID, &txModel.CreatedAt, &txModel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.CreateTransaction: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.CreateTransaction: commit: %w", err)
	}
	return nil
}

func (r *PaymentGatewayRepo) GetTransactionByOrderID(ctx context.Context, tenantID uuid.UUID, orderID string) (*domain.PaymentTransaction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetTransactionByOrderID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetTransactionByOrderID: set tenant: %w", err)
	}

	const q = `
		SELECT id, tenant_id, provider, invoice_id, order_id, amount, method, payment_link, status, is_demo, completed_at, created_at, updated_at
		FROM payment_transactions
		WHERE tenant_id = $1 AND order_id = $2`

	var txModel domain.PaymentTransaction
	err = tx.QueryRowContext(ctx, q, tenantID, orderID).Scan(
		&txModel.ID, &txModel.TenantID, &txModel.Provider, &txModel.InvoiceID, &txModel.OrderID, &txModel.Amount, &txModel.Method,
		&txModel.PaymentLink, &txModel.Status, &txModel.IsDemo, &txModel.CompletedAt, &txModel.CreatedAt, &txModel.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetTransactionByOrderID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentGatewayRepo.GetTransactionByOrderID: commit: %w", err)
	}
	return &txModel, nil
}

func (r *PaymentGatewayRepo) UpdateTransactionStatus(ctx context.Context, tenantID uuid.UUID, orderID, status string, completedAt *time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpdateTransactionStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpdateTransactionStatus: set tenant: %w", err)
	}

	const q = `
		UPDATE payment_transactions
		SET status = $3, completed_at = $4, updated_at = NOW()
		WHERE tenant_id = $1 AND order_id = $2`

	res, err := tx.ExecContext(ctx, q, tenantID, orderID, status, completedAt)
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpdateTransactionStatus: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpdateTransactionStatus: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("PaymentGatewayRepo.UpdateTransactionStatus: commit: %w", err)
	}
	return nil
}
