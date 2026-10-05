import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import Providers from './providers'

/*
 * Claude-web style typography: Inter is the closest freely-available match to
 * Claude's proprietary "Styrene" grotesque. Loaded via next/font (self-hosted)
 * and exposed as --font-inter for the design system's --font-sans token.
 */
const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
  weight: ['400', '500', '600', '700'],
})

export const metadata: Metadata = {
  title: 'Tayooli — Sistem Manajemen Ritel, Gudang & Kasir POS',
  description: 'Aplikasi kasir POS, multi-gudang, surat jalan, dan stok ritel dalam satu alur kerja yang terpadu.',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id" className={inter.variable}>
      <body suppressHydrationWarning>
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
