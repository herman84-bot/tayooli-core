package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// PaymentOrderRepo implements domain.PaymentOrderRepository using database/sql
// with parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies defined
//     in migrations/008_payment_orders.sql.
type PaymentOrderRepo struct {
	db *sql.DB
}

func NewPaymentOrderRepo(db *sql.DB) *PaymentOrderRepo {
	return &PaymentOrderRepo{db: db}
}

// ── SQL Queries ──────────────────────────────────────────────────────────────

const createPaymentOrder = `
INSERT INTO payment_orders (id, tenant_id, invoice_id, amount, currency, payment_method,
                            reference_number, status, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft', $8, $9)
RETURNING id, tenant_id, invoice_id, amount, currency, payment_method,
          reference_number, status, notes, created_by, approved_by,
          paid_at, created_at, updated_at`

const countPaymentOrdersByTenant = `SELECT COUNT(*) FROM payment_orders WHERE tenant_id = $1`

const listPaymentOrdersByTenantPaged = `
SELECT id, tenant_id, invoice_id, amount, currency, payment_method,
       reference_number, status, notes, created_by, approved_by,
       paid_at, created_at, updated_at
FROM payment_orders
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

const getPaymentOrderByID = `
SELECT id, tenant_id, invoice_id, amount, currency, payment_method,
       reference_number, status, notes, created_by, approved_by,
       paid_at, created_at, updated_at
FROM payment_orders
WHERE id = $1 AND tenant_id = $2`

const updatePaymentOrderStatus = `
UPDATE payment_orders
SET status = $1, approved_by = $2, updated_at = NOW()
WHERE id = $3 AND tenant_id = $4 AND status = $5
RETURNING id, tenant_id, invoice_id, amount, currency, payment_method,
          reference_number, status, notes, created_by, approved_by,
          paid_at, created_at, updated_at`

const markPaymentOrderPaid = `
UPDATE payment_orders
SET status = 'paid', paid_at = NOW(), updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND status = 'approved'`

const markInvoicePaid = `
UPDATE invoices
SET paid_at = $1, updated_at = NOW()
WHERE id = $2 AND tenant_id = $3`

const paymentOrderExistsByTenant = `SELECT EXISTS(SELECT 1 FROM payment_orders WHERE id = $1 AND tenant_id = $2)`

// ── Create ───────────────────────────────────────────────────────────────────

func (r *PaymentOrderRepo) Create(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.Create: set tenant: %w", err)
	}

	newID := uuid.New()
	row := tx.QueryRowContext(ctx, createPaymentOrder,
		newID,
		params.TenantID,
		params.InvoiceID,
		params.Amount,
		params.Currency,
		params.PaymentMethod,
		params.ReferenceNumber,
		params.Notes,
		params.CreatedBy,
	)
	po, err := scanPaymentOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.Create: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.Create: commit: %w", err)
	}
	return po, nil
}

// ── ListPaged ────────────────────────────────────────────────────────────────

func (r *PaymentOrderRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: set tenant: %w", err)
	}

	var total int
	if err := tx.QueryRowContext(ctx, countPaymentOrdersByTenant, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := tx.QueryContext(ctx, listPaymentOrdersByTenantPaged, tenantID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: query: %w", err)
	}
	defer rows.Close()

	var orders []domain.PaymentOrder
	for rows.Next() {
		po, err := scanPaymentOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: scan row: %w", err)
		}
		orders = append(orders, *po)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.ListPaged: commit: %w", err)
	}

	return &domain.PaymentOrderListPage{
		Data:    orders,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// ── GetByID ──────────────────────────────────────────────────────────────────

func (r *PaymentOrderRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getPaymentOrderByID, id, tenantID)
	po, err := scanPaymentOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.GetByID: commit: %w", err)
	}
	return po, nil
}

// ── UpdateStatus ─────────────────────────────────────────────────────────────

// UpdateStatus transitions a payment order from one status to another.
// Returns ErrNotFound if the payment order doesn't exist for the tenant.
// Returns ErrConflict if the current status doesn't match `from`.
func (r *PaymentOrderRepo) UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, from, to domain.PaymentOrderStatus, approvedBy *uuid.UUID) (*domain.PaymentOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.UpdateStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.UpdateStatus: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, updatePaymentOrderStatus, to, approvedBy, id, tenantID, from)
	po, err := scanPaymentOrderRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		// No rows updated: either the payment order doesn't exist (ErrNotFound)
		// or it's not in the expected source status (ErrConflict).
		var exists bool
		checkErr := tx.QueryRowContext(ctx, paymentOrderExistsByTenant, id, tenantID).Scan(&exists)
		if checkErr != nil {
			return nil, fmt.Errorf("PaymentOrderRepo.UpdateStatus: check exists: %w", checkErr)
		}
		if !exists {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.UpdateStatus: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PaymentOrderRepo.UpdateStatus: commit: %w", err)
	}
	return po, nil
}

// ── MarkInvoicePaid ──────────────────────────────────────────────────────────

// MarkInvoicePaid transitions the payment order to 'paid' and marks the
// related invoice as paid in a single transaction.
func (r *PaymentOrderRepo) MarkInvoicePaid(ctx context.Context, paymentOrderID, tenantID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: set tenant: %w", err)
	}

	// 1. Mark payment order as paid (only if currently approved).
	res, err := tx.ExecContext(ctx, markPaymentOrderPaid, paymentOrderID, tenantID)
	if err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: update payment order: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Check existence to distinguish not-found vs. wrong status.
		var exists bool
		checkErr := tx.QueryRowContext(ctx, paymentOrderExistsByTenant, paymentOrderID, tenantID).Scan(&exists)
		if checkErr != nil {
			return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: check exists: %w", checkErr)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrConflict
	}

	// 2. Look up the invoice_id from the payment order so we can mark the invoice.
	var invoiceID uuid.UUID
	var paidAt time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT invoice_id, paid_at FROM payment_orders WHERE id = $1 AND tenant_id = $2`,
		paymentOrderID, tenantID,
	).Scan(&invoiceID, &paidAt)
	if err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: lookup invoice_id: %w", err)
	}

	// 3. Mark the invoice as paid.
	if _, err := tx.ExecContext(ctx, markInvoicePaid, paidAt, invoiceID, tenantID); err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: mark invoice paid: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("PaymentOrderRepo.MarkInvoicePaid: commit: %w", err)
	}
	return nil
}

