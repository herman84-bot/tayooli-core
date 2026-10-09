package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type WMSRepo struct {
	db *sql.DB
}

func NewWMSRepo(db *sql.DB) *WMSRepo {
	return &WMSRepo{db: db}
}

// -----------------------------------------------------------------------------
// Helpers for Nullable Database Scanning
// -----------------------------------------------------------------------------

func nullUUIDToPtr(ns sql.NullString) *uuid.UUID {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	id, err := uuid.Parse(ns.String)
	if err != nil {
		return nil
	}
	return &id
}

func ptrToNullUUID(id *uuid.UUID) sql.NullString {
	if id == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: id.String(), Valid: true}
}

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	val := ns.String
	return &val
}

func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	val := nt.Time
	return &val
}

func ptrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullDecimalToPtr(nd decimal.NullDecimal) *decimal.Decimal {
	if !nd.Valid {
		return nil
	}
	val := nd.Decimal
	return &val
}

func ptrToNullDecimal(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{Valid: false}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "unique constraint") || strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "23505")
}

// -----------------------------------------------------------------------------
// 1. Regional
// -----------------------------------------------------------------------------

const createRegionalSQL = `
INSERT INTO regionals (id, tenant_id, code, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)`

func (r *WMSRepo) CreateRegional(ctx context.Context, reg *domain.Regional) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateRegional: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, reg.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateRegional: set tenant: %w", err)
	}

	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if reg.CreatedAt.IsZero() {
		reg.CreatedAt = now
	}
	if reg.UpdatedAt.IsZero() {
		reg.UpdatedAt = now
	}

	_, err = tx.ExecContext(ctx, createRegionalSQL,
		reg.ID, reg.TenantID, reg.Code, reg.Name, reg.CreatedAt, reg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateRegional: exec: %w", err)
	}

	return tx.Commit()
}

const getRegionalByIDSQL = `
SELECT id, tenant_id, code, name, created_at, updated_at
FROM regionals
WHERE id = $1 AND tenant_id = $2`

func (r *WMSRepo) GetRegionalByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Regional, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetRegionalByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetRegionalByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getRegionalByIDSQL, id, tenantID)
	var reg domain.Regional
	err = row.Scan(&reg.ID, &reg.TenantID, &reg.Code, &reg.Name, &reg.CreatedAt, &reg.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetRegionalByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetRegionalByID: commit: %w", err)
	}
	return &reg, nil
}

// -----------------------------------------------------------------------------
// 2. Warehouses
// -----------------------------------------------------------------------------

const createWarehouseSQL = `
INSERT INTO warehouses (id, tenant_id, regional_id, code, name, address, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

func (r *WMSRepo) CreateWarehouse(ctx context.Context, w *domain.Warehouse) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateWarehouse: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, w.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateWarehouse: set tenant: %w", err)
	}

	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	now := time.Now().UTC()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	if w.UpdatedAt.IsZero() {
		w.UpdatedAt = now
	}

	_, err = tx.ExecContext(ctx, createWarehouseSQL,
		w.ID, w.TenantID, ptrToNullUUID(w.RegionalID), w.Code, w.Name,
		ptrToNullString(w.Address), w.IsActive, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateWarehouse: exec: %w", err)
	}

	return tx.Commit()
}

const getWarehouseByIDSQL = `
SELECT id, tenant_id, regional_id, code, name, address, is_active, created_at, updated_at
FROM warehouses
WHERE id = $1 AND tenant_id = $2`

func (r *WMSRepo) GetWarehouseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Warehouse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWarehouseByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWarehouseByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getWarehouseByIDSQL, id, tenantID)
	var w domain.Warehouse
	var regID sql.NullString
	var addr sql.NullString

	err = row.Scan(&w.ID, &w.TenantID, &regID, &w.Code, &w.Name, &addr, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetWarehouseByID: scan: %w", err)
	}
	w.RegionalID = nullUUIDToPtr(regID)
	w.Address = nullStringToPtr(addr)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWarehouseByID: commit: %w", err)
	}
	return &w, nil
}

const listWarehousesSQL = `
SELECT id, tenant_id, regional_id, code, name, address, is_active, created_at, updated_at
FROM warehouses
WHERE tenant_id = $1
ORDER BY name ASC`

func (r *WMSRepo) ListWarehouses(ctx context.Context, tenantID uuid.UUID) ([]domain.Warehouse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehouses: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehouses: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listWarehousesSQL, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehouses: query: %w", err)
	}
	defer rows.Close()

	var result []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		var regID sql.NullString
		var addr sql.NullString
		if err := rows.Scan(&w.ID, &w.TenantID, &regID, &w.Code, &w.Name, &addr, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListWarehouses: scan: %w", err)
		}
		w.RegionalID = nullUUIDToPtr(regID)
		w.Address = nullStringToPtr(addr)
		result = append(result, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehouses: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehouses: commit: %w", err)
	}
	return result, nil
}

const listWarehousesByRegionalSQL = `
SELECT id, tenant_id, regional_id, code, name, address, is_active, created_at, updated_at
FROM warehouses
WHERE tenant_id = $1 AND regional_id = $2
ORDER BY name ASC`

func (r *WMSRepo) ListWarehousesByRegional(ctx context.Context, tenantID, regionalID uuid.UUID) ([]domain.Warehouse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listWarehousesByRegionalSQL, tenantID, regionalID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: query: %w", err)
	}
	defer rows.Close()

	var result []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		var regID sql.NullString
		var addr sql.NullString
		if err := rows.Scan(&w.ID, &w.TenantID, &regID, &w.Code, &w.Name, &addr, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: scan: %w", err)
		}
		w.RegionalID = nullUUIDToPtr(regID)
		w.Address = nullStringToPtr(addr)
		result = append(result, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByRegional: commit: %w", err)
	}
	return result, nil
}

const listWarehousesByIDsSQL = `
SELECT id, tenant_id, regional_id, code, name, address, is_active, created_at, updated_at
FROM warehouses
WHERE tenant_id = $1 AND id = ANY($2::uuid[])
ORDER BY name ASC`

func (r *WMSRepo) ListWarehousesByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Warehouse, error) {
	if len(ids) == 0 {
		return []domain.Warehouse{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: set tenant: %w", err)
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	rows, err := tx.QueryContext(ctx, listWarehousesByIDsSQL, tenantID, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: query: %w", err)
	}
	defer rows.Close()

	var result []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		var regID sql.NullString
		var addr sql.NullString
		if err := rows.Scan(&w.ID, &w.TenantID, &regID, &w.Code, &w.Name, &addr, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: scan: %w", err)
		}
		w.RegionalID = nullUUIDToPtr(regID)
		w.Address = nullStringToPtr(addr)
		result = append(result, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListWarehousesByIDs: commit: %w", err)
	}
	return result, nil
}

const getUserWarehouseIDsSQL = `
SELECT warehouse_id
FROM user_warehouses
WHERE tenant_id = $1 AND user_id = $2`

func (r *WMSRepo) GetUserWarehouseIDs(ctx context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, getUserWarehouseIDsSQL, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: query: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetUserWarehouseIDs: commit: %w", err)
	}
	return ids, nil
}

const assignUserWarehouseSQL = `
INSERT INTO user_warehouses (user_id, warehouse_id, tenant_id, assigned_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, warehouse_id) DO NOTHING`

func (r *WMSRepo) AssignUserWarehouse(ctx context.Context, uw *domain.UserWarehouse) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.AssignUserWarehouse: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, uw.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.AssignUserWarehouse: set tenant: %w", err)
	}

	if uw.AssignedAt.IsZero() {
		uw.AssignedAt = time.Now().UTC()
	}

	_, err = tx.ExecContext(ctx, assignUserWarehouseSQL, uw.UserID, uw.WarehouseID, uw.TenantID, uw.AssignedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.AssignUserWarehouse: exec: %w", err)
	}

	return tx.Commit()
}

// -----------------------------------------------------------------------------
// 3. Locations
// -----------------------------------------------------------------------------

const createLocationSQL = `
INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

func (r *WMSRepo) CreateLocation(ctx context.Context, loc *domain.WarehouseLocation) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, loc.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateLocation: set tenant: %w", err)
	}

	if loc.ID == uuid.Nil {
		loc.ID = uuid.New()
	}
	now := time.Now().UTC()
	if loc.CreatedAt.IsZero() {
		loc.CreatedAt = now
	}
	if loc.UpdatedAt.IsZero() {
		loc.UpdatedAt = now
	}
	if loc.Type == "" {
		loc.Type = domain.LocationTypeInternal
	}

	_, err = tx.ExecContext(ctx, createLocationSQL,
		loc.ID, loc.TenantID, ptrToNullUUID(loc.WarehouseID), ptrToNullUUID(loc.ParentID),
		loc.Code, ptrToNullString(loc.Barcode), loc.Name, loc.Type, loc.IsPallet,
		ptrToNullString(loc.PalletNumber), ptrToNullDecimal(loc.MaxCapacity),
		loc.CreatedAt, loc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateLocation: exec: %w", err)
	}

	return tx.Commit()
}

const getLocationByIDSQL = `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE id = $1 AND tenant_id = $2`

func (r *WMSRepo) GetLocationByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getLocationByIDSQL, id, tenantID)
	loc, err := scanLocationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLocationNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetLocationByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByID: commit: %w", err)
	}
	return loc, nil
}

const getLocationByCodeSQL = `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE tenant_id = $1 AND ($2::uuid IS NULL AND warehouse_id IS NULL OR warehouse_id = $2) AND code = $3`

func (r *WMSRepo) GetLocationByCode(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, code string) (*domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByCode: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByCode: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getLocationByCodeSQL, tenantID, ptrToNullUUID(warehouseID), code)
	loc, err := scanLocationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLocationNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetLocationByCode: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLocationByCode: commit: %w", err)
	}
	return loc, nil
}

func (r *WMSRepo) GetOrCreateSystemLocation(ctx context.Context, tenantID uuid.UUID, locType domain.LocationType) (*domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: set tenant: %w", err)
	}

	findSQL := `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE tenant_id = $1 AND type = $2 AND warehouse_id IS NULL
LIMIT 1`

	row := tx.QueryRowContext(ctx, findSQL, tenantID, locType)
	loc, err := scanLocationRow(row)
	if err == nil {
		if errCommit := tx.Commit(); errCommit != nil {
			return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: commit: %w", errCommit)
		}
		return loc, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: query existing: %w", err)
	}

	// Create virtual system location
	code := fmt.Sprintf("@%s", locType)
	name := fmt.Sprintf("System Virtual %s", locType)
	newLoc := domain.WarehouseLocation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Code:      code,
		Name:      name,
		Type:      locType,
		IsPallet:  false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err = tx.ExecContext(ctx, createLocationSQL,
		newLoc.ID, newLoc.TenantID, nil, nil,
		newLoc.Code, nil, newLoc.Name, newLoc.Type, newLoc.IsPallet,
		nil, nil, newLoc.CreatedAt, newLoc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateSystemLocation: commit: %w", err)
	}
	return &newLoc, nil
}

