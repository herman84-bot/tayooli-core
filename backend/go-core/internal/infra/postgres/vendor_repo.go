package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// VendorRepo implements domain.VendorRepository using database/sql
// with parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies defined
//     in migrations/009_vendor_management.sql.
type VendorRepo struct {
	db *sql.DB
}

func NewVendorRepo(db *sql.DB) *VendorRepo {
	return &VendorRepo{db: db}
}

// ── SQL Queries ──────────────────────────────────────────────────────────────

const createVendor = `
INSERT INTO vendors (id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active')
RETURNING id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id,
          status, avg_rating, rating_count, created_at, updated_at`

const countVendorsByTenant = `
SELECT COUNT(*) FROM vendors WHERE tenant_id = $1`

const countVendorsByTenantSearch = `
SELECT COUNT(*) FROM vendors WHERE tenant_id = $1 AND name ILIKE '%' || $2 || '%' ESCAPE '\'`

const listVendorsByTenantPaged = `
SELECT id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id,
       status, avg_rating, rating_count, created_at, updated_at
FROM vendors
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

const listVendorsByTenantSearchPaged = `
SELECT id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id,
       status, avg_rating, rating_count, created_at, updated_at
FROM vendors
WHERE tenant_id = $1 AND name ILIKE '%' || $2 || '%' ESCAPE '\'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4`

const getVendorByID = `
SELECT id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id,
       status, avg_rating, rating_count, created_at, updated_at
FROM vendors
WHERE id = $1 AND tenant_id = $2`

const updateVendor = `
UPDATE vendors
SET name = $1, email = $2, phone = $3, address = $4,
    bank_account = $5, bank_name = $6, tax_id = $7, updated_at = NOW()
WHERE id = $8 AND tenant_id = $9
RETURNING id, tenant_id, name, email, phone, address, bank_account, bank_name, tax_id,
          status, avg_rating, rating_count, created_at, updated_at`

const softDeleteVendor = `
UPDATE vendors
SET status = 'inactive', updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND status = 'active'`

const vendorExistsByTenant = `
SELECT EXISTS(SELECT 1 FROM vendors WHERE id = $1 AND tenant_id = $2)`

const existsVendorByName = `
SELECT EXISTS(
    SELECT 1 FROM vendors
    WHERE tenant_id = $1 AND LOWER(TRIM(name)) = LOWER(TRIM($2)) AND status != 'inactive'
)`

const insertVendorRating = `
INSERT INTO vendor_ratings (id, tenant_id, vendor_id, rated_by, rating, comment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, tenant_id, vendor_id, rated_by, rating, comment, created_at`

// ── Create ───────────────────────────────────────────────────────────────────

func (r *VendorRepo) Create(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("VendorRepo.Create: set tenant: %w", err)
	}

	newID := uuid.New()
	row := tx.QueryRowContext(ctx, createVendor,
		newID,
		params.TenantID,
		params.Name,
		params.Email,
		params.Phone,
		params.Address,
		params.BankAccount,
		params.BankName,
		params.TaxID,
	)
	vendor, err := scanVendorRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.Create: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("VendorRepo.Create: commit: %w", err)
	}
	return vendor, nil
}

// ── ListPaged ────────────────────────────────────────────────────────────────

func (r *VendorRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: set tenant: %w", err)
	}

	var total int
	if search != "" {
		if err := tx.QueryRowContext(ctx, countVendorsByTenantSearch, tenantID, search).Scan(&total); err != nil {
			return nil, fmt.Errorf("VendorRepo.ListPaged: count search: %w", err)
		}
	} else {
		if err := tx.QueryRowContext(ctx, countVendorsByTenant, tenantID).Scan(&total); err != nil {
			return nil, fmt.Errorf("VendorRepo.ListPaged: count: %w", err)
		}
	}

	offset := (page - 1) * perPage

	var rows *sql.Rows
	if search != "" {
		rows, err = tx.QueryContext(ctx, listVendorsByTenantSearchPaged, tenantID, search, perPage, offset)
	} else {
		rows, err = tx.QueryContext(ctx, listVendorsByTenantPaged, tenantID, perPage, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: query: %w", err)
	}
	defer rows.Close()

	var vendors []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, fmt.Errorf("VendorRepo.ListPaged: scan row: %w", err)
		}
		vendors = append(vendors, *v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("VendorRepo.ListPaged: commit: %w", err)
	}

	return &domain.VendorListPage{
		Data:    vendors,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// ── GetByID ──────────────────────────────────────────────────────────────────

func (r *VendorRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("VendorRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getVendorByID, id, tenantID)
	vendor, err := scanVendorRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("VendorRepo.GetByID: commit: %w", err)
	}
	return vendor, nil
}

// ── Update ───────────────────────────────────────────────────────────────────

func (r *VendorRepo) Update(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.Update: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("VendorRepo.Update: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, updateVendor,
		params.Name,
		params.Email,
		params.Phone,
		params.Address,
		params.BankAccount,
		params.BankName,
		params.TaxID,
		params.ID,
		params.TenantID,
	)
	vendor, err := scanVendorRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.Update: update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("VendorRepo.Update: commit: %w", err)
	}
	return vendor, nil
}

