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
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ============================================================================
// RED TEAM ADVERSARIAL SECURITY AUDIT TEST SUITE: COPILOT SUBSYSTEM
// ============================================================================

// 1. TENANT ISOLATION & IDENTITY SPOOFING
func TestSecurity_TenantIdentitySpoofingRejected(t *testing.T) {
	h := handler.NewCopilotChatHandler("", "")
	realTenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	spoofedTenantID := "99999999-9999-9999-9999-999999999999"

	handlerWithTenant := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, middleware.TenantIDKey, realTenantID)
		ctx = context.WithValue(ctx, middleware.RoleKey, "member")
		h.Chat(w, r.WithContext(ctx))
	})
	srv := httptest.NewServer(handlerWithTenant)
	t.Cleanup(srv.Close)

	// Attacker attempts to spoof tenantId and userRole in the request body
	reqPayload := map[string]any{
		"message": "buka invoice",
		"context": map[string]any{
			"tenantId": spoofedTenantID,
			"userRole": "superadmin",
		},
	}
	body, _ := json.Marshal(reqPayload)

	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify that unauthenticated requests without JWT context are 401
	unauthSrv := httptest.NewServer(http.HandlerFunc(h.Chat))
	t.Cleanup(unauthSrv.Close)

	unauthResp, err := http.Post(unauthSrv.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST unauth request failed: %v", err)
	}
	defer unauthResp.Body.Close()

	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Security Violation: Request without JWT context returned %d, expected 401 Unauthorized", unauthResp.StatusCode)
	}
}

// 2. AUTOPILOT MODE SAFETY & SILENT MUTATION:
// Verify mutating actions CANNOT have riskLevel 'low'
func TestSecurity_MutatingActionsNeverLowRisk(t *testing.T) {
	mutatingPrompts := []struct {
		prompt   string
		expected string
	}{
		{"tambah vendor PT Aman Sejahtera email info@aman.com", "create_vendor"},
		{"tambah customer PT Mitra Sukses telepon 0812345678", "create_customer"},
		{"tambah produk baru Laptop Lenovo kode LNV-01 harga 8000000", "create_product"},
		{"buatkan invoice untuk vendor PT Sumber Makmur sebesar 10 juta", "create_purchase_invoice"},
		{"ubah nama perusahaan jadi PT Nusantara Baru", "update_company_profile"},
		{"undang anggota tim budi@perusahaan.com sebagai admin", "invite_team_member"},
		{"setup payment gateway midtrans", "configure_payment_gateway"},
	}

	for _, tc := range mutatingPrompts {
		res := driveMatcher(t, tc.prompt)
		if len(res.Actions) == 0 {
			t.Fatalf("expected action for %q, got 0", tc.prompt)
		}
		for _, action := range res.Actions {
			if action.ToolName == tc.expected {
				if action.RiskLevel == "low" {
					t.Errorf("CRITICAL SECURITY FLAW: Mutating action %q has riskLevel 'low'! Autopilot could silently execute it!", action.ToolName)
				}
				if action.RiskLevel != "medium" && action.RiskLevel != "high" {
					t.Errorf("Action %q has unexpected riskLevel: %q", action.ToolName, action.RiskLevel)
				}
			}
		}
	}
}

// Verify that ONLY read-only actions (navigation, dashboard summary) can have riskLevel 'low'
func TestSecurity_OnlySafeActionsAreLowRisk(t *testing.T) {
	safePrompts := []struct {
		prompt   string
		expected string
	}{
		{"buka dashboard", "navigate_to_module"},
		{"buka vendor", "navigate_to_module"},
		{"buka invoice", "navigate_to_module"},
		{"ringkasan keuangan", "get_dashboard_summary"},
	}

	for _, tc := range safePrompts {
		res := driveMatcher(t, tc.prompt)
		if len(res.Actions) == 0 {
			t.Fatalf("expected action for %q, got 0", tc.prompt)
		}
		for _, action := range res.Actions {
			if action.ToolName == tc.expected && action.RiskLevel != "low" {
				t.Errorf("Safe action %q has riskLevel %q, expected 'low'", action.ToolName, action.RiskLevel)
			}
		}
	}
}