func (r *WMSRepo) ListLocations(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLocations: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLocations: set tenant: %w", err)
	}

	listSQL := `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE tenant_id = $1 AND ($2::uuid IS NULL OR warehouse_id = $2)
ORDER BY code ASC`

	rows, err := tx.QueryContext(ctx, listSQL, tenantID, ptrToNullUUID(warehouseID))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLocations: query: %w", err)
	}
	defer rows.Close()

	var result []domain.WarehouseLocation
	for rows.Next() {
		loc, err := scanLocationRows(rows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ListLocations: scan: %w", err)
		}
		result = append(result, *loc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLocations: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLocations: commit: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLocationRow(r rowScanner) (*domain.WarehouseLocation, error) {
	var loc domain.WarehouseLocation
	var whID, parentID, barcode, palletNum sql.NullString
	var maxCap decimal.NullDecimal

	err := r.Scan(
		&loc.ID, &loc.TenantID, &whID, &parentID,
		&loc.Code, &barcode, &loc.Name, &loc.Type,
		&loc.IsPallet, &palletNum, &maxCap,
		&loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	loc.WarehouseID = nullUUIDToPtr(whID)
	loc.ParentID = nullUUIDToPtr(parentID)
	loc.Barcode = nullStringToPtr(barcode)
	loc.PalletNumber = nullStringToPtr(palletNum)
	loc.MaxCapacity = nullDecimalToPtr(maxCap)
	return &loc, nil
}

func scanLocationRows(rows *sql.Rows) (*domain.WarehouseLocation, error) {
	return scanLocationRow(rows)
}

// -----------------------------------------------------------------------------
// 4. Barcodes & Mappings
// -----------------------------------------------------------------------------

const createBarcodeSQL = `
INSERT INTO product_barcodes (id, tenant_id, product_id, barcode, barcode_symbology, uom_name, multiplier, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

func (r *WMSRepo) CreateBarcode(ctx context.Context, b *domain.ProductBarcode) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateBarcode: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, b.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateBarcode: set tenant: %w", err)
	}

	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	if b.Multiplier.IsZero() {
		b.Multiplier = decimal.NewFromInt(1)
	}
	if b.BarcodeSymbology == "" {
		b.BarcodeSymbology = "CODE128"
	}
	if b.UOMName == "" {
		b.UOMName = "PCS"
	}

	_, err = tx.ExecContext(ctx, createBarcodeSQL,
		b.ID, b.TenantID, b.ProductID, b.Barcode, b.BarcodeSymbology, b.UOMName, b.Multiplier, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateBarcode: exec: %w", err)
	}

	return tx.Commit()
}

const createSKUMappingSQL = `
INSERT INTO product_sku_mappings (id, tenant_id, product_id, mapping_type, channel_name, external_sku, external_name, multiplier, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (tenant_id, channel_name, external_sku)
DO UPDATE SET product_id = EXCLUDED.product_id,
              mapping_type = EXCLUDED.mapping_type,
              external_name = EXCLUDED.external_name,
              multiplier = EXCLUDED.multiplier,
              updated_at = EXCLUDED.updated_at`

func (r *WMSRepo) CreateSKUMapping(ctx context.Context, m *domain.ProductSKUMapping) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateSKUMapping: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, m.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateSKUMapping: set tenant: %w", err)
	}

	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if m.Multiplier.IsZero() {
		m.Multiplier = decimal.NewFromInt(1)
	}

	_, err = tx.ExecContext(ctx, createSKUMappingSQL,
		m.ID, m.TenantID, m.ProductID, m.MappingType, m.ChannelName,
		m.ExternalSKU, ptrToNullString(m.ExternalName), m.Multiplier,
		m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateSKUMapping: exec: %w", err)
	}

	return tx.Commit()
}

// ResolveBarcode resolves a barcode, product sku, or external sku mapping.
func (r *WMSRepo) ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ResolveBarcode: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ResolveBarcode: set tenant: %w", err)
	}

	// 1. Direct SKU match in products table
	const skuSQL = `
SELECT id, sku, name
FROM products
WHERE tenant_id = $1 AND sku = $2
LIMIT 1`
	var prodID uuid.UUID
	var prodSKU, prodName string
	err = tx.QueryRowContext(ctx, skuSQL, tenantID, code).Scan(&prodID, &prodSKU, &prodName)
	if err == nil {
		_ = tx.Commit()
		return &domain.ResolvedProduct{
			ProductID:  prodID,
			SKU:        prodSKU,
			Name:       prodName,
			Barcode:    "",
			Multiplier: decimal.NewFromInt(1),
			Source:     "SKU",
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.ResolveBarcode: sku lookup: %w", err)
	}

	// 2. Physical barcode match in product_barcodes table
	const barcodeSQL = `
SELECT p.id, p.sku, p.name, b.barcode, b.multiplier
FROM product_barcodes b
JOIN products p ON p.id = b.product_id AND p.tenant_id = b.tenant_id
WHERE b.tenant_id = $1 AND b.barcode = $2
LIMIT 1`
	var barcode string
	var barcodeMult decimal.Decimal
	err = tx.QueryRowContext(ctx, barcodeSQL, tenantID, code).Scan(&prodID, &prodSKU, &prodName, &barcode, &barcodeMult)
	if err == nil {
		_ = tx.Commit()
		return &domain.ResolvedProduct{
			ProductID:  prodID,
			SKU:        prodSKU,
			Name:       prodName,
			Barcode:    barcode,
			Multiplier: barcodeMult,
			Source:     "BARCODE",
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.ResolveBarcode: barcode lookup: %w", err)
	}

	// 3. Omnichannel/Customer external SKU mapping in product_sku_mappings table
	const mappingSQL = `
SELECT p.id, p.sku, p.name, m.external_sku, m.multiplier
FROM product_sku_mappings m
JOIN products p ON p.id = m.product_id AND p.tenant_id = m.tenant_id
WHERE m.tenant_id = $1 AND m.external_sku = $2
LIMIT 1`
	var extSKU string
	var mappingMult decimal.Decimal
	err = tx.QueryRowContext(ctx, mappingSQL, tenantID, code).Scan(&prodID, &prodSKU, &prodName, &extSKU, &mappingMult)
	if err == nil {
		_ = tx.Commit()
		return &domain.ResolvedProduct{
			ProductID:   prodID,
			SKU:         prodSKU,
			Name:        prodName,
			ExternalSKU: extSKU,
			Multiplier:  mappingMult,
			Source:      "MAPPING",
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.ResolveBarcode: mapping lookup: %w", err)
	}

	return nil, domain.ErrBarcodeNotFound
}

// -----------------------------------------------------------------------------
// 5. Stock Movements & Ledger
// -----------------------------------------------------------------------------

const createStockMovementSQL = `
INSERT INTO stock_movements (id, tenant_id, movement_number, product_id, source_location_id, dest_location_id, quantity, unit_cost, status, reference_type, reference_id, executed_by, batch_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

func (r *WMSRepo) CreateStockMovement(ctx context.Context, m *domain.StockMovement) error {
	if m == nil {
		return domain.ErrInvalidInput
	}
	if m.BatchID == nil || *m.BatchID == uuid.Nil {
		return domain.ErrBatchRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovement: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, m.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovement: set tenant: %w", err)
	}

	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if m.Status == "" {
		m.Status = domain.StockMovementStatusDone
	}

	_, err = tx.ExecContext(ctx, createStockMovementSQL,
		m.ID, m.TenantID, m.MovementNumber, m.ProductID,
		m.SourceLocationID, m.DestLocationID, m.Quantity,
		m.UnitCost, m.Status, m.ReferenceType, m.ReferenceID,
		ptrToNullUUID(m.ExecutedBy), ptrToNullUUID(m.BatchID), m.CreatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovement: exec: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"movement_number":    m.MovementNumber,
		"product_id":         m.ProductID,
		"source_location_id": m.SourceLocationID,
		"dest_location_id":   m.DestLocationID,
		"quantity":           m.Quantity,
		"reference_type":     m.ReferenceType,
		"reference_id":       m.ReferenceID,
		"batch_id":           m.BatchID,
	})
	if err := r.WriteAuditTx(ctx, tx, m.TenantID, m.ExecutedBy, "stock_movement", m.ID, "created", auditDetails); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovement: audit: %w", err)
	}

	return tx.Commit()
}

// GetStockByLocation calculates aggregated inventory at a physical location:
// sum(inbound done movements) - sum(outbound done movements).
const getStockByLocationSQL = `
SELECT COALESCE(
    SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE 0 END) -
    SUM(CASE WHEN source_location_id = $2 THEN quantity ELSE 0 END),
    0
)
FROM stock_movements
WHERE tenant_id = $1
  AND product_id = $3
  AND status = 'DONE'
  AND (dest_location_id = $2 OR source_location_id = $2)`

func (r *WMSRepo) GetStockByLocation(ctx context.Context, tenantID, locationID, productID uuid.UUID) (decimal.Decimal, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetStockByLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetStockByLocation: set tenant: %w", err)
	}

	var stock decimal.Decimal
	err = tx.QueryRowContext(ctx, getStockByLocationSQL, tenantID, locationID, productID).Scan(&stock)
	if err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetStockByLocation: query: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetStockByLocation: commit: %w", err)
	}
	return stock, nil
}

// LockLocationStock acquires a transaction-scoped advisory lock for concurrency control on location stock.
func (r *WMSRepo) LockLocationStock(ctx context.Context, tx *sql.Tx, tenantID, locationID, productID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1::text || $2::text || $3::text))", tenantID, locationID, productID)
	if err != nil {
		return fmt.Errorf("WMSRepo.LockLocationStock: advisory lock: %w", err)
	}
	return nil
}

const deductStockCalcSQL = `
SELECT COALESCE(
    SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
    0
)
FROM stock_movements
WHERE tenant_id = $1
  AND product_id = $3
  AND (source_location_id = $2 OR dest_location_id = $2)
  AND status = 'DONE'`

