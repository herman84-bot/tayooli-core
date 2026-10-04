package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type SubscriptionRepo struct {
	db *sql.DB
}

func NewSubscriptionRepo(db *sql.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

// GetByTenantID retrieves the subscription for a tenant.
func (r *SubscriptionRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSubscription, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetByTenantID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetByTenantID: set tenant: %w", err)
	}

	const q = `
		SELECT id, tenant_id, plan, status, trial_started_at, trial_ends_at,
		       billing_period, current_period_start, current_period_end,
		       payment_provider, cancel_at, cancelled_at, created_at, updated_at
		FROM tenant_subscriptions
		WHERE tenant_id = $1`

	var sub domain.TenantSubscription
	err = tx.QueryRowContext(ctx, q, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.Plan, &sub.Status,
		&sub.TrialStartedAt, &sub.TrialEndsAt, &sub.BillingPeriod,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.PaymentProvider, &sub.CancelAt, &sub.CancelledAt,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetByTenantID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetByTenantID: commit: %w", err)
	}
	return &sub, nil
}

// Create creates a new subscription for a tenant.
func (r *SubscriptionRepo) Create(ctx context.Context, sub *domain.TenantSubscription) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, sub.TenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.Create: set tenant: %w", err)
	}

	const q = `
		INSERT INTO tenant_subscriptions (
			tenant_id, plan, status, trial_started_at, trial_ends_at,
			billing_period, current_period_start, current_period_end,
			payment_provider, cancel_at, cancelled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, q,
		sub.TenantID, sub.Plan, sub.Status,
		sub.TrialStartedAt, sub.TrialEndsAt,
		sub.BillingPeriod, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.PaymentProvider, sub.CancelAt, sub.CancelledAt,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.Create: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SubscriptionRepo.Create: commit: %w", err)
	}
	return nil
}

// UpdateStatus updates the subscription status.
func (r *SubscriptionRepo) UpdateStatus(ctx context.Context, tenantID uuid.UUID, status domain.SubscriptionStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateStatus: set tenant: %w", err)
	}

	const q = `
		UPDATE tenant_subscriptions
		SET status = $2, updated_at = NOW()
		WHERE tenant_id = $1`

	res, err := tx.ExecContext(ctx, q, tenantID, status)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateStatus: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateStatus: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateStatus: commit: %w", err)
	}
	return nil
}

// UpgradePlan updates the subscription plan and billing period.
func (r *SubscriptionRepo) UpgradePlan(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpgradePlan: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.UpgradePlan: set tenant: %w", err)
	}

	const q = `
		UPDATE tenant_subscriptions
		SET plan = $2, billing_period = $3, status = 'active',
		    current_period_start = NOW(),
		    current_period_end = CASE 
		        WHEN $3 = 'monthly' THEN NOW() + INTERVAL '1 month'
		        WHEN $3 = 'annual' THEN NOW() + INTERVAL '1 year'
		        ELSE NOW() + INTERVAL '1 month'
		    END,
		    updated_at = NOW()
		WHERE tenant_id = $1`

	res, err := tx.ExecContext(ctx, q, tenantID, plan, period)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpgradePlan: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpgradePlan: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SubscriptionRepo.UpgradePlan: commit: %w", err)
	}
	return nil
}

// Cancel marks a subscription for cancellation at end of current period.
func (r *SubscriptionRepo) Cancel(ctx context.Context, tenantID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.Cancel: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.Cancel: set tenant: %w", err)
	}

	const q = `
		UPDATE tenant_subscriptions
		SET status = 'cancelled', cancel_at = current_period_end, cancelled_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND status = 'active'`

	res, err := tx.ExecContext(ctx, q, tenantID)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.Cancel: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.Cancel: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SubscriptionRepo.Cancel: commit: %w", err)
	}
	return nil
}

// GetPlanLimits retrieves the limits for a specific plan.
func (r *SubscriptionRepo) GetPlanLimits(ctx context.Context, plan string) (*domain.PlanLimits, error) {
	const q = `
		SELECT id, plan, max_users, max_vendors, max_invoices_per_month,
		       max_ocr_per_month, multi_entity, api_access, created_at
		FROM plan_limits
		WHERE plan = $1`

	var limits domain.PlanLimits
	err := r.db.QueryRowContext(ctx, q, plan).Scan(
		&limits.ID, &limits.Plan, &limits.MaxUsers, &limits.MaxVendors,
		&limits.MaxInvoicesPerMonth, &limits.MaxOCRPerMonth,
		&limits.MultiEntity, &limits.APIAccess, &limits.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetPlanLimits: scan: %w", err)
	}
	return &limits, nil
}

// CreateInvoice creates a subscription billing invoice.
func (r *SubscriptionRepo) CreateInvoice(ctx context.Context, inv *domain.SubscriptionInvoice) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.CreateInvoice: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, inv.TenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.CreateInvoice: set tenant: %w", err)
	}

	const q = `
		INSERT INTO subscription_invoices (
			tenant_id, subscription_id, invoice_number, amount, currency,
			status, period_start, period_end, payment_link, payment_provider,
			payment_reference, due_date, paid_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, q,
		inv.TenantID, inv.SubscriptionID, inv.InvoiceNumber, inv.Amount,
		inv.Currency, inv.Status, inv.PeriodStart, inv.PeriodEnd,
		inv.PaymentLink, inv.PaymentProvider, inv.PaymentReference,
		inv.DueDate, inv.PaidAt,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.CreateInvoice: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SubscriptionRepo.CreateInvoice: commit: %w", err)
	}
	return nil
}

