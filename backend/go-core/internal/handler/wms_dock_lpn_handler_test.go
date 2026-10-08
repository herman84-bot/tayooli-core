package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockLPNHandlers(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	role := "admin"
	whID := uuid.New()

	t.Run("Authentication guard rejects unauthenticated requests", func(t *testing.T) {
		mock := &mockWMSUsecase{}
		router := setupWMSTestRouter(mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/docks?warehouse_id="+whID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/docks", strings.NewReader("{}"))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Docks endpoints: List, Create, Get, UpdateStatus", func(t *testing.T) {
		dockID := uuid.New()

		mock := &mockWMSUsecase{
			listDocksFn: func(ctx context.Context, tid, uid uuid.UUID, r string, warehouseID uuid.UUID, status *domain.DockStatus) ([]domain.InboundDock, error) {
				return []domain.InboundDock{
					{
						ID:          dockID,
						TenantID:    tid,
						WarehouseID: warehouseID,
						DockCode:    "DOCK-01",
						DockName:    "Dock Inbound 1",
						DockType:    domain.DockTypeInbound,
						MaxTonnage:  decimal.NewFromFloat(15.0),
						Status:      domain.DockStatusAvailable,
					},
				}, nil
			},
			createDockFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req domain.CreateDockRequest) (*domain.InboundDock, error) {
				return &domain.InboundDock{
					ID:          dockID,
					TenantID:    tid,
					WarehouseID: req.WarehouseID,
					DockCode:    "DOCK-02",
					DockName:    req.DockName,
					DockType:    req.DockType,
					MaxTonnage:  req.MaxTonnage,
					Status:      domain.DockStatusAvailable,
				}, nil
			},
			getDockFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.InboundDock, error) {
				if id == dockID {
					return &domain.InboundDock{
						ID:       dockID,
						TenantID: tid,
						DockName: "Dock Inbound 1",
						Status:   domain.DockStatusAvailable,
					}, nil
				}
				return nil, domain.ErrDockNotFound
			},
			updateDockStatusFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, req domain.UpdateDockStatusRequest) (*domain.InboundDock, error) {
				if id != dockID {
					return nil, domain.ErrDockNotFound
				}
				return &domain.InboundDock{
					ID:       dockID,
					TenantID: tid,
					Status:   req.Status,
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. ListDocks: missing warehouse_id -> 400
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/docks", nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 2. ListDocks: valid warehouse_id -> 200
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/docks?warehouse_id="+whID.String()+"&status=AVAILABLE", nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var listResp struct {
			Data []domain.InboundDock `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&listResp))
		require.Len(t, listResp.Data, 1)
		assert.Equal(t, "DOCK-01", listResp.Data[0].DockCode)

		// 3. CreateDock: payload too large -> 413
		hugeBody := `{"dock_name":"` + strings.Repeat("A", 1024*1024+10) + `"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/docks", strings.NewReader(hugeBody))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

		// 4. CreateDock: invalid JSON -> 400
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/docks", strings.NewReader("{invalid"))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 5. CreateDock: success -> 201
		createPayload := domain.CreateDockRequest{
			WarehouseID: whID,
			DockName:    "Dock 2",
			DockType:    domain.DockTypeInbound,
			MaxTonnage:  decimal.NewFromFloat(20),
		}
		bodyBytes, _ := json.Marshal(createPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/docks", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 6. GetDock: 404 ErrDockNotFound
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/docks/"+uuid.New().String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// 7. GetDock: 200 OK
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/docks/"+dockID.String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 8. UpdateDockStatus: 200 OK
		statusPayload := domain.UpdateDockStatusRequest{
			Status: domain.DockStatusOccupied,
		}
		bodyBytes, _ = json.Marshal(statusPayload)
		req = httptest.NewRequest(http.MethodPatch, "/api/v1/wms/docks/"+dockID.String()+"/status", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Appointments endpoints: List, Create, Get, AssignDock, UpdateStatus", func(t *testing.T) {
		appID := uuid.New()
		dockID := uuid.New()

		mock := &mockWMSUsecase{
			listAppointmentsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, warehouseID uuid.UUID, status *domain.AppointmentStatus) ([]domain.DockAppointment, error) {
				return []domain.DockAppointment{
					{
						ID:                appID,
						TenantID:          tid,
						WarehouseID:       warehouseID,
						AppointmentNumber: "APP-001",
						VendorName:        "Vendor A",
						Status:            domain.AppointmentStatusScheduled,
					},
				}, nil
			},
			createAppointmentFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req domain.CreateAppointmentRequest) (*domain.DockAppointment, error) {
				return &domain.DockAppointment{
					ID:                appID,
					TenantID:          tid,
					WarehouseID:       req.WarehouseID,
					AppointmentNumber: "APP-002",
					VendorName:        req.VendorName,
					Status:            domain.AppointmentStatusScheduled,
				}, nil
			},
			getAppointmentFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.DockAppointment, error) {
				if id == appID {
					return &domain.DockAppointment{
						ID:       appID,
						TenantID: tid,
						Status:   domain.AppointmentStatusScheduled,
					}, nil
				}
				return nil, domain.ErrAppointmentNotFound
			},
			assignDockToAppointmentFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id, dID uuid.UUID) (*domain.DockAppointment, error) {
				if id != appID {
					return nil, domain.ErrAppointmentNotFound
				}
				if dID == uuid.Nil {
					return nil, domain.ErrDockNotFound
				}
				// Mock dock occupied scenario
				if dID != dockID {
					return nil, domain.ErrDockOccupied
				}
				return &domain.DockAppointment{
					ID:     appID,
					DockID: &dID,
					Status: domain.AppointmentStatusScheduled,
				}, nil
			},
			updateAppointmentStatusFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, req domain.UpdateAppointmentStatusRequest) (*domain.DockAppointment, error) {
				return &domain.DockAppointment{
					ID:     id,
					Status: req.Status,
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. ListAppointments: 200 OK
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/dock-appointments?warehouse_id="+whID.String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 2. CreateAppointment: 201 Created
		createPayload := domain.CreateAppointmentRequest{
			WarehouseID:      whID,
			VendorName:       "Vendor B",
			VehiclePlate:     "B 1234 CD",
			DriverName:       "Driver 1",
			EstimatedArrival: time.Now().Add(2 * time.Hour),
		}
		bodyBytes, _ := json.Marshal(createPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/dock-appointments", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 3. GetAppointment: 404 StatusNotFound
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/dock-appointments/"+uuid.New().String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// 4. AssignDock: 409 StatusConflict (ErrDockOccupied)
		conflictDockID := uuid.New()
		assignPayload := domain.AssignDockRequest{DockID: conflictDockID}
		bodyBytes, _ = json.Marshal(assignPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/dock-appointments/"+appID.String()+"/assign", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)

		// 5. AssignDock: 200 OK
		assignPayload = domain.AssignDockRequest{DockID: dockID}
		bodyBytes, _ = json.Marshal(assignPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/dock-appointments/"+appID.String()+"/assign", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 6. UpdateAppointmentStatus: 200 OK
		statusPayload := domain.UpdateAppointmentStatusRequest{
			Status: domain.AppointmentStatusArrived,
		}
		bodyBytes, _ = json.Marshal(statusPayload)
		req = httptest.NewRequest(http.MethodPatch, "/api/v1/wms/dock-appointments/"+appID.String()+"/status", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("LPNs endpoints: List, Create, Get, AddItem, Move", func(t *testing.T) {
		lpnID := uuid.New()
		locID := uuid.New()

		mock := &mockWMSUsecase{
			listLPNsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, warehouseID uuid.UUID, status *domain.LPNStatus) ([]domain.StockLPN, error) {
				return []domain.StockLPN{
					{
						ID:          lpnID,
						TenantID:    tid,
						WarehouseID: warehouseID,
						LPNCode:     "LPN-TEST-001",
						Status:      domain.LPNStatusStaged,
					},
				}, nil
			},
			createLPNFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req domain.CreateLPNRequest) (*domain.StockLPN, error) {
				return &domain.StockLPN{
					ID:          lpnID,
					TenantID:    tid,
					WarehouseID: req.WarehouseID,
					LocationID:  req.LocationID,
					LPNCode:     "LPN-TEST-002",
					Status:      domain.LPNStatusStaged,
				}, nil
			},
			getLPNFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockLPNDetail, error) {
				if id == lpnID {
					return &domain.StockLPNDetail{
						LPN: domain.StockLPN{
							ID:       lpnID,
							TenantID: tid,
							LPNCode:  "LPN-TEST-001",
							Status:   domain.LPNStatusStaged,
						},
						Items: []domain.StockLPNItem{},
					}, nil
				}
				return nil, domain.ErrLPNNotFound
			},
			addLPNItemFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, req domain.AddLPNItemRequest) (*domain.StockLPNDetail, error) {
				return &domain.StockLPNDetail{
					LPN: domain.StockLPN{ID: id},
					Items: []domain.StockLPNItem{
						{
							ProductID: req.ProductID,
							Quantity:  req.Quantity,
						},
					},
				}, nil
			},
			moveLPNFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, req domain.MoveLPNRequest) (*domain.StockLPNDetail, error) {
				if id != lpnID {
					return nil, domain.ErrLPNNotFound
				}
				// Mock error mappings
				if req.Notes != nil && *req.Notes == "trigger_empty" {
					return nil, domain.ErrLPNEmpty
				}
				if req.Notes != nil && *req.Notes == "trigger_invalid_loc" {
					return nil, domain.ErrInvalidLocationType
				}
				return &domain.StockLPNDetail{
					LPN: domain.StockLPN{
						ID:         lpnID,
						LocationID: req.TargetLocationID,
						Status:     domain.LPNStatusStored,
					},
				}, nil
			},
		}

		router := setupWMSTestRouter(mock)

		// 1. ListLPNs: 200 OK
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/lpns?warehouse_id="+whID.String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 2. CreateLPN: 201 Created
		createPayload := domain.CreateLPNRequest{
			WarehouseID: whID,
			LocationID:  locID,
			PalletType:  domain.PalletTypeWooden,
		}
		bodyBytes, _ := json.Marshal(createPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/lpns", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 3. GetLPN: 404 StatusNotFound (ErrLPNNotFound)
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/lpns/"+uuid.New().String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// 4. GetLPN: 200 OK
		req = httptest.NewRequest(http.MethodGet, "/api/v1/wms/lpns/"+lpnID.String(), nil)
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. AddLPNItem: 200 OK
		addPayload := domain.AddLPNItemRequest{
			ProductID: uuid.New(),
			BatchID:   uuid.New(),
			Quantity:  decimal.NewFromFloat(50),
		}
		bodyBytes, _ = json.Marshal(addPayload)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/lpns/"+lpnID.String()+"/items", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 6. MoveLPN: 422 StatusUnprocessableEntity (ErrLPNEmpty)
		noteEmpty := "trigger_empty"
		movePayloadEmpty := domain.MoveLPNRequest{
			TargetLocationID: locID,
			Notes:            &noteEmpty,
		}
		bodyBytes, _ = json.Marshal(movePayloadEmpty)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/lpns/"+lpnID.String()+"/move", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		// 7. MoveLPN: 422 StatusUnprocessableEntity (ErrInvalidLocationType)
		noteInvalid := "trigger_invalid_loc"
		movePayloadInvalid := domain.MoveLPNRequest{
			TargetLocationID: locID,
			Notes:            &noteInvalid,
		}
		bodyBytes, _ = json.Marshal(movePayloadInvalid)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/lpns/"+lpnID.String()+"/move", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		// 8. MoveLPN: 200 OK
		movePayloadOK := domain.MoveLPNRequest{
			TargetLocationID: locID,
		}
		bodyBytes, _ = json.Marshal(movePayloadOK)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/wms/lpns/"+lpnID.String()+"/move", bytes.NewReader(bodyBytes))
		req = withWMSAuth(req, tenantID, userID, role)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