// DeductLocationStock atomically checks stock and inserts movement under an advisory transaction lock.
// If mov.BatchID is nil, it allocates stock via FEFO (OCA §1.3) from on-hand batches at locationID.
func (r *WMSRepo) DeductLocationStock(ctx context.Context, tenantID, locationID, productID uuid.UUID, qty decimal.Decimal, mov *domain.StockMovement) error {
	if !qty.IsPositive() {
		return domain.ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeductLocationStock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.DeductLocationStock: set tenant: %w", err)
	}

	// 1) Advisory transaction lock
	if err := r.LockLocationStock(ctx, tx, tenantID, locationID, productID); err != nil {
		return err
	}

	// 1b) Quarantine stock is blocked (ADR-014 Invariant 3, sentry-wms 1.1): it can only leave
	// the bin through the QC release / scrap flow (MoveQuarantineStock), never via
	// transfer, delivery order, POS or opname deduction.
	var srcType string
	if err := tx.QueryRowContext(ctx, `SELECT type FROM warehouse_locations WHERE id = $1 AND tenant_id = $2`, locationID, tenantID).Scan(&srcType); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("WMSRepo.DeductLocationStock: location type: %w", err)
	}
	if domain.LocationType(srcType) == domain.LocationTypeQuarantine {
		return domain.ErrQuarantineStockBlocked
	}

	// 2) If specific BatchID is requested: check that batch's balance directly
	if mov.BatchID != nil && *mov.BatchID != uuid.Nil {
		var currentStock decimal.Decimal
		batchCalcSQL := `
SELECT COALESCE(
    SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
    0
)
FROM stock_movements
WHERE tenant_id = $1 AND product_id = $3 AND batch_id = $4
  AND (source_location_id = $2 OR dest_location_id = $2)
  AND status = 'DONE'`
		if err := tx.QueryRowContext(ctx, batchCalcSQL, tenantID, locationID, productID, *mov.BatchID).Scan(&currentStock); err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: scan batch stock: %w", err)
		}
		if currentStock.LessThan(qty) {
			return domain.ErrInsufficientStock
		}

		if mov.ID == uuid.Nil {
			mov.ID = uuid.New()
		}
		if mov.CreatedAt.IsZero() {
			mov.CreatedAt = time.Now().UTC()
		}
		if mov.Status == "" {
			mov.Status = domain.StockMovementStatusDone
		}
		_, err = tx.ExecContext(ctx, createStockMovementSQL,
			mov.ID, mov.TenantID, mov.MovementNumber, mov.ProductID,
			mov.SourceLocationID, mov.DestLocationID, mov.Quantity,
			mov.UnitCost, mov.Status, mov.ReferenceType, mov.ReferenceID,
			ptrToNullUUID(mov.ExecutedBy), ptrToNullUUID(mov.BatchID), mov.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: exec movement: %w", err)
		}

		auditDetails, _ := json.Marshal(map[string]any{
			"movement_number":    mov.MovementNumber,
			"product_id":         mov.ProductID,
			"source_location_id": mov.SourceLocationID,
			"dest_location_id":   mov.DestLocationID,
			"quantity":           mov.Quantity,
			"reference_type":     mov.ReferenceType,
			"reference_id":       mov.ReferenceID,
			"batch_id":           mov.BatchID,
		})
		if err := r.WriteAuditTx(ctx, tx, tenantID, mov.ExecutedBy, "stock_movement", mov.ID, "created", auditDetails); err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: audit: %w", err)
		}
		return tx.Commit()
	}

	// 3) BatchID is nil: query all on-hand batches at this location and allocate via FEFO
	includeOnHold := (mov.ReferenceType == domain.StockRefScrap || mov.ReferenceType == domain.StockRefOpname)
	balQuery := `
SELECT b.id, b.batch_number, b.expiry_date, b.status, loc.id, loc.code, sm.product_id,
       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty,
       b.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.product_id = $2 AND loc.id = $3 AND sm.status = 'DONE'
GROUP BY b.id, b.batch_number, b.expiry_date, b.status, b.created_at, loc.id, loc.code, sm.product_id
HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
ORDER BY b.expiry_date ASC NULLS LAST, b.created_at ASC, b.id ASC`

	rows, err := tx.QueryContext(ctx, balQuery, tenantID, productID, locationID)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeductLocationStock: query batch balances: %w", err)
	}
	defer rows.Close()

	var balances []domain.BatchBalance
	for rows.Next() {
		var bal domain.BatchBalance
		var exp sql.NullTime
		if err := rows.Scan(
			&bal.BatchID, &bal.BatchNumber, &exp, &bal.Status,
			&bal.LocationID, &bal.LocationCode, &bal.ProductID,
			&bal.Quantity, &bal.CreatedAt,
		); err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: scan: %w", err)
		}
		bal.ExpiryDate = nullTimeToPtr(exp)
		balances = append(balances, bal)
	}
	_ = rows.Close()

	allocs, err := domain.AllocateFEFO(balances, qty, includeOnHold)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for i, alloc := range allocs {
		m := *mov
		m.ID = uuid.New()
		m.TenantID = tenantID
		m.ProductID = productID
		m.BatchID = &alloc.BatchID
		m.SourceLocationID = locationID
		m.Quantity = alloc.Quantity
		m.CreatedAt = now
		if m.Status == "" {
			m.Status = domain.StockMovementStatusDone
		}
		if len(allocs) > 1 {
			m.MovementNumber = fmt.Sprintf("%s-B%d", mov.MovementNumber, i+1)
		}

		_, err := tx.ExecContext(ctx, createStockMovementSQL,
			m.ID, m.TenantID, m.MovementNumber, m.ProductID,
			m.SourceLocationID, m.DestLocationID, m.Quantity,
			m.UnitCost, m.Status, m.ReferenceType, m.ReferenceID,
			ptrToNullUUID(m.ExecutedBy), ptrToNullUUID(m.BatchID), m.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: insert movement %d: %w", i, err)
		}

		auditDetails, _ := json.Marshal(map[string]any{
			"movement_number":    m.MovementNumber,
			"product_id":         m.ProductID,
			"source_location_id": m.SourceLocationID,
			"dest_location_id":   m.DestLocationID,
			"quantity":           m.Quantity,
			"reference_type":     m.ReferenceType,
			"reference_id":       m.ReferenceID,
			"batch_id":           m.BatchID,
		})
		if err := r.WriteAuditTx(ctx, tx, tenantID, m.ExecutedBy, "stock_movement", m.ID, "created", auditDetails); err != nil {
			return fmt.Errorf("WMSRepo.DeductLocationStock: audit movement %d: %w", i, err)
		}
	}

	return tx.Commit()
}

func (r *WMSRepo) ListStockMovements(ctx context.Context, tenantID uuid.UUID, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockMovements: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockMovements: set tenant: %w", err)
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `
SELECT 
    sm.id, sm.tenant_id, sm.movement_number, sm.product_id,
    COALESCE(p.name, ''), COALESCE(p.sku, ''),
    sm.source_location_id, COALESCE(sl.code, ''),
    sm.dest_location_id, COALESCE(dl.code, ''),
    sm.quantity, sm.unit_cost, sm.status, sm.reference_type, sm.reference_id,
    sm.batch_id, COALESCE(b.batch_number, ''),
    sm.executed_by, COALESCE(u.full_name, u.email, ''), sm.created_at
FROM stock_movements sm
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations sl ON sl.id = sm.source_location_id AND sl.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations dl ON dl.id = sm.dest_location_id AND dl.tenant_id = sm.tenant_id
LEFT JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
LEFT JOIN users u ON u.id = sm.executed_by
WHERE sm.tenant_id = $1
  AND ($2::uuid IS NULL OR sm.product_id = $2)
  AND ($3::uuid IS NULL OR sm.source_location_id = $3 OR sm.dest_location_id = $3)
ORDER BY sm.created_at DESC
LIMIT $4`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(productID), ptrToNullUUID(locationID), limit)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockMovements: query: %w", err)
	}
	defer rows.Close()

	var result []domain.StockMovement
	for rows.Next() {
		var m domain.StockMovement
		var execBy, batchID sql.NullString
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.MovementNumber, &m.ProductID,
			&m.ProductName, &m.SKU,
			&m.SourceLocationID, &m.SourceLocationCode,
			&m.DestLocationID, &m.DestLocationCode,
			&m.Quantity, &m.UnitCost, &m.Status, &m.ReferenceType, &m.ReferenceID,
			&batchID, &m.BatchNumber,
			&execBy, &m.ExecutedByName, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockMovements: scan: %w", err)
		}
		m.BatchID = nullUUIDToPtr(batchID)
		m.ExecutedBy = nullUUIDToPtr(execBy)
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockMovements: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockMovements: commit: %w", err)
	}
	return result, nil
}

func (r *WMSRepo) ListStockSummary(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockSummary: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockSummary: set tenant: %w", err)
	}

	query := `
WITH movement_stock AS (
    SELECT 
        sm.product_id,
        loc.warehouse_id,
        loc.id AS location_id,
        loc.code AS location_code,
        loc.type AS location_type,
        SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty
    FROM stock_movements sm
    JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
    WHERE sm.tenant_id = $1 AND sm.status = 'DONE'
    GROUP BY sm.product_id, loc.warehouse_id, loc.id, loc.code, loc.type
),
allocated_stock AS (
    SELECT
        doi.product_id,
        doi.location_id,
        SUM(doi.quantity) AS allocated_qty
    FROM delivery_order_items doi
    JOIN delivery_orders do ON do.id = doi.delivery_order_id AND do.tenant_id = doi.tenant_id
    WHERE doi.tenant_id = $1
      AND do.status NOT IN ('SHIPPED', 'CANCELLED', 'RETURNED')
    GROUP BY doi.product_id, doi.location_id
)
SELECT 
    p.id AS product_id,
    p.sku,
    p.name AS product_name,
    ms.warehouse_id,
    COALESCE(w.name, 'Gudang Utama') AS warehouse_name,
    ms.location_id,
    COALESCE(ms.location_code, 'MAIN') AS location_code,
    COALESCE(ms.qty, 0) AS quantity,
    COALESCE(als.allocated_qty, 0) AS allocated_qty,
    CASE 
        WHEN ms.location_type IN ('QUARANTINE', 'STAGING_INBOUND', 'STAGING') THEN 0
        ELSE GREATEST(0, COALESCE(ms.qty, 0) - COALESCE(als.allocated_qty, 0))
    END AS available_qty
FROM movement_stock ms
JOIN products p ON p.id = ms.product_id AND p.tenant_id = $1
LEFT JOIN warehouses w ON w.id = ms.warehouse_id AND w.tenant_id = $1
LEFT JOIN allocated_stock als ON als.product_id = ms.product_id AND als.location_id = ms.location_id
WHERE ($2::uuid IS NULL OR ms.warehouse_id = $2)
  AND ms.qty > 0
ORDER BY p.name ASC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockSummary: query: %w", err)
	}
	defer rows.Close()

	var result []domain.StockSummary
	for rows.Next() {
		var s domain.StockSummary
		var whID uuid.UUID
		var locID uuid.UUID
		if err := rows.Scan(
			&s.ProductID, &s.SKU, &s.ProductName,
			&whID, &s.WarehouseName,
			&locID, &s.LocationCode,
			&s.Quantity, &s.AllocatedQty, &s.AvailableQty,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockSummary: scan: %w", err)
		}
		s.WarehouseID = &whID
		s.LocationID = &locID
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockSummary: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockSummary: commit: %w", err)
	}
	return result, nil
}

func (r *WMSRepo) GetAvailableStock(ctx context.Context, tenantID, warehouseID uuid.UUID, locationID *uuid.UUID, productID uuid.UUID) (decimal.Decimal, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: set tenant: %w", err)
	}

	if locationID != nil && *locationID != uuid.Nil {
		var locType string
		var onHand decimal.Decimal
		err := tx.QueryRowContext(ctx, `
SELECT loc.type,
       COALESCE(SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END), 0)
