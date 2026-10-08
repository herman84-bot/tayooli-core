package postgres

// Sprint 4 Outbound Postgres Repository: Shipping Manifests, Loading Scan Verification,
// Courier Multi-DO Consolidation, Atomic Handover Dispatch & 8 SOP Outbound KPIs (ADR-014 Invariant 1, Master PRD §3.2, §4.1).

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

var _ domain.WMSManifestRepository = (*WMSRepo)(nil)

// CreateShippingManifest creates a new courier manifest, attaches the specified DOs (status PACKED or STAGED),
// sets DO status to STAGED, and records an audit log.
func (r *WMSRepo) CreateShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateShippingManifestRequest) (*domain.ShippingManifest, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: set tenant: %w", err)
	}

	// 1. Verify warehouse belongs to tenant
	var whExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id = $1 AND tenant_id = $2)`, req.WarehouseID, tenantID).Scan(&whExists); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: check warehouse: %w", err)
	}
	if !whExists {
		return nil, domain.ErrWarehouseNotFound
	}

	// 2. Fetch and lock delivery orders
	rows, err := tx.QueryContext(ctx, `
		SELECT id, warehouse_id, status, COALESCE(package_weight_kg, 0), expedition_name, manifest_id
		FROM delivery_orders
		WHERE tenant_id = $1 AND id = ANY($2)
		FOR UPDATE`, tenantID, pq.Array(req.DeliveryOrderIDs))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: query DOs: %w", err)
	}
	defer rows.Close()

	type doRow struct {
		id             uuid.UUID
		warehouseID    uuid.UUID
		status         string
		weight         decimal.Decimal
		expeditionName sql.NullString
		manifestID     sql.NullString
	}
	var dos []doRow
	totalWeight := decimal.Zero

	for rows.Next() {
		var d doRow
		if err := rows.Scan(&d.id, &d.warehouseID, &d.status, &d.weight, &d.expeditionName, &d.manifestID); err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: scan DO: %w", err)
		}
		dos = append(dos, d)
		totalWeight = totalWeight.Add(d.weight)
	}
	_ = rows.Close()

	if len(dos) != len(req.DeliveryOrderIDs) {
		return nil, domain.ErrDeliveryOrderNotFound
	}

	for _, d := range dos {
		if d.warehouseID != req.WarehouseID {
			return nil, domain.ErrInvalidInput
		}
		if d.manifestID.Valid && d.manifestID.String != "" {
			return nil, domain.ErrInvalidManifestStatus
		}
		if d.status != string(domain.DeliveryOrderStatusPacked) && d.status != string(domain.DeliveryOrderStatusStaged) {
			return nil, domain.ErrInvalidManifestStatus
		}
		if d.expeditionName.Valid && strings.TrimSpace(d.expeditionName.String) != "" && strings.TrimSpace(req.ExpeditionName) != "" {
			if !strings.EqualFold(strings.TrimSpace(d.expeditionName.String), strings.TrimSpace(req.ExpeditionName)) {
				return nil, domain.ErrDOMisload
			}
		}
	}

	// 3. Generate atomic manifest number: MAN-YYYYMMDD-XXXX
	todayStr := time.Now().UTC().Format("20060102")
	var manifestNumber string
	for i := 0; i < 10; i++ {
		var count int
		err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM shipping_manifests
			WHERE tenant_id = $1 AND manifest_number LIKE $2`,
			tenantID, "MAN-"+todayStr+"-%").Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: count manifests: %w", err)
		}
		candidate := fmt.Sprintf("MAN-%s-%04d", todayStr, count+1+i)
		var exists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM shipping_manifests WHERE tenant_id = $1 AND manifest_number = $2)`,
			tenantID, candidate).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: check manifest exists: %w", err)
		}
		if !exists {
			manifestNumber = candidate
			break
		}
	}
	if manifestNumber == "" {
		manifestNumber = fmt.Sprintf("MAN-%s-%04d", todayStr, time.Now().UnixNano()%10000)
	}

	// 4. Insert shipping manifest
	manifestID := uuid.New()
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO shipping_manifests (
			id, tenant_id, warehouse_id, manifest_number, expedition_name,
			driver_name, vehicle_plate, driver_phone, total_packages, total_weight_kg,
			status, driver_signature_svg, notes, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, NULL, $12, $13, $14, $14
		)`,
		manifestID, tenantID, req.WarehouseID, manifestNumber, strings.TrimSpace(req.ExpeditionName),
		strings.TrimSpace(req.DriverName), strings.TrimSpace(req.VehiclePlate), ptrToNullString(req.DriverPhone),
		len(dos), totalWeight,
		string(domain.ShippingManifestStatusStaged), ptrToNullString(req.Notes), ptrToNullUUID(&userID), now,
	)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: insert manifest: %w", err)
	}

	// 5. Link DOs to manifest and update status to STAGED
	_, err = tx.ExecContext(ctx, `
		UPDATE delivery_orders
		SET manifest_id = $1, status = 'STAGED', updated_at = $2
		WHERE tenant_id = $3 AND id = ANY($4)`,
		manifestID, now, tenantID, pq.Array(req.DeliveryOrderIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: link DOs: %w", err)
	}

	// 6. Audit log
	auditDetails, _ := json.Marshal(map[string]any{
		"manifest_number": manifestNumber,
		"warehouse_id":    req.WarehouseID,
		"expedition_name": req.ExpeditionName,
		"driver_name":     req.DriverName,
		"vehicle_plate":   req.VehiclePlate,
		"total_packages":  len(dos),
		"total_weight_kg": totalWeight.String(),
		"delivery_orders": req.DeliveryOrderIDs,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "shipping_manifest", manifestID, "created", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateShippingManifest: commit: %w", err)
	}

	detail, err := r.GetShippingManifestByID(ctx, tenantID, manifestID)
	if err != nil {
		return nil, err
	}
	return &detail.Manifest, nil
}

// GetShippingManifestByID retrieves manifest header and its attached delivery orders.
func (r *WMSRepo) GetShippingManifestByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ShippingManifestDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetShippingManifestByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetShippingManifestByID: set tenant: %w", err)
	}

	detail, err := r.getShippingManifestDetailTx(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetShippingManifestByID: commit: %w", err)
	}
	return detail, nil
}

