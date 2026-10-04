package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// TelegramBotDeps holds dependencies for the Telegram bot.
type TelegramBotDeps struct {
	TelegramClient BotTelegramSender
	PakasirClient  PakasirTransactionCreator
	SubUsecase     SubscriptionUsecase
	PaymentRepo    PaymentOrderRepo
}

// BotTelegramSender sends messages via Telegram for bot commands.
type BotTelegramSender interface {
	SendMessageToChat(chatID int64, text string) error
	SendPhotoToChat(chatID int64, photoURL, caption string) error
}

// PakasirTransactionCreator creates Pakasir transactions.
type PakasirTransactionCreator interface {
	CreateTransaction(ctx context.Context, project, orderID, apiKey, method string, amount int) (*PakasirTxResult, error)
}

// PakasirTxResult is the result of creating a Pakasir transaction.
type PakasirTxResult struct {
	PaymentURL string
	OrderID    string
	Amount     int
	ExpiredAt  string
}

// PaymentOrderRepo manages payment orders in the database.
type PaymentOrderRepo interface {
	CreateOrder(ctx context.Context, orderCode, tenantID, userEmail, plan string, amount int, period string) error
	GetOrderByCode(ctx context.Context, orderCode string) (*domain.TelegramPaymentOrder, error)
	UpdateStatus(ctx context.Context, orderCode, status string) error
	GetPendingOrderByCode(ctx context.Context, orderCode string) (*domain.TelegramPaymentOrder, error)
}

// TelegramUpdate represents an incoming Telegram update.
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

// TelegramBotHandler handles incoming Telegram updates.
type TelegramBotHandler struct {
	deps TelegramBotDeps
}

func NewTelegramBotHandler(deps TelegramBotDeps) *TelegramBotHandler {
	return &TelegramBotHandler{deps: deps}
}

// RegisterRoutes registers the Telegram webhook endpoint.
func (h *TelegramBotHandler) RegisterRoutes(r chi.Router) {
	r.Post("/webhooks/telegram", h.HandleWebhook)
}

