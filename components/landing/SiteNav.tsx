"use client"

import { useEffect, useRef, useState } from "react"
import Link from "next/link"
import { ArrowRight } from "lucide-react"
import { Logo } from "@/components/brand/Logo"

const NAV_ITEMS = [
  { href: "#fitur", label: "Fitur" },
  { href: "#cara-kerja", label: "Cara Kerja" },
  { href: "#harga", label: "Harga" },
  { href: "#faq", label: "FAQ" },
] as const

const container = "mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8"

/**
 * Sticky landing navbar with scroll-spy: the link for the section currently
 * in view (or last clicked) turns teal-ink with a thin underline indicator.
 * Deliberately restrained — color + 2px bar only, matching Quiet Precision.
 */
export function SiteNav() {
  const [active, setActive] = useState<string | null>(null)
  const ticking = useRef(false)

  useEffect(() => {
    const update = () => {
      if (ticking.current) return
      ticking.current = true
      requestAnimationFrame(() => {
        ticking.current = false
        // Reference line just below the sticky header (64px) + breathing room.
        const probe = 112
        let current: string | null = null
        for (const item of NAV_ITEMS) {
          const el = document.querySelector(item.href)
          if (!el) continue
          if (el.getBoundingClientRect().top <= probe) current = item.href
        }
        // Scrolled to the very bottom → force the last section active.
        if (
          window.innerHeight + window.scrollY >=
          document.documentElement.scrollHeight - 4
        ) {
          current = NAV_ITEMS[NAV_ITEMS.length - 1].href
        }
        setActive(current)
      })
    }

    update()
    window.addEventListener("scroll", update, { passive: true })
    window.addEventListener("resize", update)
    return () => {
      window.removeEventListener("scroll", update)
      window.removeEventListener("resize", update)
    }
  }, [])

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/90 backdrop-blur">
      <div className={`${container} flex h-16 items-center justify-between`}>
        <Link href="/" className="flex items-center gap-2.5">
          <Logo size={36} />
          <span className="text-lg font-semibold tracking-tight text-foreground">
            Tayooli
          </span>
        </Link>

        <nav className="hidden items-center gap-8 text-sm font-medium text-muted-foreground md:flex">
          {NAV_ITEMS.map((item) => {
            const isActive = active === item.href
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setActive(item.href)}
                aria-current={isActive ? "true" : undefined}
                className={`relative py-1 transition-colors duration-200 ${
                  isActive ? "text-primary" : "hover:text-foreground"
                }`}
              >
                {item.label}
                <span
                  aria-hidden="true"
                  className={`absolute inset-x-0 -bottom-0.5 h-0.5 rounded-full bg-primary transition-opacity duration-200 ${
                    isActive ? "opacity-100" : "opacity-0"
                  }`}
                />
              </Link>
            )
          })}
        </nav>

        <div className="flex items-center gap-3">
          <Link
            href="/login"
            className="hidden text-sm font-semibold text-foreground transition-colors hover:text-primary sm:inline-flex"
          >
            Masuk
          </Link>
          <Link
            href="/login"
            className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-card transition-colors hover:bg-primary/90"
          >
            Mulai Gratis
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    </header>
  )
}
