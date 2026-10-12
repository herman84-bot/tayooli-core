package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Regression: creating a pick wave with no eligible delivery orders used to
// surface as a generic 500 "internal server error" because handleWMSError
// had no mapping for the wave sentinel errors.
func TestHandleWMSError_PickWaveSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"no orders for wave", domain.ErrNoOrdersForWave, http.StatusUnprocessableEntity},
		{"wrapped no orders", fmt.Errorf("repo: %w", domain.ErrNoOrdersForWave), http.StatusUnprocessableEntity},
		{"wave not found", domain.ErrPickWaveNotFound, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/wms/pick-waves", nil)
			handleWMSError(rec, req, tc.err)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
