package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// driveMatcher exercises the rule-based fallback through the public Chat
// endpoint with an empty Gemini key, which forces the offline matcher path —
// exactly how the fallback runs in production when no key is configured.
func driveMatcher(t *testing.T, message string) handler.CopilotResponse {
	t.Helper()

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

	body, _ := json.Marshal(map[string]any{"message": message})
	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var res handler.CopilotResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return res
}

// ---------------------------------------------------------------------------
// Matcher parity tests (Go vs frontend fallback-matcher.ts)
// ---------------------------------------------------------------------------

func TestCopilotChat_Unauthorized_WithoutTenantContext(t *testing.T) {
	h := handler.NewCopilotChatHandler("", "")
	srv := httptest.NewServer(http.HandlerFunc(h.Chat))
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{"message": "halo"})
	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without tenant context, got %d", resp.StatusCode)
	}
}

func TestMatcherNavigatesToChartOfAccounts(t *testing.T) {
	res := driveMatcher(t, "buka chart of accounts")
	if res.ProviderUsed != "offline" {
		t.Fatalf("expected offline provider, got %q", res.ProviderUsed)
	}
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "navigate_to_module" {
		t.Fatalf("expected navigate_to_module, got %q", a.ToolName)
	}
	path, _ := a.Payload["path"].(string)
	if path != "/accounting/chart-of-accounts" {
		t.Fatalf("expected /accounting/chart-of-accounts, got %q", path)
	}
}

func TestMatcherNavigationKeywordVariants(t *testing.T) {
	cases := []struct {
		input, wantPath, wantMod string
	}{
		{"lihat coa", "/accounting/chart-of-accounts", "Chart of Accounts"},
		{"buka akun", "/accounting/chart-of-accounts", "Chart of Accounts"},
		{"buka jurnal", "/accounting/journal-entries", "Journal Entries"},
		{"lihat journal entries", "/accounting/journal-entries", "Journal Entries"},
		{"buka gr", "/dashboard/goods-receipts", "Goods Receipts"},
		{"lihat goods receipt", "/dashboard/goods-receipts", "Goods Receipts"},
		// NOTE: "buka penerimaan barang" intentionally matches create_product, not
		// navigation — "barang" is a product keyword and product creation is
		// evaluated before navigation, mirroring the frontend matcher ordering.
		{"buka penerimaan", "/dashboard/goods-receipts", "Goods Receipts"},
		{"lihat tagihan", "/dashboard/invoices", "Invoices"},
		{"buka purchase order", "/dashboard/purchase-orders", "Purchase Orders"},
		{"buka pengaturan", "/settings", "Pengaturan"},
	}
	for _, c := range cases {
		res := driveMatcher(t, c.input)
		if len(res.Actions) != 1 {
			t.Fatalf("%q: expected 1 action, got %d", c.input, len(res.Actions))
		}
		a := res.Actions[0]
		if a.ToolName != "navigate_to_module" {
			t.Fatalf("%q: expected navigate_to_module, got %q", c.input, a.ToolName)
		}
		path, _ := a.Payload["path"].(string)
		mod, _ := a.Payload["module_name"].(string)
		if path != c.wantPath || mod != c.wantMod {
			t.Fatalf("%q: got path=%q mod=%q, want path=%q mod=%q", c.input, path, mod, c.wantPath, c.wantMod)
		}
	}
}