FROM warehouse_locations loc
LEFT JOIN stock_movements sm ON (sm.dest_location_id = loc.id OR sm.source_location_id = loc.id)
    AND sm.tenant_id = $1 AND sm.product_id = $3 AND sm.status = 'DONE'
LEFT JOIN stock_batches sb ON sb.id = sm.batch_id AND sb.tenant_id = sm.tenant_id
WHERE loc.id = $2 AND loc.tenant_id = $1
  AND (sb.id IS NULL OR sb.status = 'RELEASED')
GROUP BY loc.id, loc.type`, tenantID, *locationID, productID).Scan(&locType, &onHand)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return decimal.Zero, nil
			}
			return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: query loc onhand: %w", err)
		}
		if locType != string(domain.LocationTypeInternal) {
			_ = tx.Commit()
			return decimal.Zero, nil
		}

		var allocated decimal.Decimal
		err = tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(doi.quantity), 0)
FROM delivery_order_items doi
JOIN delivery_orders do ON do.id = doi.delivery_order_id AND do.tenant_id = doi.tenant_id
WHERE doi.tenant_id = $1 AND doi.location_id = $2 AND doi.product_id = $3
  AND do.status NOT IN ('SHIPPED', 'CANCELLED', 'RETURNED')`, tenantID, *locationID, productID).Scan(&allocated)
		if err != nil {
			return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: query loc allocated: %w", err)
		}

		avail := onHand.Sub(allocated)
		if avail.IsNegative() {
			avail = decimal.Zero
		}
		if err := tx.Commit(); err != nil {
			return decimal.Zero, err
		}
		return avail, nil
	}

	// Warehouse level check across INTERNAL racks
	var onHand decimal.Decimal
	err = tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END), 0)
FROM warehouse_locations loc
JOIN stock_movements sm ON (sm.dest_location_id = loc.id OR sm.source_location_id = loc.id)
    AND sm.tenant_id = $1 AND sm.product_id = $3 AND sm.status = 'DONE'
LEFT JOIN stock_batches sb ON sb.id = sm.batch_id AND sb.tenant_id = sm.tenant_id
WHERE loc.warehouse_id = $2 AND loc.tenant_id = $1 AND loc.type = 'INTERNAL'
  AND (sb.id IS NULL OR sb.status = 'RELEASED')`, tenantID, warehouseID, productID).Scan(&onHand)
	if err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: query wh onhand: %w", err)
	}

	var allocated decimal.Decimal
	err = tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(doi.quantity), 0)
FROM delivery_order_items doi
JOIN delivery_orders do ON do.id = doi.delivery_order_id AND do.tenant_id = doi.tenant_id
WHERE doi.tenant_id = $1 AND do.warehouse_id = $2 AND doi.product_id = $3
  AND do.status NOT IN ('SHIPPED', 'CANCELLED', 'RETURNED')`, tenantID, warehouseID, productID).Scan(&allocated)
	if err != nil {
		return decimal.Zero, fmt.Errorf("WMSRepo.GetAvailableStock: query wh allocated: %w", err)
	}

	avail := onHand.Sub(allocated)
	if avail.IsNegative() {
		avail = decimal.Zero
	}
	if err := tx.Commit(); err != nil {
		return decimal.Zero, err
	}
	return avail, nil
}

// -----------------------------------------------------------------------------
// 6. Stock Transfers
// -----------------------------------------------------------------------------

const createTransferSQL = `
INSERT INTO stock_transfers (id, tenant_id, transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, approved_by, vehicle_plate, driver_name, dispatched_at, received_at, notes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

const createTransferItemSQL = `
INSERT INTO stock_transfer_items (id, tenant_id, transfer_id, product_id, requested_qty, sent_qty, received_qty, source_location_id, dest_location_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func (r *WMSRepo) CreateTransfer(ctx context.Context, t *domain.StockTransfer, items []domain.StockTransferItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateTransfer: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, t.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateTransfer: set tenant: %w", err)
	}

	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	if t.Status == "" {
		t.Status = domain.TransferStatusDraft
	}

	_, err = tx.ExecContext(ctx, createTransferSQL,
		t.ID, t.TenantID, t.TransferNumber, t.FromWarehouseID, t.ToWarehouseID,
		t.Status, t.RequestedBy, ptrToNullUUID(t.ApprovedBy),
		ptrToNullString(t.VehiclePlate), ptrToNullString(t.DriverName),
		ptrToNullTime(t.DispatchedAt), ptrToNullTime(t.ReceivedAt),
		ptrToNullString(t.Notes), t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateTransfer: exec header: %w", err)
	}

	for i := range items {
		item := &items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.TenantID = t.TenantID
		item.TransferID = t.ID
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}

		_, err = tx.ExecContext(ctx, createTransferItemSQL,
			item.ID, item.TenantID, item.TransferID, item.ProductID,
			item.RequestedQty, item.SentQty, item.ReceivedQty,
			ptrToNullUUID(item.SourceLocationID), ptrToNullUUID(item.DestLocationID),
			item.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.CreateTransfer: exec item %d: %w", i, err)
		}
	}

	return tx.Commit()
}

const getTransferByIDSQL = `
SELECT id, tenant_id, transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, approved_by, vehicle_plate, driver_name, dispatched_at, received_at, notes, rejection_reason, created_at, updated_at
FROM stock_transfers
WHERE id = $1 AND tenant_id = $2`

const getTransferItemsSQL = `
SELECT id, tenant_id, transfer_id, product_id, requested_qty, sent_qty, received_qty, source_location_id, dest_location_id, created_at
FROM stock_transfer_items
WHERE transfer_id = $1 AND tenant_id = $2`

func (r *WMSRepo) GetTransferByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getTransferByIDSQL, id, tenantID)
	var t domain.StockTransfer
	var appBy, vehPlate, drvName, notes, rejReason sql.NullString
	var dispAt, recvAt sql.NullTime

	err = row.Scan(
		&t.ID, &t.TenantID, &t.TransferNumber, &t.FromWarehouseID, &t.ToWarehouseID,
		&t.Status, &t.RequestedBy, &appBy, &vehPlate, &drvName,
		&dispAt, &recvAt, &notes, &rejReason, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, domain.ErrTransferNotFound
		}
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: scan header: %w", err)
	}
	t.ApprovedBy = nullUUIDToPtr(appBy)
	t.VehiclePlate = nullStringToPtr(vehPlate)
	t.DriverName = nullStringToPtr(drvName)
	t.DispatchedAt = nullTimeToPtr(dispAt)
	t.ReceivedAt = nullTimeToPtr(recvAt)
	t.Notes = nullStringToPtr(notes)
	t.RejectionReason = nullStringToPtr(rejReason)

	rows, err := tx.QueryContext(ctx, getTransferItemsSQL, id, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: query items: %w", err)
	}
	defer rows.Close()

	var items []domain.StockTransferItem
	for rows.Next() {
		var it domain.StockTransferItem
		var srcLoc, dstLoc sql.NullString
		if err := rows.Scan(
			&it.ID, &it.TenantID, &it.TransferID, &it.ProductID,
			&it.RequestedQty, &it.SentQty, &it.ReceivedQty,
			&srcLoc, &dstLoc, &it.CreatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: scan item: %w", err)
		}
		it.SourceLocationID = nullUUIDToPtr(srcLoc)
		it.DestLocationID = nullUUIDToPtr(dstLoc)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: items err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetTransferByID: commit: %w", err)
	}
	return &t, items, nil
}

const updateTransferStatusSQL = `
UPDATE stock_transfers
SET status = $3,
    dispatched_at = COALESCE($4, dispatched_at),
    received_at = COALESCE($5, received_at),
    approved_by = COALESCE($6, approved_by),
    rejection_reason = COALESCE($7, rejection_reason),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2`

func (r *WMSRepo) UpdateTransferStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TransferStatus, dispatchedAt, receivedAt *time.Time, approvedBy *uuid.UUID, rejectionReason *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateTransferStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateTransferStatus: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateTransferStatusSQL,
		id, tenantID, status, ptrToNullTime(dispatchedAt), ptrToNullTime(receivedAt), ptrToNullUUID(approvedBy), ptrToNullString(rejectionReason))
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateTransferStatus: exec: %w", err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateTransferStatus: rows affected: %w", err)
	}
	if rowsAff == 0 {
		return domain.ErrTransferNotFound
	}

	return tx.Commit()
}