func (r *WMSRepo) getShippingManifestDetailTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) (*domain.ShippingManifestDetail, error) {
	var m domain.ShippingManifest
	var whName sql.NullString
	var drvPhone, sigSVG, notes sql.NullString
	var uCr, uCrName, uDs, uDsName sql.NullString
	var dispAt sql.NullTime
	var statusStr string

	err := tx.QueryRowContext(ctx, `
		SELECT sm.id, sm.tenant_id, sm.warehouse_id, w.name, sm.manifest_number,
		       sm.expedition_name, sm.driver_name, sm.vehicle_plate, sm.driver_phone,
		       sm.total_packages, sm.total_weight_kg, sm.status, sm.driver_signature_svg,
		       sm.notes, sm.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
		       sm.dispatched_by, COALESCE(u_ds.full_name, u_ds.email, ''),
		       sm.dispatched_at, sm.created_at, sm.updated_at
		FROM shipping_manifests sm
		LEFT JOIN warehouses w ON w.id = sm.warehouse_id AND w.tenant_id = sm.tenant_id
		LEFT JOIN users u_cr ON u_cr.id = sm.created_by AND u_cr.tenant_id = sm.tenant_id
		LEFT JOIN users u_ds ON u_ds.id = sm.dispatched_by AND u_ds.tenant_id = sm.tenant_id
		WHERE sm.id = $1 AND sm.tenant_id = $2`, id, tenantID).Scan(
		&m.ID, &m.TenantID, &m.WarehouseID, &whName, &m.ManifestNumber,
		&m.ExpeditionName, &m.DriverName, &m.VehiclePlate, &drvPhone,
		&m.TotalPackages, &m.TotalWeightKg, &statusStr, &sigSVG,
		&notes, &uCr, &uCrName,
		&uDs, &uDsName,
		&dispAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrManifestNotFound
		}
		return nil, fmt.Errorf("WMSRepo.getShippingManifestDetailTx: query header: %w", err)
	}

	m.WarehouseName = nullStringToPtr(whName)
	m.DriverPhone = nullStringToPtr(drvPhone)
	m.DriverSignatureSVG = nullStringToPtr(sigSVG)
	m.Notes = nullStringToPtr(notes)
	m.Status = domain.ShippingManifestStatus(statusStr)
	m.CreatedBy = nullUUIDToPtr(uCr)
	m.CreatedByName = nullStringToPtr(uCrName)
	m.DispatchedBy = nullUUIDToPtr(uDs)
	m.DispatchedByName = nullStringToPtr(uDsName)
	m.DispatchedAt = nullTimeToPtr(dispAt)

	// Fetch items
	rows, err := tx.QueryContext(ctx, `
		SELECT do.id, do.do_number,
		       COALESCE(c.name, do.recipient_name, ''),
		       COALESCE(c.address, ''),
		       do.package_weight_kg, do.packaging_type,
		       (do.loading_scanned_at IS NOT NULL) AS scanned,
		       do.loading_scanned_at,
		       COALESCE(u_scan.full_name, u_scan.email, '')
		FROM delivery_orders do
		LEFT JOIN customers c ON c.id = do.customer_id AND c.tenant_id = do.tenant_id
		LEFT JOIN users u_scan ON u_scan.id = do.loading_scanned_by AND u_scan.tenant_id = do.tenant_id
		WHERE do.manifest_id = $1 AND do.tenant_id = $2
		ORDER BY do.created_at ASC, do.id ASC`, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.getShippingManifestDetailTx: query items: %w", err)
	}
	defer rows.Close()

	var items []domain.ShippingManifestItem
	for rows.Next() {
		var it domain.ShippingManifestItem
		var pWeight decimal.NullDecimal
		var pkgType, scanByName sql.NullString
		var scanAt sql.NullTime

		if err := rows.Scan(
			&it.DeliveryOrderID, &it.DONumber,
			&it.CustomerName, &it.DestinationCity,
			&pWeight, &pkgType,
			&it.Scanned, &scanAt, &scanByName,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.getShippingManifestDetailTx: scan item: %w", err)
		}
		it.PackageWeightKg = nullDecimalToPtr(pWeight)
		it.PackagingType = nullStringToPtr(pkgType)
		it.ScannedAt = nullTimeToPtr(scanAt)
		it.ScannedByName = nullStringToPtr(scanByName)

		items = append(items, it)
	}
	_ = rows.Close()

	if items == nil {
		items = []domain.ShippingManifestItem{}
	}

	return &domain.ShippingManifestDetail{
		Manifest: m,
		Items:    items,
	}, nil
}