// HandleWebhook processes incoming Telegram updates.
func (h *TelegramBotHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 1. Verify Telegram Webhook Secret Token if configured
	secretToken := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	if secretToken != "" {
		reqToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if reqToken != secretToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var update TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if update.Message == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	chatID := int64(update.Message.Chat.ID)
	userID := update.Message.From.ID
	text := strings.TrimSpace(update.Message.Text)
	userEmail := update.Message.From.Username

	switch {
	case text == "/start":
		h.handleStart(chatID, update.Message.From.FirstName)
	case text == "/bayar":
		h.handleBayar(chatID)
	case text == "/status":
		h.handleStatus(chatID, userEmail)
	case strings.HasPrefix(text, "/beli_"):
		plan := strings.TrimPrefix(text, "/beli_")
		h.handleBeli(chatID, userEmail, plan)
	case strings.HasPrefix(text, "/approve_"):
		orderCode := strings.TrimPrefix(text, "/approve_")
		h.handleApprove(chatID, userID, orderCode)
	default:
		// Ignore other messages
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TelegramBotHandler) handleStart(chatID int64, firstName string) {
	name := firstName
	if name == "" {
		name = "Kamu"
	}
	msg := fmt.Sprintf(`Halo %s! 👋

Selamat datang di <b>Tayooli Support Bot</b>.

Ketik perintah berikut:

/bayar — Beli paket langganan
/status — Cek status subscription

Paket yang tersedia:
• Starter — Rp 199.000/bulan
• Bisnis — Rp 449.000/bulan
• Enterprise — Custom

Ada yang bisa saya bantu?`, name)

	h.deps.TelegramClient.SendMessageToChat(chatID, msg)
}

func (h *TelegramBotHandler) handleBayar(chatID int64) {
	msg := `💳 <b>Pilih Paket Langganan</b>

/beli_starter — Rp 199.000/bulan
/beli_bisnis — Rp 449.000/bulan

Ketik salah satu perintah di atas untuk melanjutkan.`

	h.deps.TelegramClient.SendMessageToChat(chatID, msg)
}

func (h *TelegramBotHandler) handleStatus(chatID int64, userEmail string) {
	msg := `📋 <b>Status Subscription</b>

Untuk mengecek status, silakan login ke dashboard:
https://tayooli.id/billing`

	h.deps.TelegramClient.SendMessageToChat(chatID, msg)
}

func (h *TelegramBotHandler) handleBeli(chatID int64, userEmail string, plan string) {
	ctx := context.Background()

	// Map plan to amount
	var amount int
	var planName string
	switch plan {
	case "starter":
		amount = 199000
		planName = "Starter"
	case "bisnis":
		amount = 449000
		planName = "Bisnis"
	default:
		h.deps.TelegramClient.SendMessageToChat(chatID, "Paket tidak dikenali. Gunakan /beli_starter atau /beli_bisnis")
		return
	}

	// Generate order code
	orderCode := fmt.Sprintf("TAY-%s", strconv.FormatInt(int64(chatID), 10))

	// Create payment order in database
	pakasirProject := "tayooli"
	pakasirAPIKey := "" // will be set from env

	// Create Pakasir transaction
	pakasirOrderID := orderCode
	txResult, err := h.deps.PakasirClient.CreateTransaction(ctx, pakasirProject, pakasirOrderID, pakasirAPIKey, "qris", amount)
	if err != nil {
		// Fallback: send manual QRIS info
		msg := fmt.Sprintf(`💳 <b>Pembayaran %s</b>

<b>Nominal:</b> Rp %s
<b>Kode:</b> %s

Scan QRIS dan masukkan nominal:
<b>Rp %s</b>

Kirim bukti bayar ke sini setelah pembayaran.`, planName, formatIDR(amount), orderCode, formatIDR(amount))
		h.deps.TelegramClient.SendMessageToChat(chatID, msg)
		return
	}

	// Save order to database
	h.deps.PaymentRepo.CreateOrder(ctx, orderCode, uuid.Nil.String(), userEmail, plan, amount, "monthly")

	// Send QRIS payment info
	msg := fmt.Sprintf(`💳 <b>Pembayaran %s</b>

<b>Nominal:</b> Rp %s
<b>Kode:</b> %s
<b>Expired:</b> %s

Klik link untuk bayar:
%s

Setelah bayar, kirim bukti ke sini.`, planName, formatIDR(amount), orderCode, txResult.ExpiredAt, txResult.PaymentURL)

	h.deps.TelegramClient.SendMessageToChat(chatID, msg)
}

func (h *TelegramBotHandler) isTelegramAdmin(userID int) bool {
	adminEnv := os.Getenv("TELEGRAM_ADMIN_USER_IDS")
	if adminEnv == "" {
		return false
	}
	parts := strings.Split(adminEnv, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.Atoi(p); err == nil && id == userID {
			return true
		}
	}
	return false
}

func (h *TelegramBotHandler) handleApprove(chatID int64, userID int, orderCode string) {
	// RBAC Check: Ensure the user requesting approval is a verified admin
	if !h.isTelegramAdmin(userID) {
		h.deps.TelegramClient.SendMessageToChat(chatID, "⛔ <b>Akses ditolak:</b> Hanya administrator yang diizinkan untuk menyetujui pembayaran.")
		return
	}

	ctx := context.Background()

	// Get order from database
	order, err := h.deps.PaymentRepo.GetPendingOrderByCode(ctx, orderCode)
	if err != nil {
		h.deps.TelegramClient.SendMessageToChat(chatID, fmt.Sprintf("Order %s tidak ditemukan atau sudah diproses.", orderCode))
		return
	}

	// Activate subscription
	plan := domain.SubscriptionPlan(order.Plan)
	period := domain.BillingPeriodMonthly
	_, err = h.deps.SubUsecase.Create(ctx, order.TenantID, plan, period)
	if err != nil {
		h.deps.TelegramClient.SendMessageToChat(chatID, fmt.Sprintf("Gagal aktifkan subscription: %s", err.Error()))
		return
	}

	// Update order status
	h.deps.PaymentRepo.UpdateStatus(ctx, orderCode, "approved")

	msg := fmt.Sprintf(`✅ <b>PEMBAYARAN DITERIMA</b>

Order: %s
Paket: %s
User: %s

Subscription sudah aktif! 🎉`, orderCode, order.Plan, order.UserEmail)

	h.deps.TelegramClient.SendMessageToChat(chatID, msg)
}

func formatIDR(amount int) string {
	s := strconv.Itoa(amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}