func (r *WMSRepo) ListTransfers(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockTransfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListTransfers: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListTransfers: set tenant: %w", err)
	}

	query := `
SELECT id, tenant_id, transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, approved_by, vehicle_plate, driver_name, dispatched_at, received_at, notes, created_at, updated_at
FROM stock_transfers
WHERE tenant_id = $1
  AND ($2::uuid IS NULL OR from_warehouse_id = $2 OR to_warehouse_id = $2)
ORDER BY created_at DESC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListTransfers: query: %w", err)
	}
	defer rows.Close()

	var result []domain.StockTransfer
	for rows.Next() {
		var t domain.StockTransfer
		var appBy, vehPlate, drvName, notes sql.NullString
		var dispAt, recvAt sql.NullTime

		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.TransferNumber, &t.FromWarehouseID, &t.ToWarehouseID,
			&t.Status, &t.RequestedBy, &appBy, &vehPlate, &drvName,
			&dispAt, &recvAt, &notes, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListTransfers: scan: %w", err)
		}
		t.ApprovedBy = nullUUIDToPtr(appBy)
		t.VehiclePlate = nullStringToPtr(vehPlate)
		t.DriverName = nullStringToPtr(drvName)
		t.DispatchedAt = nullTimeToPtr(dispAt)
		t.ReceivedAt = nullTimeToPtr(recvAt)
		t.Notes = nullStringToPtr(notes)

		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListTransfers: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListTransfers: commit: %w", err)
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// 7. Delivery Orders (Surat Jalan)
// -----------------------------------------------------------------------------

const createDeliveryOrderSQL = `
INSERT INTO delivery_orders (
    id, tenant_id, sales_order_id, warehouse_id, do_number, status,
    expedition_name, tracking_number, driver_name, vehicle_plate, recipient_name, received_date,
    customer_id, created_by, confirmed_by, packed_by, dispatched_by,
    package_weight_kg, package_length_cm, package_width_cm, package_height_cm, packaging_type, order_type,
    created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)`

const createDeliveryOrderItemSQL = `
INSERT INTO delivery_order_items (
    id, tenant_id, delivery_order_id, product_id, quantity, location_id,
    batch_id, is_free_item, packed_qty, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func (r *WMSRepo) CreateDeliveryOrder(ctx context.Context, do *domain.DeliveryOrder, items []domain.DeliveryOrderItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateDeliveryOrder: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, do.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateDeliveryOrder: set tenant: %w", err)
	}

	if do.ID == uuid.Nil {
		do.ID = uuid.New()
	}
	now := time.Now().UTC()
	if do.CreatedAt.IsZero() {
		do.CreatedAt = now
	}
	if do.UpdatedAt.IsZero() {
		do.UpdatedAt = now
	}
	if do.Status == "" {
		do.Status = domain.DeliveryOrderStatusDraft
	}
	if do.OrderType == "" {
		do.OrderType = "DIRECT_DO"
	}

	_, err = tx.ExecContext(ctx, createDeliveryOrderSQL,
		do.ID, do.TenantID, ptrToNullUUID(do.SalesOrderID), do.WarehouseID, do.DONumber,
		do.Status, ptrToNullString(do.ExpeditionName), ptrToNullString(do.TrackingNumber),
		ptrToNullString(do.DriverName), ptrToNullString(do.VehiclePlate),
		ptrToNullString(do.RecipientName), ptrToNullTime(do.ReceivedDate),
		ptrToNullUUID(do.CustomerID), ptrToNullUUID(do.CreatedBy), ptrToNullUUID(do.ConfirmedBy),
		ptrToNullUUID(do.PackedBy), ptrToNullUUID(do.DispatchedBy),
		decPtr(do.PackageWeightKg), decPtr(do.PackageLengthCm), decPtr(do.PackageWidthCm), decPtr(do.PackageHeightCm),
		ptrToNullString(do.PackagingType), do.OrderType,
		do.CreatedAt, do.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23503": // FK violation
				if strings.Contains(pqErr.Constraint, "sales_order") || strings.Contains(pqErr.Detail, "sales_order_id") {
					return &domain.StockReceiptValidationError{Msg: "Ref. Sales Order tidak ditemukan. Kosongkan jika Surat Jalan tanpa Sales Order."}
				}
			case "23505": // UNIQUE (tenant_id, do_number)
				return &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Nomor Surat Jalan %s sudah dipakai.", do.DONumber)}
			}
		}
		return fmt.Errorf("WMSRepo.CreateDeliveryOrder: exec header: %w", err)
	}

	for i := range items {
		it := &items[i]
		if it.ID == uuid.Nil {
			it.ID = uuid.New()
		}
		it.TenantID = do.TenantID
		it.DeliveryOrderID = do.ID
		if it.CreatedAt.IsZero() {
			it.CreatedAt = now
		}

		_, err = tx.ExecContext(ctx, createDeliveryOrderItemSQL,
			it.ID, it.TenantID, it.DeliveryOrderID, it.ProductID, it.Quantity, it.LocationID,
			ptrToNullUUID(it.BatchID), it.IsFreeItem, it.PackedQty, it.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.CreateDeliveryOrder: exec item %d: %w", i, err)
		}
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"do_number":      do.DONumber,
		"warehouse_id":   do.WarehouseID,
		"order_type":     do.OrderType,
		"item_count":     len(items),
		"customer_id":    do.CustomerID,
		"sales_order_id": do.SalesOrderID,
	})
	if err := r.WriteAuditTx(ctx, tx, do.TenantID, do.CreatedBy, "delivery_order", do.ID, "created", auditDetails); err != nil {
		return fmt.Errorf("WMSRepo.CreateDeliveryOrder: audit: %w", err)
	}

	return tx.Commit()
}

const getDeliveryOrderByIDSQL = `
SELECT d.id, d.tenant_id, d.sales_order_id, d.warehouse_id, d.do_number, d.status,
       d.expedition_name, d.tracking_number, d.driver_name, d.vehicle_plate, d.recipient_name, d.received_date,
       d.customer_id, COALESCE(c.name, ''),
       d.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
       d.confirmed_by, COALESCE(u_cf.full_name, u_cf.email, ''),
       d.packed_by, COALESCE(u_pk.full_name, u_pk.email, ''),
       d.dispatched_by, COALESCE(u_ds.full_name, u_ds.email, ''),
       d.package_weight_kg, d.package_length_cm, d.package_width_cm, d.package_height_cm,
       d.packaging_type, COALESCE(d.order_type, 'DIRECT_DO'),
       d.created_at, d.updated_at
FROM delivery_orders d
LEFT JOIN customers c ON c.id = d.customer_id AND c.tenant_id = d.tenant_id
LEFT JOIN users u_cr ON u_cr.id = d.created_by AND u_cr.tenant_id = d.tenant_id
LEFT JOIN users u_cf ON u_cf.id = d.confirmed_by AND u_cf.tenant_id = d.tenant_id
LEFT JOIN users u_pk ON u_pk.id = d.packed_by AND u_pk.tenant_id = d.tenant_id
LEFT JOIN users u_ds ON u_ds.id = d.dispatched_by AND u_ds.tenant_id = d.tenant_id
WHERE d.id = $1 AND d.tenant_id = $2`

const getDeliveryOrderItemsSQL = `
SELECT doi.id, doi.tenant_id, doi.delivery_order_id, doi.product_id, doi.quantity, doi.location_id,
       doi.batch_id, doi.is_free_item, doi.packed_qty, doi.created_at,
       p.name, p.sku, wl.code,
       sb.batch_number, sb.expiry_date
FROM delivery_order_items doi
LEFT JOIN products p ON p.id = doi.product_id AND p.tenant_id = doi.tenant_id
LEFT JOIN warehouse_locations wl ON wl.id = doi.location_id AND wl.tenant_id = doi.tenant_id
LEFT JOIN stock_batches sb ON sb.id = doi.batch_id AND sb.tenant_id = doi.tenant_id
WHERE doi.delivery_order_id = $1 AND doi.tenant_id = $2
ORDER BY sb.expiry_date ASC NULLS LAST, doi.created_at ASC, doi.id ASC`

func (r *WMSRepo) GetDeliveryOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getDeliveryOrderByIDSQL, id, tenantID)
	var do domain.DeliveryOrder
	var soID, exp, trk, drv, veh, rec, cID, cName sql.NullString
	var uCr, uCrName, uCf, uCfName, uPk, uPkName, uDs, uDsName sql.NullString
	var pkgType sql.NullString
	var recDate sql.NullTime
	var pWeight, pLength, pWidth, pHeight decimal.NullDecimal

	err = row.Scan(
		&do.ID, &do.TenantID, &soID, &do.WarehouseID, &do.DONumber,
		&do.Status, &exp, &trk, &drv, &veh, &rec, &recDate,
		&cID, &cName,
		&uCr, &uCrName, &uCf, &uCfName, &uPk, &uPkName, &uDs, &uDsName,
		&pWeight, &pLength, &pWidth, &pHeight,
		&pkgType, &do.OrderType,
		&do.CreatedAt, &do.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, domain.ErrDeliveryOrderNotFound
		}
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: scan header: %w", err)
	}
	do.SalesOrderID = nullUUIDToPtr(soID)
	do.ExpeditionName = nullStringToPtr(exp)
	do.TrackingNumber = nullStringToPtr(trk)
	do.DriverName = nullStringToPtr(drv)
	do.VehiclePlate = nullStringToPtr(veh)
	do.RecipientName = nullStringToPtr(rec)
	do.ReceivedDate = nullTimeToPtr(recDate)
	do.CustomerID = nullUUIDToPtr(cID)
	do.CustomerName = nullStringToPtr(cName)
	do.CreatedBy = nullUUIDToPtr(uCr)
	do.CreatedByName = nullStringToPtr(uCrName)
	do.ConfirmedBy = nullUUIDToPtr(uCf)
	do.ConfirmedByName = nullStringToPtr(uCfName)
	do.PackedBy = nullUUIDToPtr(uPk)
	do.PackedByName = nullStringToPtr(uPkName)
	do.DispatchedBy = nullUUIDToPtr(uDs)
	do.DispatchedByName = nullStringToPtr(uDsName)
	do.PackageWeightKg = nullDecimalToPtr(pWeight)
	do.PackageLengthCm = nullDecimalToPtr(pLength)
	do.PackageWidthCm = nullDecimalToPtr(pWidth)
	do.PackageHeightCm = nullDecimalToPtr(pHeight)
	do.PackagingType = nullStringToPtr(pkgType)

	rows, err := tx.QueryContext(ctx, getDeliveryOrderItemsSQL, id, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: query items: %w", err)
	}
	defer rows.Close()

	var items []domain.DeliveryOrderItem
	for rows.Next() {
		var it domain.DeliveryOrderItem
		var bID, prodName, prodSKU, locCode, bNum sql.NullString
		var expDate sql.NullTime
		if err := rows.Scan(
			&it.ID, &it.TenantID, &it.DeliveryOrderID, &it.ProductID, &it.Quantity, &it.LocationID,
			&bID, &it.IsFreeItem, &it.PackedQty, &it.CreatedAt,
			&prodName, &prodSKU, &locCode,
			&bNum, &expDate,
		); err != nil {
			return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: scan item: %w", err)
		}
		it.BatchID = nullUUIDToPtr(bID)
		it.ProductName = nullStringToPtr(prodName)
		it.ProductSKU = nullStringToPtr(prodSKU)
		it.LocationCode = nullStringToPtr(locCode)
		it.BatchNumber = nullStringToPtr(bNum)
		it.ExpiryDate = nullTimeToPtr(expDate)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: items err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetDeliveryOrderByID: commit: %w", err)
	}
	return &do, items, nil
}

const updateDeliveryOrderStatusSQL = `
UPDATE delivery_orders
SET status = $3,
    received_date = COALESCE($4, received_date),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND status != $3`

func (r *WMSRepo) ConfirmDeliveryOrder(ctx context.Context, tenantID, id, userID uuid.UUID) (*domain.DeliveryOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: set tenant: %w", err)
	}

	var status string
	var doNumber string
	err = tx.QueryRowContext(ctx, `SELECT status, do_number FROM delivery_orders WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, id, tenantID).Scan(&status, &doNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeliveryOrderNotFound
		}
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: query DO: %w", err)
	}
	if status != string(domain.DeliveryOrderStatusDraft) {
		return nil, domain.ErrInvalidStatus
	}

	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
UPDATE delivery_orders
SET status = 'CONFIRMED', confirmed_by = $3, updated_at = $4
WHERE id = $1 AND tenant_id = $2`, id, tenantID, userID, now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: update status: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"do_number": doNumber,
		"status":    domain.DeliveryOrderStatusConfirmed,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "delivery_order", id, "confirmed", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmDeliveryOrder: commit: %w", err)
	}

	do, _, err := r.GetDeliveryOrderByID(ctx, tenantID, id)
	return do, err
}

func (r *WMSRepo) UpdateDeliveryOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.DeliveryOrderStatus, receivedDate *time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: set tenant: %w", err)
	}

	var currentStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM delivery_orders WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, id, tenantID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrDeliveryOrderNotFound
		}
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: lock DO: %w", err)
	}
	if currentStatus == string(domain.DeliveryOrderStatusShipped) || currentStatus == string(domain.DeliveryOrderStatusCancelled) {
		return domain.ErrInvalidStatus
	}

	res, err := tx.ExecContext(ctx, updateDeliveryOrderStatusSQL, id, tenantID, status, ptrToNullTime(receivedDate))
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: exec: %w", err)
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: rows affected: %w", err)
	}
	if rowsAff == 0 {
		return domain.ErrDeliveryOrderNotFound
	}

	action := strings.ToLower(string(status))
	if status == domain.DeliveryOrderStatusConfirmed {
		action = "confirmed"
	} else if status == domain.DeliveryOrderStatusShipped {
		action = "dispatched"
	}
	auditDetails, _ := json.Marshal(map[string]any{
		"status": status,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, nil, "delivery_order", id, action, auditDetails); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDeliveryOrderStatus: audit: %w", err)
	}

	return tx.Commit()
}

