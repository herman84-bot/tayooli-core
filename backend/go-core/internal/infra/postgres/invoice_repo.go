package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// InvoiceRepo implements domain.InvoiceRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies defined
//     in migrations/001_init_schema.sql.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type InvoiceRepo struct {
	db *sql.DB
}

func NewInvoiceRepo(db *sql.DB) *InvoiceRepo {
	return &InvoiceRepo{db: db}
}

const listInvoicesByTenant = `
SELECT id, tenant_id, vendor_id, invoice_number, amount, currency, status,
       ai_confidence_score, due_date, po_id, match_result, created_at, updated_at
FROM invoices
WHERE tenant_id = $1
ORDER BY created_at DESC`

func (r *InvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listInvoicesByTenant, tenantID)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("InvoiceRepo.List: scan row: %w", err)
		}
		invoices = append(invoices, *inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: rows err: %w", err)
	}

	// Close rows before committing to release the server-side cursor first.
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.List: commit: %w", err)
	}
	return invoices, nil
}

const countInvoicesByTenant = `SELECT COUNT(*) FROM invoices WHERE tenant_id = $1`

const listInvoicesByTenantPaged = `
SELECT id, tenant_id, vendor_id, invoice_number, amount, currency, status,
       ai_confidence_score, due_date, po_id, match_result, created_at, updated_at
FROM invoices
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

// ListPaged returns a page of invoices plus the total count for the tenant.
func (r *InvoiceRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: set tenant: %w", err)
	}

	// Count total rows for this tenant.
	var total int
	if err := tx.QueryRowContext(ctx, countInvoicesByTenant, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := tx.QueryContext(ctx, listInvoicesByTenantPaged, tenantID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("InvoiceRepo.ListPaged: scan row: %w", err)
		}
		invoices = append(invoices, *inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.ListPaged: commit: %w", err)
	}

	return &domain.InvoiceListPage{
		Data:    invoices,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

const getInvoiceByID = `
SELECT id, tenant_id, vendor_id, invoice_number, amount, currency, status,
       ai_confidence_score, due_date, po_id, match_result, created_at, updated_at
FROM invoices
WHERE id = $1 AND tenant_id = $2`

func (r *InvoiceRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getInvoiceByID, id, tenantID)
	inv, err := scanInvoiceRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.GetByID: commit: %w", err)
	}
	return inv, nil
}

const createInvoice = `
INSERT INTO invoices (id, tenant_id, vendor_id, invoice_number, amount, currency, status, due_date, po_id)
VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8)
RETURNING id, tenant_id, vendor_id, invoice_number, amount, currency, status,
          ai_confidence_score, due_date, po_id, match_result, created_at, updated_at`

func (r *InvoiceRepo) Create(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Create: set tenant: %w", err)
	}

	newID := uuid.New()
	row := tx.QueryRowContext(ctx, createInvoice,
		newID,
		params.TenantID,
		params.VendorID,
		params.InvoiceNumber,
		params.Amount,
		params.Currency,
		params.DueDate,
		params.POID,
	)
	inv, err := scanInvoiceRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Create: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Create: commit: %w", err)
	}
	return inv, nil
}

const approveInvoice = `
UPDATE invoices
SET status = 'approved', updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND status IN ('ai_processed', 'pending_review', 'ai_failed')
RETURNING id, tenant_id, vendor_id, invoice_number, amount, currency, status,
          ai_confidence_score, due_date, po_id, match_result, created_at, updated_at`

const invoiceExistsByTenantQuery = `SELECT EXISTS(SELECT 1 FROM invoices WHERE id = $1 AND tenant_id = $2)`

func (r *InvoiceRepo) Approve(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Approve: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Approve: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, approveInvoice, id, tenantID)
	inv, err := scanInvoiceRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		// No rows updated means either the invoice does not belong to this tenant
		// (ErrNotFound) or it is already in a non-pending state (ErrConflict).
		// Distinguish by checking existence separately.
		var exists bool
		checkErr := tx.QueryRowContext(ctx, invoiceExistsByTenantQuery, id, tenantID).Scan(&exists)
		if checkErr != nil {
			return nil, fmt.Errorf("InvoiceRepo.Approve: check exists: %w", checkErr)
		}
		if !exists {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Approve: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Approve: commit: %w", err)
	}
	return inv, nil
}

const rejectInvoice = `
UPDATE invoices
SET status = 'rejected', updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND status IN ('pending', 'pending_review')
RETURNING id, tenant_id, vendor_id, invoice_number, amount, currency, status,
          ai_confidence_score, due_date, po_id, match_result, created_at, updated_at`

func (r *InvoiceRepo) Reject(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Reject: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Reject: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, rejectInvoice, id, tenantID)
	inv, err := scanInvoiceRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		// No rows updated means either the invoice does not belong to this tenant
		// (ErrNotFound) or it is already in a non-rejectable state (ErrConflict).
		// Distinguish by checking existence separately.
		var exists bool
		checkErr := tx.QueryRowContext(ctx, invoiceExistsByTenantQuery, id, tenantID).Scan(&exists)
		if checkErr != nil {
			return nil, fmt.Errorf("InvoiceRepo.Reject: check exists: %w", checkErr)
		}
		if !exists {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Reject: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InvoiceRepo.Reject: commit: %w", err)
	}
	return inv, nil
}

const updateOCRResult = `
UPDATE invoices
SET ai_confidence_score = $1,
    extracted_text      = $2,
    status              = $3,
    ocr_processed_at    = NOW()
