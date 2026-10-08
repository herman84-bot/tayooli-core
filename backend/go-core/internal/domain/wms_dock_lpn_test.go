package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDockRequest_Validate(t *testing.T) {
	validWH := uuid.New()

	tests := []struct {
		name    string
		req     domain.CreateDockRequest
		wantErr error
	}{
		{
			name: "valid request with all fields",
			req: domain.CreateDockRequest{
				WarehouseID: validWH,
				DockCode:    "DOCK-01",
				DockName:    "Dock Inbound 1",
				DockType:    domain.DockTypeInbound,
				MaxTonnage:  decimal.NewFromFloat(15.5),
			},
			wantErr: nil,
		},
		{
			name: "valid request minimal",
			req: domain.CreateDockRequest{
				WarehouseID: validWH,
				DockName:    "Dock Inbound 2",
			},
			wantErr: nil,
		},
		{
			name: "missing warehouse id",
			req: domain.CreateDockRequest{
				WarehouseID: uuid.Nil,
				DockName:    "Dock 1",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty dock name",
			req: domain.CreateDockRequest{
				WarehouseID: validWH,
				DockName:    "   ",
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "invalid dock type",
			req: domain.CreateDockRequest{
				WarehouseID: validWH,
				DockName:    "Dock 1",
				DockType:    domain.DockType("UNKNOWN"),
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "negative max tonnage",
			req: domain.CreateDockRequest{
				WarehouseID: validWH,
				DockName:    "Dock 1",
				MaxTonnage:  decimal.NewFromFloat(-5.0),
			},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateDockStatusRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.UpdateDockStatusRequest
		wantErr error
	}{
		{
			name:    "valid AVAILABLE",
			req:     domain.UpdateDockStatusRequest{Status: domain.DockStatusAvailable},
			wantErr: nil,
		},
		{
			name:    "valid OCCUPIED",
			req:     domain.UpdateDockStatusRequest{Status: domain.DockStatusOccupied},
			wantErr: nil,
		},
		{
			name:    "valid MAINTENANCE",
			req:     domain.UpdateDockStatusRequest{Status: domain.DockStatusMaintenance},
			wantErr: nil,
		},
		{
			name:    "invalid status",
			req:     domain.UpdateDockStatusRequest{Status: domain.DockStatus("INVALID")},
			wantErr: domain.ErrInvalidStatus,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateAppointmentRequest_Validate(t *testing.T) {
	validWH := uuid.New()
	validETA := time.Now().Add(2 * time.Hour)

	tests := []struct {
		name    string
		req     domain.CreateAppointmentRequest
		wantErr error
	}{
		{
			name: "valid appointment request",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      validWH,
				VendorName:       "PT Ekspedisi Jaya",
				VehiclePlate:     "B 9876 XYZ",
				DriverName:       "Agus Subroto",
				EstimatedArrival: validETA,
			},
			wantErr: nil,
		},
		{
			name: "missing warehouse id",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      uuid.Nil,
				VendorName:       "PT Ekspedisi Jaya",
				VehiclePlate:     "B 9876 XYZ",
				DriverName:       "Agus Subroto",
				EstimatedArrival: validETA,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty vendor name",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      validWH,
				VendorName:       "  ",
				VehiclePlate:     "B 9876 XYZ",
				DriverName:       "Agus Subroto",
				EstimatedArrival: validETA,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty vehicle plate",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      validWH,
				VendorName:       "PT Ekspedisi Jaya",
				VehiclePlate:     "",
				DriverName:       "Agus Subroto",
				EstimatedArrival: validETA,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty driver name",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      validWH,
				VendorName:       "PT Ekspedisi Jaya",
				VehiclePlate:     "B 9876 XYZ",
				DriverName:       "   ",
				EstimatedArrival: validETA,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "zero estimated arrival",
			req: domain.CreateAppointmentRequest{
				WarehouseID:      validWH,
				VendorName:       "PT Ekspedisi Jaya",
				VehiclePlate:     "B 9876 XYZ",
				DriverName:       "Agus Subroto",
				EstimatedArrival: time.Time{},
			},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAssignDockRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.AssignDockRequest
		wantErr error
	}{
		{
			name:    "valid dock id",
			req:     domain.AssignDockRequest{DockID: uuid.New()},
			wantErr: nil,
		},
		{
			name:    "missing dock id",
			req:     domain.AssignDockRequest{DockID: uuid.Nil},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateAppointmentStatusRequest_Validate(t *testing.T) {
	validStatuses := []domain.AppointmentStatus{
		domain.AppointmentStatusScheduled,
		domain.AppointmentStatusArrived,
		domain.AppointmentStatusUnloading,
		domain.AppointmentStatusCompleted,
		domain.AppointmentStatusCancelled,
	}

	for _, s := range validStatuses {
		t.Run("valid status "+string(s), func(t *testing.T) {
			req := domain.UpdateAppointmentStatusRequest{Status: s}
			assert.NoError(t, req.Validate())
		})
	}

	t.Run("invalid status", func(t *testing.T) {
		req := domain.UpdateAppointmentStatusRequest{Status: domain.AppointmentStatus("INVALID")}
		assert.ErrorIs(t, req.Validate(), domain.ErrInvalidStatus)
	})
}

func TestCreateLPNRequest_Validate(t *testing.T) {
	validWH := uuid.New()
	validLoc := uuid.New()

	tests := []struct {
		name    string
		req     domain.CreateLPNRequest
		wantErr error
	}{
		{
			name: "valid lpn request",
			req: domain.CreateLPNRequest{
				WarehouseID: validWH,
				LocationID:  validLoc,
				PalletType:  domain.PalletTypeWooden,
				MaxWeightKg: decimal.NewFromFloat(1200.0),
			},
			wantErr: nil,
		},
		{
			name: "missing warehouse id",
			req: domain.CreateLPNRequest{
				WarehouseID: uuid.Nil,
				LocationID:  validLoc,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "missing location id",
			req: domain.CreateLPNRequest{
				WarehouseID: validWH,
				LocationID:  uuid.Nil,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "invalid pallet type",
			req: domain.CreateLPNRequest{
				WarehouseID: validWH,
				LocationID:  validLoc,
				PalletType:  domain.PalletType("PAPER"),
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "negative max weight",
			req: domain.CreateLPNRequest{
				WarehouseID: validWH,
				LocationID:  validLoc,
				MaxWeightKg: decimal.NewFromFloat(-10.0),
			},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddLPNItemRequest_Validate(t *testing.T) {
	validProd := uuid.New()
	validBatch := uuid.New()

	tests := []struct {
		name    string
		req     domain.AddLPNItemRequest
		wantErr error
	}{
		{
			name: "valid add item request",
			req: domain.AddLPNItemRequest{
				ProductID: validProd,
				BatchID:   validBatch,
				Quantity:  decimal.NewFromFloat(50.0),
			},
			wantErr: nil,
		},
		{
			name: "missing product id",
			req: domain.AddLPNItemRequest{
				ProductID: uuid.Nil,
				BatchID:   validBatch,
				Quantity:  decimal.NewFromFloat(50.0),
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "missing batch id",
			req: domain.AddLPNItemRequest{
				ProductID: validProd,
				BatchID:   uuid.Nil,
				Quantity:  decimal.NewFromFloat(50.0),
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "zero quantity",
			req: domain.AddLPNItemRequest{
				ProductID: validProd,
				BatchID:   validBatch,
				Quantity:  decimal.Zero,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "negative quantity",
			req: domain.AddLPNItemRequest{
				ProductID: validProd,
				BatchID:   validBatch,
				Quantity:  decimal.NewFromFloat(-5.0),
			},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMoveLPNRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.MoveLPNRequest
		wantErr error
	}{
		{
			name:    "valid move request",
			req:     domain.MoveLPNRequest{TargetLocationID: uuid.New()},
			wantErr: nil,
		},
		{
			name:    "missing target location id",
			req:     domain.MoveLPNRequest{TargetLocationID: uuid.Nil},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
