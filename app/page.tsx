import Link from "next/link"
import { cookies } from "next/headers"
import { redirect } from "next/navigation"
import { AUTH_COOKIE } from "@/lib/auth/demo-auth"
import { PricingSection } from "@/components/landing/PricingSection"
import { SiteNav } from "@/components/landing/SiteNav"
import { Logo } from "@/components/brand/Logo"
import {
  ArrowRight,
  CalendarClock,
  Check,
  CheckCircle2,
  ChevronDown,
  FileSpreadsheet,
  FileWarning,
  FileText,
  Hourglass,
  Landmark,
  ScanLine,
  ShieldCheck,
  ShoppingCart,
  Wallet,
  Workflow,
} from "lucide-react"

/* ── Reusable styles ─────────────────────────────────────────────────────── */

const container = "mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8"
const eyebrow =
  "text-xs font-semibold uppercase tracking-[0.18em] text-primary"
const sectionTitle = "text-2xl sm:text-3xl font-bold tracking-tight text-foreground"
const sectionLead = "text-base sm:text-lg text-muted-foreground leading-relaxed"

const ctaPrimary =
  "inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-card transition-colors hover:bg-primary/90"
const ctaOutline =
  "inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border bg-background px-6 text-sm font-semibold text-foreground transition-colors hover:bg-muted"

const checkItem = "flex items-start gap-3 text-sm text-muted-foreground"
const checkIcon =
  "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary"

/* ── Sections ────────────────────────────────────────────────────────────── */