// ListShippingManifests lists manifests matching the optional filters.
func (r *WMSRepo) ListShippingManifests(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.ShippingManifestStatus, expeditionName *string) ([]domain.ShippingManifest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListShippingManifests: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListShippingManifests: set tenant: %w", err)
	}

	query := `
		SELECT sm.id, sm.tenant_id, sm.warehouse_id, w.name, sm.manifest_number,
		       sm.expedition_name, sm.driver_name, sm.vehicle_plate, sm.driver_phone,
		       sm.total_packages, sm.total_weight_kg, sm.status, sm.driver_signature_svg,
		       sm.notes, sm.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
		       sm.dispatched_by, COALESCE(u_ds.full_name, u_ds.email, ''),
		       sm.dispatched_at, sm.created_at, sm.updated_at
		FROM shipping_manifests sm
		LEFT JOIN warehouses w ON w.id = sm.warehouse_id AND w.tenant_id = sm.tenant_id
		LEFT JOIN users u_cr ON u_cr.id = sm.created_by AND u_cr.tenant_id = sm.tenant_id
		LEFT JOIN users u_ds ON u_ds.id = sm.dispatched_by AND u_ds.tenant_id = sm.tenant_id
		WHERE sm.tenant_id = $1`

	args := []any{tenantID}
	argIdx := 2

	if warehouseID != nil {
		query += fmt.Sprintf(" AND sm.warehouse_id = $%d", argIdx)
		args = append(args, *warehouseID)
		argIdx++
	}
	if status != nil && *status != "" {
		query += fmt.Sprintf(" AND sm.status = $%d", argIdx)
		args = append(args, string(*status))
		argIdx++
	}
	if expeditionName != nil && strings.TrimSpace(*expeditionName) != "" {
		query += fmt.Sprintf(" AND sm.expedition_name ILIKE $%d", argIdx)
		args = append(args, "%"+strings.TrimSpace(*expeditionName)+"%")
		argIdx++
	}

	query += " ORDER BY sm.created_at DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListShippingManifests: query: %w", err)
	}
	defer rows.Close()

	var list []domain.ShippingManifest
	for rows.Next() {
		var m domain.ShippingManifest
		var whName sql.NullString
		var drvPhone, sigSVG, notes sql.NullString
		var uCr, uCrName, uDs, uDsName sql.NullString
		var dispAt sql.NullTime
		var statusStr string

		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.WarehouseID, &whName, &m.ManifestNumber,
			&m.ExpeditionName, &m.DriverName, &m.VehiclePlate, &drvPhone,
			&m.TotalPackages, &m.TotalWeightKg, &statusStr, &sigSVG,
			&notes, &uCr, &uCrName,
			&uDs, &uDsName,
			&dispAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListShippingManifests: scan: %w", err)
		}

		m.WarehouseName = nullStringToPtr(whName)
		m.DriverPhone = nullStringToPtr(drvPhone)
		m.DriverSignatureSVG = nullStringToPtr(sigSVG)
		m.Notes = nullStringToPtr(notes)
		m.Status = domain.ShippingManifestStatus(statusStr)
		m.CreatedBy = nullUUIDToPtr(uCr)
		m.CreatedByName = nullStringToPtr(uCrName)
		m.DispatchedBy = nullUUIDToPtr(uDs)
		m.DispatchedByName = nullStringToPtr(uDsName)
		m.DispatchedAt = nullTimeToPtr(dispAt)

		list = append(list, m)
	}
	_ = rows.Close()

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListShippingManifests: commit: %w", err)
	}

	if list == nil {
		list = []domain.ShippingManifest{}
	}
	return list, nil
}

