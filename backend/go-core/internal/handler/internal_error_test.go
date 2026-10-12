package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Regression: unexpected 500s must never echo the internal cause (SQL,
// driver, upstream text) to the client.
func TestRespondInternalError_DoesNotLeakCause(t *testing.T) {
	secret := `pq: duplicate key value violates unique constraint "users_pkey" host=10.0.0.5`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", nil)
	rr := httptest.NewRecorder()

	respondInternalError(rr, req, "test op", errors.New(secret))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "pq:") || strings.Contains(body, "users_pkey") || strings.Contains(body, "10.0.0.5") {
		t.Fatalf("response leaks internal error: %s", body)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var env ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
	}
	if env.Error.Message != internalErrorMessage {
		t.Fatalf("message = %q, want %q", env.Error.Message, internalErrorMessage)
	}
}
