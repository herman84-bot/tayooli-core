package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	groq "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/groq"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// emailRegex matches standard email addresses in user messages.
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// priceRegex matches Indonesian-style prices: Rp 50.000 / 50000 / 50,000
var priceRegex = regexp.MustCompile(`(?i)(?:rp\.?\s*)?(\d[\d.,]*)\s*(?:ribu|rb)?`)

// skuRegex matches "sku <CODE>" or "kode <CODE>" tokens.
var skuRegex = regexp.MustCompile(`(?i)(?:sku|kode)\s+([A-Za-z0-9\-_]+)`)

// phoneRegex matches Indonesian phone numbers: 08xx, +62xx, (021) xxx
var phoneRegex = regexp.MustCompile(`(?:\+62|62|0)\d[\d\-\s]{7,14}\d`)

// poWordRegex matches standalone "po" as a distinct word boundary token.
var poWordRegex = regexp.MustCompile(`(?i)\bpo\b`)

// interrogativeRegex matches question words immediately following an entity prefix
// to avoid false-triggering creation on queries like "vendor apa yang aktif?".
var interrogativeRegex = regexp.MustCompile(`(?i)^(?:vendor|customer|pelanggan|pemasok|supplier)\s+(?:apa|siapa|mana|kenapa|mengapa|bagaimana|berapa)\b`)

// addressRegex matches address keywords and captures remainder
var addressRegex = regexp.MustCompile(`(?i)(?:\balamat\b|\bberalamat\b|\blokasi\b|\baddress\b)\s*:?\s*(.+)`)

// entityMarkerRegex identifies formal company entities
var entityMarkerRegex = regexp.MustCompile(`(?i)\b(PT|CV|UD|TB|PD|FA|Toko|Koperasi|Yayasan)\b\.?`)

// nameIntroRegex checks if name was explicitly introduced
var nameIntroRegex = regexp.MustCompile(`(?i)\b(?:bernama|dengan nama|nama\s*:|nama\s+\S|yaitu)\b`)

// prepositionPrefixRegex checks if a string begins with a preposition
var prepositionPrefixRegex = regexp.MustCompile(`(?i)^(?:untuk|buat|buatkan|dengan|dari|ke|kepada|di)\b`)


// CopilotChatHandler handles AI Copilot chat requests
type CopilotChatHandler struct {
	geminiAPIKey string
	geminiModel  string
	httpClient   *http.Client

	groqAPIKey string
	groqClient *groq.Client

	// geminiDegradedUntil gates Gemini calls after a 429 (RPM rate limit)
	// so every chat message does not pay a full failing round-trip.
	geminiDegradedUntil time.Time
}

// NewCopilotChatHandler creates a new handler with Gemini + Groq API keys.
// Groq acts as a fallback provider when Gemini rate-limits (429) or fails.
func NewCopilotChatHandler(geminiAPIKey, groqAPIKey string) *CopilotChatHandler {
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	h := &CopilotChatHandler{
		geminiAPIKey: geminiAPIKey,
		geminiModel:  model,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		groqAPIKey:   groqAPIKey,
	}
	if groqAPIKey != "" {
		h.groqClient = groq.NewClient(groqAPIKey)
	}
	return h
}

// CopilotRequest represents the incoming chat request
type CopilotRequest struct {
	Message string `json:"message"`
	Context struct {
		TenantID      string   `json:"tenantId"`
		UserRole      string   `json:"userRole"`
		ActiveRoute   string   `json:"activeRoute"`
		ActiveModule  string   `json:"activeModule"`
		AutonomyLevel string   `json:"autonomyLevel"`
		AllowedScopes []string `json:"allowedScopes"`
		EmergencyStop bool     `json:"emergencyStop"`
	} `json:"context"`
}

