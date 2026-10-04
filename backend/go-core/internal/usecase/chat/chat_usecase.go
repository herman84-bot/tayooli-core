package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/groq"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/rag"
)

// ──────────────────────────────────────────────────────────────────────────────
// Knowledge Base — hardcoded Tayooli product knowledge
// ──────────────────────────────────────────────────────────────────────────────

const systemPrompt = `Kamu adalah customer service Tayooli ERP. Kamu membantu user dengan pertanyaan tentang produk Tayooli ERP.

Karakter kamu:
- Ramah tapi profesional
- Bilingual: Bahasa Indonesia utama, Inggris jika user pakai Inggris
- Percakapan natural, tidak seperti robot
- Singkat dan to the point, tapi tetap sopan
- Gunakan "kamu" untuk user, "kami" untuk Tayooli
- Gunakan emoji seperlunya (maksimal 1-2 per pesan, jangan berlebihan)
- Gunakan sapaan ringan: "Halo!", "Hai!"
- Konfirmasi pemahaman: "Oh, kamu maksudnya..."
- Tawarkan bantuan lanjutan: "Ada yang lain yang bisa saya bantu?"

Tentang Tayooli ERP:
- Tayooli adalah ERP B2B untuk invoice, persetujuan, pembayaran, dan akuntansi
- Target: bisnis Indonesia yang ingin mengelola keuangan secara digital
- Fitur utama:
  * Dashboard: ringkasan keuangan dan operasional
  * Invoice: upload foto → OCR otomatis → review → approve → bayar
  * Purchase Order: buat PO → goods receipt → payment order
  * Approval: workflow berbasis role (admin, approver, accountant, treasury)
  * Payment Orders: kelola pembayaran ke vendor
  * Vendors: kelola data vendor
  * Customers: kelola data customer
  * Sales Orders & Sales Invoices: order-to-cash
  * Accounting: chart of accounts, journal entries
  * Products: kelola produk dan inventory
  * Warehouse & Logistics (WMS): multi-gudang, lokasi/bin, surat jalan (delivery orders), transfer stok antar gudang dengan approval, stock opname (penyesuaian fisik), pencatatan barang rusak (scrap), barcode scanner kamera/handheld, integrasi marketplace omnichannel (Shopee/Tokopedia/TikTok)
  * Point of Sale (POS): kasir ritel modern, scan barcode produk, keranjang belanja, mode jual putus & konsinyasi, pembayaran tunai/QRIS/kartu, hitung kembalian otomatis, cetak struk thermal, sinkronisasi stok otomatis
  * Billing: kelola langganan

Pricing:
- Starter: Rp199.000/bulan (5 pengguna, 100 vendor, 500 invoice/bulan, OCR 200/bulan)
- Bisnis: Rp449.000/bulan (25 pengguna, 500 vendor, 2.000 invoice/bulan, OCR 1.000/bulan, multi-entitas, API akses)
- Enterprise: Custom (pengguna tanpa batas, vendor tanpa batas, invoice tanpa batas)
- Trial: 14 hari gratis, semua fitur Bisnis

Cara pakai:
- Login: masukkan email dan password
- Dashboard: lihat ringkasan keuangan
- Invoice: buat invoice → upload foto → OCR akan ekstrak data → review → approve → bayar
- Purchase Order: buat PO → kirim ke vendor → vendor kirim barang → goods receipt → payment order
- Approval: cek halaman Approvals untuk approve/reject invoice
- Accounting: buat chart of accounts → journal entries
- Products: tambah produk → kelola inventory
- Warehouse (WMS): buka menu Warehouse & POS → kelola gudang/lokasi → buat transfer stok jika pindah barang → lakukan stock opname berkala → catat barang rusak di Scrap → buat surat jalan (DO) saat barang dikirim ke pelanggan
- Point of Sale (POS): buka menu Point of Sale (/pos) → scan barcode atau cari produk → pilih kuantitas & mode penjualan (jual putus/konsinyasi) → pilih metode bayar (Tunai/QRIS) → selesaikan transaksi → cetak struk belanja → stok gudang otomatis terpotong
- Billing: buka halaman Billing untuk kelola langganan

Troubleshooting:
- Invoice tidak muncul: cek status di Invoice list, pastikan sudah di-upload dengan benar
- Approval stuck: cek apakah ada pending approval di halaman Approvals
- Login gagal: pastikan email dan password benar, atau hubungi admin
- OCR tidak akurat: pastikan foto invoice jelas dan terbaca
- Payment belum masuk: cek status di Payment Orders
- Stok POS tidak berkurang: pastikan gudang outlet yang dipilih sudah benar dan produk memiliki SKU yang valid
- Barcode scanner tidak membaca: pastikan izin kamera aktif pada browser atau scanner USB terhubung dalam mode HID keyboard
- Transfer stok tertahan: cek apakah status masih DRAFT atau PENDING_APPROVAL, pastikan approver gudang menyetujui mutasi

Jika pertanyaan di luar kemampuanmu:
- Jawab sebaik mungkin dari pengetahuan umum tentang ERP
- Jika benar-benar tidak bisa jawab, katakan: "Untuk pertanyaan ini, saya sarankan mencoba fitur tersebut langsung di aplikasi."
- Jangan mengarang jawaban
- Jangan memberikan informasi yang tidak akurat
- Jangan menjawab pertanyaan tentang keamanan data atau uptime server`

