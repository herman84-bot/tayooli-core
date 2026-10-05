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

const nonBlank = (v: unknown): v is string => typeof v === 'string' && v.trim() !== ''

/**
 * Ubah payload error API apa pun menjadi string yang aman ditampilkan.
 * Tidak pernah menghasilkan "[object Object]": nilai non-string diabaikan
 * dan jatuh ke `fallback`.
 */
export function extractErrorMessage(
  payload: unknown,
  fallback = 'Terjadi kesalahan pada sistem.'
): string {
  if (nonBlank(payload)) return payload
  if (!payload || typeof payload !== 'object') return fallback

  const data = payload as Record<string, unknown>
  const err = data.error
  if (err && typeof err === 'object' && !Array.isArray(err)) {
    const e = err as Record<string, unknown>
    if (nonBlank(e.message)) return e.message
    if (nonBlank(e.code)) return e.code
  }
  if (nonBlank(err)) return err
  if (nonBlank(data.message)) return data.message

  const errors = data.errors
  if (errors && typeof errors === 'object') {
    const parts = (Array.isArray(errors) ? errors : Object.values(errors))
      .flat()
      .map((e) => (nonBlank(e) ? e : e && typeof e === 'object' && nonBlank((e as { message?: unknown }).message) ? (e as { message: string }).message : ''))
      .filter(Boolean)
    if (parts.length) return parts.join(', ')
  }
  return fallback
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
    return extractErrorMessage((await res.json()) as ApiErrorEnvelope, fallback)
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