// ActionDiff represents a field diff in a proposed action
type ActionDiff struct {
	Field    string `json:"field"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// CopilotAction represents a proposed action
type CopilotAction struct {
	ID                       string         `json:"id"`
	ToolName                 string         `json:"toolName"`
	Title                    string         `json:"title"`
	Description              string         `json:"description"`
	RiskLevel                string         `json:"riskLevel"`
	RequiresConfirmationText string         `json:"requiresConfirmationText,omitempty"`
	Diff                     []ActionDiff   `json:"diff"`
	Payload                  map[string]any `json:"payload"`
	Status                   string         `json:"status"`
}

// CopilotResponse represents the chat response
type CopilotResponse struct {
	Reply        string          `json:"reply"`
	Actions      []CopilotAction `json:"actions"`
	ProviderUsed string          `json:"providerUsed"`
}

// Chat processes a copilot chat request
func (h *CopilotChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		RespondError(w, r, http.StatusBadRequest, "message cannot be empty")
		return
	}

	// SECURITY: Override tenant context from JWT — never trust client-supplied values.
	tid, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	req.Context.TenantID = tid.String()
	req.Context.UserRole = appMiddleware.GetRole(r.Context())

	// 1. Emergency Kill-Switch check
	if req.Context.EmergencyStop {
		respondJSON(w, http.StatusOK, CopilotResponse{
			Reply:        "Emergency Kill-Switch aktif: Semua pemanggilan fungsi asisten AI dinonaktifkan oleh administrator workspace. Harap hubungi admin untuk mengaktifkannya kembali.",
			Actions:      []CopilotAction{},
			ProviderUsed: "offline",
		})
		return
	}

	// 2. Try Gemini API (skipped while degraded: 429 = RPM rate limit,
	//    cooldown 15s to match Google's retry-after suggestion)
	if h.geminiAPIKey != "" && time.Now().After(h.geminiDegradedUntil) {
		resp, err := h.callGemini(r.Context(), req)
		if err == nil {
			applyAutonomyEnforcement(resp, req.Context.AutonomyLevel)
			respondJSON(w, http.StatusOK, resp)
			return
		}
		if strings.Contains(err.Error(), "status 429") {
			h.geminiDegradedUntil = time.Now().Add(15 * time.Second)
		}
		// Gemini failed  → fall through to Groq
	}

	// 3. Try Groq as backup provider (LLM reply + offline-rule action detection
	//    so action preview cards still render regardless of provider).
	if h.groqClient != nil {
		resp, err := h.callGroq(r.Context(), req)
		if err == nil {
			applyAutonomyEnforcement(resp, req.Context.AutonomyLevel)
			respondJSON(w, http.StatusOK, resp)
			return
		}
		// Groq also failed  → fall through to rule-based only
	}

	// 4. Fallback to rule-based matcher
	fallback := matchRuleBasedAction(req.Message, req.Context.AutonomyLevel)
	respondJSON(w, http.StatusOK, CopilotResponse{
		Reply:        fallback.Reply,
		Actions:      fallback.Actions,
		ProviderUsed: "offline",
	})
}

// callGroq generates a reply via Groq and attaches offline-rule-detected
// actions so action preview cards render the same as the Gemini path.
func (h *CopilotChatHandler) callGroq(ctx context.Context, req CopilotRequest) (*CopilotResponse, error) {
	system := fmt.Sprintf(`Anda adalah Tayooli Copilot, asisten AI workspace ERP.
Konteks: Tenant=%s, Role=%s, Route=%s, Module=%s, Level=%s
Scopes: %s
Jawab dengan bahasa Indonesia profesional, ringkas, dan ramah.`,
		req.Context.TenantID,
		req.Context.UserRole,
		req.Context.ActiveRoute,
		req.Context.ActiveModule,
		req.Context.AutonomyLevel,
		strings.Join(req.Context.AllowedScopes, ", "),
	)

	msg := groq.ChatMessage{Role: "user", Content: req.Message}
	messages := []groq.ChatMessage{
		{Role: "system", Content: system},
		msg,
	}
	reply, model, err := h.groqClient.ChatCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("groq call: %w", err)
	}

	// Attach offline-rule-detected actions so ActionPreviewCard still appears.
	fallback := matchRuleBasedAction(req.Message, req.Context.AutonomyLevel)
	replyText := reply
	actions := []CopilotAction{}
	if len(fallback.Actions) > 0 {
		actions = fallback.Actions
	}

	if replyText == "" {
		replyText = fallback.Reply
	}

	return &CopilotResponse{
		Reply:        replyText,
		Actions:      actions,
		ProviderUsed: "groq/" + model,
	}, nil
}

// callGemini calls the Gemini API for chat completion
func (h *CopilotChatHandler) callGemini(ctx context.Context, req CopilotRequest) (*CopilotResponse, error) {
	systemInstruction := fmt.Sprintf(`Anda adalah Tayooli Copilot, asisten AI workspace ERP.
Konteks: Tenant=%s, Role=%s, Route=%s, Module=%s, Level=%s
Scopes: %s
Jawab dengan bahasa Indonesia profesional, ringkas, dan ramah.
Jika pengguna meminta tindakan, panggil tool yang sesuai.`,
		req.Context.TenantID,
		req.Context.UserRole,
		req.Context.ActiveRoute,
		req.Context.ActiveModule,
		req.Context.AutonomyLevel,
		strings.Join(req.Context.AllowedScopes, ", "),
	)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": req.Message},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		},
		"tools": copilotToolDeclarations(),
		"generationConfig": map[string]any{
			"temperature": 0.7,
		},
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", h.geminiModel, h.geminiAPIKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("gemini decode: %w", err)
	}

	reply := ""
	proposed := []CopilotAction{}
	if len(geminiResp.Candidates) > 0 {
		parts := geminiResp.Candidates[0].Content.Parts
		for _, p := range parts {
			if p.Text != "" {
				reply += p.Text
			}
			if p.FunctionCall != nil && p.FunctionCall.Name != "" {
				action := buildProposedAction(p.FunctionCall.Name, p.FunctionCall.Args)
				if action != nil {
					proposed = append(proposed, *action)
				}
			}
		}
	}

	if reply == "" && len(proposed) > 0 {
		reply = fmt.Sprintf("Saya telah menyiapkan %d proposal aksi untuk Anda tinjau di bawah ini.", len(proposed))
	} else if reply == "" {
		reply = "Tugas dipahami. Tidak ada aksi perubahan yang diperlukan untuk permintaan ini."
	}

	return &CopilotResponse{
		Reply:        reply,
		Actions:      proposed,
		ProviderUsed: "gemini",
	}, nil
}

// copilotToolDeclarations mirrors lib/copilot/tools.ts (frontend registry) so
// the Gemini function-calling contract is identical on both provider paths.
func copilotToolDeclarations() []map[string]any {
	return []map[string]any{
		{
			"functionDeclarations": []map[string]any{
				{
					"name":        "update_company_profile",
					"description": "Memperbarui profil perusahaan: nama PT, alamat, NPWP.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"company_name": map[string]any{"type": "STRING", "description": "Nama resmi perusahaan"},
							"address":      map[string]any{"type": "STRING", "description": "Alamat kantor"},
							"npwp":         map[string]any{"type": "STRING", "description": "Nomor NPWP"},
						},
					},
				},
				{
					"name":        "invite_team_member",
					"description": "Mengundang anggota tim baru ke workspace dengan peran tertentu.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"email": map[string]any{"type": "STRING", "description": "Email undangan"},
							"role": map[string]any{
								"type": "STRING",
								"description": "Peran anggota",
								"enum": []string{"admin", "accountant", "approver", "treasury", "cfo", "member"},
							},
						},
						"required": []string{"email", "role"},
					},
				},
				{
					"name":        "configure_payment_gateway",
					"description": "Mengatur payment gateway aktif (pakasir atau midtrans). Berisiko tinggi, butuh konfirmasi.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"provider": map[string]any{
								"type": "STRING",
								"description": "Penyedia gateway",
								"enum": []string{"pakasir", "midtrans"},
							},
							"is_active": map[string]any{"type": "BOOLEAN", "description": "Aktif atau tidak"},
						},
						"required": []string{"provider", "is_active"},
					},
				},
				{
					"name":        "create_vendor",
					"description": "Mendaftarkan vendor/pemasok baru ke master data.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"name":         map[string]any{"type": "STRING", "description": "Nama vendor"},
							"email":        map[string]any{"type": "STRING", "description": "Email vendor"},
							"phone":        map[string]any{"type": "STRING", "description": "Telepon vendor"},
							"bank_account": map[string]any{"type": "STRING", "description": "Nomor rekening"},
							"bank_name":    map[string]any{"type": "STRING", "description": "Nama bank"},
							"tax_id":       map[string]any{"type": "STRING", "description": "NPWP vendor"},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "create_customer",
					"description": "Mendaftarkan pelanggan/klien baru ke CRM.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"name":    map[string]any{"type": "STRING", "description": "Nama pelanggan"},
							"email":   map[string]any{"type": "STRING", "description": "Email pelanggan"},
							"phone":   map[string]any{"type": "STRING", "description": "Telepon pelanggan"},
							"address": map[string]any{"type": "STRING", "description": "Alamat pelanggan"},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "create_product",
					"description": "Membuat produk baru dengan SKU di katalog.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"sku":         map[string]any{"type": "STRING", "description": "Kode SKU unik"},
							"name":        map[string]any{"type": "STRING", "description": "Nama produk"},
							"price":       map[string]any{"type": "NUMBER", "description": "Harga satuan"},
							"description": map[string]any{"type": "STRING", "description": "Deskripsi produk"},
						},
						"required": []string{"sku", "name"},
					},
				},
				{
					"name":        "create_purchase_invoice",
					"description": "Membuat draf purchase invoice (tagihan vendor/pemasok) baru dengan nama vendor dan nominal.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"vendor_name": map[string]any{"type": "STRING", "description": "Nama vendor pemasok"},
							"amount":      map[string]any{"type": "NUMBER", "description": "Total nominal tagihan invoice"},
							"currency":    map[string]any{"type": "STRING", "description": "Mata uang (default IDR)"},
							"description": map[string]any{"type": "STRING", "description": "Keterangan tagihan invoice"},
						},
						"required": []string{"vendor_name"},
					},
				},
				{
					"name":        "get_dashboard_summary",
					"description": "Mengambil ringkasan metrik keuangan dan operasional dashboard workspace.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"metric": map[string]any{"type": "STRING", "description": "Jenis metrik ringkasan (overview, cash_flow, invoices)"},
							"period": map[string]any{"type": "STRING", "description": "Periode waktu (current_month, today, year_to_date)"},
						},
					},
				},
				{
					"name":        "navigate_to_module",
					"description": "Membuka halaman modul ERP tertentu di aplikasi.",
					"parameters": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"path": map[string]any{
								"type": "STRING",
								"description": "Rute halaman tujuan",
								"enum": []string{"/dashboard", "/dashboard/invoices", "/dashboard/purchase-orders", "/dashboard/goods-receipts", "/dashboard/payment-orders", "/dashboard/vendors", "/dashboard/customers", "/approvals", "/accounting/chart-of-accounts", "/accounting/journal-entries", "/settings", "/pos", "/wms", "/wms/delivery-orders", "/wms/marketplace", "/wms/transfers", "/wms/opname", "/wms/scrap", "/wms/scanner"},
							},
							"module_name": map[string]any{"type": "STRING", "description": "Nama modul yang ditampilkan ke user"},
						},
						"required": []string{"path"},
					},
				},
			},
		},
	}
}

// buildProposedAction converts a Gemini function call into a reviewable
// CopilotAction with the same risk levels as the rule-based fallback.
func buildProposedAction(toolName string, args map[string]interface{}) *CopilotAction {
	id := fmt.Sprintf("gemini-%s-%d", toolName, time.Now().UnixMilli())

	switch toolName {
	case "update_company_profile":
		companyName := strOr(strVal(args["company_name"]), "")
		if isRejectedName(companyName) {
			return nil
		}
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:                    "Pembaruan Profil Perusahaan",
			Description:              fmt.Sprintf("Memperbarui nama profil perusahaan menjadi '%s'.", companyName),
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     []ActionDiff{{Field: "company_name", OldValue: "(Profil Saat Ini)", NewValue: companyName}},
			Payload:                  args,
			Status:                   "pending",
		}
	case "invite_team_member", "invite_member":
		email, _ := args["email"].(string)
		role, _ := args["role"].(string)
		email = strings.TrimSpace(strings.ToLower(email))
		if !emailRegex.MatchString(email) {
			return nil
		}
		if mappedRole, ok := roleKeywordMap[strings.ToLower(role)]; ok {
			role = mappedRole
		} else if role == "" {
			role = "member"
		}
		args["email"] = email
		args["role"] = role
		return &CopilotAction{
			ID: id, ToolName: "invite_team_member",
			Title:                    "Undang Anggota Tim",
			Description:              fmt.Sprintf("Mengundang %s sebagai %s.", email, role),
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     []ActionDiff{{Field: "email", OldValue: "-", NewValue: email}, {Field: "role", OldValue: "-", NewValue: role}},
			Payload:                  args,
			Status:                   "pending",
		}
	case "configure_payment_gateway":
		provider, _ := args["provider"].(string)
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:                    fmt.Sprintf("Konfigurasi Gateway: %s", strings.ToUpper(provider)),
			Description:              fmt.Sprintf("Mengatur penyedia %s sebagai gateway pembayaran aktif.", provider),
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     []ActionDiff{{Field: "provider", OldValue: "-", NewValue: provider}},
			Payload:                  args,
			Status:                   "pending",
		}
	case "create_vendor":
		vendorName, _ := args["name"].(string)
		vendorName = strings.TrimSpace(vendorName)
		if isRejectedName(vendorName) {
			return nil
		}
		diffs := []ActionDiff{}
		for _, f := range []string{"name", "email", "phone", "address", "bank_account", "bank_name", "tax_id"} {
			if v, ok := args[f].(string); ok && v != "" {
				diffs = append(diffs, ActionDiff{Field: f, OldValue: "-", NewValue: v})
			}
		}
		if len(diffs) == 0 && vendorName != "" {
			diffs = append(diffs, ActionDiff{Field: "name", OldValue: "-", NewValue: vendorName})
		}
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:       "Pendaftaran Rekanan Vendor Baru",
			Description: fmt.Sprintf("Mendaftarkan mitra vendor baru '%s' ke dalam direktori rekanan bisnis.", vendorName),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     args,
			Status:      "pending",
		}
	case "create_customer":
		customerName, _ := args["name"].(string)
		customerName = strings.TrimSpace(customerName)
		if isRejectedName(customerName) {
			return nil
		}
		diffs := []ActionDiff{}
		for _, f := range []string{"name", "email", "phone", "address"} {
			if v, ok := args[f].(string); ok && v != "" {
				diffs = append(diffs, ActionDiff{Field: f, OldValue: "-", NewValue: v})
			}
		}
		if len(diffs) == 0 && customerName != "" {
			diffs = append(diffs, ActionDiff{Field: "name", OldValue: "-", NewValue: customerName})
		}
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:       "Pendaftaran Pelanggan Baru",
			Description: fmt.Sprintf("Mendaftarkan pelanggan baru '%s' ke dalam sistem CRM dan penjualan.", customerName),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     args,
			Status:      "pending",
		}
	case "create_product":
		sku, _ := args["sku"].(string)
		prodName, _ := args["name"].(string)
		sku = strings.TrimSpace(sku)
		if sku == "" {
			sku = fmt.Sprintf("PRD-%d", time.Now().Unix()%10000)
			args["sku"] = sku
		}
		diffs := []ActionDiff{
			{Field: "sku", OldValue: "-", NewValue: sku},
			{Field: "name", OldValue: "-", NewValue: strOr(prodName, "Nama Produk")},
		}
		if p, ok := args["price"]; ok {
			var priceNum float64
			switch v := p.(type) {
			case float64:
				priceNum = v
			case int:
				priceNum = float64(v)
			case string:
				priceNum, _ = strconv.ParseFloat(v, 64)
			}
			if priceNum > 0 {
				diffs = append(diffs, ActionDiff{Field: "price", OldValue: "-", NewValue: fmt.Sprintf("Rp %s", formatPrice(priceNum))})
			}
		}
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:       "Penambahan Produk Baru",
			Description: fmt.Sprintf("Membuat SKU produk baru '%s' (%s) di katalog barang.", strOr(prodName, "Nama Produk"), sku),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     args,
			Status:      "pending",
		}
	case "create_purchase_invoice":
		vendorName, _ := args["vendor_name"].(string)
		if vendorName == "" {
			if v, ok := args["vendor"].(string); ok {
				vendorName = v
			}
		}
		var amount float64
		switch v := args["amount"].(type) {
		case float64:
			amount = v
		case int:
			amount = float64(v)
		case string:
			amount, _ = strconv.ParseFloat(v, 64)
		}
		currency, _ := args["currency"].(string)
		if currency == "" {
			currency = "IDR"
		}
		args["currency"] = currency
		args["vendor_name"] = vendorName
		args["amount"] = amount

		diffs := []ActionDiff{
			{Field: "vendor_name", OldValue: "-", NewValue: strOr(vendorName, "(Pilih Vendor)")},
		}
		desc := fmt.Sprintf("Membuat draf purchase invoice untuk vendor %s.", strOr(vendorName, "(Pilih Vendor)"))
		if amount > 0 {
			diffs = append(diffs, ActionDiff{Field: "amount", OldValue: "-", NewValue: fmt.Sprintf("Rp %s", formatPrice(amount))})
			desc = fmt.Sprintf("Membuat draf purchase invoice untuk vendor %s senilai Rp %s.", strOr(vendorName, "(Pilih Vendor)"), formatPrice(amount))
		}
		return &CopilotAction{
			ID:          id,
			ToolName:    toolName,
			Title:       "Pembuatan Purchase Invoice Baru",
			Description: desc,
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     args,
			Status:      "pending",
		}
	case "get_dashboard_summary":
		return &CopilotAction{
			ID:          id,
			ToolName:    toolName,
			Title:       "Ringkasan Dashboard Finansial",
			Description: "Mengambil dan menampilkan ringkasan metrik keuangan dan operasional workspace.",
			RiskLevel:   "low",
			Diff:        []ActionDiff{{Field: "modul", OldValue: "-", NewValue: "Dashboard Finansial"}},
			Payload:     args,
			Status:      "pending",
		}
	case "navigate_to_module":
		path, _ := args["path"].(string)
		mod, _ := args["module_name"].(string)
		if mod == "" {
			mod = path
		}
		return &CopilotAction{
			ID: id, ToolName: toolName,
			Title:       fmt.Sprintf("Navigasi ke %s", mod),
			Description: fmt.Sprintf("Membuka dan mengarahkan tampilan langsung ke modul %s.", mod),
			RiskLevel:   "low",
			Diff:        []ActionDiff{{Field: "rute", OldValue: "Halaman Saat Ini", NewValue: path}},
			Payload:     args,
			Status:      "pending",
		}
	default:
		return nil
	}
}

// strVal safely coerces an interface{} from Gemini args to string.
func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// strOr returns fallback when s is empty.
func strOr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// extractAfterPatterns tries to find an entity name following common Indonesian patterns:
// "nama X", "bernama X", "yaitu X", "untuk X", or the word after the trigger keyword.
// Patterns are matched longest-first so "bernama" wins over bare "vendor ".
func extractAfterPatterns(message string, patterns ...string) string {
	lower := strings.ToLower(message)
	// Sort patterns by descending length so more specific phrases match first.
	sorted := append([]string{}, patterns...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && len(sorted[j]) > len(sorted[j-1]); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	for _, p := range sorted {
		idx := strings.Index(lower, p)
		if idx == -1 {
			continue
		}
		rest := strings.TrimSpace(message[idx+len(p):])
		if rest == "" {
			continue
		}
		// Take everything until "email", "telepon", "phone", "hp", "telp",
		// "npwp", "bank", "rekening", "sku", "harga", or end of string.
		separators := []string{
			" sebesar", " sejumlah", " senilai", " nominal", " harga", " total",
			" dengan", " di ", " nilai", " rp.", " rp",
			" email", " telepon", " phone", " hp", " telp",
			" alamat", " beralamat", " lokasi", " address",
			" npwp", " bank", " rekening", " sku", ",",
		}
		best := len(rest)
		for _, sep := range separators {
			if i := strings.Index(strings.ToLower(rest), sep); i > 0 && i < best {
				best = i
			}
		}
		name := strings.TrimSpace(rest[:best])
		if name != "" {
			return name
		}
	}
	return ""
}

// extractEmailRegex finds the first email address in the message via regex.
func extractEmailRegex(message string) string {
	m := emailRegex.FindString(message)
	return strings.ToLower(m)
}

// extractPhone finds the first phone number in the message.
func extractPhone(message string) string {
	m := phoneRegex.FindString(message)
	// Clean up spaces and dashes
	m = strings.ReplaceAll(m, " ", "")
	m = strings.ReplaceAll(m, "-", "")
	return m
}

// extractSKU finds a SKU/kode token in the message.
func extractSKU(message string) string {
	m := skuRegex.FindStringSubmatch(message)
	if len(m) >= 2 {
		return strings.ToUpper(m[1])
	}
	return ""
}

// extractAddress finds the address in the message if present.
func extractAddress(message string) string {
	m := addressRegex.FindStringSubmatch(message)
	if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
		return ""
	}
	rest := strings.TrimSpace(m[1])
	separators := []string{"email:", "telepon:", "telp:", "phone:", "hp:", "npwp:", "bank:"}
	low := strings.ToLower(rest)
	best := len(rest)
	for _, sep := range separators {
		if idx := strings.Index(low, sep); idx >= 0 && idx < best {
			best = idx
		}
	}
	rest = rest[:best]
	rest = emailRegex.ReplaceAllString(rest, " ")
	rest = phoneRegex.ReplaceAllString(rest, " ")
	rest = regexp.MustCompile(`(?i)\b(?:e-?mail|telepon|telp|phone|hp)\s*:?\s*`).ReplaceAllString(rest, " ")
	rest = regexp.MustCompile(`\s{2,}`).ReplaceAllString(rest, " ")
	return strings.Trim(rest, " \t\r\n,;:-")
}

// stripAddressFromName removes address content and labels from extracted name.
func stripAddressFromName(name, address string) string {
	if address != "" {
		idx := strings.Index(strings.ToLower(name), strings.ToLower(address))
		if idx >= 0 {
			name = strings.TrimSpace(name[:idx] + name[idx+len(address):])
		}
	}
	cut := regexp.MustCompile(`(?i)\b(?:alamat|beralamat|lokasi|address)\b`).FindStringIndex(name)
	if len(cut) > 0 {
		name = strings.TrimSpace(name[:cut[0]])
	}
	return strings.Trim(name, " \t\r\n,;:-")
}

// isRejectedName checks if a candidate name is generic junk or starts with a preposition.
func isRejectedName(candidate string) bool {
	t := strings.Trim(candidate, " \t\r\n,;:-")
	if len(t) < 2 {
		return true
	}
	low := strings.ToLower(t)
	switch low {
	case "baru", "perusahaan", "vendor", "customer", "produk", "pelanggan", "item", "barang", "(isi nama vendor)", "(isi nama pelanggan)":
		return true
	}
	return prepositionPrefixRegex.MatchString(t)
}

// isGroundedName checks whether the name is accompanied by an entity marker, introducer, contact details, or is a valid multi-word name.
func isGroundedName(candidate, message string) bool {
	if isRejectedName(candidate) {
		return false
	}
	if nameIntroRegex.MatchString(message) ||
		entityMarkerRegex.MatchString(message) ||
		emailRegex.MatchString(message) ||
		phoneRegex.MatchString(message) {
		return true
	}
	words := strings.Fields(candidate)
	if len(words) >= 2 {
		firstWord := strings.ToLower(words[0])
		if !isRejectedName(firstWord) && !prepositionPrefixRegex.MatchString(firstWord) {
			return true
		}
	} else if len(words) == 1 {
		if len(candidate) >= 3 && unicode.IsUpper(rune(candidate[0])) && !isRejectedName(candidate) {
			return true
		}
	}
	return false
}

// isGroundedCompanyName checks whether company name was clearly specified.
func isGroundedCompanyName(candidate, message string) bool {
	if isRejectedName(candidate) {
		return false
	}
	low := strings.ToLower(message)
	hasIntro := strings.Contains(low, "jadi") ||
		strings.Contains(low, "menjadi") ||
		strings.Contains(low, "yaitu") ||
		strings.Contains(message, `"`) ||
		regexp.MustCompile(`(?i)\bnama\s*:`).MatchString(message)
	return hasIntro || entityMarkerRegex.MatchString(message)
}

