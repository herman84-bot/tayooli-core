"use client"

import { useEffect } from "react"

// Last-resort boundary: replaces the root layout, so it must render its own
// <html>/<body> and cannot rely on globals.css. Inline styles only.
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error("Global error boundary:", error)
  }, [error])

  return (
    <html lang="id">
      <body style={{ margin: 0, fontFamily: "system-ui, sans-serif", background: "#F8FAFC" }}>
        <div
          role="alert"
          style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", padding: 16 }}
        >
          <div
            style={{
              maxWidth: 420,
              width: "100%",
              background: "#fff",
              border: "1px solid #FECACA",
              borderRadius: 12,
              padding: 32,
              textAlign: "center",
            }}
          >
            <h1 style={{ fontSize: 18, fontWeight: 700, color: "#0F172A", margin: 0 }}>Aplikasi mengalami kendala</h1>
            <p style={{ fontSize: 14, color: "#64748B", marginTop: 8 }}>
              Terjadi kesalahan tak terduga. Data Anda aman. Silakan muat ulang halaman.
            </p>
            {error.digest && (
              <p style={{ fontSize: 12, color: "#94A3B8", fontFamily: "monospace" }}>Kode referensi: {error.digest}</p>
            )}
            <div style={{ display: "flex", gap: 8, justifyContent: "center", marginTop: 24, flexWrap: "wrap" }}>
              <button
                type="button"
                onClick={() => reset()}
                style={{
                  padding: "10px 16px",
                  minHeight: 44,
                  borderRadius: 8,
                  border: "none",
                  background: "#2563EB",
                  color: "#fff",
                  fontWeight: 600,
                  cursor: "pointer",
                }}
              >
                Coba Lagi
              </button>
              <a
                href="/dashboard"
                style={{
                  padding: "10px 16px",
                  minHeight: 44,
                  boxSizing: "border-box",
                  borderRadius: 8,
                  border: "1px solid #E2E8F0",
                  color: "#334155",
                  fontWeight: 600,
                  textDecoration: "none",
                  display: "inline-flex",
                  alignItems: "center",
                }}
              >
                Kembali ke Dasbor
              </a>
            </div>
          </div>
        </div>
      </body>
    </html>
  )
}