// 3. ADVISORY MODE ENFORCEMENT
func TestSecurity_AdvisoryModeStrictEnforcement(t *testing.T) {
	h := handler.NewCopilotChatHandler("", "")
	tenantID := "00000000-0000-0000-0000-000000000001"
	handlerWithTenant := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, middleware.TenantIDKey, uuid.MustParse(tenantID))
		ctx = context.WithValue(ctx, middleware.RoleKey, "admin")
		h.Chat(w, r.WithContext(ctx))
	})
	srv := httptest.NewServer(handlerWithTenant)
	t.Cleanup(srv.Close)

	testCases := []struct {
		prompt       string
		shouldAction bool
		expectedTool string
	}{
		{"tambah vendor PT Berkah", false, "create_vendor"},
		{"tambah customer Budi Santoso", false, "create_customer"},
		{"tambah produk HP Samsung kode SAM-01", false, "create_product"},
		{"buatkan invoice untuk vendor PT Maju sebesar 10 juta", false, "create_purchase_invoice"},
		{"ubah nama perusahaan jadi PT Baru", false, "update_company_profile"},
		{"undang anggota tim user@test.com sebagai admin", false, "invite_team_member"},
		{"setup payment gateway pakasir", false, "configure_payment_gateway"},
		{"buka halaman vendor", true, "navigate_to_module"},
		{"buka chart of accounts", true, "navigate_to_module"},
	}

	for _, tc := range testCases {
		reqBody := map[string]any{
			"message": tc.prompt,
			"context": map[string]any{
				"autonomyLevel": "advisory",
			},
		}
		body, _ := json.Marshal(reqBody)
		resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}

		var res handler.CopilotResponse
		_ = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()

		if !tc.shouldAction {
			if len(res.Actions) > 0 {
				t.Errorf("CRITICAL SECURITY FLAW: Advisory mode allowed mutating action %v for prompt %q", res.Actions, tc.prompt)
			}
			if !strings.Contains(res.Reply, "Advisory") {
				t.Errorf("Advisory mode expected warning message in reply for %q, got: %s", tc.prompt, res.Reply)
			}
		} else {
			if len(res.Actions) != 1 || res.Actions[0].ToolName != tc.expectedTool {
				t.Errorf("Advisory mode blocked permitted read-only action %q, got: %v", tc.expectedTool, res.Actions)
			}
		}
	}
}

// 4. REGEX DENIAL OF SERVICE (ReDoS) / TIMING ATTACKS / SPECIAL CHARACTERS
func TestSecurity_PlusInPhoneNumberDoesNotCrashGoBackend(t *testing.T) {
	// Adversarial input with international phone number starting with +62
	prompt := "tambah vendor baru PT ABC telepon +628123456789"
	res := driveMatcher(t, prompt)
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	if res.Actions[0].ToolName != "create_vendor" {
		t.Fatalf("expected create_vendor, got %q", res.Actions[0].ToolName)
	}
	phone, _ := res.Actions[0].Payload["phone"].(string)
	if phone != "+628123456789" {
		t.Fatalf("expected phone '+628123456789', got %q", phone)
	}
}

func TestSecurity_ReDoSResistance(t *testing.T) {
	// Adversarial patterns designed to stress regexes
	longRepeatingA := strings.Repeat("a", 50000)
	longRepeatingSpecial := strings.Repeat("@-._+*", 10000)
	nestedPatterns := strings.Repeat("vendor vendor vendor ", 2000)
	longNumeric := "Rp " + strings.Repeat("9.", 5000) + "000"

	payloads := []string{
		longRepeatingA,
		longRepeatingSpecial,
		nestedPatterns,
		longNumeric,
		"tambah vendor " + longRepeatingA + "@" + longRepeatingA + ".com",
		"tambah vendor " + strings.Repeat("0812", 2000),
		"buatkan invoice untuk vendor " + strings.Repeat("sebesar ", 2000) + " 5000000",
	}

	for i, p := range payloads {
		start := time.Now()
		_ = driveMatcher(t, p)
		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			t.Errorf("POTENTIAL ReDoS / PERFORMANCE DEGRADATION: payload %d took %v (> 100ms)", i, elapsed)
		}
	}
}