// ScanDOLoading verifies barcode matching a DO in this manifest, marks it scanned,
// and updates manifest status to LOADED if all DOs are scanned.
func (r *WMSRepo) ScanDOLoading(ctx context.Context, tenantID, manifestID uuid.UUID, barcode string, userID uuid.UUID) (*domain.ShippingManifestDetail, error) {
	bc := strings.TrimSpace(barcode)
	if bc == "" {
		return nil, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: set tenant: %w", err)
	}

	// 1. Lock manifest and verify status
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM shipping_manifests
		WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, manifestID, tenantID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrManifestNotFound
		}
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: check manifest: %w", err)
	}

	if domain.ShippingManifestStatus(status) == domain.ShippingManifestStatusDispatched ||
		domain.ShippingManifestStatus(status) == domain.ShippingManifestStatusCancelled {
		return nil, domain.ErrInvalidManifestStatus
	}

	// 2. Find matching DO in this manifest by do_number or tracking_number
	var doID uuid.UUID
	var doNumber string
	err = tx.QueryRowContext(ctx, `
		SELECT id, do_number FROM delivery_orders
		WHERE manifest_id = $1 AND tenant_id = $2 AND (do_number = $3 OR tracking_number = $3)
		FOR UPDATE`, manifestID, tenantID, bc).Scan(&doID, &doNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDOMisload
		}
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: find DO: %w", err)
	}

	// 3. Mark DO as loaded/scanned
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		UPDATE delivery_orders
		SET loading_scanned_at = $1,
		    loading_scanned_by = $2,
		    updated_at = $1
		WHERE id = $3 AND tenant_id = $4`,
		now, userID, doID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: update DO scan: %w", err)
	}

	// 4. Check if all DOs in this manifest are now scanned
	var unscannedCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM delivery_orders
		WHERE manifest_id = $1 AND tenant_id = $2 AND loading_scanned_at IS NULL`,
		manifestID, tenantID).Scan(&unscannedCount)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: count unscanned: %w", err)
	}

	if unscannedCount == 0 {
		_, err = tx.ExecContext(ctx, `
			UPDATE shipping_manifests
			SET status = $1, updated_at = $2
			WHERE id = $3 AND tenant_id = $4`,
			string(domain.ShippingManifestStatusLoaded), now, manifestID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ScanDOLoading: update manifest loaded: %w", err)
		}
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"do_id":       doID,
		"do_number":   doNumber,
		"barcode":     bc,
		"all_scanned": (unscannedCount == 0),
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "shipping_manifest", manifestID, "loading_scan", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ScanDOLoading: commit: %w", err)
	}

	return r.GetShippingManifestByID(ctx, tenantID, manifestID)
}

