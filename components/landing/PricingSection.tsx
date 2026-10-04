"use client"

import { useState } from "react"
import Link from "next/link"
import { ArrowRight, Check } from "lucide-react"
import { formatCurrency } from "@/lib/currency"

type BillingPeriod = "monthly" | "annual"

interface Tier {
  name: string
  tagline: string
  monthly: number | null // null = custom (Enterprise)
  featured?: boolean
  cta: { label: string; href: string }
  features: string[]
}

const TIERS: Tier[] = [
  {
    name: "Starter",
    tagline: "Untuk usaha kecil yang baru rapi.",
    monthly: 199000,
    cta: { label: "Mulai Gratis", href: "/login" },
    features: [
      "Invoice & OCR tanpa batas",
      "1 pengguna, hingga 100 vendor",
      "Alur persetujuan dasar",
      "Status invoice real-time",
      "Dashboard keuangan",
    ],
  },
  {
    name: "Bisnis",
    tagline: "Untuk tim finance yang sedang tumbuh.",
    monthly: 449000,
    featured: true,
    cta: { label: "Mulai Gratis", href: "/login" },
    features: [
      "Semua di Starter",
      "10 pengguna, vendor tak terbatas",
      "Persetujuan multi-level & limit",
      "Accounting otomatis (COA & jurnal)",
      "Payment gateway Pakasir & Midtrans",
      "Akses API",
    ],
  },
  {
    name: "Enterprise",
    tagline: "Untuk kebutuhan kompleks & multi-entitas.",
    monthly: null,
    cta: { label: "Hubungi Sales", href: "/login" },
    features: [
      "Pengguna tak terbatas",
      "Multi-entitas dengan RLS lanjutan",
      "Workflow & integrasi kustom",
      "SLA & dukungan khusus",
      "Onboarding terpandu",
    ],
  },
]

export function PricingSection() {
  const [period, setPeriod] = useState<BillingPeriod>("annual")
  const [selected, setSelected] = useState<string>("Bisnis")

  return (
    <section id="harga" className="scroll-mt-20 bg-muted/40 py-16 sm:py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-primary">Harga</p>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            Harga yang jelas, tanpa biaya per transaksi.
          </h2>
          <p className="mt-4 text-base text-muted-foreground sm:text-lg">
            Langganan flat per bulan. Mulai gratis 14 hari — tanpa kartu kredit.
          </p>

          {/* Billing toggle */}
          <div className="mt-8 inline-flex items-center gap-1 rounded-lg border border-border bg-card p-1">
            {(
              [
                { key: "monthly", label: "Bulanan" },
                { key: "annual", label: "Tahunan · Hemat 20%" },
              ] as const
            ).map((opt) => (
              <button
                key={opt.key}
                type="button"
                onClick={() => setPeriod(opt.key)}
                className={`rounded-md px-4 py-1.5 text-sm font-medium transition-colors ${
                  period === opt.key
                    ? "bg-primary text-primary-foreground shadow-card"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        <div role="radiogroup" aria-label="Pilih paket" className="mt-12 grid gap-4 lg:grid-cols-3">
          {TIERS.map((tier) => {
            const price =
              tier.monthly === null
                ? null
                : period === "annual"
                  ? Math.round(tier.monthly * 0.8)
                  : tier.monthly
            const isSelected = selected === tier.name
            return (
              <div
                key={tier.name}
                role="radio"
                aria-checked={isSelected}
                tabIndex={0}
                onClick={() => setSelected(tier.name)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault()
                    setSelected(tier.name)
                  }
                }}
                className={`relative flex cursor-pointer flex-col rounded-xl border bg-card p-7 shadow-card transition-all duration-200 select-none ${
                  isSelected
                    ? "border-primary shadow-card-hover ring-1 ring-primary/15"
                    : tier.featured
                      ? "border-primary/40 hover:border-primary/70 hover:shadow-card-hover"
                      : "border-border hover:border-primary/40 hover:shadow-card-hover"
                } ${tier.featured ? "lg:-my-2 lg:py-9" : ""}`}
              >
                {tier.featured && (
                  <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-primary px-3 py-1 text-[11px] font-semibold text-primary-foreground">
                    Paling Populer
                  </span>
                )}
                {isSelected && (
                  <span className="absolute right-4 top-4 inline-flex items-center gap-1 rounded-full bg-primary px-2.5 py-0.5 text-[11px] font-semibold text-primary-foreground">
                    <Check className="h-3 w-3" />
                    Terpilih
                  </span>
                )}
                <h3 className="text-base font-semibold text-foreground">{tier.name}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{tier.tagline}</p>

                <div className="mt-6 flex items-baseline gap-1">
                  {price === null ? (
                    <span className="text-3xl font-bold tracking-tight text-foreground">Custom</span>
                  ) : (
                    <>
                      <span className="text-3xl font-bold tracking-tight text-foreground">
                        {formatCurrency(price)}
                      </span>
                      <span className="text-sm text-muted-foreground">/bulan</span>
                    </>
                  )}
                </div>
                {price !== null && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    {period === "annual" ? "Ditagih tahunan (hemat 20%)" : "Ditagih bulanan"}
                  </p>
                )}

                <ul className="mt-6 flex-1 space-y-3">
                  {tier.features.map((f) => (
                    <li key={f} className="flex items-start gap-3 text-sm text-muted-foreground">
                      <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
                        <Check className="h-3 w-3" />
                      </span>
                      {f}
                    </li>
                  ))}
                </ul>

                <Link
                  href={tier.cta.href}
                  className={`mt-7 inline-flex h-11 items-center justify-center gap-2 rounded-lg px-6 text-sm font-semibold transition-colors ${
                    isSelected || tier.featured
                      ? "bg-primary text-primary-foreground shadow-card hover:bg-primary/90"
                      : "border border-border bg-background text-foreground hover:bg-muted"
                  }`}
                >
                  {tier.cta.label}
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
            )
          })}
        </div>

        <p className="mt-8 text-center text-xs text-muted-foreground">
          Klik kartu untuk memilih paket · Gratis 14 hari · Tanpa kartu kredit · Batalkan kapan saja
        </p>
      </div>
    </section>
  )
}
