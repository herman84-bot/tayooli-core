package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Verify interface implementation at compile time.
var (
	_ domain.WMSDockLPNRepository = (*postgres.WMSRepo)(nil)
)

func TestWMSRepo_DockLPNValidationFastPaths(t *testing.T) {
	repo := postgres.NewWMSRepo(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("CreateDock rejects missing warehouse", func(t *testing.T) {
		req := domain.CreateDockRequest{
			WarehouseID: uuid.Nil,
			DockName:    "Dock 1",
		}
		_, err := repo.CreateDock(ctx, tenantID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("CreateDock rejects empty dock name", func(t *testing.T) {
		req := domain.CreateDockRequest{
			WarehouseID: uuid.New(),
			DockName:    "   ",
		}
		_, err := repo.CreateDock(ctx, tenantID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("UpdateDockStatus rejects invalid status", func(t *testing.T) {
		req := domain.UpdateDockStatusRequest{
			Status: domain.DockStatus("INVALID_STATUS"),
		}
		_, err := repo.UpdateDockStatus(ctx, tenantID, uuid.New(), req)
		require.ErrorIs(t, err, domain.ErrInvalidStatus)
	})

	t.Run("CreateAppointment rejects missing vendor name", func(t *testing.T) {
		req := domain.CreateAppointmentRequest{
			WarehouseID:      uuid.New(),
			VendorName:       "",
			VehiclePlate:     "B 1234 CD",
			DriverName:       "Agus",
			EstimatedArrival: time.Now(),
		}
		_, err := repo.CreateAppointment(ctx, tenantID, userID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("CreateAppointment rejects zero estimated arrival", func(t *testing.T) {
		req := domain.CreateAppointmentRequest{
			WarehouseID:      uuid.New(),
			VendorName:       "PT Maju",
			VehiclePlate:     "B 1234 CD",
			DriverName:       "Agus",
			EstimatedArrival: time.Time{},
		}
		_, err := repo.CreateAppointment(ctx, tenantID, userID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("AssignDockToAppointment rejects missing dock ID", func(t *testing.T) {
		req := domain.AssignDockRequest{
			DockID: uuid.Nil,
		}
		_, err := repo.AssignDockToAppointment(ctx, tenantID, uuid.New(), req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("UpdateAppointmentStatus rejects invalid status", func(t *testing.T) {
		req := domain.UpdateAppointmentStatusRequest{
			Status: domain.AppointmentStatus("UNKNOWN_STATUS"),
		}
		_, err := repo.UpdateAppointmentStatus(ctx, tenantID, uuid.New(), req)
		require.ErrorIs(t, err, domain.ErrInvalidStatus)
	})

	t.Run("CreateLPN rejects missing location ID", func(t *testing.T) {
		req := domain.CreateLPNRequest{
			WarehouseID: uuid.New(),
			LocationID:  uuid.Nil,
		}
		_, err := repo.CreateLPN(ctx, tenantID, userID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("AddLPNItem rejects non-positive quantity", func(t *testing.T) {
		req := domain.AddLPNItemRequest{
			ProductID: uuid.New(),
			BatchID:   uuid.New(),
			Quantity:  decimal.Zero,
		}
		_, err := repo.AddLPNItem(ctx, tenantID, uuid.New(), req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("MoveLPN rejects missing target location ID", func(t *testing.T) {
		req := domain.MoveLPNRequest{
			TargetLocationID: uuid.Nil,
		}
		_, err := repo.MoveLPN(ctx, tenantID, userID, uuid.New(), req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}