func (r *WMSRepo) ListDeliveryOrders(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: set tenant: %w", err)
	}

	query := `
SELECT d.id, d.tenant_id, d.sales_order_id, d.warehouse_id, d.do_number, d.status,
       d.expedition_name, d.tracking_number, d.driver_name, d.vehicle_plate, d.recipient_name, d.received_date,
       d.customer_id, COALESCE(c.name, ''),
       d.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
       d.confirmed_by, COALESCE(u_cf.full_name, u_cf.email, ''),
       d.packed_by, COALESCE(u_pk.full_name, u_pk.email, ''),
       d.dispatched_by, COALESCE(u_ds.full_name, u_ds.email, ''),
       d.package_weight_kg, d.package_length_cm, d.package_width_cm, d.package_height_cm,
       d.packaging_type, COALESCE(d.order_type, 'DIRECT_DO'),
       d.created_at, d.updated_at
FROM delivery_orders d
LEFT JOIN customers c ON c.id = d.customer_id AND c.tenant_id = d.tenant_id
LEFT JOIN users u_cr ON u_cr.id = d.created_by AND u_cr.tenant_id = d.tenant_id
LEFT JOIN users u_cf ON u_cf.id = d.confirmed_by AND u_cf.tenant_id = d.tenant_id
LEFT JOIN users u_pk ON u_pk.id = d.packed_by AND u_pk.tenant_id = d.tenant_id
LEFT JOIN users u_ds ON u_ds.id = d.dispatched_by AND u_ds.tenant_id = d.tenant_id
WHERE d.tenant_id = $1
  AND ($2::uuid IS NULL OR d.warehouse_id = $2)
ORDER BY d.created_at DESC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: query: %w", err)
	}
	defer rows.Close()

	var result []domain.DeliveryOrder
	for rows.Next() {
		var do domain.DeliveryOrder
		var soID, exp, trk, drv, veh, rec, cID, cName sql.NullString
		var uCr, uCrName, uCf, uCfName, uPk, uPkName, uDs, uDsName sql.NullString
		var pkgType sql.NullString
		var recDate sql.NullTime
		var pWeight, pLength, pWidth, pHeight decimal.NullDecimal

		if err := rows.Scan(
			&do.ID, &do.TenantID, &soID, &do.WarehouseID, &do.DONumber,
			&do.Status, &exp, &trk, &drv, &veh, &rec, &recDate,
			&cID, &cName,
			&uCr, &uCrName, &uCf, &uCfName, &uPk, &uPkName, &uDs, &uDsName,
			&pWeight, &pLength, &pWidth, &pHeight,
			&pkgType, &do.OrderType,
			&do.CreatedAt, &do.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: scan: %w", err)
		}
		do.SalesOrderID = nullUUIDToPtr(soID)
		do.ExpeditionName = nullStringToPtr(exp)
		do.TrackingNumber = nullStringToPtr(trk)
		do.DriverName = nullStringToPtr(drv)
		do.VehiclePlate = nullStringToPtr(veh)
		do.RecipientName = nullStringToPtr(rec)
		do.ReceivedDate = nullTimeToPtr(recDate)
		do.CustomerID = nullUUIDToPtr(cID)
		do.CustomerName = nullStringToPtr(cName)
		do.CreatedBy = nullUUIDToPtr(uCr)
		do.CreatedByName = nullStringToPtr(uCrName)
		do.ConfirmedBy = nullUUIDToPtr(uCf)
		do.ConfirmedByName = nullStringToPtr(uCfName)
		do.PackedBy = nullUUIDToPtr(uPk)
		do.PackedByName = nullStringToPtr(uPkName)
		do.DispatchedBy = nullUUIDToPtr(uDs)
		do.DispatchedByName = nullStringToPtr(uDsName)
		do.PackageWeightKg = nullDecimalToPtr(pWeight)
		do.PackageLengthCm = nullDecimalToPtr(pLength)
		do.PackageWidthCm = nullDecimalToPtr(pWidth)
		do.PackageHeightCm = nullDecimalToPtr(pHeight)
		do.PackagingType = nullStringToPtr(pkgType)

		result = append(result, do)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDeliveryOrders: commit: %w", err)
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// 8. Stock Opname (Physical Inventory Counting)
// -----------------------------------------------------------------------------

const createStockOpnameSQL = `
INSERT INTO stock_opnames (id, tenant_id, warehouse_id, opname_number, status, conducted_by, approved_by, notes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func (r *WMSRepo) CreateStockOpname(ctx context.Context, op *domain.StockOpname) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockOpname: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, op.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockOpname: set tenant: %w", err)
	}

	if op.ID == uuid.Nil {
		op.ID = uuid.New()
	}
	if op.Status == "" {
		op.Status = domain.StockOpnameStatusDraft
	}
	now := time.Now().UTC()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = now
	}
	if op.UpdatedAt.IsZero() {
		op.UpdatedAt = now
	}

	_, err = tx.ExecContext(ctx, createStockOpnameSQL,
		op.ID, op.TenantID, op.WarehouseID, op.OpnameNumber, string(op.Status),
		op.ConductedBy, ptrToNullUUID(op.ApprovedBy), ptrToNullString(op.Notes),
		op.CreatedAt, op.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockOpname: exec: %w", err)
	}

	return tx.Commit()
}

const getStockOpnameByIDSQL = `
SELECT id, tenant_id, warehouse_id, opname_number, status, conducted_by, approved_by, notes, created_at, updated_at
FROM stock_opnames
WHERE id = $1 AND tenant_id = $2`

func (r *WMSRepo) GetStockOpnameByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockOpname, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetStockOpnameByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetStockOpnameByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getStockOpnameByIDSQL, id, tenantID)
	var op domain.StockOpname
	var status string
	var appBy, notes sql.NullString
	if err := row.Scan(
		&op.ID, &op.TenantID, &op.WarehouseID, &op.OpnameNumber, &status,
		&op.ConductedBy, &appBy, &notes, &op.CreatedAt, &op.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOpnameNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetStockOpnameByID: scan: %w", err)
	}
	op.Status = domain.StockOpnameStatus(status)
	op.ApprovedBy = nullUUIDToPtr(appBy)
	op.Notes = nullStringToPtr(notes)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetStockOpnameByID: commit: %w", err)
	}
	return &op, nil
}

const listStockOpnamesSQL = `
SELECT id, tenant_id, warehouse_id, opname_number, status, conducted_by, approved_by, notes, created_at, updated_at
FROM stock_opnames
WHERE tenant_id = $1
ORDER BY created_at DESC`

const listStockOpnamesByWarehouseSQL = `
SELECT id, tenant_id, warehouse_id, opname_number, status, conducted_by, approved_by, notes, created_at, updated_at
FROM stock_opnames
WHERE tenant_id = $1 AND warehouse_id = $2
ORDER BY created_at DESC`

func (r *WMSRepo) ListStockOpnames(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockOpname, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnames: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnames: set tenant: %w", err)
	}

	var rows *sql.Rows
	var errQuery error
	if warehouseID != nil {
		rows, errQuery = tx.QueryContext(ctx, listStockOpnamesByWarehouseSQL, tenantID, *warehouseID)
	} else {
		rows, errQuery = tx.QueryContext(ctx, listStockOpnamesSQL, tenantID)
	}
	if errQuery != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnames: query: %w", errQuery)
	}
	defer rows.Close()

	result := make([]domain.StockOpname, 0)
	for rows.Next() {
		var op domain.StockOpname
		var status string
		var appBy, notes sql.NullString
		if err := rows.Scan(
			&op.ID, &op.TenantID, &op.WarehouseID, &op.OpnameNumber, &status,
			&op.ConductedBy, &appBy, &notes, &op.CreatedAt, &op.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockOpnames: scan: %w", err)
		}
		op.Status = domain.StockOpnameStatus(status)
		op.ApprovedBy = nullUUIDToPtr(appBy)
		op.Notes = nullStringToPtr(notes)
		result = append(result, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnames: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnames: commit: %w", err)
	}
	return result, nil
}

const addStockOpnameItemSQL = `
INSERT INTO stock_opname_items (id, opname_id, tenant_id, product_id, location_id, system_qty, physical_qty, discrepancy_qty, notes, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func (r *WMSRepo) AddStockOpnameItem(ctx context.Context, item *domain.StockOpnameItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.AddStockOpnameItem: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, item.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.AddStockOpnameItem: set tenant: %w", err)
	}

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}

	_, err = tx.ExecContext(ctx, addStockOpnameItemSQL,
		item.ID, item.OpnameID, item.TenantID, item.ProductID, item.LocationID,
		item.SystemQty, item.PhysicalQty, item.DiscrepancyQty, ptrToNullString(item.Notes),
		item.CreatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.AddStockOpnameItem: exec: %w", err)
	}

	return tx.Commit()
}

const listStockOpnameItemsSQL = `
SELECT id, opname_id, tenant_id, product_id, location_id, system_qty, physical_qty, discrepancy_qty, notes, created_at
FROM stock_opname_items
WHERE tenant_id = $1 AND opname_id = $2
ORDER BY created_at ASC`

func (r *WMSRepo) ListStockOpnameItems(ctx context.Context, tenantID, opnameID uuid.UUID) ([]domain.StockOpnameItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listStockOpnameItemsSQL, tenantID, opnameID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: query: %w", err)
	}
	defer rows.Close()

	result := make([]domain.StockOpnameItem, 0)
	for rows.Next() {
		var item domain.StockOpnameItem
		var notes sql.NullString
		if err := rows.Scan(
			&item.ID, &item.OpnameID, &item.TenantID, &item.ProductID, &item.LocationID,
			&item.SystemQty, &item.PhysicalQty, &item.DiscrepancyQty, &notes, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: scan: %w", err)
		}
		item.Notes = nullStringToPtr(notes)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockOpnameItems: commit: %w", err)
	}
	return result, nil
}

const updateStockOpnameStatusSQL = `
UPDATE stock_opnames
SET status = $1, approved_by = $2, updated_at = $3
WHERE id = $4 AND tenant_id = $5`

func (r *WMSRepo) UpdateStockOpnameStatus(ctx context.Context, tenantID, opnameID uuid.UUID, status domain.StockOpnameStatus, approvedBy *uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateStockOpnameStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateStockOpnameStatus: set tenant: %w", err)
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, updateStockOpnameStatusSQL,
		string(status), ptrToNullUUID(approvedBy), now, opnameID, tenantID)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateStockOpnameStatus: exec: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateStockOpnameStatus: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrOpnameNotFound
	}

	return tx.Commit()
}

// -----------------------------------------------------------------------------
// 9. Stock Scrap (Damaged Goods & Scrap Quarantine)
// -----------------------------------------------------------------------------

const createStockScrapSQL = `
INSERT INTO stock_scraps (id, tenant_id, scrap_number, warehouse_id, product_id, source_location_id, scrap_location_id, quantity, reason, reported_by, approved_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

func (r *WMSRepo) CreateStockScrap(ctx context.Context, scrap *domain.StockScrap) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockScrap: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, scrap.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockScrap: set tenant: %w", err)
	}

	if scrap.ID == uuid.Nil {
		scrap.ID = uuid.New()
	}
	if scrap.CreatedAt.IsZero() {
		scrap.CreatedAt = time.Now().UTC()
	}

	_, err = tx.ExecContext(ctx, createStockScrapSQL,
		scrap.ID, scrap.TenantID, scrap.ScrapNumber, scrap.WarehouseID, scrap.ProductID,
		scrap.SourceLocationID, scrap.ScrapLocationID, scrap.Quantity, scrap.Reason,
		scrap.ReportedBy, scrap.ApprovedBy, scrap.CreatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockScrap: exec: %w", err)
	}

	return tx.Commit()
}