// DispatchShippingManifest executes atomic handover: validates driver signature, deducts stock
// double-entry to @CUSTOMER for all items in linked DOs, updates DOs to SHIPPED, manifest to DISPATCHED.
func (r *WMSRepo) DispatchShippingManifest(ctx context.Context, tenantID, manifestID, userID uuid.UUID, req domain.DispatchShippingManifestRequest) (*domain.ShippingManifest, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: set tenant: %w", err)
	}

	// 1. Lock and validate manifest status
	var mStatus string
	var manifestNum string
	err = tx.QueryRowContext(ctx, `
		SELECT manifest_number, status
		FROM shipping_manifests
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE`, manifestID, tenantID).Scan(&manifestNum, &mStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrManifestNotFound
		}
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: get manifest: %w", err)
	}

	curStatus := domain.ShippingManifestStatus(mStatus)
	if curStatus != domain.ShippingManifestStatusLoaded && curStatus != domain.ShippingManifestStatusStaged {
		return nil, domain.ErrInvalidManifestStatus
	}

	// 2. Fetch DOs in this manifest
	rows, err := tx.QueryContext(ctx, `
		SELECT id, do_number
		FROM delivery_orders
		WHERE manifest_id = $1 AND tenant_id = $2
		FOR UPDATE`, manifestID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: query DOs: %w", err)
	}
	defer rows.Close()

	type doItemRef struct {
		id       uuid.UUID
		doNumber string
	}
	var dos []doItemRef
	for rows.Next() {
		var d doItemRef
		if err := rows.Scan(&d.id, &d.doNumber); err != nil {
			return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: scan DO: %w", err)
		}
		dos = append(dos, d)
	}
	_ = rows.Close()

	if len(dos) == 0 {
		return nil, domain.ErrManifestEmpty
	}

	// 3. Resolve @CUSTOMER virtual destination location
	custLocID, err := r.getOrCreateCustomerLocationTx(ctx, tx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: customer location: %w", err)
	}

	now := time.Now().UTC()

	// 4. For each DO: process stock deduction for each line item and transition status to SHIPPED
	for _, do := range dos {
		itemRows, err := tx.QueryContext(ctx, `
			SELECT doi.id, doi.product_id, doi.quantity, doi.location_id, doi.batch_id
			FROM delivery_order_items doi
			WHERE doi.delivery_order_id = $1 AND doi.tenant_id = $2
			ORDER BY doi.created_at ASC, doi.id ASC`, do.id, tenantID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: query DO items for %s: %w", do.doNumber, err)
		}

		type doiRow struct {
			id         uuid.UUID
			productID  uuid.UUID
			quantity   decimal.Decimal
			locationID uuid.UUID
			batchID    *uuid.UUID
		}
		var doiItems []doiRow
		for itemRows.Next() {
			var it doiRow
			var bID sql.NullString
			if err := itemRows.Scan(&it.id, &it.productID, &it.quantity, &it.locationID, &bID); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: scan item: %w", err)
			}
			it.batchID = nullUUIDToPtr(bID)
			doiItems = append(doiItems, it)
		}
		_ = itemRows.Close()

		for idx, item := range doiItems {
			// Lock location stock
			if err := r.LockLocationStock(ctx, tx, tenantID, item.locationID, item.productID); err != nil {
				return nil, err
			}

			// Validate location is not quarantine
			var locType string
			if err := tx.QueryRowContext(ctx, `SELECT type FROM warehouse_locations WHERE id = $1 AND tenant_id = $2`, item.locationID, tenantID).Scan(&locType); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: location type: %w", err)
			}
			if domain.LocationType(locType) == domain.LocationTypeQuarantine {
				return nil, domain.ErrQuarantineStockBlocked
			}

			movNumber := fmt.Sprintf("DO-SHIP-%s-%d", do.doNumber, idx+1)

			// If specific batch requested
			if item.batchID != nil && *item.batchID != uuid.Nil {
				var currentStock decimal.Decimal
				err := tx.QueryRowContext(ctx, `
					SELECT COALESCE(
						SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
						0
					)
					FROM stock_movements
					WHERE tenant_id = $1 AND product_id = $3 AND batch_id = $4
					  AND (source_location_id = $2 OR dest_location_id = $2)
					  AND status = 'DONE'`, tenantID, item.locationID, item.productID, *item.batchID).Scan(&currentStock)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: scan batch stock: %w", err)
				}
				if currentStock.LessThan(item.quantity) {
					return nil, domain.ErrInsufficientStock
				}

				movID := uuid.New()
				_, err = tx.ExecContext(ctx, createStockMovementSQL,
					movID, tenantID, movNumber, item.productID,
					item.locationID, custLocID, item.quantity,
					decimal.Zero, domain.StockMovementStatusDone,
					domain.StockRefDeliveryOrder, do.id,
					ptrToNullUUID(&userID), ptrToNullUUID(item.batchID), now)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: insert movement: %w", err)
				}

				auditDetails, _ := json.Marshal(map[string]any{
					"movement_number":    movNumber,
					"product_id":         item.productID,
					"source_location_id": item.locationID,
					"dest_location_id":   custLocID,
					"quantity":           item.quantity,
					"reference_type":     domain.StockRefDeliveryOrder,
					"reference_id":       do.id,
					"batch_id":           item.batchID,
				})
				if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_movement", movID, "created", auditDetails); err != nil {
					return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: audit movement: %w", err)
				}
			} else {
				// FEFO allocation
				balRows, err := tx.QueryContext(ctx, `
					SELECT b.id, b.batch_number, b.expiry_date, b.status, loc.id, loc.code, sm.product_id,
					       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty,
					       b.created_at
					FROM stock_movements sm
					JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
					JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
					WHERE sm.tenant_id = $1 AND sm.product_id = $2 AND loc.id = $3 AND sm.status = 'DONE'
					GROUP BY b.id, b.batch_number, b.expiry_date, b.status, b.created_at, loc.id, loc.code, sm.product_id
					HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
					ORDER BY b.expiry_date ASC NULLS LAST, b.created_at ASC, b.id ASC`, tenantID, item.productID, item.locationID)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: query batches: %w", err)
				}

				var balances []domain.BatchBalance
				for balRows.Next() {
					var bal domain.BatchBalance
					var exp sql.NullTime
					if err := balRows.Scan(
						&bal.BatchID, &bal.BatchNumber, &exp, &bal.Status,
						&bal.LocationID, &bal.LocationCode, &bal.ProductID,
						&bal.Quantity, &bal.CreatedAt,
					); err != nil {
						balRows.Close()
						return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: scan batch: %w", err)
					}
					bal.ExpiryDate = nullTimeToPtr(exp)
					balances = append(balances, bal)
				}
				_ = balRows.Close()

				allocs, err := domain.AllocateFEFO(balances, item.quantity, false)
				if err != nil {
					return nil, err
				}

				for aIdx, alloc := range allocs {
					mID := uuid.New()
					subMovNumber := movNumber
					if len(allocs) > 1 {
						subMovNumber = fmt.Sprintf("%s-B%d", movNumber, aIdx+1)
					}
					_, err := tx.ExecContext(ctx, createStockMovementSQL,
						mID, tenantID, subMovNumber, item.productID,
						item.locationID, custLocID, alloc.Quantity,
						decimal.Zero, domain.StockMovementStatusDone,
						domain.StockRefDeliveryOrder, do.id,
						ptrToNullUUID(&userID), ptrToNullUUID(&alloc.BatchID), now)
					if err != nil {
						return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: insert fefo movement: %w", err)
					}

					auditDetails, _ := json.Marshal(map[string]any{
						"movement_number":    subMovNumber,
						"product_id":         item.productID,
						"source_location_id": item.locationID,
						"dest_location_id":   custLocID,
						"quantity":           alloc.Quantity,
						"reference_type":     domain.StockRefDeliveryOrder,
						"reference_id":       do.id,
						"batch_id":           alloc.BatchID,
					})
					if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_movement", mID, "created", auditDetails); err != nil {
						return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: audit fefo movement: %w", err)
					}
				}
			}
		}

		// Update DO to SHIPPED
		_, err = tx.ExecContext(ctx, `
			UPDATE delivery_orders
			SET status = 'SHIPPED',
			    dispatched_by = $1,
			    dispatched_at = $2,
			    updated_at = $2
			WHERE id = $3 AND tenant_id = $4`,
			userID, now, do.id, tenantID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: update DO status: %w", err)
		}

		doAuditDetails, _ := json.Marshal(map[string]any{
			"do_number":   do.doNumber,
			"manifest_id": manifestID,
			"status":      "SHIPPED",
		})
		if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "delivery_order", do.id, "dispatched", doAuditDetails); err != nil {
			return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: audit DO: %w", err)
		}
	}

	// 5. Update shipping manifest to DISPATCHED
	_, err = tx.ExecContext(ctx, `
		UPDATE shipping_manifests
		SET status = 'DISPATCHED',
		    driver_signature_svg = $1,
		    notes = COALESCE($2, notes),
		    dispatched_by = $3,
		    dispatched_at = $4,
		    updated_at = $4
		WHERE id = $5 AND tenant_id = $6`,
		strings.TrimSpace(req.DriverSignatureSVG), ptrToNullString(req.Notes),
		userID, now, manifestID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: update manifest: %w", err)
	}

	manifestAuditDetails, _ := json.Marshal(map[string]any{
		"manifest_number": manifestNum,
		"status":          "DISPATCHED",
		"total_dos":       len(dos),
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "shipping_manifest", manifestID, "dispatched", manifestAuditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: audit manifest: %w", err)
	}

	// 6. Commit atomic transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.DispatchShippingManifest: commit: %w", err)
	}

	detail, err := r.GetShippingManifestByID(ctx, tenantID, manifestID)
	if err != nil {
		return nil, err
	}
	return &detail.Manifest, nil
}