func TestMatcherProductCreation(t *testing.T) {
	// Ungrounded requests must return 0 actions and ask clarifying question
	for _, input := range []string{"tambah produk baru", "daftarkan barang", "buat item katalog", "tambahkan sku baru"} {
		res := driveMatcher(t, input)
		if len(res.Actions) != 0 {
			t.Fatalf("%q: expected 0 actions for ungrounded request, got %d", input, len(res.Actions))
		}
		if !strings.Contains(strings.ToLower(res.Reply), "produk") && !strings.Contains(strings.ToLower(res.Reply), "sku") {
			t.Fatalf("%q: expected clarification question in reply, got %q", input, res.Reply)
		}
	}

	// Grounded request creates proposal
	grounded := "tambah produk baru bernama Meja Kantor SKU FUR-01 harga 750000"
	res := driveMatcher(t, grounded)
	if len(res.Actions) != 1 {
		t.Fatalf("%q: expected 1 action, got %d", grounded, len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_product" {
		t.Fatalf("%q: expected create_product, got %q", grounded, a.ToolName)
	}
	if a.RiskLevel != "medium" {
		t.Fatalf("%q: expected medium risk, got %q", grounded, a.RiskLevel)
	}
}

func TestMatcherCustomerCreation(t *testing.T) {
	// Ungrounded requests must return 0 actions and ask clarifying question
	for _, input := range []string{"tambah customer baru", "daftarkan pelanggan", "tambah klien"} {
		res := driveMatcher(t, input)
		if len(res.Actions) != 0 {
			t.Fatalf("%q: expected 0 actions for ungrounded request, got %d", input, len(res.Actions))
		}
		if !strings.Contains(strings.ToLower(res.Reply), "nama") {
			t.Fatalf("%q: expected clarification question in reply, got %q", input, res.Reply)
		}
	}

	// Grounded request creates proposal
	grounded := "tambah customer baru bernama PT Mitra Bahagia"
	res := driveMatcher(t, grounded)
	if len(res.Actions) != 1 {
		t.Fatalf("%q: expected 1 action, got %d", grounded, len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_customer" {
		t.Fatalf("%q: expected create_customer, got %q", grounded, a.ToolName)
	}
}

func TestMatcherSmallTalkParity(t *testing.T) {
	cases := []struct{ input, wantSnippet string }{
		{"apa kabar?", "Kabar baik"},
		{"kamu siapa sih?", "Tayooli Copilot"},
		{"terima kasih banyak", "Sama-sama"},
		{"makasih ya", "Sama-sama"},
	}
	for _, c := range cases {
		res := driveMatcher(t, c.input)
		if !strings.Contains(res.Reply, c.wantSnippet) {
			t.Fatalf("%q: reply %q does not contain %q", c.input, res.Reply, c.wantSnippet)
		}
		if len(res.Actions) != 0 {
			t.Fatalf("%q: expected no actions, got %d", c.input, len(res.Actions))
		}
	}
}

func TestMatcherProductNavigation(t *testing.T) {
	// "buka produk" navigates to /products
	res := driveMatcher(t, "buka produk")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	if res.Actions[0].ToolName != "navigate_to_module" {
		t.Fatalf("expected navigate_to_module, got %q", res.Actions[0].ToolName)
	}
	path, _ := res.Actions[0].Payload["path"].(string)
	if path != "/products" {
		t.Fatalf("expected /products, got %q", path)
	}
}

func TestMatcherInviteMemberExtraction(t *testing.T) {
	res := driveMatcher(t, "undang anggota tim email budi@test.com sebagai finance")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "invite_team_member" {
		t.Fatalf("expected invite_team_member, got %q", a.ToolName)
	}
	email, _ := a.Payload["email"].(string)
	role, _ := a.Payload["role"].(string)
	if email != "budi@test.com" {
		t.Fatalf("expected email budi@test.com, got %q", email)
	}
	if role != "accountant" {
		t.Fatalf("expected role accountant, got %q", role)
	}
}

func TestMatcherInviteMemberWithoutEmailFails(t *testing.T) {
	res := driveMatcher(t, "undang anggota tim sebagai admin")
	if len(res.Actions) != 0 {
		t.Fatalf("expected 0 actions for invite without email, got %d", len(res.Actions))
	}
	if !strings.Contains(res.Reply, "email yang valid") {
		t.Fatalf("expected prompt for valid email, got %q", res.Reply)
	}
}

func TestMatcherCreateVendorSubFields(t *testing.T) {
	res := driveMatcher(t, "vendor baru bernama PT Maju Lancar email contact@maju.id telepon 0812345678")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_vendor" {
		t.Fatalf("expected create_vendor, got %q", a.ToolName)
	}
	name, _ := a.Payload["name"].(string)
	email, _ := a.Payload["email"].(string)
	phone, _ := a.Payload["phone"].(string)
	if name != "PT Maju Lancar" {
		t.Fatalf("expected name 'PT Maju Lancar', got %q", name)
	}
	if email != "contact@maju.id" {
		t.Fatalf("expected email 'contact@maju.id', got %q", email)
	}
	if phone != "0812345678" {
		t.Fatalf("expected phone '0812345678', got %q", phone)
	}
}

func TestMatcherCreateProductSubFields(t *testing.T) {
	res := driveMatcher(t, "buat produk baru bernama Laptop Asus kode ASUS-01 harga 15.000.000")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_product" {
		t.Fatalf("expected create_product, got %q", a.ToolName)
	}
	sku, _ := a.Payload["sku"].(string)
	name, _ := a.Payload["name"].(string)
	price, _ := a.Payload["price"].(float64)
	if sku != "ASUS-01" {
		t.Fatalf("expected sku 'ASUS-01', got %q", sku)
	}
	if name != "Laptop Asus" {
		t.Fatalf("expected name 'Laptop Asus', got %q", name)
	}
	if price != 15000000 {
		t.Fatalf("expected price 15000000, got %v", price)
	}
}

