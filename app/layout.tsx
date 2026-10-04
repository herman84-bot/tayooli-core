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
  title: 'Tayooli — ERP Order-to-Pay & Order-to-Cash untuk Bisnis Indonesia',
  description: 'Invoice (OCR + AI), persetujuan, pembayaran, dan akuntansi dalam satu alur yang tenang. Gratis 14 hari, tanpa kartu kredit.',
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
