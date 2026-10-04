# PRD: AI Customer Service — Tayooli ERP

## Problem

Tayooli ERP belum memiliki customer service yang responsif. User baru membutuhkan bantuan saat:
- Setup awal (cara membuat invoice, approve, bayar)
- Troubleshooting (kenapa invoice tidak muncul, kenapa approval stuck)
- Pertanyaan pricing/subscription
- Pertanyaan fitur

Saat ini semua pertanyaan harus dijawab manual oleh tim. Ketika user bertumbuh, ini menjadi bottleneck.

## Goal

Bangun AI customer service yang:
1. **Terlihat natural** — user tidak sadar ini AI
2. **Menguasai produk Tayooli** — bisa jawab pertanyaan spesifik tentang fitur
3. **Tahu kapan harus eskalasi** — jika tidak bisa jawab, forward ke human
4. **Gratis untuk mulai** — pakai Groq API (free tier)

## Scope

### IN
- Groq API client dengan auto-model fallback
- Chat backend (Go endpoints)
- Chat widget UI (floating button di frontend)
- Knowledge base produk Tayooli (hardcoded, bukan vector DB)
- Conversation history (in-memory per session)
- Escalation ke human agent

### OUT
- Voice/phone support
- Multi-language (untuk sekarang hanya Bahasa Indonesia)
- Video support
- Screen sharing
- Ticketing system
- Payment via chat
- Authentication admin dashboard untuk manage chat

## Tech Requirements

### Backend (Go)

```
Endpoint:
POST /api/v1/chat/message     — kirim pesan, dapat respons AI
GET  /api/v1/chat/history     — ambil riwayat percakapan
POST /api/v1/chat/escalate    — eskalasi ke human agent
GET  /api/v1/chat/health      — cek status Groq API

Request:
{
  "message": "Cara buat invoice baru?",
  "session_id": "optional-session-id"
}

Response:
{
  "reply": "Untuk membuat invoice baru...",
  "session_id": "abc-123",
  "escalated": false,
  "model_used": "llama-3.3-70b-versatile"
}
```

### Groq API Client

```
- Endpoint: https://api.groq.com/openai/v1/chat/completions
- Auth: Bearer token (GROQ_API_KEY env var)
- Auto-model fallback:
  1. llama-3.3-70b-versatile (primary)
  2. llama-3.1-8b-instant (fallback 1)
  3. mixtral-8x7b-32768 (fallback 2)
  
- Jika primary down → otomatis coba fallback 1
- Jika fallback 1 down → otomatis coba fallback 2
- Jika semua down → return "Layanan chat sedang tidak tersedia"
- Tidak perlu retry berulang — cukup fallback sekali
```

### Chat Widget (Frontend)

```
- Floating button di pojok kanan bawah
- Klik → buka chat panel
- Panel berisi:
  - Header: "Tayooli Support"
  - Message list (user + AI)
  - Input field + send button
  - "Hubungi Human" button (escalate)
- Responsive: mobile-friendly
- Style: minimal, professional, tidak mengganggu
```

### Knowledge Base

```
Hardcoded knowledge tentang Tayooli ERP:

1. Produk:
   - Tayooli adalah ERP B2B untuk invoice, persetujuan, pembayaran
   - Fitur: OCR invoice, approval workflow, payment orders, accounting
   - Pricing: Starter Rp199rb/bulan, Bisnis Rp449rb/bulan, Enterprise custom
   - Trial 14 hari gratis

2. Cara pakai:
   - Dashboard: ringkasan keuangan
   - Invoice: upload foto → OCR → review → approve → bayar
   - Purchase Order: buat PO → goods receipt → payment order
   - Approval: workflow berbasis role
   - Accounting: chart of accounts, journal entries

3. Troubleshooting:
   - Invoice tidak muncul: cek status di Invoice list
   - Approval stuck: cek apakah ada pending approval
   - Login gagal: cek email/password, atau reset password

4. Billing:
   - Upgrade: Billing page → pilih paket → bayar via Pakasir
   - Cancel: Billing page → cancel subscription
   - Trial: 14 hari, bisa upgrade kapan saja
```

## Architecture

```
User Browser
    │
    ▼
Chat Widget (React)
    │ POST /api/v1/chat/message
    ▼
Go Backend (Chi Router)
    │
    ▼
Groq API Client (Go)
    │ POST https://api.groq.com/openai/v1/chat/completions
    ▼
Groq Cloud (LLM)
    │
    ▼
Response → Go Backend → Chat Widget → User
```

## Security Requirements

1. **Rate limiting**: Max 30 messages per minute per IP
2. **Input sanitization**: Strip HTML/script tags dari user message
3. **Output sanitization**: AI response di-escape sebelum ditampilkan
4. **No sensitive data in prompts**: Jangan kirim password, token, atau data sensitif ke Groq
5. **Session isolation**: Setiap session hanya bisa akses riwayat sendiri
6. **API key**: GROQ_API_KEY di environment variable, tidak di kode

## Prompt Engineering

### System Prompt