// isGroundedProductName checks whether product name was clearly grounded.
func isGroundedProductName(candidate, message string, price float64) bool {
	t := strings.Trim(candidate, " \t\r\n,;:-")
	if len(t) < 2 || isRejectedName(t) {
		return false
	}
	if nameIntroRegex.MatchString(message) || regexp.MustCompile(`(?i)\b(?:sku|kode)\b`).MatchString(message) || price > 0 {
		return true
	}
	return len(t) >= 4 && regexp.MustCompile(`[A-Z0-9]`).MatchString(t)
}

// extractPrice finds a numeric price value in the message.
// Returns 0 if not found. Handles "harga 50000", "Rp 50.000", "50000".
func extractPrice(message string) float64 {
	lower := strings.ToLower(message)
	// Look for "harga" keyword first to scope the search
	idx := strings.Index(lower, "harga")
	search := message
	if idx >= 0 {
		search = message[idx:]
	}
	m := priceRegex.FindStringSubmatch(search)
	if len(m) >= 2 {
		numStr := m[1]
		// Remove thousand separators (dots in Indonesian, commas)
		numStr = strings.ReplaceAll(numStr, ".", "")
		numStr = strings.ReplaceAll(numStr, ",", "")
		val, err := strconv.ParseFloat(numStr, 64)
		if err == nil {
			// Handle "ribu" / "rb" suffix
			if strings.Contains(strings.ToLower(search), "ribu") || strings.Contains(strings.ToLower(search), " rb") {
				val *= 1000
			}
			return val
		}
	}
	return 0
}

