package wms_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockLPNUsecase(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminID := uuid.New()
	auditorID := uuid.New()
	stafID := uuid.New()
	whID := uuid.New()
	unassignedWhID := uuid.New()

	repo := newMockWMSRepo()
	repo.warehouses[whID] = domain.Warehouse{
		ID:       whID,
		TenantID: tenantID,
		Code:     "WH-01",
		Name:     "Main Distribution Center",
		IsActive: true,
	}
	repo.warehouses[unassignedWhID] = domain.Warehouse{
		ID:       unassignedWhID,
		TenantID: tenantID,
		Code:     "WH-UNASSIGNED",
		Name:     "Restricted Facility",
		IsActive: true,
	}
	// Assign stafID to whID only
	repo.userWarehouses[tenantID.String()+":"+stafID.String()] = []uuid.UUID{whID}

	usecase := uc.New(repo)

	// Set global mock warehouse IDs
	testDockWarehouseID = whID
	testAppointmentWarehouseID = whID
	testLPNWarehouseID = whID

	// Reset mock hooks after tests
	defer func() {
		testDockWarehouseID = uuid.Nil
		testAppointmentWarehouseID = uuid.Nil
		testLPNWarehouseID = uuid.Nil
		mockGetDockByIDFn = nil
		mockGetAppointmentByIDFn = nil
		mockGetLPNByIDFn = nil
	}()

	t.Run("CreateDock - validation and access control", func(t *testing.T) {
		// 1. Validation failure: empty dock name
		_, err := usecase.CreateDock(ctx, tenantID, adminID, "admin", domain.CreateDockRequest{
			WarehouseID: whID,
			DockName:    "   ",
			DockType:    domain.DockTypeInbound,
			MaxTonnage:  decimal.NewFromFloat(20),
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// 2. Validation failure: invalid dock type
		_, err = usecase.CreateDock(ctx, tenantID, adminID, "admin", domain.CreateDockRequest{
			WarehouseID: whID,
			DockName:    "Dock A",
			DockType:    domain.DockType("UNKNOWN"),
			MaxTonnage:  decimal.NewFromFloat(20),
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// 3. Auditor role forbidden
		_, err = usecase.CreateDock(ctx, tenantID, auditorID, "auditor", domain.CreateDockRequest{
			WarehouseID: whID,
			DockName:    "Dock A",
			DockType:    domain.DockTypeInbound,
			MaxTonnage:  decimal.NewFromFloat(20),
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// 4. Warehouse staff accessing unassigned warehouse
		_, err = usecase.CreateDock(ctx, tenantID, stafID, "warehouse", domain.CreateDockRequest{
			WarehouseID: unassignedWhID,
			DockName:    "Dock Unassigned",
			DockType:    domain.DockTypeInbound,
			MaxTonnage:  decimal.NewFromFloat(20),
		})
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		// 5. Success by admin
		dock, err := usecase.CreateDock(ctx, tenantID, adminID, "admin", domain.CreateDockRequest{
			WarehouseID: whID,
			DockCode:    "DOCK-01",
			DockName:    "Inbound Dock 1",
			DockType:    domain.DockTypeInbound,
			MaxTonnage:  decimal.NewFromFloat(25.5),
		})
		require.NoError(t, err)
		assert.Equal(t, "Inbound Dock 1", dock.DockName)
		assert.Equal(t, domain.DockStatusAvailable, dock.Status)
	})

	t.Run("GetDock & ListDocks - read access", func(t *testing.T) {
		dockID := uuid.New()

		// GetDock: repo error
		mockGetDockByIDFn = func(ctx context.Context, tid, id uuid.UUID) (*domain.InboundDock, error) {
			return nil, domain.ErrDockNotFound
		}
		_, err := usecase.GetDock(ctx, tenantID, adminID, "admin", dockID)
		require.ErrorIs(t, err, domain.ErrDockNotFound)
		mockGetDockByIDFn = nil

		// GetDock: unauthorized warehouse for staff
		testDockWarehouseID = unassignedWhID
		_, err = usecase.GetDock(ctx, tenantID, stafID, "warehouse", dockID)
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)
		testDockWarehouseID = whID

		// GetDock: success
		dock, err := usecase.GetDock(ctx, tenantID, stafID, "warehouse", dockID)
		require.NoError(t, err)
		assert.Equal(t, dockID, dock.ID)

		// ListDocks: unauthorized warehouse for staff
		_, err = usecase.ListDocks(ctx, tenantID, stafID, "warehouse", unassignedWhID, nil)
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		// ListDocks: success with status filter
		st := domain.DockStatusAvailable
		docks, err := usecase.ListDocks(ctx, tenantID, stafID, "warehouse", whID, &st)
		require.NoError(t, err)
		assert.NotNil(t, docks)
	})

	t.Run("UpdateDockStatus - validation and write access", func(t *testing.T) {
		dockID := uuid.New()

		// Invalid status
		_, err := usecase.UpdateDockStatus(ctx, tenantID, adminID, "admin", dockID, domain.UpdateDockStatusRequest{
			Status: domain.DockStatus("INVALID_STATUS"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidStatus)

		// Auditor forbidden
		_, err = usecase.UpdateDockStatus(ctx, tenantID, auditorID, "auditor", dockID, domain.UpdateDockStatusRequest{
			Status: domain.DockStatusOccupied,
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		updated, err := usecase.UpdateDockStatus(ctx, tenantID, adminID, "admin", dockID, domain.UpdateDockStatusRequest{
			Status: domain.DockStatusOccupied,
		})
		require.NoError(t, err)
		assert.Equal(t, domain.DockStatusOccupied, updated.Status)
	})

	t.Run("CreateAppointment - validation and access", func(t *testing.T) {
		// Validation failure: missing plate
		_, err := usecase.CreateAppointment(ctx, tenantID, adminID, "admin", domain.CreateAppointmentRequest{
			WarehouseID:      whID,
			VendorName:       "PT Vendor",
			VehiclePlate:     "   ",
			DriverName:       "Supir",
			EstimatedArrival: time.Now().Add(2 * time.Hour),
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Validation failure: zero ETA
		_, err = usecase.CreateAppointment(ctx, tenantID, adminID, "admin", domain.CreateAppointmentRequest{
			WarehouseID:  whID,
			VendorName:   "PT Vendor",
			VehiclePlate: "B 1234 CD",
			DriverName:   "Supir",
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Auditor forbidden
		_, err = usecase.CreateAppointment(ctx, tenantID, auditorID, "auditor", domain.CreateAppointmentRequest{
			WarehouseID:      whID,
			VendorName:       "PT Vendor",
			VehiclePlate:     "B 1234 CD",
			DriverName:       "Supir",
			EstimatedArrival: time.Now().Add(2 * time.Hour),
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Staff unauthorized warehouse
		_, err = usecase.CreateAppointment(ctx, tenantID, stafID, "warehouse", domain.CreateAppointmentRequest{
			WarehouseID:      unassignedWhID,
			VendorName:       "PT Vendor",
			VehiclePlate:     "B 1234 CD",
			DriverName:       "Supir",
			EstimatedArrival: time.Now().Add(2 * time.Hour),
		})
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		// Success
		app, err := usecase.CreateAppointment(ctx, tenantID, stafID, "warehouse", domain.CreateAppointmentRequest{
			WarehouseID:      whID,
			VendorName:       "PT Pemasok Utama",
			VehiclePlate:     "B 9876 XYZ",
			DriverName:       "Joko",
			EstimatedArrival: time.Now().Add(3 * time.Hour),
		})
		require.NoError(t, err)
		assert.Equal(t, "PT Pemasok Utama", app.VendorName)
		assert.Equal(t, domain.AppointmentStatusScheduled, app.Status)
	})

	t.Run("GetAppointment & ListAppointments - read access", func(t *testing.T) {
		appID := uuid.New()

		// GetAppointment: repo error
		mockGetAppointmentByIDFn = func(ctx context.Context, tid, id uuid.UUID) (*domain.DockAppointment, error) {
			return nil, domain.ErrAppointmentNotFound
		}
		_, err := usecase.GetAppointment(ctx, tenantID, adminID, "admin", appID)
		require.ErrorIs(t, err, domain.ErrAppointmentNotFound)
		mockGetAppointmentByIDFn = nil

		// GetAppointment: staff unauthorized warehouse
		testAppointmentWarehouseID = unassignedWhID
		_, err = usecase.GetAppointment(ctx, tenantID, stafID, "warehouse", appID)
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)
		testAppointmentWarehouseID = whID

		// GetAppointment: success
		app, err := usecase.GetAppointment(ctx, tenantID, stafID, "warehouse", appID)
		require.NoError(t, err)
		assert.Equal(t, appID, app.ID)

		// ListAppointments: success
		st := domain.AppointmentStatusScheduled
		apps, err := usecase.ListAppointments(ctx, tenantID, stafID, "warehouse", whID, &st)
		require.NoError(t, err)
		assert.NotNil(t, apps)
	})

	t.Run("AssignDockToAppointment - validation and write access", func(t *testing.T) {
		appID := uuid.New()
		dockID := uuid.New()

		// Invalid dock ID
		_, err := usecase.AssignDockToAppointment(ctx, tenantID, adminID, "admin", appID, uuid.Nil)
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Auditor forbidden
		_, err = usecase.AssignDockToAppointment(ctx, tenantID, auditorID, "auditor", appID, dockID)
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Appointment not found
		mockGetAppointmentByIDFn = func(ctx context.Context, tid, id uuid.UUID) (*domain.DockAppointment, error) {
			return nil, domain.ErrAppointmentNotFound
		}
		_, err = usecase.AssignDockToAppointment(ctx, tenantID, adminID, "admin", appID, dockID)
		require.ErrorIs(t, err, domain.ErrAppointmentNotFound)
		mockGetAppointmentByIDFn = nil

		// Success
		assigned, err := usecase.AssignDockToAppointment(ctx, tenantID, adminID, "admin", appID, dockID)
		require.NoError(t, err)
		assert.Equal(t, &dockID, assigned.DockID)
	})

	t.Run("UpdateAppointmentStatus - validation and write access", func(t *testing.T) {
		appID := uuid.New()

		// Invalid status
		_, err := usecase.UpdateAppointmentStatus(ctx, tenantID, adminID, "admin", appID, domain.UpdateAppointmentStatusRequest{
			Status: domain.AppointmentStatus("INVALID"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidStatus)

		// Auditor forbidden
		_, err = usecase.UpdateAppointmentStatus(ctx, tenantID, auditorID, "auditor", appID, domain.UpdateAppointmentStatusRequest{
			Status: domain.AppointmentStatusArrived,
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		updated, err := usecase.UpdateAppointmentStatus(ctx, tenantID, adminID, "admin", appID, domain.UpdateAppointmentStatusRequest{
			Status: domain.AppointmentStatusArrived,
		})
		require.NoError(t, err)
		assert.Equal(t, domain.AppointmentStatusArrived, updated.Status)
	})

	t.Run("CreateLPN - validation and write access", func(t *testing.T) {
		locID := uuid.New()

		// Invalid DTO: missing warehouse ID
		_, err := usecase.CreateLPN(ctx, tenantID, adminID, "admin", domain.CreateLPNRequest{
			LocationID: locID,
			PalletType: domain.PalletTypeWooden,
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Invalid DTO: invalid pallet type
		_, err = usecase.CreateLPN(ctx, tenantID, adminID, "admin", domain.CreateLPNRequest{
			WarehouseID: whID,
			LocationID:  locID,
			PalletType:  domain.PalletType("PAPER"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Auditor forbidden
		_, err = usecase.CreateLPN(ctx, tenantID, auditorID, "auditor", domain.CreateLPNRequest{
			WarehouseID: whID,
			LocationID:  locID,
			PalletType:  domain.PalletTypeWooden,
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		lpn, err := usecase.CreateLPN(ctx, tenantID, stafID, "warehouse", domain.CreateLPNRequest{
			WarehouseID: whID,
			LocationID:  locID,
			PalletType:  domain.PalletTypeWooden,
			MaxWeightKg: decimal.NewFromFloat(1500),
		})
		require.NoError(t, err)
		assert.Equal(t, domain.LPNStatusStaged, lpn.Status)
		assert.Equal(t, domain.PalletTypeWooden, lpn.PalletType)
	})

	t.Run("GetLPN & ListLPNs - read access", func(t *testing.T) {
		lpnID := uuid.New()

		// GetLPN: repo error
		mockGetLPNByIDFn = func(ctx context.Context, tid, id uuid.UUID) (*domain.StockLPNDetail, error) {
			return nil, domain.ErrLPNNotFound
		}
		_, err := usecase.GetLPN(ctx, tenantID, adminID, "admin", lpnID)
		require.ErrorIs(t, err, domain.ErrLPNNotFound)
		mockGetLPNByIDFn = nil

		// GetLPN: unauthorized warehouse for staff
		testLPNWarehouseID = unassignedWhID
		_, err = usecase.GetLPN(ctx, tenantID, stafID, "warehouse", lpnID)
		require.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)
		testLPNWarehouseID = whID

		// GetLPN: success
		detail, err := usecase.GetLPN(ctx, tenantID, stafID, "warehouse", lpnID)
		require.NoError(t, err)
		assert.Equal(t, lpnID, detail.LPN.ID)

		// ListLPNs: success
		st := domain.LPNStatusStaged
		lpns, err := usecase.ListLPNs(ctx, tenantID, stafID, "warehouse", whID, &st)
		require.NoError(t, err)
		assert.NotNil(t, lpns)
	})

	t.Run("AddLPNItem - validation and write access", func(t *testing.T) {
		lpnID := uuid.New()
		prodID := uuid.New()
		batchID := uuid.New()

		// Validation failure: non-positive quantity
		_, err := usecase.AddLPNItem(ctx, tenantID, adminID, "admin", lpnID, domain.AddLPNItemRequest{
			ProductID: prodID,
			BatchID:   batchID,
			Quantity:  decimal.NewFromFloat(0),
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Auditor forbidden
		_, err = usecase.AddLPNItem(ctx, tenantID, auditorID, "auditor", lpnID, domain.AddLPNItemRequest{
			ProductID: prodID,
			BatchID:   batchID,
			Quantity:  decimal.NewFromFloat(10),
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		detail, err := usecase.AddLPNItem(ctx, tenantID, stafID, "warehouse", lpnID, domain.AddLPNItemRequest{
			ProductID: prodID,
			BatchID:   batchID,
			Quantity:  decimal.NewFromFloat(100),
		})
		require.NoError(t, err)
		assert.NotNil(t, detail)
	})

	t.Run("MoveLPN - validation, access and repo delegation", func(t *testing.T) {
		lpnID := uuid.New()
		targetLocID := uuid.New()

		// Validation failure: missing target location
		_, err := usecase.MoveLPN(ctx, tenantID, adminID, "admin", lpnID, domain.MoveLPNRequest{
			TargetLocationID: uuid.Nil,
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// LPN not found
		mockGetLPNByIDFn = func(ctx context.Context, tid, id uuid.UUID) (*domain.StockLPNDetail, error) {
			return nil, domain.ErrLPNNotFound
		}
		_, err = usecase.MoveLPN(ctx, tenantID, adminID, "admin", lpnID, domain.MoveLPNRequest{
			TargetLocationID: targetLocID,
		})
		require.ErrorIs(t, err, domain.ErrLPNNotFound)
		mockGetLPNByIDFn = nil

		// Auditor forbidden
		_, err = usecase.MoveLPN(ctx, tenantID, auditorID, "auditor", lpnID, domain.MoveLPNRequest{
			TargetLocationID: targetLocID,
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		detail, err := usecase.MoveLPN(ctx, tenantID, stafID, "warehouse", lpnID, domain.MoveLPNRequest{
			TargetLocationID: targetLocID,
		})
		require.NoError(t, err)
		assert.Equal(t, targetLocID, detail.LPN.LocationID)
		assert.Equal(t, domain.LPNStatusStored, detail.LPN.Status)
	})
}