```
Kamu adalah customer service Tayooli ERP. Kamu membantu user dengan pertanyaan 
tentang produk Tayooli ERP.

Karakter kamu:
- Ramah tapi profesional
- Bilingual: Bahasa Indonesia utama, Inggris jika user pakai Inggris
- Percakapan natural, tidak seperti robot
- Singkat dan to the point, tapi tetap sopan
- Gunakan "kamu" untuk user, "kami" untuk Tayooli

Kamu menguasai:
- Fitur Tayooli: invoice, approval, payment, accounting, products
- Pricing: Starter Rp199rb/bulan, Bisnis Rp449rb/bulan, Enterprise custom
- Trial: 14 hari gratis
- Troubleshooting umum

Jika pertanyaan di luar kemampuanmu:
- "Untuk pertanyaan ini, saya sarankan menghubungi tim support kami."
- Jangan mengarang jawaban
- Jangan memberikan informasi yang tidak akurat

Gaya bicara:
- Gunakan sapaan ringan: "Halo!", "Hai!"
- Gunakan emoji seperlunya (1-2 per pesan, jangan berlebihan)
- Gunakan bullet point untuk penjelasan panjang
- Konfirmasi pemahaman: "Oh, kamu maksudnya..."
- Tawarkan bantuan lanjutan: "Ada yang lain yang bisa saya bantu?"
```

### Temperature

```
temperature: 0.7 (natural tapi tidak terlalu random)
max_tokens: 1024 (cukup untuk penjelasan detail)
top_p: 0.9
```

## Conversation Memory

```
- In-memory per session (bukan database)
- Max 20 pesan terakhir per session
- Session ID dari frontend (UUID atau random string)
- Session expired setelah 30 menit idle
- Tidak ada persistent storage untuk chat history
```

## Escalation

```
Trigger eskalasi:
1. User klik "Hubungi Human"
2. AI merasa tidak bisa membantu (3x berturut-turut)
3. User marah/frustrasi (deteksi dari kata kunci)

Saat eskalasi:
- Tampilkan pesan: "Saya menghubungkan kamu dengan tim support kami."
- Simpan conversation history
- Dalam MVP: kirim notification ke admin (email/log)
- Nanti: integrate dengan ticketing system
```

## Error Handling

```
1. Groq API down:
   "Layanan chat sedang tidak tersedia. Silakan coba lagi dalam beberapa menit."

2. Rate limit:
   "Terlalu banyak pesan. Silakan tunggu sebentar."

3. Invalid input:
   "Pesan tidak valid. Silakan coba lagi."

4. Session expired:
   "Sesi telah berakhir. Memulai percakapan baru."
```

## Acceptance Criteria

1. ✅ User bisa klik chat widget dan mulai percakapan
2. ✅ AI merespon dalam < 3 detik (p95)
3. ✅ AI bisa menjawab pertanyaan tentang fitur Tayooli
4. ✅ AI bisa menjawab pertanyaan tentang pricing
5. ✅ AI bisa melakukan eskalasi ke human
6. ✅ Auto-model fallback bekerja (primary → fallback1 → fallback2)
7. ✅ Chat widget responsive di mobile
8. ✅ Tidak ada console error
9. ✅ Rate limiting aktif
10. ✅ Input/output sanitization aktif

## Performance Targets

- Response time: < 3 detik (p95)
- Throughput: 100 concurrent chats
- Memory: < 50MB untuk conversation storage
- Groq API calls: < 500ms (p95)

## Schema Changes

- Tidak ada database schema changes
- Conversation history in-memory only

## Dependencies

| Dependency | Purpose | Cost |
|---|---|---|
| Groq API | LLM inference | Free tier (14K req/day) |
| Go chi router | HTTP routing | Already in project |
| Lucide React | Chat icons | Already in project |
| Tailwind CSS | Chat styling | Already in project |

## Implementation Phases

### Phase 1: Groq API Client (Go)
- Buat `internal/infra/groq/client.go`
- Auto-model fallback logic
- Rate limiting
- Input/output sanitization

### Phase 2: Chat Backend (Go)
- POST /api/v1/chat/message
- GET /api/v1/chat/history
- POST /api/v1/chat/escalate
- Session management
- System prompt + knowledge base

### Phase 3: Chat Widget (Frontend)
- Floating chat button
- Chat panel component
- Message list + input
- "Hubungi Human" button
- Responsive design

### Phase 4: Integration + Testing
- Wire ke main.go
- End-to-end testing
- Performance testing
- Security review

## Files to Create/Modify

### Backend (Go)
| File | Aksi |
|---|---|
| `internal/infra/groq/client.go` | BARU — Groq API client |
| `internal/handler/chat_handler.go` | BARU — Chat HTTP endpoints |
| `internal/usecase/chat/chat_usecase.go` | BARU — Chat business logic |
| `cmd/api/main.go` | DIMODIFIKASI — Wire chat routes |
| `.env` / deployment | DIMODIFIKASI — Add GROQ_API_KEY |

### Frontend (Next.js)
| File | Aksi |
|---|---|
| `components/chat/ChatWidget.tsx` | BARU — Floating chat widget |
| `components/chat/ChatPanel.tsx` | BARU — Chat panel component |
| `components/chat/ChatMessage.tsx` | BARU — Message bubble component |
| `hooks/useChat.ts` | BARU — Chat React Query hook |
| `app/(app)/layout.tsx` | DIMODIFIKASI — Add ChatWidget |

## Risk

| Risk | Mitigation |
|---|---|
| Groq free tier limit | Auto-model fallback + rate limiting |
| AI gives wrong answer | Knowledge base hardcoded + escalation |
| Groq API down | Graceful degradation message |
| Cost overrun | Monitor usage, set alerts |
| Security: prompt injection | Input sanitization + system prompt guardrails |