func (r *WMSRepo) getOrCreateCustomerLocationTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (uuid.UUID, error) {
	var locID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM warehouse_locations
		WHERE tenant_id = $1 AND type = 'CUSTOMER' AND warehouse_id IS NULL
		LIMIT 1`, tenantID).Scan(&locID)
	if err == nil {
		return locID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}

	newID := uuid.New()
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO warehouse_locations (id, tenant_id, code, name, type, is_pallet, created_at, updated_at)
		VALUES ($1, $2, '@CUSTOMER', 'System Virtual CUSTOMER', 'CUSTOMER', false, $3, $3)`,
		newID, tenantID, now)
	if err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

// GetWMSOutboundKPIs computes 8 WMS operational metrics across inbound receipts, picking, dispatch and backlogs.
func (r *WMSRepo) GetWMSOutboundKPIs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) (*domain.WMSOutboundKPISummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: set tenant: %w", err)
	}

	var summary domain.WMSOutboundKPISummary
	summary.ReceivingAccuracyPct = 100.0
	summary.POCompliancePct = 100.0
	summary.PickingAccuracyPct = 100.0
	summary.OnTimeShipmentPct = 100.0

	// 1. DockToStockAvgMinutes: avg time from stock_receipts created_at to posted_at (minutes)
	q1 := `SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (posted_at - created_at)) / 60.0), 0.0)
	       FROM stock_receipts
	       WHERE tenant_id = $1 AND status = 'POSTED' AND posted_at IS NOT NULL`
	args1 := []any{tenantID}
	if warehouseID != nil {
		q1 += ` AND warehouse_id = $2`
		args1 = append(args1, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q1, args1...).Scan(&summary.DockToStockAvgMinutes); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: dock to stock: %w", err)
	}

	// 2. ReceivingAccuracyPct: accepted / (accepted + rejected) * 100
	q2 := `SELECT COALESCE(SUM(sri.accepted_qty) / NULLIF(SUM(sri.accepted_qty + sri.rejected_qty), 0) * 100.0, 100.0)
	       FROM stock_receipt_items sri
	       JOIN stock_receipts sr ON sr.id = sri.receipt_id AND sr.tenant_id = sri.tenant_id
	       WHERE sri.tenant_id = $1 AND sr.status = 'POSTED'`
	args2 := []any{tenantID}
	if warehouseID != nil {
		q2 += ` AND sr.warehouse_id = $2`
		args2 = append(args2, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q2, args2...).Scan(&summary.ReceivingAccuracyPct); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: receiving accuracy: %w", err)
	}

	// 3. POCompliancePct: accepted vs expected
	q3 := `SELECT COALESCE(SUM(LEAST(sri.accepted_qty, sri.expected_qty)) / NULLIF(SUM(sri.expected_qty), 0) * 100.0, 100.0)
	       FROM stock_receipt_items sri
	       JOIN stock_receipts sr ON sr.id = sri.receipt_id AND sr.tenant_id = sri.tenant_id
	       WHERE sri.tenant_id = $1 AND sr.status = 'POSTED' AND sri.expected_qty IS NOT NULL AND sri.expected_qty > 0`
	args3 := []any{tenantID}
	if warehouseID != nil {
		q3 += ` AND sr.warehouse_id = $2`
		args3 = append(args3, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q3, args3...).Scan(&summary.POCompliancePct); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: po compliance: %w", err)
	}

	// 4. BacklogInboundCount: count DRAFT stock_receipts
	q4 := `SELECT COUNT(*) FROM stock_receipts WHERE tenant_id = $1 AND status = 'DRAFT'`
	args4 := []any{tenantID}
	if warehouseID != nil {
		q4 += ` AND warehouse_id = $2`
		args4 = append(args4, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q4, args4...).Scan(&summary.BacklogInboundCount); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: backlog inbound: %w", err)
	}

	// 5. OrderToDispatchAvgHours: avg time from DO created_at to dispatched_at (hours)
	q5 := `SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (dispatched_at - created_at)) / 3600.0), 0.0)
	       FROM delivery_orders
	       WHERE tenant_id = $1 AND dispatched_at IS NOT NULL`
	args5 := []any{tenantID}
	if warehouseID != nil {
		q5 += ` AND warehouse_id = $2`
		args5 = append(args5, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q5, args5...).Scan(&summary.OrderToDispatchAvgHours); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: order to dispatch: %w", err)
	}

	// 6. PickingAccuracyPct: picking tasks lines with damaged_qty == 0 vs total
	q6 := `SELECT COALESCE(SUM(CASE WHEN pti.damaged_qty = 0 THEN 1.0 ELSE 0.0 END) / NULLIF(COUNT(*), 0) * 100.0, 100.0)
	       FROM picking_task_items pti
	       JOIN picking_tasks pt ON pt.id = pti.task_id AND pt.tenant_id = pti.tenant_id
	       JOIN delivery_orders do ON do.id = pt.delivery_order_id AND do.tenant_id = pt.tenant_id
	       WHERE pti.tenant_id = $1`
	args6 := []any{tenantID}
	if warehouseID != nil {
		q6 += ` AND do.warehouse_id = $2`
		args6 = append(args6, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q6, args6...).Scan(&summary.PickingAccuracyPct); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: picking accuracy: %w", err)
	}

	// 7. OnTimeShipmentPct: shipped / delivered DOs vs total non-cancelled
	q7 := `SELECT COALESCE(SUM(CASE WHEN status IN ('SHIPPED', 'DELIVERED') THEN 1.0 ELSE 0.0 END) / NULLIF(COUNT(*), 0) * 100.0, 100.0)
	       FROM delivery_orders
	       WHERE tenant_id = $1 AND status != 'CANCELLED'`
	args7 := []any{tenantID}
	if warehouseID != nil {
		q7 += ` AND warehouse_id = $2`
		args7 = append(args7, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q7, args7...).Scan(&summary.OnTimeShipmentPct); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: on time shipment: %w", err)
	}

	// 8. BacklogOutboundCount: DOs in DRAFT, CONFIRMED, PICKED, PACKED, STAGED
	q8 := `SELECT COUNT(*) FROM delivery_orders WHERE tenant_id = $1 AND status IN ('DRAFT', 'CONFIRMED', 'PICKED', 'PACKED', 'STAGED')`
	args8 := []any{tenantID}
	if warehouseID != nil {
		q8 += ` AND warehouse_id = $2`
		args8 = append(args8, *warehouseID)
	}
	if err := tx.QueryRowContext(ctx, q8, args8...).Scan(&summary.BacklogOutboundCount); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: backlog outbound: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSOutboundKPIs: commit: %w", err)
	}

	// Safe rounding
	summary.DockToStockAvgMinutes = math.Round(summary.DockToStockAvgMinutes*10) / 10
	summary.ReceivingAccuracyPct = math.Round(summary.ReceivingAccuracyPct*10) / 10
	summary.POCompliancePct = math.Round(summary.POCompliancePct*10) / 10
	summary.OrderToDispatchAvgHours = math.Round(summary.OrderToDispatchAvgHours*10) / 10
	summary.PickingAccuracyPct = math.Round(summary.PickingAccuracyPct*10) / 10
	summary.OnTimeShipmentPct = math.Round(summary.OnTimeShipmentPct*10) / 10

	return &summary, nil
}
