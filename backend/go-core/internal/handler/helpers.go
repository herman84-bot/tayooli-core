package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ErrorResponse is the standard error envelope returned by all API endpoints.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody holds the machine-readable code, human-readable message,
// and optional structured details (validation errors, field hints, etc.).
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// RespondError writes a standardised JSON error response.
// Usage:
//
//	handler.RespondError(w, r, http.StatusBadRequest, "invalid input", validationErrors)
func RespondError(w http.ResponseWriter, r *http.Request, status int, message string, details ...any) {
	resp := ErrorResponse{
		Error: ErrorBody{
			Code:    http.StatusText(status),
			Message: message,
		},
	}
	if len(details) > 0 {
		resp.Error.Details = details[0]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// parseUUIDParam extracts and validates a UUID URL parameter.
// Returns (uuid, false) if param is missing or not a valid UUID.
func parseUUIDParam(r *http.Request, param string) (uuid.UUID, bool) {
	s := chi.URLParam(r, param)
	id, err := uuid.Parse(s)
	return id, err == nil
}

var maxDecimalAmount, _ = decimal.NewFromString("9999999999999999.9999")

// parseDecimalAmount validates an amount string for use in financial fields.
// Rules: must be a valid decimal, positive, <= NUMERIC(20,4) max, max 4 decimal places.
func parseDecimalAmount(s string) (decimal.Decimal, error) {
	amt, err := decimal.NewFromString(s)
	if err != nil || amt.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("amount must be a valid positive decimal string (e.g. \"1234.56\")")
	}
	if amt.GreaterThan(maxDecimalAmount) {
		return decimal.Zero, fmt.Errorf("amount exceeds maximum allowed value")
	}
	if !amt.Equal(amt.Truncate(4)) {
		return decimal.Zero, fmt.Errorf("amount may not have more than 4 decimal places")
	}
	return amt, nil
}
