package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWMSManifestEndpoints(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	role := "admin"

	t.Run("GET /api/v1/wms/manifests returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listShippingManifestsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID, status *domain.ShippingManifestStatus, expName *string) ([]domain.ShippingManifest, error) {
				return []domain.ShippingManifest{
					{
						ID:             uuid.New(),
						TenantID:       tid,
						ManifestNumber: "SM-2026-001",
						ExpeditionName: "JNE",
						DriverName:     "Budi",
						VehiclePlate:   "B 1234 CD",
						TotalPackages:  3,
						TotalWeightKg:  decimal.NewFromFloat(15.5),
						Status:         domain.ShippingManifestStatusStaged,
					},
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/manifests?status=STAGED&expedition_name=JNE", nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []domain.ShippingManifest `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, "SM-2026-001", resp.Data[0].ManifestNumber)
	})

	t.Run("POST /api/v1/wms/manifests success and invalid validation", func(t *testing.T) {
		whID := uuid.New()
		doID := uuid.New()

		mock := &mockWMSUsecase{
			createShippingManifestFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req domain.CreateShippingManifestRequest) (*domain.ShippingManifest, error) {
				if len(req.DeliveryOrderIDs) == 0 {
					return nil, domain.ErrManifestEmpty
				}
				return &domain.ShippingManifest{
					ID:             uuid.New(),
					TenantID:       tid,
					WarehouseID:    req.WarehouseID,
					ManifestNumber: "SM-2026-002",
					ExpeditionName: req.ExpeditionName,
					DriverName:     req.DriverName,
					VehiclePlate:   req.VehiclePlate,
					TotalPackages:  len(req.DeliveryOrderIDs),
					Status:         domain.ShippingManifestStatusStaged,
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. Success case
		payload := domain.CreateShippingManifestRequest{
			WarehouseID:      whID,
			ExpeditionName:   "SiCepat",
			DriverName:       "Siti",
			VehiclePlate:     "D 5678 EF",
			DeliveryOrderIDs: []uuid.UUID{doID},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/wms/manifests", bytes.NewReader(body))
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var successResp struct {
			Data domain.ShippingManifest `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&successResp)
		require.NoError(t, err)
		assert.Equal(t, "SM-2026-002", successResp.Data.ManifestNumber)
		assert.Equal(t, domain.ShippingManifestStatusStaged, successResp.Data.Status)

		// 2. Invalid validation (empty DOs)
		invalidPayload := domain.CreateShippingManifestRequest{
			WarehouseID:      whID,
			ExpeditionName:   "SiCepat",
			DriverName:       "Siti",
			VehiclePlate:     "D 5678 EF",
			DeliveryOrderIDs: []uuid.UUID{},
		}
		invBody, _ := json.Marshal(invalidPayload)
		invReq := httptest.NewRequest(http.MethodPost, "/api/v1/wms/manifests", bytes.NewReader(invBody))
		invReq = withWMSAuth(invReq, tenantID, userID, role)
		invW := httptest.NewRecorder()
		router.ServeHTTP(invW, invReq)

		assert.Equal(t, http.StatusBadRequest, invW.Code)

		// 3. Malformed JSON
		badReq := httptest.NewRequest(http.MethodPost, "/api/v1/wms/manifests", bytes.NewReader([]byte("{invalid-json")))
		badReq = withWMSAuth(badReq, tenantID, userID, role)
		badW := httptest.NewRecorder()
		router.ServeHTTP(badW, badReq)

		assert.Equal(t, http.StatusBadRequest, badW.Code)
	})

	t.Run("GET /api/v1/wms/manifests/{id} success, invalid UUID, not found", func(t *testing.T) {
		manifestID := uuid.New()
		now := time.Now().UTC()

		mock := &mockWMSUsecase{
			getShippingManifestFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.ShippingManifestDetail, error) {
				if id == manifestID {
					return &domain.ShippingManifestDetail{
						Manifest: domain.ShippingManifest{
							ID:             id,
							TenantID:       tid,
							ManifestNumber: "SM-2026-003",
							Status:         domain.ShippingManifestStatusStaged,
							CreatedAt:      now,
						},
						Items: []domain.ShippingManifestItem{
							{
								DONumber:        "DO-2026-001",
								CustomerName:    "PT Maju Jaya",
								DestinationCity: "Jakarta",
								Scanned:         false,
							},
						},
					}, nil
				}
				return nil, domain.ErrManifestNotFound
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. Success
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wms/manifests/%s", manifestID), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data domain.ShippingManifestDetail `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, manifestID, resp.Data.Manifest.ID)
		require.Len(t, resp.Data.Items, 1)

		// 2. Invalid UUID
		invReq := httptest.NewRequest(http.MethodGet, "/api/v1/wms/manifests/not-a-uuid", nil)
		invReq = withWMSAuth(invReq, tenantID, userID, role)
		invW := httptest.NewRecorder()
		router.ServeHTTP(invW, invReq)
		assert.Equal(t, http.StatusBadRequest, invW.Code)

		// 3. Not found
		missingReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wms/manifests/%s", uuid.New()), nil)
		missingReq = withWMSAuth(missingReq, tenantID, userID, role)
		missingW := httptest.NewRecorder()
		router.ServeHTTP(missingW, missingReq)
		assert.Equal(t, http.StatusNotFound, missingW.Code)
	})

	t.Run("POST /api/v1/wms/manifests/{id}/loading-scan success & misload error", func(t *testing.T) {
		manifestID := uuid.New()
		now := time.Now().UTC()

		mock := &mockWMSUsecase{
			scanDOLoadingFn: func(ctx context.Context, tid, uid uuid.UUID, r string, mID uuid.UUID, req domain.LoadingScanRequest) (*domain.ShippingManifestDetail, error) {
				if req.Barcode == "DO-WRONG" {
					return nil, domain.ErrDOMisload
				}
				return &domain.ShippingManifestDetail{
					Manifest: domain.ShippingManifest{
						ID:             mID,
						TenantID:       tid,
						ManifestNumber: "SM-2026-004",
						Status:         domain.ShippingManifestStatusLoaded,
						CreatedAt:      now,
					},
					Items: []domain.ShippingManifestItem{
						{
							DONumber: req.Barcode,
							Scanned:  true,
						},
					},
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. Success
		body, _ := json.Marshal(domain.LoadingScanRequest{Barcode: "DO-2026-001"})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/wms/manifests/%s/loading-scan", manifestID), bytes.NewReader(body))
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data domain.ShippingManifestDetail `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, domain.ShippingManifestStatusLoaded, resp.Data.Manifest.Status)
		assert.True(t, resp.Data.Items[0].Scanned)

		// 2. Misload error -> 422 Unprocessable Entity
		misloadBody, _ := json.Marshal(domain.LoadingScanRequest{Barcode: "DO-WRONG"})
		misReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/wms/manifests/%s/loading-scan", manifestID), bytes.NewReader(misloadBody))
		misReq = withWMSAuth(misReq, tenantID, userID, role)
		misW := httptest.NewRecorder()
		router.ServeHTTP(misW, misReq)

		assert.Equal(t, http.StatusUnprocessableEntity, misW.Code)
	})

	t.Run("POST /api/v1/wms/manifests/{id}/dispatch success & signature error", func(t *testing.T) {
		manifestID := uuid.New()
		now := time.Now().UTC()

		mock := &mockWMSUsecase{
			dispatchShippingManifestFn: func(ctx context.Context, tid, uid uuid.UUID, r string, mID uuid.UUID, req domain.DispatchShippingManifestRequest) (*domain.ShippingManifest, error) {
				if len(req.DriverSignatureSVG) <= 10 {
					return nil, domain.ErrManifestSignatureRequired
				}
				return &domain.ShippingManifest{
					ID:                 mID,
					TenantID:           tid,
					ManifestNumber:     "SM-2026-005",
					Status:             domain.ShippingManifestStatusDispatched,
					DriverSignatureSVG: &req.DriverSignatureSVG,
					DispatchedBy:       &uid,
					DispatchedAt:       &now,
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. Success dispatch with valid SVG signature
		validSig := "<svg viewBox='0 0 100 100'><line x1='0' y1='0' x2='100' y2='100'/></svg>"
		body, _ := json.Marshal(domain.DispatchShippingManifestRequest{DriverSignatureSVG: validSig})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/wms/manifests/%s/dispatch", manifestID), bytes.NewReader(body))
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data domain.ShippingManifest `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, domain.ShippingManifestStatusDispatched, resp.Data.Status)
		assert.Equal(t, validSig, *resp.Data.DriverSignatureSVG)

		// 2. Missing/short signature -> 400 Bad Request
		invBody, _ := json.Marshal(domain.DispatchShippingManifestRequest{DriverSignatureSVG: ""})
		invReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/wms/manifests/%s/dispatch", manifestID), bytes.NewReader(invBody))
		invReq = withWMSAuth(invReq, tenantID, userID, role)
		invW := httptest.NewRecorder()
		router.ServeHTTP(invW, invReq)

		assert.Equal(t, http.StatusBadRequest, invW.Code)
	})

	t.Run("GET /api/v1/wms/kpi returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			getWMSOutboundKPIsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID) (*domain.WMSOutboundKPISummary, error) {
				return &domain.WMSOutboundKPISummary{
					DockToStockAvgMinutes:   domain.KPIAvg(85, 2),
					ReceivingAccuracyPct:    domain.KPIRatioPct(992, 1000),
					POCompliancePct:         domain.KPIRatioPct(978, 1000),
					BacklogInboundCount:     3,
					OrderToDispatchAvgHours: domain.KPIAvg(5.6, 2),
					PickingAccuracyPct:      domain.KPIRatioPct(999, 1000),
					OnTimeShipmentPct:       domain.KPIRatioPct(981, 1000),
					BacklogOutboundCount:    4,
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/kpi", nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data domain.WMSOutboundKPISummary `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Data.DockToStockAvgMinutes)
		assert.Equal(t, 42.5, *resp.Data.DockToStockAvgMinutes)
		assert.Equal(t, 99.9, *resp.Data.PickingAccuracyPct)
		assert.Equal(t, 98.1, *resp.Data.OnTimeShipmentPct)
	})

	// Contract: a tenant with no transactions must get JSON null for every
	// rate/average KPI (rendered "belum ada data"), never 100 or 0.
	t.Run("GET /api/v1/wms/kpi with no data returns null rates", func(t *testing.T) {
		mock := &mockWMSUsecase{
			getWMSOutboundKPIsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID) (*domain.WMSOutboundKPISummary, error) {
				return &domain.WMSOutboundKPISummary{
					WindowDays:              domain.KPIWindowDays,
					DockToStockAvgMinutes:   domain.KPIAvg(0, 0),
					ReceivingAccuracyPct:    domain.KPIRatioPct(0, 0),
					POCompliancePct:         domain.KPIRatioPct(0, 0),
					OrderToDispatchAvgHours: domain.KPIAvg(0, 0),
					PickingAccuracyPct:      domain.KPIRatioPct(0, 0),
					OnTimeShipmentPct:       domain.KPIRatioPct(0, 0),
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/kpi", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var raw struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&raw))
		for _, k := range []string{"dock_to_stock_avg_minutes", "receiving_accuracy_pct", "po_compliance_pct", "order_to_dispatch_avg_hours", "picking_accuracy_pct", "on_time_shipment_pct"} {
			v, present := raw.Data[k]
			assert.True(t, present, "%s must be present", k)
			assert.Nil(t, v, "%s must be null when there is no data", k)
		}
		assert.EqualValues(t, 0, raw.Data["backlog_inbound_count"])
		assert.EqualValues(t, 30, raw.Data["window_days"])
	})

	t.Run("Alias endpoints /delivery-orders/manifests work identically", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listShippingManifestsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID, status *domain.ShippingManifestStatus, expName *string) ([]domain.ShippingManifest, error) {
				return []domain.ShippingManifest{{ManifestNumber: "SM-ALIAS-001"}}, nil
			},
		}

		router := setupWMSTestRouter(mock)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/delivery-orders/manifests", nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []domain.ShippingManifest `json:"data"`
		}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, "SM-ALIAS-001", resp.Data[0].ManifestNumber)
	})
}
