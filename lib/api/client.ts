import axios, { type AxiosError } from 'axios'

// Satu-satunya pintu API: proxy Next.js /api/v1/* → backend Go (lihat app/api/v1/[...path]/route.ts).
// Semua request lewat same-origin agar cookie HttpOnly (tayooli_auth) otomatis terkirim —
// auth dikelola cookie session, bukan Bearer token di memori.
const apiClient = axios.create({
  // NOTE: semua pemanggil (lib/queries/*) sudah menulis path lengkap '/api/v1/...'.
  // Jangan tambahkan baseURL '/api/v1' di sini — akan menghasilkan prefix dobel
  // (/api/v1/api/v1/...) yang membuat proxy Next.js gagal (404/502).
  timeout: 10000,
})

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // 401 dari backend diteruskan — UI menangani error ini per halaman.
    return Promise.reject(error)
  }
)

export default apiClient