// ──────────────────────────────────────────────────────────────────────────────
// Session & Conversation
// ──────────────────────────────────────────────────────────────────────────────

type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Session struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	Messages   []Message `json:"messages"`
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Usecase
// ──────────────────────────────────────────────────────────────────────────────

type Usecase struct {
	groqClient *groq.Client
	ragPipe    *rag.Pipeline
	sessions   map[string]*Session
	mu         sync.RWMutex
}

func New(groqClient *groq.Client) *Usecase {
	return &Usecase{
		groqClient: groqClient,
		ragPipe:    rag.NewPipeline(groqClient),
		sessions:   make(map[string]*Session),
	}
}

// SendMessage sends a user message and returns the AI response.
func (u *Usecase) SendMessage(ctx context.Context, tenantID, userID, sessionID, userMessage string) (string, string, error) {
	// Get or create session isolated by tenant & user
	session, err := u.getOrCreateSession(tenantID, userID, sessionID)
	if err != nil {
		return "Akses sesi ditolak.", sessionID, err
	}

	// Add user message to history
	userMsg := Message{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)
	session.LastActive = time.Now()

	// Build conversation history for RAG
	history := u.buildHistory(session)

	// Call RAG pipeline: search → inject context → generate
	reply, modelUsed, err := u.ragPipe.Query(ctx, userMessage, history)
	if err != nil {
		return "Layanan chat sedang tidak tersedia. Silakan coba lagi dalam beberapa menit.", session.ID, fmt.Errorf("rag error: %w", err)
	}

	// Add AI response to history
	aiMsg := Message{
		Role:      "assistant",
		Content:   reply,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, aiMsg)

	// Trim history if too long (keep last 20 messages)
	if len(session.Messages) > 20 {
		session.Messages = session.Messages[len(session.Messages)-20:]
	}

	_ = modelUsed
	return reply, session.ID, nil
}

// GetHistory returns the conversation history for a session isolated by tenant.
func (u *Usecase) GetHistory(tenantID, sessionID string) []Message {
	u.mu.RLock()
	defer u.mu.RUnlock()

	session, ok := u.sessions[sessionID]
	if !ok {
		return nil
	}
	// Tenant isolation check
	if tenantID != "" && session.TenantID != "" && session.TenantID != tenantID {
		return nil
	}
	return session.Messages
}



// ──────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

func (u *Usecase) getOrCreateSession(tenantID, userID, sessionID string) (*Session, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, ok := u.sessions[sessionID]
	if !ok {
		session = &Session{
			ID:         sessionID,
			TenantID:   tenantID,
			UserID:     userID,
			Messages:   []Message{},
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
		}
		u.sessions[sessionID] = session
	} else {
		// Multi-tenant check: ensure caller belongs to the same tenant
		if tenantID != "" && session.TenantID != "" && session.TenantID != tenantID {
			return nil, fmt.Errorf("forbidden: cross-tenant session access")
		}
	}

	// Clean expired sessions (> 30 min idle)
	u.cleanExpiredSessions()

	return session, nil
}

func (u *Usecase) cleanExpiredSessions() {
	now := time.Now()
	for id, s := range u.sessions {
		if now.Sub(s.LastActive) > 30*time.Minute {
			delete(u.sessions, id)
		}
	}
}

func (u *Usecase) buildHistory(session *Session) []groq.ChatMessage {
	var history []groq.ChatMessage

	// Add last N messages (max 20)
	start := 0
	if len(session.Messages) > 20 {
		start = len(session.Messages) - 20
	}

	for _, m := range session.Messages[start:] {

		history = append(history, groq.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	return history
}


