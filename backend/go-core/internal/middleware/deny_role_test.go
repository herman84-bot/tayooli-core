package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDenyRole(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := DenyRole("cashier")(ok)
	cases := map[string]int{"cashier": 403, "admin": 200, "warehouse": 200, "owner": 200}
	for role, want := range cases {
		req := httptest.NewRequest(http.MethodGet, "/wms/warehouses", nil)
		req = req.WithContext(context.WithValue(req.Context(), RoleKey, role))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("role %s: want %d got %d", role, want, rec.Code)
		}
	}
}