func TestMatcherAdvisoryModeBlocksMutations(t *testing.T) {
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

	reqBody := map[string]any{
		"message": "tambah vendor baru bernama PT Advisory Test",
		"context": map[string]any{
			"autonomyLevel": "advisory",
		},
	}
	body, _ := json.Marshal(reqBody)
	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST chat: %v", err)
	}
	defer resp.Body.Close()

	var res handler.CopilotResponse
	json.NewDecoder(resp.Body).Decode(&res)

	if len(res.Actions) != 0 {
		t.Fatalf("expected 0 actions in advisory mode, got %d", len(res.Actions))
	}
	if !strings.Contains(res.Reply, "Advisory") {
		t.Fatalf("expected reply to mention Advisory mode, got %q", res.Reply)
	}
}

func TestMatcherCreatePurchaseInvoiceNotVendor(t *testing.T) {
	res := driveMatcher(t, "buatkan invoice untuk vendor PT Maju Jaya sebesar 5 juta rupiah")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_purchase_invoice" {
		t.Fatalf("expected create_purchase_invoice, got %q", a.ToolName)
	}
	if a.RiskLevel != "medium" {
		t.Fatalf("expected medium risk, got %q", a.RiskLevel)
	}
	vendorName, _ := a.Payload["vendor_name"].(string)
	if vendorName != "PT Maju Jaya" {
		t.Fatalf("expected vendor_name 'PT Maju Jaya', got %q", vendorName)
	}
	amount, _ := a.Payload["amount"].(float64)
	if amount != 5000000 {
		t.Fatalf("expected amount 5000000, got %v", amount)
	}
}

func TestMatcherBukaVendorNavigates(t *testing.T) {
	for _, phrase := range []string{"buka vendor", "buka halaman vendors", "lihat vendor"} {
		res := driveMatcher(t, phrase)
		if len(res.Actions) != 1 {
			t.Fatalf("%q: expected 1 action, got %d", phrase, len(res.Actions))
		}
		a := res.Actions[0]
		if a.ToolName != "navigate_to_module" {
			t.Fatalf("%q: expected navigate_to_module, got %q", phrase, a.ToolName)
		}
		path, _ := a.Payload["path"].(string)
		if path != "/dashboard/vendors" {
			t.Fatalf("%q: expected /dashboard/vendors, got %q", phrase, path)
		}
	}
}

func TestMatcherInterrogativeQueryDoesNotTriggerCreateVendor(t *testing.T) {
	res := driveMatcher(t, "vendor apa yang paling sering digunakan?")
	if len(res.Actions) != 0 {
		t.Fatalf("expected 0 actions for interrogative query, got %d", len(res.Actions))
	}
	if strings.TrimSpace(res.Reply) == "" {
		t.Fatalf("expected non-empty reply for interrogative query")
	}
	for _, a := range res.Actions {
		if a.ToolName == "create_vendor" {
			t.Fatalf("must NOT trigger create_vendor for interrogative query")
		}
	}
}

