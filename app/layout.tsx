import type { Metadata } from 'next'
// Inter is bundled from npm (@fontsource-variable/inter), not fetched from
// Google Fonts at build time. next/font/google downloads font files during
// `next build`; when that fetch flakes inside the Zeabur Docker build, the
// loader crashes with "Cannot read properties of null (reading '1')" and the
// whole deploy fails. A local package makes the build hermetic.
import '@fontsource-variable/inter'
import './globals.css'
import Providers from './providers'

export const metadata: Metadata = {
  title: 'Tayooli — Sistem Manajemen Ritel, Gudang & Kasir POS',
  description: 'Aplikasi kasir POS, multi-gudang, surat jalan, dan stok ritel dalam satu alur kerja yang terpadu.',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body suppressHydrationWarning>
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