// ListInvoices retrieves all subscription invoices for a tenant.
func (r *SubscriptionRepo) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionInvoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: set tenant: %w", err)
	}

	const q = `
		SELECT id, tenant_id, subscription_id, invoice_number, amount, currency,
		       status, period_start, period_end, payment_link, payment_provider,
		       payment_reference, due_date, paid_at, created_at, updated_at
		FROM subscription_invoices
		WHERE tenant_id = $1
		ORDER BY created_at DESC`

	rows, err := tx.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.SubscriptionInvoice
	for rows.Next() {
		var inv domain.SubscriptionInvoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.InvoiceNumber,
			&inv.Amount, &inv.Currency, &inv.Status, &inv.PeriodStart,
			&inv.PeriodEnd, &inv.PaymentLink, &inv.PaymentProvider,
			&inv.PaymentReference, &inv.DueDate, &inv.PaidAt,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: scan: %w", err)
		}
		invoices = append(invoices, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: rows error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListInvoices: commit: %w", err)
	}
	return invoices, nil
}

// GetInvoiceByID retrieves a subscription invoice by its ID, scoped to tenant.
func (r *SubscriptionRepo) GetInvoiceByID(ctx context.Context, tenantID, invoiceID uuid.UUID) (*domain.SubscriptionInvoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetInvoiceByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetInvoiceByID: set tenant: %w", err)
	}

	const q = `
		SELECT id, tenant_id, subscription_id, invoice_number, amount, currency,
		       status, period_start, period_end, payment_link, payment_provider,
		       payment_reference, due_date, paid_at, created_at, updated_at
		FROM subscription_invoices
		WHERE id = $1 AND tenant_id = $2`

	var inv domain.SubscriptionInvoice
	err = tx.QueryRowContext(ctx, q, invoiceID, tenantID).Scan(
		&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.InvoiceNumber,
		&inv.Amount, &inv.Currency, &inv.Status, &inv.PeriodStart,
		&inv.PeriodEnd, &inv.PaymentLink, &inv.PaymentProvider,
		&inv.PaymentReference, &inv.DueDate, &inv.PaidAt,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetInvoiceByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.GetInvoiceByID: commit: %w", err)
	}
	return &inv, nil
}

// UpdateInvoiceStatus updates the status and optionally paid_at of an invoice, scoped to tenant.
func (r *SubscriptionRepo) UpdateInvoiceStatus(ctx context.Context, tenantID uuid.UUID, invoiceID uuid.UUID, status string, paidAt *time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateInvoiceStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateInvoiceStatus: set tenant: %w", err)
	}

	const q = `
		UPDATE subscription_invoices
		SET status = $1, paid_at = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4`

	res, err := tx.ExecContext(ctx, q, status, paidAt, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateInvoiceStatus: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("SubscriptionRepo.UpdateInvoiceStatus: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return tx.Commit()
}

// ListAllInvoices retrieves all subscription invoices across all tenants (for webhook processing).
func (r *SubscriptionRepo) ListAllInvoices(ctx context.Context) ([]domain.SubscriptionInvoice, error) {
	const q = `
		SELECT id, tenant_id, subscription_id, invoice_number, amount, currency,
		       status, period_start, period_end, payment_link, payment_provider,
		       payment_reference, due_date, paid_at, created_at, updated_at
		FROM subscription_invoices
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("SubscriptionRepo.ListAllInvoices: query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.SubscriptionInvoice
	for rows.Next() {
		var inv domain.SubscriptionInvoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.InvoiceNumber,
			&inv.Amount, &inv.Currency, &inv.Status, &inv.PeriodStart,
			&inv.PeriodEnd, &inv.PaymentLink, &inv.PaymentProvider,
			&inv.PaymentReference, &inv.DueDate, &inv.PaidAt,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("SubscriptionRepo.ListAllInvoices: scan: %w", err)
		}
		invoices = append(invoices, inv)
	}

	return invoices, rows.Err()
}
