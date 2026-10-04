package handler

import (
	"encoding/json"
	"net/http"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/shopspring/decimal"
)

type PaymentGatewayHandler struct {
	repo domain.PaymentGatewayRepository
}

func NewPaymentGatewayHandler(repo domain.PaymentGatewayRepository) *PaymentGatewayHandler {
	return &PaymentGatewayHandler{repo: repo}
}

type UpsertPaymentConfigRequest struct {
	Provider              string          `json:"provider"`
	Slug                  *string         `json:"slug"`
	APIKey                *string         `json:"api_key"`
	ServerKey             *string         `json:"server_key"`
	ClientKey             *string         `json:"client_key"`
	IsProduction          bool            `json:"is_production"`
	IsActive              bool            `json:"is_active"`
	SettlementBankName    *string         `json:"settlement_bank_name"`
	SettlementBankAccount *string         `json:"settlement_bank_account"`
	SettlementHolderName  *string         `json:"settlement_holder_name"`
	GatewayFeePercent     decimal.Decimal `json:"gateway_fee_percent"`
}

func (h *PaymentGatewayHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "pakasir"
	}

	cfg, err := h.repo.GetConfig(r.Context(), tenantID, provider)
	if err == domain.ErrNotFound {
		respondJSON(w, http.StatusOK, map[string]any{
			"provider":       provider,
			"hasCredentials": false,
			"is_active":      false,
		})
		return
	}
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to get payment config")
		return
	}

	// Mask sensitive keys
	hasCreds := (cfg.APIKey != nil && *cfg.APIKey != "") || (cfg.ServerKey != nil && *cfg.ServerKey != "")
	respondJSON(w, http.StatusOK, map[string]any{
		"provider":                cfg.Provider,
		"slug":                    cfg.Slug,
		"client_key":              cfg.ClientKey,
		"hasCredentials":          hasCreds,
		"is_production":           cfg.IsProduction,
		"is_active":               cfg.IsActive,
		"settlement_bank_name":    cfg.SettlementBankName,
		"settlement_bank_account": cfg.SettlementBankAccount,
		"settlement_holder_name":  cfg.SettlementHolderName,
		"gateway_fee_percent":     cfg.GatewayFeePercent,
	})
}

func (h *PaymentGatewayHandler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpsertPaymentConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" {
		req.Provider = "pakasir"
	}

	cfg := &domain.TenantPaymentConfig{
		TenantID:              tenantID,
		Provider:              req.Provider,
		Slug:                  req.Slug,
		APIKey:                req.APIKey,
		ServerKey:             req.ServerKey,
		ClientKey:             req.ClientKey,
		IsProduction:          req.IsProduction,
		IsActive:              req.IsActive,
		SettlementBankName:    req.SettlementBankName,
		SettlementBankAccount: req.SettlementBankAccount,
		SettlementHolderName:  req.SettlementHolderName,
		GatewayFeePercent:     req.GatewayFeePercent,
	}

	if cfg.GatewayFeePercent.IsZero() {
		cfg.GatewayFeePercent = decimal.NewFromFloat(0.70)
	}

	if err := h.repo.UpsertConfig(r.Context(), cfg); err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to save payment config")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"message":  "payment config saved successfully",
		"provider": cfg.Provider,
	})
}
