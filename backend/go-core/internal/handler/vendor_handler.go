package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// escapeILIKEWildcards escapes % and _ in user input so they are treated as
// literal characters in ILIKE patterns (which treats them as wildcards).
func escapeILIKEWildcards(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// vendorUsecase groups the operations the handler needs.
type vendorUsecase interface {
	CreateVendor(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error)
	ListVendors(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error)
	GetVendor(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error)
	UpdateVendor(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error)
	DeleteVendor(ctx context.Context, id, tenantID uuid.UUID, deletedBy *uuid.UUID) error
	RateVendor(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error)
}

// VendorHandler handles HTTP requests for vendor resources.
type VendorHandler struct {
	uc vendorUsecase
}

func NewVendorHandler(uc vendorUsecase) *VendorHandler {
	return &VendorHandler{uc: uc}
}

// vendorView is the JSON representation returned to clients.
type vendorView struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	Name        string  `json:"name"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Address     *string `json:"address,omitempty"`
	BankAccount *string `json:"bank_account,omitempty"`
	BankName    *string `json:"bank_name,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
	Status      string  `json:"status"`
	AvgRating   string  `json:"avg_rating"`
	RatingCount int     `json:"rating_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toVendorView(v *domain.Vendor) vendorView {
	return vendorView{
		ID:          v.ID.String(),
		TenantID:    v.TenantID.String(),
		Name:        v.Name,
		Email:       v.Email,
		Phone:       v.Phone,
		Address:     v.Address,
		BankAccount: v.BankAccount,
		BankName:    v.BankName,
		TaxID:       v.TaxID,
		Status:      string(v.Status),
		AvgRating:   v.AvgRating.StringFixed(2),
		RatingCount: v.RatingCount,
		CreatedAt:   v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   v.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// vendorRatingView is the JSON representation for a rating.
type vendorRatingView struct {
	ID        string  `json:"id"`
	TenantID  string  `json:"tenant_id"`
	VendorID  string  `json:"vendor_id"`
	RatedBy   *string `json:"rated_by,omitempty"`
	Rating    int     `json:"rating"`
	Comment   *string `json:"comment,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func toVendorRatingView(r *domain.VendorRating) vendorRatingView {
	v := vendorRatingView{
		ID:        r.ID.String(),
		TenantID:  r.TenantID.String(),
		VendorID:  r.VendorID.String(),
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.RatedBy != nil {
		s := r.RatedBy.String()
		v.RatedBy = &s
	}
	return v
}

// ── ListVendors ──────────────────────────────────────────────────────────────

// ListVendors handles GET /api/v1/vendors?q=&page=&per_page=
func (h *VendorHandler) ListVendors(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	search := escapeILIKEWildcards(r.URL.Query().Get("q"))
	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 100 {
		perPage = 100
	}

	result, err := h.uc.ListVendors(r.Context(), tenantID, search, page, perPage)
	if err != nil {
		log.Error().Err(err).Str("handler", "ListVendors").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []vendorView `json:"data"`
		Total   int          `json:"total"`
		Page    int          `json:"page"`
		PerPage int          `json:"per_page"`
	}

	views := make([]vendorView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toVendorView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

// ── GetVendor ────────────────────────────────────────────────────────────────

// GetVendor handles GET /api/v1/vendors/{id}
func (h *VendorHandler) GetVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid vendor id")
		return
	}
	vendor, err := h.uc.GetVendor(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "vendor not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetVendor").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, toVendorView(vendor))
}

// ── CreateVendor ─────────────────────────────────────────────────────────────

// createVendorRequest is the JSON body for POST /api/v1/vendors.
type createVendorRequest struct {
	Name        string  `json:"name"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Address     *string `json:"address,omitempty"`
	BankAccount *string `json:"bank_account,omitempty"`
	BankName    *string `json:"bank_name,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
}

// CreateVendor handles POST /api/v1/vendors
func (h *VendorHandler) CreateVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64 KB limit
	var req createVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	params := domain.CreateVendorParams{
		TenantID:    tenantID,
		Name:        req.Name,
		Email:       req.Email,
		Phone:       req.Phone,
		Address:     req.Address,
		BankAccount: req.BankAccount,
		BankName:    req.BankName,
		TaxID:       req.TaxID,
	}

	vendor, err := h.uc.CreateVendor(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, err.Error())
			return
		}
		log.Error().Err(err).Str("handler", "CreateVendor").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toVendorView(vendor))
}

// ── UpdateVendor ─────────────────────────────────────────────────────────────

// updateVendorRequest is the JSON body for PUT /api/v1/vendors/{id}.
type updateVendorRequest struct {
	Name        string  `json:"name"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Address     *string `json:"address,omitempty"`
	BankAccount *string `json:"bank_account,omitempty"`
	BankName    *string `json:"bank_name,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
}

// UpdateVendor handles PUT /api/v1/vendors/{id}
func (h *VendorHandler) UpdateVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid vendor id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64 KB limit
	var req updateVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	params := domain.UpdateVendorParams{
		ID:          id,
		TenantID:    tenantID,
		Name:        req.Name,
		Email:       req.Email,
		Phone:       req.Phone,
		Address:     req.Address,
		BankAccount: req.BankAccount,
		BankName:    req.BankName,
		TaxID:       req.TaxID,
	}

	vendor, err := h.uc.UpdateVendor(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "vendor not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, err.Error())
			return
		}
		log.Error().Err(err).Str("handler", "UpdateVendor").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toVendorView(vendor))
}

// ── DeleteVendor ─────────────────────────────────────────────────────────────

// DeleteVendor handles DELETE /api/v1/vendors/{id}
func (h *VendorHandler) DeleteVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid vendor id")
		return
	}

	// Extract user_id from context for audit.
	var deletedBy *uuid.UUID
	if userID, uok := appMiddleware.GetUserID(r.Context()); uok {
		deletedBy = &userID
	}

	err := h.uc.DeleteVendor(r.Context(), id, tenantID, deletedBy)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "vendor not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "vendor is not active")
			return
		}
		log.Error().Err(err).Str("handler", "DeleteVendor").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "vendor deleted"})
}

// ── RateVendor ───────────────────────────────────────────────────────────────

// rateVendorRequest is the JSON body for POST /api/v1/vendors/{id}/rate.
type rateVendorRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment,omitempty"`
}

// RateVendor handles POST /api/v1/vendors/{id}/rate
func (h *VendorHandler) RateVendor(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	vendorID, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid vendor id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64 KB limit
	var req rateVendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	// Extract user_id from context for rated_by.
	var ratedBy *uuid.UUID
	if userID, uok := appMiddleware.GetUserID(r.Context()); uok {
		ratedBy = &userID
	}

	params := domain.AddVendorRatingParams{
		TenantID: tenantID,
		VendorID: vendorID,
		RatedBy:  ratedBy,
		Rating:   req.Rating,
		Comment:  req.Comment,
	}

	rating, err := h.uc.RateVendor(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, err.Error())
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "vendor not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "you have already rated this vendor")
			return
		}
		log.Error().Err(err).Str("handler", "RateVendor").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toVendorRatingView(rating))
}