WHERE id = $4
  AND tenant_id = $5`

// UpdateOCRResult persists the OCR pipeline output produced by the Python AI worker.
// The update runs inside a transaction so that SET LOCAL app.current_tenant_id
// activates RLS policies, enforcing tenant isolation at the database layer as well.
func (r *InvoiceRepo) UpdateOCRResult(ctx context.Context, tenantID, invoiceID uuid.UUID, score float64, extractedText, status string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InvoiceRepo.UpdateOCRResult: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("InvoiceRepo.UpdateOCRResult: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateOCRResult, score, extractedText, status, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("InvoiceRepo.UpdateOCRResult: exec: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("InvoiceRepo.UpdateOCRResult: rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("InvoiceRepo.UpdateOCRResult: commit: %w", err)
	}
	return nil
}

func scanInvoice(s *sql.Rows) (*domain.Invoice, error) {
	var inv domain.Invoice
	var aiScore sql.NullFloat64
	var dueDate sql.NullTime
	var poIDStr sql.NullString
	var matchResult sql.NullString
	var status string
	err := s.Scan(
		&inv.ID,
		&inv.TenantID,
		&inv.VendorID,
		&inv.InvoiceNumber,
		&inv.Amount,
		&inv.Currency,
		&status,
		&aiScore,
		&dueDate,
		&poIDStr,
		&matchResult,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.Status = domain.InvoiceStatus(status)
	if aiScore.Valid {
		inv.AIConfidenceScore = aiScore.Float64
	}
	if dueDate.Valid {
		t := dueDate.Time
		inv.DueDate = &t
	}
	if poIDStr.Valid {
		id, err := uuid.Parse(poIDStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanInvoice: malformed po_id %q: %w", poIDStr.String, err)
		}
		inv.POID = &id
	}
	if matchResult.Valid {
		inv.MatchResult = &matchResult.String
	}
	return &inv, nil
}

func scanInvoiceRow(row *sql.Row) (*domain.Invoice, error) {
	var inv domain.Invoice
	var aiScore sql.NullFloat64
	var dueDate sql.NullTime
	var poIDStr sql.NullString
	var matchResult sql.NullString
	var status string
	err := row.Scan(
		&inv.ID,
		&inv.TenantID,
		&inv.VendorID,
		&inv.InvoiceNumber,
		&inv.Amount,
		&inv.Currency,
		&status,
		&aiScore,
		&dueDate,
		&poIDStr,
		&matchResult,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.Status = domain.InvoiceStatus(status)
	if aiScore.Valid {
		inv.AIConfidenceScore = aiScore.Float64
	}
	if dueDate.Valid {
		t := dueDate.Time
		inv.DueDate = &t
	}
	if poIDStr.Valid {
		id, err := uuid.Parse(poIDStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanInvoiceRow: malformed po_id %q: %w", poIDStr.String, err)
		}
		inv.POID = &id
	}
	if matchResult.Valid {
		inv.MatchResult = &matchResult.String
	}
	return &inv, nil
}