// roleKeywordMap maps Indonesian/English keywords to valid role enum values.
var roleKeywordMap = map[string]string{
	"admin":      "admin",
	"finance":    "accountant",
	"keuangan":   "accountant",
	"akuntan":    "accountant",
	"accountant": "accountant",
	"approver":   "approver",
	"persetujuan": "approver",
	"treasury":   "treasury",
	"bendahara":  "treasury",
	"cfo":        "cfo",
	"member":     "member",
	"anggota":    "member",
	"viewer":     "member",
}

// extractRole finds a role keyword in the message and maps it to the valid enum.
func extractRole(message string) string {
	lower := strings.ToLower(message)
	// Check "sebagai <role>" pattern first (most specific)
	for _, prefix := range []string{"sebagai ", "sebagai: ", "role ", "peran ", "jadi "} {
		idx := strings.Index(lower, prefix)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(lower[idx+len(prefix):])
		word := strings.Fields(rest)
		if len(word) > 0 {
			if mapped, ok := roleKeywordMap[word[0]]; ok {
				return mapped
			}
		}
	}
	// Fallback: scan entire message for any known role keyword
	for keyword, role := range roleKeywordMap {
		if strings.Contains(lower, keyword) {
			return role
		}
	}
	return ""
}

// cleanVendorName removes email, phone, and keyword artifacts from vendor name.
func cleanVendorName(name, email, phone string) string {
	if email != "" {
		name = strings.ReplaceAll(name, email, "")
	}
	if phone != "" {
		name = strings.ReplaceAll(name, phone, "")
	}
	strips := []string{
		"email:", "email", "e-mail:", "e-mail",
		"telepon:", "telepon", "telp:", "telp", "phone:", "phone", "hp:", "hp",
		"no.", "nomor", "sebesar", "sejumlah", "rp.", "rp", "senilai", "nominal",
		"dengan", "dan",
	}
	lower := strings.ToLower(name)
	for _, s := range strips {
		for strings.Contains(lower, s) {
			idx := strings.Index(lower, s)
			name = name[:idx] + name[idx+len(s):]
			lower = strings.ToLower(name)
		}
	}
	name = strings.Trim(name, " \t\r\n,;:-")
	name = regexp.MustCompile(`(?i)^baru\s+`).ReplaceAllString(name, "")
	return strings.Trim(name, " \t\r\n,;:-")
}

// cleanProductName strips SKU, kode, harga, and Rp tokens from product name.
func cleanProductName(name, sku string) string {
	if sku != "" {
		name = strings.ReplaceAll(name, sku, "")
	}
	name = skuRegex.ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)(?:seharga|harga)?\s*rp\.?\s*\d[\d.,]*\s*(?:ribu|rb)?`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)\bharga\s+\d[\d.,]*\b`).ReplaceAllString(name, "")
	strips := []string{"kode:", "kode", "sku:", "sku", "harga:", "harga", "seharga", "dengan", "rp.", "rp"}
	lower := strings.ToLower(name)
	for _, s := range strips {
		for strings.Contains(lower, s) {
			idx := strings.Index(lower, s)
			name = name[:idx] + name[idx+len(s):]
			lower = strings.ToLower(name)
		}
	}
	return strings.Trim(name, " \t\r\n,;:-")
}

// formatPrice formats numeric price with dots (e.g. 75000 -> 75.000).
func formatPrice(val float64) string {
	intVal := int64(val)
	str := strconv.FormatInt(intVal, 10)
	n := len(str)
	if n <= 3 {
		return str
	}
	var res []byte
	rem := n % 3
	if rem > 0 {
		res = append(res, str[:rem]...)
	}
	for i := rem; i < n; i += 3 {
		if len(res) > 0 {
			res = append(res, '.')
		}
		res = append(res, str[i:i+3]...)
	}
	return string(res)
}

