import { AxiosError } from 'axios'

/**
 * Standard API error envelope dari backend Go
 * (backend/go-core/internal/handler/helpers.go):
 *   { "error": { "code": "Bad Request", "message": "email sudah terdaftar" } }
 * Beberapa fallback proxy Next.js mengembalikan string biasa:
 *   { "error": "backend unreachable" }
 */
interface ApiErrorEnvelope {
  error?: string | { code?: string; message?: string; details?: unknown }
  message?: string
}

/**
 * Ekstrak pesan error yang bisa dibaca manusia dari respons fetch /api/v1/*.
 * Menangani envelope objek backend Go (error.message), string biasa, dan
 * body non-JSON (mis. halaman HTML dari proxy) — tanpa pernah melempar.
 */
export async function extractApiErrorMessage(
  res: Response,
  fallback: string
): Promise<string> {
  try {
    const body = (await res.json()) as ApiErrorEnvelope
    if (typeof body.error === 'string' && body.error.trim()) {
      return body.error
    }
    if (body.error && typeof body.error === 'object') {
      const msg = body.error.message
      if (typeof msg === 'string' && msg.trim()) {
        return msg
      }
    }
    if (typeof body.message === 'string' && body.message.trim()) {
      return body.message
    }
  } catch {
    // Body bukan JSON — jatuh ke pesan fallback di bawah.
  }
  return fallback
}

export function getClientErrorMessage(error: unknown): string {
  if (error instanceof AxiosError) {
    switch (error.response?.status) {
      case 401: return 'Sesi tidak valid. Masukkan token yang benar.'
      case 403: return 'Anda tidak punya akses ke data ini.'
      case 404: return 'Data tidak ditemukan.'
      case 422: return 'Data tidak valid.'
      case 500: return 'Terjadi kesalahan pada server. Coba lagi nanti.'
      default: return 'Gagal memuat data. Periksa koneksi Anda.'
    }
  }
  return 'Terjadi kesalahan yang tidak diketahui.'
}