func TestMatcherCreateVendorCleanName(t *testing.T) {
	res := driveMatcher(t, "tambah vendor PT Berkah email vendor@berkah.com")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_vendor" {
		t.Fatalf("expected create_vendor, got %q", a.ToolName)
	}
	if a.RiskLevel != "medium" {
		t.Fatalf("expected medium risk, got %q", a.RiskLevel)
	}
	name, _ := a.Payload["name"].(string)
	if name != "PT Berkah" {
		t.Fatalf("expected name 'PT Berkah', got %q", name)
	}
	email, _ := a.Payload["email"].(string)
	if email != "vendor@berkah.com" {
		t.Fatalf("expected email 'vendor@berkah.com', got %q", email)
	}
}

func TestMatcherScenario4_StopWordsInVendorRegistration(t *testing.T) {
	res := driveMatcher(t, "tambah vendor baru PT Sinar Abadi sebesar 10jt telepon 08123456789")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	a := res.Actions[0]
	if a.ToolName != "create_vendor" {
		t.Fatalf("expected create_vendor, got %q", a.ToolName)
	}
	name, _ := a.Payload["name"].(string)
	phone, _ := a.Payload["phone"].(string)
	t.Logf("EXTRACTED NAME: %q, PHONE: %q", name, phone)
	if name != "PT Sinar Abadi" {
		t.Fatalf("expected name 'PT Sinar Abadi', got %q", name)
	}
	if phone != "08123456789" {
		t.Fatalf("expected phone '08123456789', got %q", phone)
	}
}

func TestMatcherDashboardSummary(t *testing.T) {
	for _, phrase := range []string{"ringkasan keuangan", "summary kas"} {
		res := driveMatcher(t, phrase)
		if len(res.Actions) != 1 {
			t.Fatalf("%q: expected 1 action, got %d", phrase, len(res.Actions))
		}
		a := res.Actions[0]
		if a.ToolName != "get_dashboard_summary" {
			t.Fatalf("%q: expected get_dashboard_summary, got %q", phrase, a.ToolName)
		}
		if a.RiskLevel != "low" {
			t.Fatalf("%q: expected low risk, got %q", phrase, a.RiskLevel)
		}
	}
}