// filterAdvisoryActions removes any mutating actions if autonomy level is advisory.
// In advisory mode, only navigation or read-only actions are permitted.
func filterAdvisoryActions(actions []CopilotAction, autonomyLevel string) []CopilotAction {
	if strings.ToLower(strings.TrimSpace(autonomyLevel)) != "advisory" {
		return actions
	}
	filtered := make([]CopilotAction, 0)
	for _, a := range actions {
		if a.ToolName == "navigate_to_module" {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// applyAutonomyEnforcement ensures advisory mode blocks mutating action cards across any provider.
func applyAutonomyEnforcement(resp *CopilotResponse, autonomyLevel string) {
	if resp == nil || strings.ToLower(strings.TrimSpace(autonomyLevel)) != "advisory" {
		return
	}
	origCount := len(resp.Actions)
	resp.Actions = filterAdvisoryActions(resp.Actions, autonomyLevel)
	if origCount > 0 && len(resp.Actions) < origCount {
		resp.Reply += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
	}
}

// parseAmount parses numeric amounts in Indonesian and standard formats:
// e.g. "5 juta" -> 5000000, "2.5 juta" -> 2500000, "5.000.000" -> 5000000, "500 ribu" -> 500000, "10jt" -> 10000000.
func parseAmount(message string) float64 {
	lower := strings.ToLower(message)

	// 1. Multiplier patterns: "5 juta", "2.5 jt", "500 ribu", "500rb", "10jt", "1 miliar"
	multRegex := regexp.MustCompile(`(?i)(?:rp\.?\s*)?(\d+(?:[.,]\d+)?)\s*(miliar|milyar|juta|jt|ribu|rb)\b`)
	if m := multRegex.FindStringSubmatch(lower); len(m) >= 3 {
		numStr := strings.ReplaceAll(m[1], ",", ".")
		val, err := strconv.ParseFloat(numStr, 64)
		if err == nil {
			unit := strings.ToLower(m[2])
			switch unit {
			case "miliar", "milyar":
				return val * 1_000_000_000
			case "juta", "jt":
				return val * 1_000_000
			case "ribu", "rb":
				return val * 1_000
			}
		}
	}

	// 2. Explicit formatted numbers: "5.000.000", "5,000,000", or "5000000"
	search := message
	for _, kw := range []string{"sebesar", "sejumlah", "rp", "nominal", "total", "nilai", "senilai", "harga"} {
		if idx := strings.Index(lower, kw); idx >= 0 {
			search = message[idx:]
			break
		}
	}
	currRegex := regexp.MustCompile(`(?i)(?:rp\.?\s*)?(\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d{1,3}(?:,\d{3})+(?:\.\d+)?|\d{4,})`)
	if m := currRegex.FindStringSubmatch(search); len(m) >= 2 {
		numStr := m[1]
		if strings.Contains(numStr, ".") && strings.Contains(numStr, ",") {
			if strings.LastIndex(numStr, ",") > strings.LastIndex(numStr, ".") {
				numStr = strings.ReplaceAll(numStr, ".", "")
				numStr = strings.ReplaceAll(numStr, ",", ".")
			} else {
				numStr = strings.ReplaceAll(numStr, ",", "")
			}
		} else if strings.Contains(numStr, ".") {
			numStr = strings.ReplaceAll(numStr, ".", "")
		} else if strings.Contains(numStr, ",") {
			numStr = strings.ReplaceAll(numStr, ",", "")
		}
		val, err := strconv.ParseFloat(numStr, 64)
		if err == nil && val > 0 {
			return val
		}
	}

	return 0
}

// extractInvoiceVendor extracts the vendor name from an invoice creation prompt.
// It terminates the vendor name before stop words: sebesar, sejumlah, rp, nominal, harga, total, dengan, di, nilai, senilai.
func extractInvoiceVendor(message string) string {
	lower := strings.ToLower(message)
	prefixes := []string{
		"untuk vendor ", "dari vendor ", "kepada vendor ", "ke vendor ", "pada vendor ",
		"buatkan invoice untuk ", "buat invoice untuk ", "buatkan tagihan untuk ", "buat tagihan untuk ",
		"catat invoice untuk ", "catat tagihan untuk ",
		"invoice untuk vendor ", "tagihan untuk vendor ", "faktur untuk vendor ",
		"invoice untuk ", "tagihan untuk ", "faktur untuk ",
		"invoice vendor ", "tagihan vendor ", "faktur vendor ",
		"vendor ", "pemasok ", "supplier ",
		"untuk ", "dari ", "kepada ",
	}
	var rest string
	for _, p := range prefixes {
		if idx := strings.Index(lower, p); idx != -1 {
			candidate := strings.TrimSpace(message[idx+len(p):])
			if candidate != "" {
				rest = candidate
				break
			}
		}
	}
	if rest == "" {
		return ""
	}

	stopWords := []string{
		" sebesar", " sejumlah", " senilai", " nominal", " harga", " total",
		" dengan", " di ", " nilai", " rp.", " rp", " seharga",
		" tanggal", " jatuh tempo", " email", " telepon", " telp", " phone", " hp",
		",", ";",
	}

	best := len(rest)
	lowerRest := strings.ToLower(rest)
	for _, sw := range stopWords {
		if i := strings.Index(lowerRest, sw); i >= 0 && i < best {
			best = i
		}
	}

	name := strings.TrimSpace(rest[:best])
	name = strings.Trim(name, " \t\r\n,;:-'\"")
	return name
}

// matchRuleBasedAction is the deterministic rule-based fallback matcher
func matchRuleBasedAction(message string, autonomyLevel string) CopilotResponse {
	normalized := strings.ToLower(strings.TrimSpace(message))
	actions := []CopilotAction{}

	// 1. Greetings & Small Talk
	if strings.HasPrefix(normalized, "halo") || strings.HasPrefix(normalized, "hallo") ||
		strings.HasPrefix(normalized, "hai") || strings.HasPrefix(normalized, "hi") ||
		strings.HasPrefix(normalized, "hey") || strings.HasPrefix(normalized, "selamat") {
		return CopilotResponse{
			Reply:        "Halo! Senang bisa menyapa Anda. Ada yang bisa saya bantu untuk operasional workspace Anda hari ini? Anda bisa meminta saya mengelola profil perusahaan, mengundang tim, mendaftarkan vendor atau produk, hingga membuka modul transaksi.",
			Actions:      []CopilotAction{},
			ProviderUsed: "offline",
		}
	}

	// Well-being & identity small talk (aligned with frontend fallback-matcher)
	if strings.Contains(normalized, "apa kabar") || strings.Contains(normalized, "gimana kabarnya") || strings.Contains(normalized, "bagaimana kabarmu") {
		return CopilotResponse{
			Reply:        "Kabar baik dan sistem siap membantu Anda kapan saja! Ada tugas atau pengaturan workspace yang ingin kita selesaikan hari ini?",
			Actions:      []CopilotAction{},
			ProviderUsed: "offline",
		}
	}

	if strings.Contains(normalized, "siapa kamu") || strings.Contains(normalized, "kamu siapa") || strings.Contains(normalized, "bisa apa") || strings.Contains(normalized, "bisa ngapain") || strings.Contains(normalized, "bantuan apa") {
		return CopilotResponse{
			Reply:        "Saya Tayooli Copilot, asisten yang siap membantu Anda mengelola pengaturan dan data di Tayooli ERP. Anda cukup memerintahkan saya dengan bahasa sehari-hari, misalnya: 'ubah nama perusahaan', 'tambah vendor baru', 'undang anggota tim', atau 'buka modul akuntansi'.",
			Actions:      []CopilotAction{},
			ProviderUsed: "offline",
		}
	}

	if strings.Contains(normalized, "terima kasih") || strings.Contains(normalized, "makasih") || strings.Contains(normalized, "thanks") || strings.Contains(normalized, "tengkyu") {
		return CopilotResponse{
			Reply:        "Sama-sama! Senang bisa membantu kelancaran kerja Anda. Beri tahu saya jika ada hal lain yang ingin diselesaikan ya.",
			Actions:      []CopilotAction{},
			ProviderUsed: "offline",
		}
	}

	// 2. Navigation evaluated FIRST
	isNavigation := strings.Contains(normalized, "buka") ||
		strings.Contains(normalized, "lihat") ||
		strings.Contains(normalized, "menu") ||
		strings.Contains(normalized, "ke halaman") ||
		strings.Contains(normalized, "tampilkan") ||
		strings.Contains(normalized, "navigasi")

	if isNavigation {
		if strings.Contains(normalized, "ringkasan") || strings.Contains(normalized, "summary") {
			actions = append(actions, CopilotAction{
				ID:          fmt.Sprintf("rule-summary-%d", time.Now().UnixMilli()),
				ToolName:    "get_dashboard_summary",
				Title:       "Ringkasan Dashboard Finansial",
				Description: "Mengambil dan menampilkan ringkasan metrik keuangan dan operasional workspace.",
				RiskLevel:   "low",
				Diff:        []ActionDiff{{Field: "modul", OldValue: "-", NewValue: "Dashboard Finansial"}},
				Payload:     map[string]any{"metric": "overview", "period": "current_month"},
				Status:      "pending",
			})
			return CopilotResponse{
				Reply:        "Saya telah menyiapkan aksi untuk mengambil ringkasan dashboard dan metrik keuangan workspace Anda. Silakan jalankan aksi di bawah.",
				Actions:      actions,
				ProviderUsed: "offline",
			}
		}

		path := "/dashboard"
		mod := "Dashboard"
		if strings.Contains(normalized, "vendor") || strings.Contains(normalized, "pemasok") || strings.Contains(normalized, "supplier") {
			path, mod = "/dashboard/vendors", "Vendors"
		} else if strings.Contains(normalized, "customer") || strings.Contains(normalized, "pelanggan") || strings.Contains(normalized, "klien") {
			path, mod = "/dashboard/customers", "Customers"
		} else if strings.Contains(normalized, "invoice") || strings.Contains(normalized, "tagihan") || strings.Contains(normalized, "faktur") {
			path, mod = "/dashboard/invoices", "Invoices"
		} else if strings.Contains(normalized, "purchase order") || strings.Contains(normalized, "purchase") || poWordRegex.MatchString(normalized) {
			path, mod = "/dashboard/purchase-orders", "Purchase Orders"
		} else if strings.Contains(normalized, "gr") || strings.Contains(normalized, "goods receipt") || strings.Contains(normalized, "penerimaan") {
			path, mod = "/dashboard/goods-receipts", "Goods Receipts"
		} else if strings.Contains(normalized, "gateway") || strings.Contains(normalized, "pakasir") || strings.Contains(normalized, "midtrans") {
			path, mod = "/dashboard/payment-gateways", "Payment Gateways"
		} else if strings.Contains(normalized, "pembayaran") || strings.Contains(normalized, "payment order") || strings.Contains(normalized, "bayar") {
			path, mod = "/dashboard/payment-orders", "Payment Orders"
		} else if strings.Contains(normalized, "coa") || strings.Contains(normalized, "akun") || strings.Contains(normalized, "chart of account") {
			path, mod = "/accounting/chart-of-accounts", "Chart of Accounts"
		} else if strings.Contains(normalized, "jurnal") || strings.Contains(normalized, "journal") {
			path, mod = "/accounting/journal-entries", "Journal Entries"
		} else if strings.Contains(normalized, "pos") || strings.Contains(normalized, "kasir") || strings.Contains(normalized, "point of sale") {
			path, mod = "/pos", "Point of Sale"
		} else if strings.Contains(normalized, "surat jalan") || strings.Contains(normalized, "delivery order") {
			path, mod = "/wms/delivery-orders", "Surat Jalan (DO)"
		} else if strings.Contains(normalized, "marketplace") || strings.Contains(normalized, "omnichannel") {
			path, mod = "/wms/marketplace", "Marketplace Omnichannel"
		} else if strings.Contains(normalized, "transfer") || strings.Contains(normalized, "mutasi") {
			path, mod = "/wms/transfers", "Stock Transfers"
		} else if strings.Contains(normalized, "opname") {
			path, mod = "/wms/opname", "Stock Opname"
		} else if strings.Contains(normalized, "scrap") || strings.Contains(normalized, "rusak") || strings.Contains(normalized, "afkir") {
			path, mod = "/wms/scrap", "Barang Rusak / Scrap"
		} else if strings.Contains(normalized, "scanner") || strings.Contains(normalized, "scan") {
			path, mod = "/wms/scanner", "Barcode Scanner"
		} else if strings.Contains(normalized, "gudang") || strings.Contains(normalized, "warehouse") || strings.Contains(normalized, "wms") {
			path, mod = "/wms", "Warehouse & Stock"
		} else if strings.Contains(normalized, "produk") || strings.Contains(normalized, "barang") || strings.Contains(normalized, "katalog") || strings.Contains(normalized, "item") {
			path, mod = "/products", "Katalog Produk"
		} else if strings.Contains(normalized, "setting") || strings.Contains(normalized, "pengaturan") {
			path, mod = "/settings", "Pengaturan"
		}

		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-nav-%d", time.Now().UnixMilli()),
			ToolName:    "navigate_to_module",
			Title:       fmt.Sprintf("Navigasi ke %s", mod),
			Description: fmt.Sprintf("Membuka dan mengarahkan tampilan langsung ke modul %s.", mod),
			RiskLevel:   "low",
			Diff:        []ActionDiff{{Field: "rute", OldValue: "Halaman Saat Ini", NewValue: path}},
			Payload:     map[string]any{"path": path, "module_name": mod},
			Status:      "pending",
		})
		return CopilotResponse{
			Reply:        fmt.Sprintf("Saya menyiapkan navigasi cepat ke %s. Klik jalankan untuk membuka halaman.", mod),
			Actions:      actions,
			ProviderUsed: "offline",
		}
	}

	// 3. Transactions (invoice, tagihan, faktur, purchase order, po, ringkasan, summary) evaluated BEFORE Master Data
	isInvoiceCreation := (strings.Contains(normalized, "invoice") || strings.Contains(normalized, "tagihan") || strings.Contains(normalized, "faktur")) &&
		(strings.Contains(normalized, "buat") || strings.Contains(normalized, "buatkan") ||
			strings.Contains(normalized, "tambah") || strings.Contains(normalized, "tambahkan") ||
			strings.Contains(normalized, "catat") || strings.Contains(normalized, "terbitkan") ||
			strings.Contains(normalized, "bikin") || strings.Contains(normalized, "input") ||
			strings.Contains(normalized, "sebesar") || strings.Contains(normalized, "sejumlah") ||
			strings.Contains(normalized, "senilai") || strings.Contains(normalized, "nominal") ||
			strings.Contains(normalized, "rp"))

	if isInvoiceCreation {
		vendorName := extractInvoiceVendor(message)
		amount := parseAmount(message)
		if vendorName == "" || isRejectedName(vendorName) {
			return CopilotResponse{
				Reply:        "Untuk membuat purchase invoice, saya membutuhkan nama vendor rekanan yang jelas. Contoh: 'buatkan invoice untuk vendor PT Maju Jaya sebesar 5 juta rupiah'. Vendor mana yang ingin dibuatkan invoice?",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}

		payload := map[string]any{
			"vendor_name": vendorName,
			"amount":      amount,
			"currency":    "IDR",
		}
		diffs := []ActionDiff{
			{Field: "vendor_name", OldValue: "-", NewValue: vendorName},
		}
		desc := fmt.Sprintf("Membuat draf purchase invoice untuk vendor %s.", vendorName)
		if amount > 0 {
			diffs = append(diffs, ActionDiff{Field: "amount", OldValue: "-", NewValue: fmt.Sprintf("Rp %s", formatPrice(amount))})
			desc = fmt.Sprintf("Membuat draf purchase invoice untuk vendor %s senilai Rp %s.", vendorName, formatPrice(amount))
		}

		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-invoice-%d", time.Now().UnixMilli()),
			ToolName:    "create_purchase_invoice",
			Title:       "Pembuatan Purchase Invoice Baru",
			Description: desc,
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     payload,
			Status:      "pending",
		})

		replyText := fmt.Sprintf("Proposal pembuatan draf purchase invoice untuk vendor '%s' telah disiapkan. Silakan tinjau dan jalankan aksi berikut.", vendorName)
		if amount > 0 {
			replyText = fmt.Sprintf("Proposal pembuatan draf purchase invoice untuk vendor '%s' senilai Rp %s telah disiapkan. Silakan tinjau dan jalankan aksi berikut.", vendorName, formatPrice(amount))
		}

		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	isSummaryRequest := strings.Contains(normalized, "ringkasan") ||
		strings.Contains(normalized, "summary") ||
		strings.Contains(normalized, "total tagihan") ||
		strings.Contains(normalized, "rekapitulasi")

	if isSummaryRequest {
		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-summary-%d", time.Now().UnixMilli()),
			ToolName:    "get_dashboard_summary",
			Title:       "Ringkasan Dashboard Finansial",
			Description: "Mengambil dan menampilkan ringkasan metrik keuangan dan operasional workspace.",
			RiskLevel:   "low",
			Diff:        []ActionDiff{{Field: "modul", OldValue: "-", NewValue: "Dashboard Finansial"}},
			Payload:     map[string]any{"metric": "overview", "period": "current_month"},
			Status:      "pending",
		})
		replyText := "Saya telah menyiapkan aksi untuk mengambil ringkasan dashboard dan metrik keuangan workspace Anda. Silakan jalankan aksi di bawah."
		return CopilotResponse{
			Reply:        replyText,
			Actions:      actions,
			ProviderUsed: "offline",
		}
	}

	// 4. Company profile
	if strings.Contains(normalized, "ubah nama") || strings.Contains(normalized, "nama perusahaan") || strings.Contains(normalized, "profil") {
		companyName := extractAfterPatterns(message, "nama perusahaan jadi ", "nama perusahaan menjadi ", "nama perusahaan ", "ubah nama jadi ", "ubah nama menjadi ", "ubah nama ")
		companyName = cleanVendorName(companyName, "", "")
		if !isGroundedCompanyName(companyName, message) {
			return CopilotResponse{
				Reply:        "Untuk memperbarui profil perusahaan, saya membutuhkan nama legal baru perusahaan yang jelas. Contoh: 'ubah nama perusahaan jadi PT Sukses Sejahtera'. Nama perusahaan baru apa yang ingin digunakan?",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}
		actions = append(actions, CopilotAction{
			ID:                       fmt.Sprintf("rule-profile-%d", time.Now().UnixMilli()),
			ToolName:                 "update_company_profile",
			Title:                    "Pembaruan Profil Perusahaan",
			Description:              fmt.Sprintf("Memperbarui nama profil perusahaan menjadi '%s'.", companyName),
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     []ActionDiff{{Field: "company_name", OldValue: "(Profil Saat Ini)", NewValue: companyName}},
			Payload:                  map[string]any{"company_name": companyName},
			Status:                   "pending",
		})
		replyText := fmt.Sprintf("Saya mendeteksi permintaan pengaturan profil perusahaan. Nama baru: '%s'. Karena ini perubahan profil berisiko tinggi, ketik KONFIRMASI pada kartu aksi untuk melanjutkan.", companyName)
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// 5. Team invite
	if strings.Contains(normalized, "undang") || strings.Contains(normalized, "tambah anggota") || strings.Contains(normalized, "invite") {
		email := extractEmailRegex(message)
		role := extractRole(message)
		if role == "" {
			role = "member"
		}

		if email == "" {
			return CopilotResponse{
				Reply:        "Untuk mengundang anggota tim baru, mohon sertakan alamat email yang valid (contoh: 'undang anggota tim email budi@test.com sebagai finance').",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}

		payload := map[string]any{"email": email, "role": role}
		diffs := []ActionDiff{
			{Field: "email", OldValue: "-", NewValue: email},
			{Field: "role", OldValue: "-", NewValue: role},
		}
		desc := fmt.Sprintf("Mengundang %s sebagai %s.", email, role)
		actions = append(actions, CopilotAction{
			ID:                       fmt.Sprintf("rule-team-%d", time.Now().UnixMilli()),
			ToolName:                 "invite_team_member",
			Title:                    "Undang Anggota Tim",
			Description:              desc,
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     diffs,
			Payload:                  payload,
			Status:                   "pending",
		})
		replyText := fmt.Sprintf("Undangan untuk %s (peran: %s) telah disiapkan. Karena ini perubahan hak akses workspace, ketik KONFIRMASI untuk melanjutkan.", email, role)
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// 6. Payment gateway
	if strings.Contains(normalized, "gateway") || strings.Contains(normalized, "pakasir") || strings.Contains(normalized, "midtrans") || strings.Contains(normalized, "setup payment gateway") {
		if !strings.Contains(normalized, "midtrans") && !strings.Contains(normalized, "pakasir") {
			return CopilotResponse{
				Reply:        "Mohon tentukan penyedia payment gateway yang ingin dikonfigurasi (pilihan: Pakasir atau Midtrans). Contoh: 'setup payment gateway midtrans'.",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}
		provider := "pakasir"
		if strings.Contains(normalized, "midtrans") {
			provider = "midtrans"
		}
		actions = append(actions, CopilotAction{
			ID:                       fmt.Sprintf("rule-gw-%d", time.Now().UnixMilli()),
			ToolName:                 "configure_payment_gateway",
			Title:                    fmt.Sprintf("Konfigurasi Gateway: %s", strings.ToUpper(provider)),
			Description:              fmt.Sprintf("Mengatur penyedia %s sebagai gateway pembayaran aktif.", provider),
			RiskLevel:                "high",
			RequiresConfirmationText: "KONFIRMASI",
			Diff:                     []ActionDiff{{Field: "provider", OldValue: "-", NewValue: provider}},
			Payload:                  map[string]any{"provider": provider, "is_active": true},
			Status:                   "pending",
		})
		replyText := fmt.Sprintf("Saya telah menyiapkan proposal konfigurasi %s. Karena ini pengaturan finansial berisiko tinggi, konfirmasi teks wajib dilakukan.", strings.ToUpper(provider))
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// 7. Master Data: Vendor, Customer, Product
	// Never trigger master data creation if transactional words are present.
	hasTransactionalWords := strings.Contains(normalized, "invoice") ||
		strings.Contains(normalized, "tagihan") ||
		strings.Contains(normalized, "faktur") ||
		poWordRegex.MatchString(normalized) ||
		strings.Contains(normalized, "purchase order") ||
		strings.Contains(normalized, "bayar") ||
		strings.Contains(normalized, "pembayaran")

	// Master Data: Vendor
	isVendorCreation := !hasTransactionalWords && !interrogativeRegex.MatchString(normalized) && (
		strings.Contains(normalized, "tambah vendor") ||
			strings.Contains(normalized, "tambahkan vendor") ||
			strings.Contains(normalized, "daftar vendor") ||
			strings.Contains(normalized, "daftarkan vendor") ||
			strings.Contains(normalized, "registrasi vendor") ||
			strings.Contains(normalized, "vendor baru") ||
			strings.Contains(normalized, "buat vendor") ||
			strings.Contains(normalized, "buatkan vendor") ||
			strings.Contains(normalized, "rekanan baru") ||
			strings.Contains(normalized, "mitra baru") ||
			strings.Contains(normalized, "tambah pemasok") ||
			strings.Contains(normalized, "pemasok baru") ||
			strings.Contains(normalized, "daftar pemasok") ||
			strings.Contains(normalized, "daftarkan pemasok") ||
			strings.Contains(normalized, "tambah supplier") ||
			strings.Contains(normalized, "supplier baru") ||
			strings.Contains(normalized, "daftar supplier") ||
			strings.HasPrefix(normalized, "vendor baru") ||
			strings.HasPrefix(normalized, "vendor "))

	if isVendorCreation {
		vendorEmail := extractEmailRegex(message)
		vendorPhone := extractPhone(message)
		address := extractAddress(message)

		vendorName := extractAfterPatterns(message,
			"vendor baru bernama ", "vendor bernama ", "pemasok baru bernama ", "pemasok bernama ",
			"supplier baru bernama ", "supplier bernama ",
			"tambah vendor baru dengan nama ", "tambah vendor dengan nama ",
			"vendor baru dengan nama ", "vendor dengan nama ",
			"dengan nama ", "bernama ", "nama: ", "nama ",
			"tambah vendor baru bernama ", "tambah vendor baru ", "tambah pemasok baru ", "tambah supplier baru ",
			"daftarkan vendor baru ", "daftar vendor baru ", "buat vendor baru ", "buatkan vendor baru ",
			"vendor baru ", "pemasok baru ", "supplier baru ",
			"tambah vendor ", "daftarkan vendor ", "daftar vendor ", "buat vendor ", "buatkan vendor ",
			"vendor ", "pemasok ", "supplier ",
		)
		vendorName = stripAddressFromName(vendorName, address)
		vendorName = cleanVendorName(vendorName, vendorEmail, vendorPhone)
		vendorName = regexp.MustCompile(`(?i)^(?:dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*`).ReplaceAllString(vendorName, "")
		vendorName = strings.Trim(vendorName, " \t\r\n,;:-")

		if !isGroundedName(vendorName, message) {
			return CopilotResponse{
				Reply:        "Saya belum menemukan nama vendor pada pesan Anda. Mohon sebutkan nama vendor yang ingin didaftarkan (contoh: 'tambah vendor baru bernama PT Sinar Terang Sejati').",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}

		payload := map[string]any{"name": vendorName}
		diffs := []ActionDiff{{Field: "name", OldValue: "-", NewValue: vendorName}}
		if vendorEmail != "" {
			payload["email"] = vendorEmail
			diffs = append(diffs, ActionDiff{Field: "email", OldValue: "-", NewValue: vendorEmail})
		}
		if vendorPhone != "" {
			payload["phone"] = vendorPhone
			diffs = append(diffs, ActionDiff{Field: "phone", OldValue: "-", NewValue: vendorPhone})
		}
		if address != "" {
			payload["address"] = address
			diffs = append(diffs, ActionDiff{Field: "address", OldValue: "-", NewValue: address})
		}
		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-vendor-%d", time.Now().UnixMilli()),
			ToolName:    "create_vendor",
			Title:       "Pendaftaran Rekanan Vendor Baru",
			Description: fmt.Sprintf("Mendaftarkan mitra vendor baru '%s' ke dalam direktori rekanan bisnis.", vendorName),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     payload,
			Status:      "pending",
		})
		replyText := fmt.Sprintf("Proposal penambahan vendor '%s' disiapkan. Tinjau detail pada kartu aksi di bawah.", vendorName)
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// Master Data: Customer
	isCustomerCreation := !hasTransactionalWords && !interrogativeRegex.MatchString(normalized) && (
		strings.Contains(normalized, "tambah customer") ||
			strings.Contains(normalized, "tambahkan customer") ||
			strings.Contains(normalized, "customer baru") ||
			strings.Contains(normalized, "daftar customer") ||
			strings.Contains(normalized, "daftarkan customer") ||
			strings.Contains(normalized, "buat customer") ||
			strings.Contains(normalized, "buatkan customer") ||
			strings.Contains(normalized, "tambah pelanggan") ||
			strings.Contains(normalized, "tambahkan pelanggan") ||
			strings.Contains(normalized, "pelanggan baru") ||
			strings.Contains(normalized, "daftar pelanggan") ||
			strings.Contains(normalized, "daftarkan pelanggan") ||
			strings.Contains(normalized, "buat pelanggan") ||
			strings.Contains(normalized, "tambah klien") ||
			strings.Contains(normalized, "klien baru") ||
			strings.Contains(normalized, "daftar klien") ||
			strings.Contains(normalized, "buat klien") ||
			strings.HasPrefix(normalized, "customer baru") ||
			strings.HasPrefix(normalized, "customer "))

	if isCustomerCreation {
		custEmail := extractEmailRegex(message)
		custPhone := extractPhone(message)
		address := extractAddress(message)

		custName := extractAfterPatterns(message,
			"customer baru bernama ", "customer bernama ", "pelanggan baru bernama ", "pelanggan bernama ",
			"klien baru bernama ", "klien bernama ",
			"tambah customer baru dengan nama ", "tambah customer dengan nama ",
			"customer baru dengan nama ", "customer dengan nama ",
			"dengan nama ", "bernama ", "nama: ", "nama ",
			"tambah customer baru bernama ", "tambah customer baru ", "tambah pelanggan baru ", "tambah klien baru ",
			"daftarkan customer baru ", "daftar customer baru ", "buat customer baru ", "buatkan customer baru ",
			"customer baru ", "pelanggan baru ", "klien baru ",
			"tambah customer ", "daftarkan customer ", "daftar customer ", "tambah pelanggan ", "daftar pelanggan ",
			"buat customer ", "buatkan customer ",
			"customer ", "pelanggan ", "klien ",
		)
		custName = stripAddressFromName(custName, address)
		custName = cleanVendorName(custName, custEmail, custPhone)
		custName = regexp.MustCompile(`(?i)^(?:dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*`).ReplaceAllString(custName, "")
		custName = strings.Trim(custName, " \t\r\n,;:-")

		if !isGroundedName(custName, message) {
			return CopilotResponse{
				Reply:        "Saya belum menemukan nama pelanggan pada pesan Anda. Mohon sebutkan nama pelanggan yang ingin didaftarkan (contoh: 'tambah customer baru bernama PT Mitra Sejahtera').",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}

		payload := map[string]any{"name": custName}
		diffs := []ActionDiff{{Field: "name", OldValue: "-", NewValue: custName}}
		if custEmail != "" {
			payload["email"] = custEmail
			diffs = append(diffs, ActionDiff{Field: "email", OldValue: "-", NewValue: custEmail})
		}
		if custPhone != "" {
			payload["phone"] = custPhone
			diffs = append(diffs, ActionDiff{Field: "phone", OldValue: "-", NewValue: custPhone})
		}
		if address != "" {
			payload["address"] = address
			diffs = append(diffs, ActionDiff{Field: "address", OldValue: "-", NewValue: address})
		}
		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-cust-%d", time.Now().UnixMilli()),
			ToolName:    "create_customer",
			Title:       "Pendaftaran Pelanggan Baru",
			Description: fmt.Sprintf("Mendaftarkan pelanggan baru '%s' ke dalam sistem CRM dan penjualan.", custName),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     payload,
			Status:      "pending",
		})
		replyText := fmt.Sprintf("Proposal penambahan pelanggan '%s' disiapkan. Tinjau detail pada kartu aksi di bawah.", custName)
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// Master Data: Product
	isProductCreation := strings.Contains(normalized, "tambah produk") ||
		strings.Contains(normalized, "tambahkan produk") ||
		strings.Contains(normalized, "produk baru") ||
		strings.Contains(normalized, "buat produk") ||
		strings.Contains(normalized, "buatkan produk") ||
		strings.Contains(normalized, "tambah barang") ||
		strings.Contains(normalized, "tambahkan barang") ||
		strings.Contains(normalized, "barang baru") ||
		strings.Contains(normalized, "buat barang") ||
		strings.Contains(normalized, "daftarkan barang") ||
		strings.Contains(normalized, "daftar barang") ||
		strings.Contains(normalized, "tambah item") ||
		strings.Contains(normalized, "item baru") ||
		strings.Contains(normalized, "buat item") ||
		strings.Contains(normalized, "tambah sku") ||
		strings.Contains(normalized, "tambahkan sku") ||
		strings.Contains(normalized, "sku baru")

	if isProductCreation {
		prodSKU := extractSKU(message)
		prodPrice := extractPrice(message)
		prodName := extractAfterPatterns(message,
			"produk baru bernama ", "produk bernama ", "barang baru bernama ", "barang bernama ",
			"tambah produk baru bernama ", "tambah produk baru ", "tambah barang baru ", "tambah item baru ",
			"daftarkan produk baru ", "daftar produk baru ", "buat produk baru ", "buatkan produk baru ",
			"produk baru ", "barang baru ", "item baru ",
			"tambah produk ", "tambah barang ", "tambah item ",
			"produk ", "barang ", "item ",
		)
		prodName = cleanProductName(prodName, prodSKU)
		prodName = regexp.MustCompile(`(?i)^(?:dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*`).ReplaceAllString(prodName, "")
		prodName = strings.Trim(prodName, " \t\r\n,;:-")

		if !isGroundedProductName(prodName, message, prodPrice) {
			return CopilotResponse{
				Reply:        "Untuk menambahkan produk baru, mohon sertakan nama dan SKU atau harga produk. Contoh: 'buat produk baru bernama Laptop Asus kode ASUS-01 harga 15.000.000'.",
				Actions:      []CopilotAction{},
				ProviderUsed: "offline",
			}
		}

		if prodSKU == "" {
			prodSKU = fmt.Sprintf("PRD-%d", time.Now().Unix()%10000)
		}

		payload := map[string]any{"sku": prodSKU, "name": prodName, "price": prodPrice}
		diffs := []ActionDiff{
			{Field: "sku", OldValue: "-", NewValue: prodSKU},
			{Field: "name", OldValue: "-", NewValue: prodName},
		}
		if prodPrice > 0 {
			diffs = append(diffs, ActionDiff{Field: "price", OldValue: "-", NewValue: fmt.Sprintf("Rp %s", formatPrice(prodPrice))})
		}
		actions = append(actions, CopilotAction{
			ID:          fmt.Sprintf("rule-prod-%d", time.Now().UnixMilli()),
			ToolName:    "create_product",
			Title:       "Penambahan Produk Baru",
			Description: fmt.Sprintf("Membuat SKU produk baru '%s' (%s) di katalog barang.", prodName, prodSKU),
			RiskLevel:   "medium",
			Diff:        diffs,
			Payload:     payload,
			Status:      "pending",
		})
		replyText := fmt.Sprintf("Proposal pembuatan produk '%s' (SKU: %s) telah disiapkan. Tinjau detail pada kartu aksi di bawah.", prodName, prodSKU)
		finalActions := filterAdvisoryActions(actions, autonomyLevel)
		if strings.ToLower(strings.TrimSpace(autonomyLevel)) == "advisory" {
			replyText += "\n\n*(Mode Otonomi: Advisory - Tindakan mutasi data dibatasi dalam mode Read-Only. Proposal aksi ditiadakan.)*"
		}
		return CopilotResponse{
			Reply:        replyText,
			Actions:      finalActions,
			ProviderUsed: "offline",
		}
	}

	// Default
	return CopilotResponse{
		Reply:        "Saya siap membantu Anda di workspace ini. Anda dapat meminta saya untuk mengelola profil perusahaan, mengundang rekan tim, mendaftarkan vendor atau pelanggan baru, hingga membuka menu transaksi. Apa yang ingin Anda kerjakan saat ini?",
		Actions:      []CopilotAction{},
		ProviderUsed: "offline",
	}
}
