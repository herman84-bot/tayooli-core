package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShippingManifestRequest_Validate(t *testing.T) {
	validWH := uuid.New()
	validDOs := []uuid.UUID{uuid.New()}

	tests := []struct {
		name    string
		req     domain.CreateShippingManifestRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      validWH,
				ExpeditionName:   "JNE",
				DriverName:       "Budi Santoso",
				VehiclePlate:     "B 1234 CD",
				DeliveryOrderIDs: validDOs,
			},
			wantErr: nil,
		},
		{
			name: "missing warehouse",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      uuid.Nil,
				ExpeditionName:   "JNE",
				DriverName:       "Budi Santoso",
				VehiclePlate:     "B 1234 CD",
				DeliveryOrderIDs: validDOs,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty expedition name",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      validWH,
				ExpeditionName:   "   ",
				DriverName:       "Budi Santoso",
				VehiclePlate:     "B 1234 CD",
				DeliveryOrderIDs: validDOs,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty driver name",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      validWH,
				ExpeditionName:   "JNE",
				DriverName:       "",
				VehiclePlate:     "B 1234 CD",
				DeliveryOrderIDs: validDOs,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty vehicle plate",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      validWH,
				ExpeditionName:   "JNE",
				DriverName:       "Budi Santoso",
				VehiclePlate:     "   ",
				DeliveryOrderIDs: validDOs,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "empty delivery orders list",
			req: domain.CreateShippingManifestRequest{
				WarehouseID:      validWH,
				ExpeditionName:   "JNE",
				DriverName:       "Budi Santoso",
				VehiclePlate:     "B 1234 CD",
				DeliveryOrderIDs: nil,
			},
			wantErr: domain.ErrManifestEmpty,
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

func TestDispatchShippingManifestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.DispatchShippingManifestRequest
		wantErr error
	}{
		{
			name: "valid svg signature",
			req: domain.DispatchShippingManifestRequest{
				DriverSignatureSVG: "<svg><path d=\"M10 10\"/></svg>",
			},
			wantErr: nil,
		},
		{
			name: "empty signature",
			req: domain.DispatchShippingManifestRequest{
				DriverSignatureSVG: "",
			},
			wantErr: domain.ErrManifestSignatureRequired,
		},
		{
			name: "too short signature",
			req: domain.DispatchShippingManifestRequest{
				DriverSignatureSVG: "<svg></svg>", // 11 chars but trimmed len <= 10 if short
			},
			wantErr: nil,
		},
		{
			name: "whitespace only signature",
			req: domain.DispatchShippingManifestRequest{
				DriverSignatureSVG: "          ",
			},
			wantErr: domain.ErrManifestSignatureRequired,
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