func TestMatcher_GroundingOverhaul(t *testing.T) {
	// 1. Vendor with address extracted cleanly
	res := driveMatcher(t, "Tambah vendor baru dengan nama PT Sinar Terang Sejati, email halo@sinarterang.id, telepon 081222333444, alamat Jl Merdeka 1 Bandung")
	if len(res.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(res.Actions))
	}
	vAction := res.Actions[0]
	if vAction.ToolName != "create_vendor" {
		t.Fatalf("expected create_vendor, got %q", vAction.ToolName)
	}
	if vAction.Payload["name"] != "PT Sinar Terang Sejati" {
		t.Errorf("expected name 'PT Sinar Terang Sejati', got %v", vAction.Payload["name"])
	}
	if vAction.Payload["email"] != "halo@sinarterang.id" {
		t.Errorf("expected email 'halo@sinarterang.id', got %v", vAction.Payload["email"])
	}
	if vAction.Payload["phone"] != "081222333444" {
		t.Errorf("expected phone '081222333444', got %v", vAction.Payload["phone"])
	}
	addr, _ := vAction.Payload["address"].(string)
	if !strings.Contains(addr, "Jl Merdeka 1 Bandung") {
		t.Errorf("expected address to contain 'Jl Merdeka 1 Bandung', got %q", addr)
	}

	// 2. Ungrounded vendor requests return 0 actions and ask clarifying question
	for _, ungrounded := range []string{"tambah vendor baru", "buat vendor baru", "daftarkan vendor", "Saya mau tambah vendor baru untuk suplai kertas"} {
		resU := driveMatcher(t, ungrounded)
		if len(resU.Actions) != 0 {
			t.Fatalf("%q: expected 0 actions for ungrounded request, got %d", ungrounded, len(resU.Actions))
		}
		if !strings.Contains(strings.ToLower(resU.Reply), "nama") {
			t.Errorf("%q: expected reply to prompt for name, got: %s", ungrounded, resU.Reply)
		}
	}

	// 3. Company profile update with ungrounded name returns 0 actions
	resCompUngrounded := driveMatcher(t, "ubah nama perusahaan")
	if len(resCompUngrounded.Actions) != 0 {
		t.Fatalf("expected 0 actions for ungrounded company update, got %d", len(resCompUngrounded.Actions))
	}
	if !strings.Contains(strings.ToLower(resCompUngrounded.Reply), "nama") {
		t.Errorf("expected prompt for company name, got: %s", resCompUngrounded.Reply)
	}

	// 4. Company profile update with grounded name is High Risk with KONFIRMASI
	resCompGrounded := driveMatcher(t, "ubah nama perusahaan jadi PT Nusantara Maju")
	if len(resCompGrounded.Actions) != 1 {
		t.Fatalf("expected 1 action for grounded company update, got %d", len(resCompGrounded.Actions))
	}
	cA := resCompGrounded.Actions[0]
	if cA.ToolName != "update_company_profile" {
		t.Errorf("expected update_company_profile, got %q", cA.ToolName)
	}
	if cA.RiskLevel != "high" {
		t.Errorf("expected high risk level, got %q", cA.RiskLevel)
	}
	if cA.RequiresConfirmationText != "KONFIRMASI" {
		t.Errorf("expected KONFIRMASI confirmation text, got %q", cA.RequiresConfirmationText)
	}
	if cA.Payload["company_name"] != "PT Nusantara Maju" {
		t.Errorf("expected company_name 'PT Nusantara Maju', got %v", cA.Payload["company_name"])
	}

	// 5. Team invite is High Risk with KONFIRMASI
	resTeam := driveMatcher(t, "undang anggota tim user@perusahaan.com sebagai admin")
	if len(resTeam.Actions) != 1 {
		t.Fatalf("expected 1 action for team invite, got %d", len(resTeam.Actions))
	}
	tA := resTeam.Actions[0]
	if tA.RiskLevel != "high" {
		t.Errorf("expected high risk level for invite, got %q", tA.RiskLevel)
	}
	if tA.RequiresConfirmationText != "KONFIRMASI" {
		t.Errorf("expected KONFIRMASI confirmation text for invite, got %q", tA.RequiresConfirmationText)
	}

	// 6. Payment gateway without provider asks clarification
	resGWMissing := driveMatcher(t, "setup payment gateway")
	if len(resGWMissing.Actions) != 0 {
		t.Fatalf("expected 0 actions when payment provider missing, got %d", len(resGWMissing.Actions))
	}
	if !strings.Contains(strings.ToLower(resGWMissing.Reply), "pakasir") && !strings.Contains(strings.ToLower(resGWMissing.Reply), "midtrans") {
		t.Errorf("expected provider choices in reply, got: %s", resGWMissing.Reply)
	}
}

func TestMatcherWMSAndPOSNavigation(t *testing.T) {
	tests := []struct {
		message    string
		expectPath string
	}{
		{"buka kasir pos", "/pos"},
		{"ke menu point of sale", "/pos"},
		{"buka gudang", "/wms"},
		{"buka menu transfer stok", "/wms/transfers"},
		{"buka stock opname", "/wms/opname"},
		{"lihat barang rusak scrap", "/wms/scrap"},
		{"buka barcode scanner", "/wms/scanner"},
		{"buka surat jalan delivery order", "/wms/delivery-orders"},
		{"buka marketplace omnichannel", "/wms/marketplace"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			res := driveMatcher(t, tt.message)
			if len(res.Actions) == 0 {
				t.Fatalf("expected 1 navigation action for %q, got 0", tt.message)
			}
			act := res.Actions[0]
			if act.ToolName != "navigate_to_module" {
				t.Errorf("expected tool navigate_to_module, got %s", act.ToolName)
			}
			path, _ := act.Payload["path"].(string)
			if path != tt.expectPath {
				t.Errorf("for message %q, expected path %s, got %s", tt.message, tt.expectPath, path)
			}
		})
	}
}
