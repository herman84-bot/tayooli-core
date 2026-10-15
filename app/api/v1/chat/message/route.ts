import { NextRequest, NextResponse } from 'next/server'
import { getCustomerServiceReply, handleAmbiguousQuery } from '@/lib/chat/knowledge-engine'

interface ChatRequestBody {
  message?: string
  session_id?: string
}

async function queryExternalAI(prompt: string): Promise<string | null> {
  const systemInstruction = `Anda adalah Customer Service resmi "Tayooli Support" untuk Tayooli ERP Core di Indonesia.
Sistem ini beroperasi dengan 12 modul inti yang bersih (clean architecture):
1. Dasbor (/dashboard): Ringkasan omzet kasir, metrik bisnis harian, total stok fisik, notifikasi stok menipis.
2. Produk (/products): Master katalog SKU unik, harga modal (HPP), harga jual, batas minimum stok, dan barcode.
3. Barang Masuk & Keluar (/wms/arus-barang): Penerimaan supplier (Inbound) dan penerbitan Surat Jalan resmi (Delivery Order/DO) dengan tanda tangan & kurir.
4. Gudang & Stok (/wms): Manajemen multi-gudang dan tata letak rak/bin penyimpanan.
5. Marketplace Omnichannel (/wms/marketplace): Integrasi 2-arah Tokopedia, Shopee, TikTok Shop, Lazada untuk pencegahan overselling.
6. Transfer Stok (/wms/transfers): Perpindahan barang antar cabang gudang dengan siklus DRAFT -> PENDING_APPROVAL -> IN_TRANSIT -> RECEIVED.
7. Stok Opname (/wms/opname): Audit fisik stok berkala dan penyesuaian varians/selisih otomatis.
8. Barang Rusak / Scrap (/wms/scrap): Pencatatan barang cacat/pecah/expired dan write-off stok agar tidak ikut terjual.
9. Pemindai Barcode (/wms/scanner): Scan barcode produk menggunakan kamera ponsel/webcam atau USB/Bluetooth barcode gun.
10. Kasir POS (/pos): Transaksi penjualan ritel cepat, QRIS dinamis, tunai dengan hitung kembalian presisi, potong stok instan, dan cetak struk thermal (58mm/80mm).
11. Pengaturan (/settings): Profil toko, PPN, format struk, dan keamanan akun.
12. Bantuan (/help): Pusat panduan operasional dan chat customer service.

Aturan Respon:
- Bersikap ramah, sopan, empatik, dan profesional layaknya Customer Service asli manusia di Indonesia (sapa dengan "Halo kak!", sapaan hangat, dan bahasa yang mengayomi).
- Jawab tepat sasaran sesuai pertanyaan pengguna. Jika pengguna bertanya tentang modul tertentu, berikan panduan langkah demi langkah yang terstruktur.
- Jika pengguna bertanya tentang cetak, bedakan dengan jelas antara Cetak Struk Kasir di POS dan Cetak Surat Jalan di Arus Barang/DO.
- Setiap mutasi stok barang masuk atau keluar wajib menggunakan nomor batch (batch_id) untuk ketertelusuran produk (KO-1).
- Jangan menampilkan karakter markdown rusak atau syntax tag mentah.`

  // 1. Coba Gemini API jika GEMINI_API_KEY tersedia
  const geminiKey = process.env.GEMINI_API_KEY
  if (geminiKey) {
    const candidateModels = ['gemini-2.0-flash', 'gemini-1.5-flash']
    for (const model of candidateModels) {
      try {
        const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${geminiKey}`
        const res = await fetch(endpoint, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            systemInstruction: {
              parts: [{ text: systemInstruction }],
            },
            contents: [
              {
                role: 'user',
                parts: [{ text: prompt }],
              },
            ],
          }),
          signal: AbortSignal.timeout(3500),
        })

        if (res.status === 429) {
          // Quota exhausted for this API key, skip remaining Gemini models
          break
        }

        if (res.ok) {
          const data = await res.json()
          const text = data?.candidates?.[0]?.content?.parts?.[0]?.text
          if (text && typeof text === 'string' && text.trim().length > 0) {
            return text.trim()
          }
        }
      } catch {
        // Coba model berikutnya atau fallback ke local engine
      }
    }
  }

  // 2. Coba Groq API jika GROQ_API_KEY tersedia
  const groqKey = process.env.GROQ_API_KEY
  if (groqKey) {
    try {
      const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${groqKey}`,
        },
        body: JSON.stringify({
          model: 'llama-3.3-70b-versatile',
          messages: [
            { role: 'system', content: systemInstruction },
            { role: 'user', content: prompt },
          ],
          temperature: 0.3,
        }),
        signal: AbortSignal.timeout(3500),
      })

      if (res.ok) {
        const data = await res.json()
        const text = data?.choices?.[0]?.message?.content
        if (text && typeof text === 'string' && text.trim().length > 0) {
          return text.trim()
        }
      }
    } catch {
      // Fallback ke local engine
    }
  }

  return null
}

export async function POST(req: NextRequest) {
  try {
    const body: ChatRequestBody = await req.json().catch(() => ({}))
    const userMessage = (body.message ?? '').trim().slice(0, 1000)
    const sessionId = body.session_id || `session_${Date.now()}`

    if (!userMessage) {
      return NextResponse.json({ error: 'Pesan tidak boleh kosong' }, { status: 400 })
    }

    // 1. Cek penanganan query ambigu deterministik terlebih dahulu
    const ambiguousReply = handleAmbiguousQuery(userMessage)
    if (ambiguousReply) {
      return NextResponse.json({
        reply: ambiguousReply,
        session_id: sessionId,
      })
    }

    // 2. Coba external AI jika API key tersedia
    const aiReply = await queryExternalAI(userMessage)
    if (aiReply) {
      return NextResponse.json({
        reply: aiReply,
        session_id: sessionId,
      })
    }

    // 3. Gunakan built-in weighted knowledge & intent engine Tayooli ERP
    const reply = getCustomerServiceReply(userMessage)

    return NextResponse.json({
      reply,
      session_id: sessionId,
    })
  } catch (err) {
    console.error('Tayooli support chat error:', err)
    return NextResponse.json({
      reply: 'Halo kak, layanan Customer Service Tayooli siap membantu. Silakan sampaikan pertanyaan Anda kembali ya.',
      session_id: `session_${Date.now()}`,
    })
  }
}