// ── Scan helpers ─────────────────────────────────────────────────────────────

func scanPaymentOrder(s *sql.Rows) (*domain.PaymentOrder, error) {
	var po domain.PaymentOrder
	var amountStr string
	var status string
	var refNum, notes sql.NullString
	var createdByStr, approvedByStr sql.NullString
	var paidAt sql.NullTime

	err := s.Scan(
		&po.ID,
		&po.TenantID,
		&po.InvoiceID,
		&amountStr,
		&po.Currency,
		&po.PaymentMethod,
		&refNum,
		&status,
		&notes,
		&createdByStr,
		&approvedByStr,
		&paidAt,
		&po.CreatedAt,
		&po.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	amt, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("scanPaymentOrder: parse amount %q: %w", amountStr, err)
	}
	po.Amount = amt
	po.Status = domain.PaymentOrderStatus(status)

	if refNum.Valid {
		po.ReferenceNumber = &refNum.String
	}
	if notes.Valid {
		po.Notes = &notes.String
	}
	if createdByStr.Valid {
		id, err := uuid.Parse(createdByStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanPaymentOrder: parse created_by %q: %w", createdByStr.String, err)
		}
		po.CreatedBy = &id
	}
	if approvedByStr.Valid {
		id, err := uuid.Parse(approvedByStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanPaymentOrder: parse approved_by %q: %w", approvedByStr.String, err)
		}
		po.ApprovedBy = &id
	}
	if paidAt.Valid {
		t := paidAt.Time
		po.PaidAt = &t
	}

	return &po, nil
}

func scanPaymentOrderRow(row *sql.Row) (*domain.PaymentOrder, error) {
	var po domain.PaymentOrder
	var amountStr string
	var status string
	var refNum, notes sql.NullString
	var createdByStr, approvedByStr sql.NullString
	var paidAt sql.NullTime

	err := row.Scan(
		&po.ID,
		&po.TenantID,
		&po.InvoiceID,
		&amountStr,
		&po.Currency,
		&po.PaymentMethod,
		&refNum,
		&status,
		&notes,
		&createdByStr,
		&approvedByStr,
		&paidAt,
		&po.CreatedAt,
		&po.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	amt, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("scanPaymentOrderRow: parse amount %q: %w", amountStr, err)
	}
	po.Amount = amt
	po.Status = domain.PaymentOrderStatus(status)

	if refNum.Valid {
		po.ReferenceNumber = &refNum.String
	}
	if notes.Valid {
		po.Notes = &notes.String
	}
	if createdByStr.Valid {
		id, err := uuid.Parse(createdByStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanPaymentOrderRow: parse created_by %q: %w", createdByStr.String, err)
		}
		po.CreatedBy = &id
	}
	if approvedByStr.Valid {
		id, err := uuid.Parse(approvedByStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanPaymentOrderRow: parse approved_by %q: %w", approvedByStr.String, err)
		}
		po.ApprovedBy = &id
	}
	if paidAt.Valid {
		t := paidAt.Time
		po.PaidAt = &t
	}

	return &po, nil
}