function HeroMockup() {
  const stats = [
    { label: "Total Invoice", value: "Rp 1,2 M", sub: "128 invoice bulan ini" },
    { label: "Disetujui", value: "Rp 890 Jt", sub: "62% dari total" },
    { label: "Menunggu Persetujuan", value: "12", sub: "3 butuh tindakan Anda" },
  ]
  const rows = [
    { no: "INV-2026-0142", vendor: "PT Nusantara Niaga", amount: "Rp 45.000.000", status: "Review AI", cls: "bg-blue-50 text-blue-700" },
    { no: "INV-2026-0141", vendor: "CV Karya Mandiri", amount: "Rp 8.750.000", status: "Menunggu", cls: "bg-amber-50 text-amber-700" },
    { no: "INV-2026-0140", vendor: "PT Maju Jaya", amount: "Rp 24.500.000", status: "Disetujui", cls: "bg-emerald-50 text-emerald-700" },
    { no: "INV-2026-0139", vendor: "Toko Berkah", amount: "Rp 12.300.000", status: "Ditolak", cls: "bg-rose-50 text-rose-700" },
  ]
  const bars = [40, 62, 48, 78, 55, 88, 70, 96]

  return (
    <div className="relative mx-auto mt-10 max-w-4xl">
      {/* Browser frame */}
      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-card-hover">
        {/* Window chrome */}
        <div className="flex items-center gap-3 border-b border-border bg-muted/60 px-4 py-3">
          <div className="flex gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-zinc-300" />
            <span className="h-2.5 w-2.5 rounded-full bg-zinc-300" />
            <span className="h-2.5 w-2.5 rounded-full bg-zinc-300" />
          </div>
          <div className="flex-1 rounded-md border border-border bg-background px-3 py-1 text-center text-xs text-muted-foreground">
            app.tayooli.id/dashboard
          </div>
          <span className="hidden rounded-full bg-primary/10 px-2.5 py-0.5 text-[11px] font-medium text-primary sm:block">
            Demo
          </span>
        </div>

        <div className="flex">
          {/* Fake sidebar */}
          <div className="hidden w-44 shrink-0 flex-col gap-1 border-r border-border bg-background p-3 sm:flex">
            <div className="mb-2 flex items-center gap-2 px-2">
              <Logo size={24} />
              <span className="text-xs font-semibold text-foreground">Tayooli</span>
            </div>
            {[
              { icon: Landmark, label: "Dashboard", active: true },
              { icon: FileText, label: "Invoice" },
              { icon: ShoppingCart, label: "Purchase Order" },
              { icon: Workflow, label: "Persetujuan" },
              { icon: Wallet, label: "Pembayaran" },
              { icon: Landmark, label: "Akuntansi" },
            ].map(({ icon: Icon, label, active }) => (
              <div
                key={label}
                className={`flex items-center gap-2.5 rounded-lg px-2 py-1.5 text-xs ${
                  active
                    ? "bg-primary/10 font-semibold text-primary"
                    : "text-muted-foreground"
                }`}
              >
                <Icon className="h-3.5 w-3.5" />
                {label}
              </div>
            ))}
          </div>

          {/* Fake main */}
          <div className="flex-1 space-y-4 bg-background p-4 sm:p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-bold text-foreground">Dashboard</div>
                <div className="text-xs text-muted-foreground">
                  Ringkasan keuangan September 2026
                </div>
              </div>
              <div className="hidden rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground sm:block">
                + Invoice Baru
              </div>
            </div>

            {/* Stat cards */}
            <div className="grid grid-cols-3 gap-3">
              {stats.map((s) => (
                <div key={s.label} className="rounded-lg border border-border bg-card p-3">
                  <div className="text-[11px] text-muted-foreground">{s.label}</div>
                  <div className="mt-0.5 text-sm font-bold text-foreground sm:text-base">{s.value}</div>
                  <div className="mt-0.5 hidden text-[10px] text-muted-foreground lg:block">{s.sub}</div>
                </div>
              ))}
            </div>

            <div className="grid gap-3 lg:grid-cols-5">
              {/* Invoice table */}
              <div className="rounded-lg border border-border bg-card p-3 lg:col-span-3">
                <div className="mb-2 text-xs font-semibold text-foreground">Invoice Terbaru</div>
                <div className="space-y-1.5">
                  {rows.map((r) => (
                    <div
                      key={r.no}
                      className="flex items-center justify-between gap-2 rounded-md border border-border/60 bg-background px-2.5 py-1.5"
                    >
                      <div className="min-w-0">
                        <div className="font-mono text-[11px] font-medium text-foreground">{r.no}</div>
                        <div className="truncate text-[10px] text-muted-foreground">{r.vendor}</div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="hidden text-[11px] font-medium text-zinc-700 sm:block">{r.amount}</span>
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${r.cls}`}>
                          {r.status}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Mini bar chart */}
              <div className="rounded-lg border border-border bg-card p-3 lg:col-span-2">
                <div className="mb-2 text-xs font-semibold text-foreground">Arus Kas 8 Bulan</div>
                <div className="flex h-28 items-end gap-1.5">
                  {bars.map((h, i) => (
                    <div
                      key={i}
                      className="flex-1 rounded-t-sm bg-primary/70"
                      style={{ height: `${h}%` }}
                    />
                  ))}
                </div>
                <div className="mt-2 flex justify-between text-[9px] text-muted-foreground">
                  {["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"].map((m) => (
                    <span key={m}>{m}</span>
                  ))}
                </div>
                <div className="mt-3 rounded-md bg-success/10 px-2.5 py-1.5 text-[10px] font-medium text-success">
                  ▲ +18% dibanding kuartal lalu
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Floating approval card */}
      <div className="absolute -right-4 -top-8 hidden w-64 rounded-xl border border-border bg-card p-4 shadow-popover lg:block">
        <div className="flex items-center gap-2">
          <CheckCircle2 className="h-4 w-4 text-success" />
          <span className="text-xs font-semibold text-foreground">Persetujuan diterima</span>
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
          PO-2026-0142 · PT Nusantara Niaga
          <br />
          Rp 45.000.000
        </p>
        <p className="mt-2 text-[10px] text-muted-foreground">oleh Andi · 2 menit lalu</p>
      </div>

      {/* Floating AI review card */}
      <div className="absolute -bottom-8 -left-4 hidden w-64 rounded-xl border border-border bg-card p-4 shadow-popover lg:block">
        <div className="flex items-center gap-2">
          <ScanLine className="h-4 w-4 text-primary" />
          <span className="text-xs font-semibold text-foreground">OCR selesai</span>
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
          INV-2026-0142 terbaca otomatis dari PDF — tanpa salah ketik.
        </p>
      </div>
    </div>
  )
}

function StatsBand() {
  const stats = [
    { value: "3×", label: "lebih cepat siklus invoice, dari terima hingga bayar" },
    { value: "98%", label: "akurasi pembacaan OCR dengan review AI" },
    { value: "24 jam", label: "rata-rata invoice disetujui dengan workflow" },
    { value: "15+", label: "kanal pembayaran & integrasi siap pakai" },
  ]
  return (
    <section className="py-12 sm:py-14">
      <div className={`${container} grid grid-cols-2 gap-x-8 gap-y-8 sm:gap-y-10 lg:grid-cols-4`}>
        {stats.map((s) => (
          <div key={s.value}>
            <div className="text-3xl font-bold tracking-tight text-primary sm:text-4xl">{s.value}</div>
            <p className="mt-1.5 max-w-[220px] text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {s.label}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}

function ProblemSection() {
  const pains = [
    {
      icon: FileWarning,
      title: "Invoice tersebar di mana-mana",
      desc: "Email, Excel, chat — rawan salah input, hilang, dan duplikat. Tidak ada satu sumber kebenaran.",
    },
    {
      icon: Hourglass,
      title: "Persetujuan berlarut-larut",
      desc: "Tagihan menunggu tanda tangan tanpa jejak yang jelas. Tak ada yang tahu di mana proses berhenti.",
    },
    {
      icon: CalendarClock,
      title: "Pembayaran telat, cash flow tersendat",
      desc: "Jatuh tempo terlewat, denda menumpuk, dan hubungan dengan vendor mulai renggang.",
    },
    {
      icon: FileSpreadsheet,
      title: "Laporan dirapikan manual",
      desc: "Jurnal, rekonsiliasi, dan laporan dikerjakan larut malam di spreadsheet yang gampang salah.",
    },
  ]
  return (
    <section className="bg-card py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Masalahnya</p>
          <h2 className={`${sectionTitle} mt-3`}>Finance manual itu mahal — diam-diam.</h2>
          <p className={`${sectionLead} mt-4`}>
            Setiap jam yang dipakai mengetik ulang invoice adalah jam yang tidak dipakai untuk
            mengelola bisnis. Ini yang terjadi hampir setiap minggu:
          </p>
        </div>
        <div className="mt-12 grid gap-x-8 gap-y-10 sm:grid-cols-2 lg:grid-cols-4">
          {pains.map(({ icon: Icon, title, desc }) => (
            <div key={title}>
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/8 text-primary">
                <Icon className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-4 text-sm font-semibold text-foreground">{title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function FeatureRow({
  eyebrowLabel,
  title,
  desc,
  bullets,
  visual,
  reversed = false,
}: {
  eyebrowLabel: string
  title: string
  desc: string
  bullets: string[]
  visual: React.ReactNode
  reversed?: boolean
}) {
  const content = (
    <div className="flex flex-col justify-center">
      <p className={eyebrow}>{eyebrowLabel}</p>
      <h3 className="mt-3 text-xl font-bold tracking-tight text-foreground sm:text-2xl">
        {title}
      </h3>
      <p className="mt-3 text-sm leading-relaxed text-muted-foreground sm:text-base">{desc}</p>
      <ul className="mt-6 space-y-3">
        {bullets.map((b) => (
          <li key={b} className={checkItem}>
            <span className={checkIcon}>
              <Check className="h-3 w-3" />
            </span>
            {b}
          </li>
        ))}
      </ul>
    </div>
  )

  return (
    <div
      className={`grid items-center gap-8 sm:gap-12 lg:grid-cols-2 lg:gap-16 ${
        reversed ? "lg:[&>:first-child]:order-2" : ""
      }`}
    >
      {content}
      {visual}
    </div>
  )
}

/** Mini product panels that hint at actual UI without being abstract decorations. */
function FeatureVisualOCR() {
  return (
    <div className="rounded-xl border border-border bg-card p-5 shadow-card">
      <div className="flex items-center gap-2">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <ScanLine className="h-4 w-4" />
        </div>
        <span className="text-xs font-semibold text-foreground">OCR & Validation</span>
        <span className="ml-auto rounded-full bg-success/10 px-2 py-0.5 text-[10px] font-medium text-success">
          Tervalidasi
        </span>
      </div>
      <div className="mt-4 space-y-2">
        {[
          { field: "Vendor", val: "PT Nusantara Niaga" },
          { field: "No. Invoice", val: "INV-2026-0142" },
          { field: "Total", val: "Rp 45.000.000" },
          { field: "PPN", val: "Rp 4.500.000" },
        ].map((f) => (
          <div key={f.field} className="flex items-center justify-between rounded-md bg-background px-3 py-2">
            <span className="text-xs text-muted-foreground">{f.field}</span>
            <span className="font-mono text-xs font-medium text-foreground">{f.val}</span>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center gap-2 rounded-md bg-primary/5 px-3 py-2">
        <Check className="h-3.5 w-3.5 text-success" />
        <span className="text-[11px] text-muted-foreground">Duplikat tidak terdeteksi · Angka cocok 100%</span>
      </div>
    </div>
  )
}

function FeatureVisualApproval() {
  return (
    <div className="rounded-xl border border-border bg-card p-5 shadow-card">
      <div className="flex items-center gap-2">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Workflow className="h-4 w-4" />
        </div>
        <span className="text-xs font-semibold text-foreground">Approval Pipeline</span>
      </div>
      <div className="mt-4 space-y-2">
        {[
          { step: "Admin", status: "Disetujui", color: "bg-success/10 text-success" },
          { step: "Akuntan", status: "Disetujui", color: "bg-success/10 text-success" },
          { step: "Approver", status: "Menunggu", color: "bg-amber-50 text-amber-700" },
        ].map((s) => (
          <div key={s.step} className="flex items-center justify-between rounded-md bg-background px-3 py-2">
            <span className="text-xs font-medium text-foreground">{s.step}</span>
            <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${s.color}`}>{s.status}</span>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center gap-2 rounded-md bg-background px-3 py-2">
        <span className="text-[11px] text-muted-foreground">Limit: &gt;Rp 10.000.000 → ApproverRequired</span>
      </div>
    </div>
  )
}

function FeaturesSection() {
  return (
    <section id="fitur" className="scroll-mt-20 border-t border-border bg-muted/40 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Satu sistem, seluruh alur</p>
          <h2 className={`${sectionTitle} mt-3`}>
            Dari invoice diterima sampai jurnal tercatat — tanpa bolak-balik.
          </h2>
          <p className={`${sectionLead} mt-4`}>
            Tayooli menutup seluruh siklus order-to-pay dan order-to-cash untuk bisnis Indonesia,
            dengan otomatisasi di titik-titik yang paling sering menguras waktu.
          </p>
        </div>

        <div className="mt-14 space-y-16 sm:space-y-20">
          <FeatureRow
            eyebrowLabel="Hemat Waktu"
            title="Invoice dibaca mesin, bukan diketik"
            desc="Upload PDF atau foto — OCR membaca, AI memvalidasi, lalu invoice masuk alur persetujuan. Tanpa salah ketik, tanpa input ulang."
            bullets={[
              "OCR otomatis dari PDF dan foto, angka terbaca presisi",
              "AI mendeteksi anomali, duplikat, dan selisih jumlah",
              "Data tervalidasi sebelum pernah menyentuh pembukuan",
            ]}
            visual={<FeatureVisualOCR />}
          />
          <FeatureRow
            eyebrowLabel="Alur Terkendali"
            title="Persetujuan yang jelas, tanpa kejar-kejaran"
            desc="Aturan approval berbasis peran: siapa menyetujui, berapa limitnya, dan apa yang terjadi setelahnya — semua tercatat."
            bullets={[
              "Alur PO → GR → persetujuan → payment order yang utuh",
              "Peran admin, akuntan, dan approver dengan limit berbeda",
              "Jejak audit lengkap untuk setiap keputusan",
            ]}
            visual={<FeatureVisualApproval />}
            reversed
          />
          <FeatureRow
            eyebrowLabel="Keuangan Akurat"
            title="Jurnal terisi sendiri dari setiap transaksi"
            desc="Setiap invoice dan pembayaran yang disetujui otomatis membentuk jurnal — chart of accounts dan laporan selalu selaras."
            bullets={[
              "Chart of accounts & jurnal otomatis tanpa entri ganda",
              "Rekonsiliasi pembayaran real-time via webhook",
              "Dashboard: status invoice, arus kas, top vendor",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Landmark className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Jurnal Otomatis</span>
                </div>
                <div className="mt-4 space-y-2">
                  <div className="rounded-md bg-background px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-muted-foreground">Debit</div>
                    <div className="mt-0.5 flex items-center justify-between">
                      <span className="text-xs text-foreground">Beban Operasional</span>
                      <span className="font-mono text-xs font-medium text-foreground">45.000.000</span>
                    </div>
                  </div>
                  <div className="rounded-md bg-background px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-muted-foreground">Kredit</div>
                    <div className="mt-0.5 flex items-center justify-between">
                      <span className="text-xs text-foreground">Piutang Usaha</span>
                      <span className="font-mono text-xs font-medium text-foreground">45.000.000</span>
                    </div>
                  </div>
                </div>
                <div className="mt-3 flex items-center gap-2 rounded-md bg-success/5 px-3 py-2">
                  <Check className="h-3.5 w-3.5 text-success" />
                  <span className="text-[11px] text-muted-foreground">Jurnal tercatat otomatis · COA sinkron</span>
                </div>
              </div>
            }
          />
          <FeatureRow
            eyebrowLabel="Siap Bayar"
            title="Dibayar lewat kanal favorit pelanggan"
            desc="Terhubung dengan payment gateway lokal, dengan status pembayaran yang sinkron sampai ke pembukuan."
            bullets={[
              "Pakasir & Midtrans terpasang, tinggal isi kredensial",
              "Status pembayaran terpantau real-time",
              "Multi-tenant dengan isolasi data (RLS) di setiap lapisan",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Wallet className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Payment Channels</span>
                </div>
                <div className="mt-4 space-y-2">
                  {[
                    { name: "Pakasir", status: "Terhubung", color: "bg-success/10 text-success" },
                    { name: "Midtrans", status: "Terhubung", color: "bg-success/10 text-success" },
                    { name: "Transfer Bank", status: "Aktif", color: "bg-primary/10 text-primary" },
                  ].map((g) => (
                    <div key={g.name} className="flex items-center justify-between rounded-md bg-background px-3 py-2">
                      <span className="text-xs font-medium text-foreground">{g.name}</span>
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${g.color}`}>{g.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            }
            reversed
          />
        </div>
      </div>
    </section>
  )
}

function HowItWorks() {
  const steps = [
    {
      n: "01",
      title: "Daftar & undang tim",
      desc: "Buat workspace, undang admin, akuntan, dan approver. Atur peran dan limit persetujuan sekali.",
    },
    {
      n: "02",
      title: "Masukkan data tanpa mengetik",
      desc: "Upload invoice (OCR membacanya otomatis) atau buat PO dan goods receipt langsung di sistem.",
    },
    {
      n: "03",
      title: "Setujui, bayar, lacak",
      desc: "Alur approval berjalan, payment order terbit, dan setiap status terpantau real-time sampai jurnal.",
    },
  ]
  return (
    <section id="cara-kerja" className="scroll-mt-20 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Cara Kerja</p>
          <h2 className={`${sectionTitle} mt-3`}>Produktif dalam tiga langkah.</h2>
          <p className={`${sectionLead} mt-4`}>
            Tidak perlu implementasi berbulan-bulan. Alur yang sama dengan tim finance Anda, hanya
            tanpa kerja manual.
          </p>
        </div>
        <div className="mt-12 grid gap-4 md:grid-cols-3">
          {steps.map((s) => (
            <div key={s.n} className="relative rounded-xl border border-border bg-card p-7 shadow-card">
              <span className="font-mono text-sm font-semibold text-primary">{s.n}</span>
              <h3 className="mt-3 text-base font-semibold text-foreground">{s.title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{s.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function Testimonials() {
  const items = [
    {
      quote:
        "Sebelumnya invoice kami numpuk di email dan Excel. Sekarang OCR langsung baca, approver tinggal klik — siklus pembayaran turun dari dua minggu jadi tiga hari.",
      name: "Rina Puspitasari",
      role: "Finance Manager, PT Nusantara Niaga",
    },
    {
      quote:
        "Yang saya suka: alur persetujuannya jelas. Tidak ada lagi invoice hilang di meja orang. Semua ada jejaknya, dan laporan akuntansi terisi sendiri.",
      name: "Budi Santoso",
      role: "Owner, CV Karya Mandiri",
    },
    {
      quote:
        "Kami menjalankan tiga entitas. Isolasi data per tenant bikin audit tenang, dan pembayaran via gateway lokal langsung tersinkron ke pembukuan.",
      name: "Dewi Lestari",
      role: "Head of Finance, PT IndoLogistik",
    },
  ]
  return (
    <section className="py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Testimoni</p>
          <h2 className={`${sectionTitle} mt-3`}>Dipakai tim finance yang butuh ketenangan.</h2>
        </div>
        <div className="mt-12 grid gap-8 sm:gap-6 md:grid-cols-3">
          {items.map((t) => (
            <blockquote key={t.name} className="flex flex-col">
              <p className="flex-1 text-sm leading-relaxed text-foreground/80 sm:text-base">
                “{t.quote}”
              </p>
              <figcaption className="mt-5 pt-4">
                <div className="text-sm font-semibold text-foreground">{t.name}</div>
                <div className="mt-0.5 text-xs text-muted-foreground">{t.role}</div>
              </figcaption>
            </blockquote>
          ))}
        </div>
      </div>
    </section>
  )
}

function Faq() {
  const items = [
    {
      q: "Apakah Tayooli bisa membaca invoice dari PDF atau foto?",
      a: "Bisa. Upload file dan OCR membacanya otomatis, lalu AI memvalidasi angka, mendeteksi anomali atau duplikat sebelum invoice masuk alur persetujuan. Anda tetap bisa mengoreksi sebelum disetujui.",
    },
    {
      q: "Bagaimana alur persetujuan bekerja?",
      a: "Anda mengatur peran (admin, akuntan, approver) beserta limitnya. Invoice atau PO yang melewati limit otomatis masuk antrean approver yang sesuai — semua keputusan terekam dalam jejak audit.",
    },
    {
      q: "Apakah ada biaya per transaksi?",
      a: "Tidak. Langganan bersifat flat per bulan. Biaya dari payment gateway hanya muncul jika Anda memakainya, sesuai tarif gateway itu sendiri.",
    },
    {
      q: "Bagaimana keamanan data antar-perusahaan?",
      a: "Tayooli multi-tenant dengan row-level security di PostgreSQL — data setiap tenant terisolasi di tingkat database, bukan sekadar di aplikasi.",
    },
    {
      q: "Bisa dicoba tanpa kartu kredit?",
      a: "Tentu. 14 hari gratis, tanpa kartu kredit, batalkan kapan saja. Untuk melihat langsung, masuk dengan akun demo: admin@test.com / password123.",
    },
    {
      q: "Bagaimana integrasi dengan bank atau gateway lain?",
      a: "Pakasir dan Midtrans sudah terpasang. Untuk kebutuhan khusus, tersedia akses API — tim kami bisa membahas kustomisasi untuk Enterprise.",
    },
  ]
  return (
    <section id="faq" className="scroll-mt-20 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>FAQ</p>
          <h2 className={`${sectionTitle} mt-3`}>Pertanyaan yang sering ditanyakan.</h2>
        </div>
        <div className="mt-10 max-w-3xl space-y-3">
          {items.map((item) => (
            <details
              key={item.q}
              className="group rounded-xl border border-border bg-card shadow-card open:shadow-card-hover"
            >
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 px-5 py-4 text-sm font-semibold text-foreground [&::-webkit-details-marker]:hidden">
                {item.q}
                <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180" />
              </summary>
              <p className="px-5 pb-5 text-sm leading-relaxed text-muted-foreground">{item.a}</p>
            </details>
          ))}
        </div>
      </div>
    </section>
  )
}

function FinalCta() {
  return (
    <section className="px-4 pb-20 sm:px-6 sm:pb-24">
      <div className="mx-auto max-w-6xl">
        <div className="relative overflow-hidden rounded-xl bg-foreground px-6 py-16 text-center sm:px-12 sm:py-20">
          <div className="mx-auto max-w-2xl">
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-primary-foreground/60">
              Mulai Sekarang
            </p>
            <h2 className="mt-4 text-2xl font-bold tracking-tight text-background sm:text-4xl">
              Keuangan yang tenang dan terkendali.
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-background/70 sm:text-base">
              Gratis 14 hari. Tanpa kartu kredit. Batalkan kapan saja — invoice, persetujuan, dan
              pembayaran Anda beres sebelum akhir pekan.
            </p>
            <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Link
                href="/login"
                className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-background px-7 text-sm font-semibold text-foreground shadow-card transition-colors hover:bg-background/90"
              >
                Mulai Gratis
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="#faq"
                className="inline-flex h-11 items-center justify-center rounded-lg border border-background/25 px-7 text-sm font-semibold text-background transition-colors hover:bg-background/10"
              >
                Tanya Jawab
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function Footer() {
  const cols = [
    {
      title: "Produk",
      links: [
        { label: "Fitur", href: "#fitur" },
        { label: "Cara Kerja", href: "#cara-kerja" },
        { label: "Harga", href: "#harga" },
        { label: "FAQ", href: "#faq" },
      ],
    },
    {
      title: "Akses",
      links: [
        { label: "Masuk", href: "/login" },
        { label: "Mulai Gratis", href: "/login" },
        { label: "Demo", href: "/login" },
      ],
    },
    {
      title: "Perusahaan",
      links: [
        { label: "Tentang", href: "#" },
        { label: "Keamanan", href: "#" },
        { label: "Kontak", href: "#" },
      ],
    },
  ]
  return (
    <footer className="border-t border-border bg-card">
      <div className={`${container} py-14`}>
        <div className="grid gap-10 md:grid-cols-[1.5fr_1fr_1fr_1fr]">
          <div>
            <div className="flex items-center gap-2.5">
              <Logo size={36} />
              <span className="text-lg font-semibold tracking-tight text-foreground">Tayooli</span>
            </div>
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-muted-foreground">
              ERP order-to-pay & order-to-cash untuk bisnis Indonesia — invoice, persetujuan, dan
              pembayaran dalam satu alur yang tenang.
            </p>
          </div>
          {cols.map((col) => (
            <div key={col.title}>
              <h3 className="text-sm font-semibold text-foreground">{col.title}</h3>
              <ul className="mt-4 space-y-3">
                {col.links.map((l) => (
                  <li key={l.label}>
                    <Link
                      href={l.href}
                      className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {l.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-12 flex flex-col items-start justify-between gap-4 border-t border-border pt-6 sm:flex-row sm:items-center">
          <p className="text-xs text-muted-foreground">© 2026 Tayooli. Dibuat untuk finance yang tenang.</p>
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <ShieldCheck className="h-3.5 w-3.5 text-primary" />
            Data terisolasi per tenant · TLS & RLS
          </p>
        </div>
      </div>
    </footer>
  )
}

/* ── Page ─────────────────────────────────────────────────────────────────── */

/**
 * LandingPage component: marketing landing page for Tayooli ERP.
 * Configurable toggle: by default redirects root (/) to /dashboard or /login
 * for client demos. Can be re-enabled by setting NEXT_PUBLIC_DISABLE_LANDING_PAGE='false'.
 */
export const dynamic = "force-dynamic"

export default async function LandingPage() {
  const isLandingDisabled =
    process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE !== "false"

  if (isLandingDisabled) {
    const cookieStore = await cookies()
    const authCookie = cookieStore.get(AUTH_COOKIE)?.value
    if (authCookie) {
      redirect("/dashboard")
    }
    redirect("/login")
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <SiteNav />

      {/* Hero */}
      <section className="overflow-hidden">
        <div className={`${container} pt-12 sm:pt-16 lg:pt-20`}>
          <div className="mx-auto max-w-3xl text-center">
            <span className="inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-3.5 py-1.5 text-xs font-medium text-primary">
              <span className="h-1.5 w-1.5 rounded-full bg-success" />
              AI-driven ERP · Order-to-Pay & Order-to-Cash
            </span>
            <h1 className="mt-5 text-4xl font-bold leading-[1.1] tracking-tight text-foreground sm:text-5xl lg:text-5xl xl:text-[3.4rem]">
              Tagihan beres, pembayaran lancar, laporan akurat.
            </h1>
            <p className="mx-auto mt-5 max-w-2xl text-base leading-relaxed text-muted-foreground sm:text-lg">
              Tayooli menyatukan invoice (OCR + AI), persetujuan, dan pembayaran dalam satu alur —
              dari PO hingga jurnal. Otomatis, aman, dan dibangun untuk bisnis Indonesia.
            </p>
            <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Link href="/login" className={ctaPrimary}>
                Mulai Gratis 14 Hari
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link href="#produk" className={ctaOutline}>
                Lihat Demo
              </Link>
            </div>
            <p className="mt-5 text-xs text-muted-foreground">
              Tanpa kartu kredit · Batalkan kapan saja · Coba akun demo:{" "}
              <span className="font-mono text-foreground/80">admin@test.com</span>
            </p>
          </div>

          <div id="produk" className="scroll-mt-20">
            <HeroMockup />
          </div>
        </div>
      </section>

      <StatsBand />
      <ProblemSection />
      <FeaturesSection />
      <HowItWorks />
      <Testimonials />
      <PricingSection />
      <Faq />
      <FinalCta />
      <Footer />
    </div>
  )
}