const listStockScrapsSQL = `
SELECT id, tenant_id, scrap_number, warehouse_id, product_id, source_location_id, scrap_location_id, quantity, reason, reported_by, approved_by, created_at
FROM stock_scraps
WHERE tenant_id = $1
ORDER BY created_at DESC`

const listStockScrapsByWarehouseSQL = `
SELECT id, tenant_id, scrap_number, warehouse_id, product_id, source_location_id, scrap_location_id, quantity, reason, reported_by, approved_by, created_at
FROM stock_scraps
WHERE tenant_id = $1 AND warehouse_id = $2
ORDER BY created_at DESC`

func (r *WMSRepo) ListStockScraps(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockScrap, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockScraps: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockScraps: set tenant: %w", err)
	}

	var rows *sql.Rows
	var errQuery error
	if warehouseID != nil {
		rows, errQuery = tx.QueryContext(ctx, listStockScrapsByWarehouseSQL, tenantID, *warehouseID)
	} else {
		rows, errQuery = tx.QueryContext(ctx, listStockScrapsSQL, tenantID)
	}
	if errQuery != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockScraps: query: %w", errQuery)
	}
	defer rows.Close()

	result := make([]domain.StockScrap, 0)
	for rows.Next() {
		var scrap domain.StockScrap
		if err := rows.Scan(
			&scrap.ID, &scrap.TenantID, &scrap.ScrapNumber, &scrap.WarehouseID,
			&scrap.ProductID, &scrap.SourceLocationID, &scrap.ScrapLocationID,
			&scrap.Quantity, &scrap.Reason, &scrap.ReportedBy, &scrap.ApprovedBy, &scrap.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockScraps: scan: %w", err)
		}
		result = append(result, scrap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockScraps: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockScraps: commit: %w", err)
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// 10. Marketplace Sales Orders & SKU Mappings
// -----------------------------------------------------------------------------

const createMarketplaceBatchSQL = `
INSERT INTO marketplace_import_batches (
    id, tenant_id, batch_number, channel, warehouse_id, file_name,
    total_orders, processed_orders, failed_orders, unmapped_skus, status, uploaded_by, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

func (r *WMSRepo) CreateMarketplaceBatch(ctx context.Context, batch *domain.MarketplaceImportBatch) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateMarketplaceBatch: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, batch.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateMarketplaceBatch: set tenant: %w", err)
	}

	if batch.ID == uuid.Nil {
		batch.ID = uuid.New()
	}
	if batch.CreatedAt.IsZero() {
		batch.CreatedAt = time.Now().UTC()
	}
	if batch.Status == "" {
		batch.Status = domain.MarketplaceBatchStatusCompleted
	}

	_, err = tx.ExecContext(ctx, createMarketplaceBatchSQL,
		batch.ID, batch.TenantID, batch.BatchNumber, string(batch.Channel),
		batch.WarehouseID, batch.FileName, batch.TotalOrders, batch.ProcessedOrders,
		batch.FailedOrders, batch.UnmappedSKUs, string(batch.Status),
		batch.UploadedBy, batch.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateMarketplaceBatch: exec: %w", err)
	}

	return tx.Commit()
}

const getMarketplaceBatchByIDSQL = `
SELECT id, tenant_id, batch_number, channel, warehouse_id, file_name,
       total_orders, processed_orders, failed_orders, unmapped_skus, status, uploaded_by, created_at
FROM marketplace_import_batches
WHERE tenant_id = $1 AND id = $2`

func (r *WMSRepo) GetMarketplaceBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MarketplaceImportBatch, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceBatchByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceBatchByID: set tenant: %w", err)
	}

	var b domain.MarketplaceImportBatch
	var channel, status string
	err = tx.QueryRowContext(ctx, getMarketplaceBatchByIDSQL, tenantID, id).Scan(
		&b.ID, &b.TenantID, &b.BatchNumber, &channel, &b.WarehouseID, &b.FileName,
		&b.TotalOrders, &b.ProcessedOrders, &b.FailedOrders, &b.UnmappedSKUs,
		&status, &b.UploadedBy, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrBatchNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceBatchByID: query: %w", err)
	}
	b.Channel = domain.MarketplaceChannel(channel)
	b.Status = domain.MarketplaceBatchStatus(status)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceBatchByID: commit: %w", err)
	}
	return &b, nil
}

const listMarketplaceBatchesSQL = `
SELECT id, tenant_id, batch_number, channel, warehouse_id, file_name,
       total_orders, processed_orders, failed_orders, unmapped_skus, status, uploaded_by, created_at
FROM marketplace_import_batches
WHERE tenant_id = $1
ORDER BY created_at DESC`

const listMarketplaceBatchesByWarehouseSQL = `
SELECT id, tenant_id, batch_number, channel, warehouse_id, file_name,
       total_orders, processed_orders, failed_orders, unmapped_skus, status, uploaded_by, created_at
FROM marketplace_import_batches
WHERE tenant_id = $1 AND warehouse_id = $2
ORDER BY created_at DESC`

func (r *WMSRepo) ListMarketplaceBatches(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: set tenant: %w", err)
	}

	var rows *sql.Rows
	var qErr error
	if warehouseID != nil {
		rows, qErr = tx.QueryContext(ctx, listMarketplaceBatchesByWarehouseSQL, tenantID, *warehouseID)
	} else {
		rows, qErr = tx.QueryContext(ctx, listMarketplaceBatchesSQL, tenantID)
	}
	if qErr != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: query: %w", qErr)
	}
	defer rows.Close()

	res := make([]domain.MarketplaceImportBatch, 0)
	for rows.Next() {
		var b domain.MarketplaceImportBatch
		var channel, status string
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.BatchNumber, &channel, &b.WarehouseID, &b.FileName,
			&b.TotalOrders, &b.ProcessedOrders, &b.FailedOrders, &b.UnmappedSKUs,
			&status, &b.UploadedBy, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: scan: %w", err)
		}
		b.Channel = domain.MarketplaceChannel(channel)
		b.Status = domain.MarketplaceBatchStatus(status)
		res = append(res, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceBatches: commit: %w", err)
	}
	return res, nil
}

const updateMarketplaceBatchSQL = `
UPDATE marketplace_import_batches
SET total_orders = $1, processed_orders = $2, failed_orders = $3, unmapped_skus = $4, status = $5
WHERE tenant_id = $6 AND id = $7`

func (r *WMSRepo) UpdateMarketplaceBatch(ctx context.Context, batch *domain.MarketplaceImportBatch) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceBatch: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, batch.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceBatch: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateMarketplaceBatchSQL,
		batch.TotalOrders, batch.ProcessedOrders, batch.FailedOrders,
		batch.UnmappedSKUs, string(batch.Status), batch.TenantID, batch.ID,
	)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceBatch: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrBatchNotFound
	}

	return tx.Commit()
}

func scanMarketplaceOrderRow(scanner interface{ Scan(dest ...any) error }) (*domain.MarketplaceOrder, error) {
	var o domain.MarketplaceOrder
	var batchID, soID sql.NullString
	var custName, custPhone, shipAddr, courier, trkNum sql.NullString
	var channel, status string

	err := scanner.Scan(
		&o.ID, &o.TenantID, &batchID, &o.WarehouseID, &channel, &o.ExternalOrderID,
		&o.OrderDate, &custName, &custPhone, &shipAddr, &courier,
		&trkNum, &o.TotalAmount, &o.ShippingFee, &o.MarketplaceFee, &o.NetAmount,
		&status, &soID, &o.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	o.BatchID = nullUUIDToPtr(batchID)
	o.SalesOrderID = nullUUIDToPtr(soID)
	o.CustomerName = nullStringToPtr(custName)
	o.CustomerPhone = nullStringToPtr(custPhone)
	o.ShippingAddress = nullStringToPtr(shipAddr)
	o.Courier = nullStringToPtr(courier)
	o.TrackingNumber = nullStringToPtr(trkNum)
	o.Channel = domain.MarketplaceChannel(channel)
	o.Status = domain.MarketplaceOrderStatus(status)
	o.Items = []domain.MarketplaceOrderItem{}
	return &o, nil
}

func fetchMarketplaceOrderItems(ctx context.Context, tx *sql.Tx, tenantID, orderID uuid.UUID) ([]domain.MarketplaceOrderItem, error) {
	const itemsSQL = `
SELECT id, tenant_id, order_id, external_sku, product_id, item_name, quantity, unit_price, subtotal, is_mapped
FROM marketplace_order_items
WHERE tenant_id = $1 AND order_id = $2
ORDER BY id ASC`

	rows, err := tx.QueryContext(ctx, itemsSQL, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.MarketplaceOrderItem, 0)
	for rows.Next() {
		var item domain.MarketplaceOrderItem
		var prodID sql.NullString
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.OrderID, &item.ExternalSKU,
			&prodID, &item.ItemName, &item.Quantity, &item.UnitPrice,
			&item.Subtotal, &item.IsMapped,
		); err != nil {
			return nil, err
		}
		item.ProductID = nullUUIDToPtr(prodID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const createMarketplaceOrderSQL = `
INSERT INTO marketplace_orders (
    id, tenant_id, batch_id, warehouse_id, channel, external_order_id,
    order_date, customer_name, customer_phone, shipping_address, courier,
    tracking_number, total_amount, shipping_fee, marketplace_fee, net_amount,
    status, sales_order_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

const createMarketplaceOrderItemSQL = `
INSERT INTO marketplace_order_items (
    id, tenant_id, order_id, external_sku, product_id, item_name,
    quantity, unit_price, subtotal, is_mapped
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func (r *WMSRepo) CreateMarketplaceOrder(ctx context.Context, order *domain.MarketplaceOrder) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateMarketplaceOrder: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, order.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateMarketplaceOrder: set tenant: %w", err)
	}

	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	if order.OrderDate.IsZero() {
		order.OrderDate = order.CreatedAt
	}
	if order.Status == "" {
		order.Status = domain.MarketplaceOrderStatusCompleted
	}

	_, err = tx.ExecContext(ctx, createMarketplaceOrderSQL,
		order.ID, order.TenantID, ptrToNullUUID(order.BatchID), order.WarehouseID,
		string(order.Channel), order.ExternalOrderID, order.OrderDate,
		ptrToNullString(order.CustomerName), ptrToNullString(order.CustomerPhone),
		ptrToNullString(order.ShippingAddress), ptrToNullString(order.Courier),
		ptrToNullString(order.TrackingNumber), order.TotalAmount, order.ShippingFee,
		order.MarketplaceFee, order.NetAmount, string(order.Status),
		ptrToNullUUID(order.SalesOrderID), order.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateMarketplaceOrder
		}
		return fmt.Errorf("WMSRepo.CreateMarketplaceOrder: exec order: %w", err)
	}

	for i := range order.Items {
		item := &order.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.TenantID = order.TenantID
		item.OrderID = order.ID

		_, err = tx.ExecContext(ctx, createMarketplaceOrderItemSQL,
			item.ID, item.TenantID, item.OrderID, item.ExternalSKU,
			ptrToNullUUID(item.ProductID), item.ItemName, item.Quantity,
			item.UnitPrice, item.Subtotal, item.IsMapped,
		)
		if err != nil {
			return fmt.Errorf("WMSRepo.CreateMarketplaceOrder: exec item: %w", err)
		}
	}

	return tx.Commit()
}

const getMarketplaceOrderByExternalIDSQL = `
SELECT id, tenant_id, batch_id, warehouse_id, channel, external_order_id,
       order_date, customer_name, customer_phone, shipping_address, courier,
       tracking_number, total_amount, shipping_fee, marketplace_fee, net_amount,
       status, sales_order_id, created_at
FROM marketplace_orders
WHERE tenant_id = $1 AND channel = $2 AND external_order_id = $3`

func (r *WMSRepo) GetMarketplaceOrderByExternalID(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalID string) (*domain.MarketplaceOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByExternalID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByExternalID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getMarketplaceOrderByExternalIDSQL, tenantID, string(channel), externalID)
	o, err := scanMarketplaceOrderRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMarketplaceOrderNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByExternalID: query: %w", err)
	}

	items, err := fetchMarketplaceOrderItems(ctx, tx, tenantID, o.ID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByExternalID: fetch items: %w", err)
	}
	o.Items = items

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByExternalID: commit: %w", err)
	}
	return o, nil
}

const getMarketplaceOrderByIDSQL = `
SELECT id, tenant_id, batch_id, warehouse_id, channel, external_order_id,
       order_date, customer_name, customer_phone, shipping_address, courier,
       tracking_number, total_amount, shipping_fee, marketplace_fee, net_amount,
       status, sales_order_id, created_at
FROM marketplace_orders
WHERE tenant_id = $1 AND id = $2`

func (r *WMSRepo) GetMarketplaceOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MarketplaceOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getMarketplaceOrderByIDSQL, tenantID, id)
	o, err := scanMarketplaceOrderRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMarketplaceOrderNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByID: query: %w", err)
	}

	items, err := fetchMarketplaceOrderItems(ctx, tx, tenantID, o.ID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByID: fetch items: %w", err)
	}
	o.Items = items

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetMarketplaceOrderByID: commit: %w", err)
	}
	return o, nil
}

func (r *WMSRepo) ListMarketplaceOrders(ctx context.Context, tenantID uuid.UUID, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: set tenant: %w", err)
	}

	query := `
SELECT id, tenant_id, batch_id, warehouse_id, channel, external_order_id,
       order_date, customer_name, customer_phone, shipping_address, courier,
       tracking_number, total_amount, shipping_fee, marketplace_fee, net_amount,
       status, sales_order_id, created_at
FROM marketplace_orders
WHERE tenant_id = $1`

	args := []any{tenantID}
	argIdx := 2

	if warehouseID != nil {
		query += fmt.Sprintf(" AND warehouse_id = $%d", argIdx)
		args = append(args, *warehouseID)
		argIdx++
	}
	if batchID != nil {
		query += fmt.Sprintf(" AND batch_id = $%d", argIdx)
		args = append(args, *batchID)
		argIdx++
	}
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*status))
		argIdx++
	}

	query += " ORDER BY order_date DESC, created_at DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: query: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.MarketplaceOrder, 0)
	for rows.Next() {
		o, err := scanMarketplaceOrderRow(rows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: scan: %w", err)
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: rows err: %w", err)
	}

	for i := range orders {
		items, err := fetchMarketplaceOrderItems(ctx, tx, tenantID, orders[i].ID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: fetch items: %w", err)
		}
		orders[i].Items = items
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMarketplaceOrders: commit: %w", err)
	}
	return orders, nil
}

const updateMarketplaceOrderStatusSQL = `
UPDATE marketplace_orders
SET status = $1
WHERE tenant_id = $2 AND id = $3`

func (r *WMSRepo) UpdateMarketplaceOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.MarketplaceOrderStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceOrderStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceOrderStatus: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateMarketplaceOrderStatusSQL, string(status), tenantID, id)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateMarketplaceOrderStatus: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrMarketplaceOrderNotFound
	}

	return tx.Commit()
}

const getSKUMappingSQL = `
SELECT id, tenant_id, product_id, mapping_type, channel_name, external_sku, external_name, multiplier, created_at, updated_at
FROM product_sku_mappings
WHERE tenant_id = $1 AND channel_name = $2 AND external_sku = $3
LIMIT 1`

func (r *WMSRepo) GetSKUMapping(ctx context.Context, tenantID uuid.UUID, channelName, externalSKU string) (*domain.ProductSKUMapping, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetSKUMapping: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetSKUMapping: set tenant: %w", err)
	}

	var m domain.ProductSKUMapping
	var extName sql.NullString
	var mappingType string
	err = tx.QueryRowContext(ctx, getSKUMappingSQL, tenantID, channelName, externalSKU).Scan(
		&m.ID, &m.TenantID, &m.ProductID, &mappingType, &m.ChannelName,
		&m.ExternalSKU, &extName, &m.Multiplier, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSKUMappingNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetSKUMapping: query: %w", err)
	}
	m.MappingType = domain.SKUMappingType(mappingType)
	m.ExternalName = nullStringToPtr(extName)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetSKUMapping: commit: %w", err)
	}
	return &m, nil
}

const listSKUMappingsSQL = `
SELECT id, tenant_id, product_id, mapping_type, channel_name, external_sku, external_name, multiplier, created_at, updated_at
FROM product_sku_mappings
WHERE tenant_id = $1
ORDER BY created_at DESC`

const listSKUMappingsByChannelSQL = `
SELECT id, tenant_id, product_id, mapping_type, channel_name, external_sku, external_name, multiplier, created_at, updated_at
FROM product_sku_mappings
WHERE tenant_id = $1 AND channel_name = $2
ORDER BY created_at DESC`

func (r *WMSRepo) ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListSKUMappings: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListSKUMappings: set tenant: %w", err)
	}

	var rows *sql.Rows
	var qErr error
	if channelName != "" {
		rows, qErr = tx.QueryContext(ctx, listSKUMappingsByChannelSQL, tenantID, channelName)
	} else {
		rows, qErr = tx.QueryContext(ctx, listSKUMappingsSQL, tenantID)
	}
	if qErr != nil {
		return nil, fmt.Errorf("WMSRepo.ListSKUMappings: query: %w", qErr)
	}
	defer rows.Close()

	res := make([]domain.ProductSKUMapping, 0)
	for rows.Next() {
		var m domain.ProductSKUMapping
		var extName sql.NullString
		var mappingType string
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.ProductID, &mappingType, &m.ChannelName,
			&m.ExternalSKU, &extName, &m.Multiplier, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListSKUMappings: scan: %w", err)
		}
		m.MappingType = domain.SKUMappingType(mappingType)
		m.ExternalName = nullStringToPtr(extName)
		res = append(res, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListSKUMappings: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListSKUMappings: commit: %w", err)
	}
	return res, nil
}

const updateUnmappedOrderItemsSQL = `
UPDATE marketplace_order_items oi
SET product_id = $1, is_mapped = true
FROM marketplace_orders o
WHERE oi.order_id = o.id
  AND oi.tenant_id = $2
  AND o.tenant_id = $2
  AND o.channel = $3
  AND oi.external_sku = $4
  AND oi.is_mapped = false`

func (r *WMSRepo) UpdateUnmappedOrderItems(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalSKU string, productID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateUnmappedOrderItems: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateUnmappedOrderItems: set tenant: %w", err)
	}

	_, err = tx.ExecContext(ctx, updateUnmappedOrderItemsSQL, productID, tenantID, string(channel), externalSKU)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateUnmappedOrderItems: exec: %w", err)
	}

	return tx.Commit()
}

const getPendingUnmappedOrdersBySKUSQL = `
SELECT DISTINCT o.id, o.tenant_id, o.batch_id, o.warehouse_id, o.channel, o.external_order_id,
       o.order_date, o.customer_name, o.customer_phone, o.shipping_address, o.courier,
       o.tracking_number, o.total_amount, o.shipping_fee, o.marketplace_fee, o.net_amount,
       o.status, o.sales_order_id, o.created_at
FROM marketplace_orders o
JOIN marketplace_order_items oi ON oi.order_id = o.id AND oi.tenant_id = o.tenant_id
WHERE o.tenant_id = $1
  AND o.channel = $2
  AND o.status = 'UNMAPPED_SKU'
  AND oi.external_sku = $3
ORDER BY o.created_at ASC`

func (r *WMSRepo) GetPendingUnmappedOrdersBySKU(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalSKU string) ([]domain.MarketplaceOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, getPendingUnmappedOrdersBySKUSQL, tenantID, string(channel), externalSKU)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: query: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.MarketplaceOrder, 0)
	for rows.Next() {
		o, err := scanMarketplaceOrderRow(rows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: scan: %w", err)
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: rows err: %w", err)
	}

	for i := range orders {
		items, err := fetchMarketplaceOrderItems(ctx, tx, tenantID, orders[i].ID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: fetch items: %w", err)
		}
		orders[i].Items = items
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPendingUnmappedOrdersBySKU: commit: %w", err)
	}
	return orders, nil
}

const getProductBySKUSQL = `
SELECT id, tenant_id, name, description, sku, price, created_at, updated_at
FROM products
WHERE tenant_id = $1 AND sku = $2
LIMIT 1`

func (r *WMSRepo) GetProductBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetProductBySKU: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetProductBySKU: set tenant: %w", err)
	}

	var p domain.Product
	var desc sql.NullString
	err = tx.QueryRowContext(ctx, getProductBySKUSQL, tenantID, sku).Scan(
		&p.ID, &p.TenantID, &p.Name, &desc, &p.SKU, &p.Price, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetProductBySKU: query: %w", err)
	}
	p.Description = desc.String

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetProductBySKU: commit: %w", err)
	}
	return &p, nil
}