// ── SoftDelete ───────────────────────────────────────────────────────────────

func (r *VendorRepo) SoftDelete(ctx context.Context, id, tenantID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("VendorRepo.SoftDelete: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("VendorRepo.SoftDelete: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, softDeleteVendor, id, tenantID)
	if err != nil {
		return fmt.Errorf("VendorRepo.SoftDelete: exec: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("VendorRepo.SoftDelete: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Check existence to distinguish not-found vs. already inactive.
		var exists bool
		checkErr := tx.QueryRowContext(ctx, vendorExistsByTenant, id, tenantID).Scan(&exists)
		if checkErr != nil {
			return fmt.Errorf("VendorRepo.SoftDelete: check exists: %w", checkErr)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrConflict
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("VendorRepo.SoftDelete: commit: %w", err)
	}
	return nil
}

// ── AddRating ────────────────────────────────────────────────────────────────

func (r *VendorRepo) AddRating(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.AddRating: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("VendorRepo.AddRating: set tenant: %w", err)
	}

	newID := uuid.New()
	row := tx.QueryRowContext(ctx, insertVendorRating,
		newID,
		params.TenantID,
		params.VendorID,
		params.RatedBy,
		params.Rating,
		params.Comment,
	)
	rating, err := scanVendorRatingRow(row)
	if err != nil {
		return nil, fmt.Errorf("VendorRepo.AddRating: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("VendorRepo.AddRating: commit: %w", err)
	}
	return rating, nil
}

// ── Scan helpers ─────────────────────────────────────────────────────────────

func scanVendor(s *sql.Rows) (*domain.Vendor, error) {
	var v domain.Vendor
	var status string
	var avgRatingStr string
	var email, phone, address, bankAccount, bankName, taxID sql.NullString

	err := s.Scan(
		&v.ID,
		&v.TenantID,
		&v.Name,
		&email,
		&phone,
		&address,
		&bankAccount,
		&bankName,
		&taxID,
		&status,
		&avgRatingStr,
		&v.RatingCount,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	avgRating, err := decimal.NewFromString(avgRatingStr)
	if err != nil {
		return nil, fmt.Errorf("scanVendor: parse avg_rating %q: %w", avgRatingStr, err)
	}
	v.AvgRating = avgRating
	v.Status = domain.VendorStatus(status)

	if email.Valid {
		v.Email = &email.String
	}
	if phone.Valid {
		v.Phone = &phone.String
	}
	if address.Valid {
		v.Address = &address.String
	}
	if bankAccount.Valid {
		v.BankAccount = &bankAccount.String
	}
	if bankName.Valid {
		v.BankName = &bankName.String
	}
	if taxID.Valid {
		v.TaxID = &taxID.String
	}

	return &v, nil
}

func scanVendorRow(row *sql.Row) (*domain.Vendor, error) {
	var v domain.Vendor
	var status string
	var avgRatingStr string
	var email, phone, address, bankAccount, bankName, taxID sql.NullString

	err := row.Scan(
		&v.ID,
		&v.TenantID,
		&v.Name,
		&email,
		&phone,
		&address,
		&bankAccount,
		&bankName,
		&taxID,
		&status,
		&avgRatingStr,
		&v.RatingCount,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	avgRating, err := decimal.NewFromString(avgRatingStr)
	if err != nil {
		return nil, fmt.Errorf("scanVendorRow: parse avg_rating %q: %w", avgRatingStr, err)
	}
	v.AvgRating = avgRating
	v.Status = domain.VendorStatus(status)

	if email.Valid {
		v.Email = &email.String
	}
	if phone.Valid {
		v.Phone = &phone.String
	}
	if address.Valid {
		v.Address = &address.String
	}
	if bankAccount.Valid {
		v.BankAccount = &bankAccount.String
	}
	if bankName.Valid {
		v.BankName = &bankName.String
	}
	if taxID.Valid {
		v.TaxID = &taxID.String
	}

	return &v, nil
}

func scanVendorRatingRow(row *sql.Row) (*domain.VendorRating, error) {
	var r domain.VendorRating
	var ratedByStr sql.NullString
	var comment sql.NullString

	err := row.Scan(
		&r.ID,
		&r.TenantID,
		&r.VendorID,
		&ratedByStr,
		&r.Rating,
		&comment,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if ratedByStr.Valid {
		id, err := uuid.Parse(ratedByStr.String)
		if err != nil {
			return nil, fmt.Errorf("scanVendorRatingRow: parse rated_by %q: %w", ratedByStr.String, err)
		}
		r.RatedBy = &id
	}
	if comment.Valid {
		r.Comment = &comment.String
	}

	return &r, nil
}

// ── ExistsByName ─────────────────────────────────────────────────────────────

func (r *VendorRepo) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, existsVendorByName, tenantID, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("VendorRepo.ExistsByName: %w", err)
	}
	return exists, nil
}

