import Link from "next/link"
import { cookies } from "next/headers"
import { redirect } from "next/navigation"
import { AUTH_COOKIE } from "@/lib/auth/demo-auth"
import { PricingSection } from "@/components/landing/PricingSection"
import { SiteNav } from "@/components/landing/SiteNav"
import { Logo } from "@/components/brand/Logo"
import {
  ArrowRight,
  Boxes,
  CalendarClock,
  Check,
  CheckCircle2,
  ChevronDown,
  ClipboardCheck,
  FileSpreadsheet,
  FileWarning,
  Hourglass,
  LayoutDashboard,
  Package,
  ScanLine,
  Settings,
  ShieldCheck,
  ShoppingBag,
  Store,
  Truck,
  Warehouse,
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
    { label: "Omzet POS Hari Ini", value: "Rp 4,2 Jt", sub: "14 transaksi kasir" },
    { label: "Fisik di Gudang", value: "1.240 Unit", sub: "3 gudang aktif" },
    { label: "Peringatan Stok", value: "2 SKU", sub: "Stok di bawah batas aman" },
  ]
  const rows = [
    { no: "POS-2026-0042", desc: "Budi Santoso · Kasir Utama", amount: "Rp 450.000", status: "QRIS Lunas", cls: "bg-emerald-50 text-emerald-700" },
    { no: "DO-2026-0018", desc: "Toko Sinar Jaya · Surat Jalan", amount: "120 Unit", status: "Terkirim", cls: "bg-blue-50 text-blue-700" },
    { no: "POS-2026-0041", desc: "Pelanggan Tunai", amount: "Rp 125.000", status: "Tunai Lunas", cls: "bg-emerald-50 text-emerald-700" },
    { no: "TRF-2026-0005", desc: "Gudang Utama → Cabang", amount: "50 Unit", status: "Dalam Pengiriman", cls: "bg-amber-50 text-amber-700" },
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
          <div className="flex-1 rounded-md border border-border bg-background px-3 py-1 text-center text-xs text-muted-foreground font-mono">
            app.tayooli.my.id/dashboard
          </div>
          <span className="hidden rounded-full bg-primary/10 px-2.5 py-0.5 text-[11px] font-medium text-primary sm:block">
            Standalone ERP
          </span>
        </div>

        <div className="flex">
          {/* Fake sidebar */}
          <div className="hidden w-48 shrink-0 flex-col gap-1 border-r border-border bg-background p-3 sm:flex">
            <div className="mb-2 flex items-center gap-2 px-2">
              <Logo size={24} />
              <span className="text-xs font-semibold text-foreground">Tayooli</span>
            </div>
            {[
              { icon: LayoutDashboard, label: "Dashboard", active: true },
              { icon: Package, label: "Products" },
              { icon: Warehouse, label: "Warehouse & Stock" },
              { icon: Truck, label: "Surat Jalan (DO)" },
              { icon: ShoppingBag, label: "Marketplace" },
              { icon: Store, label: "Point of Sale" },
              { icon: Settings, label: "Settings" },
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
                <div className="text-sm font-bold text-foreground">Dashboard Operasional</div>
                <div className="text-xs text-muted-foreground">
                  Kasir POS, pergudangan, dan mutasi barang hari ini
                </div>
              </div>
              <div className="hidden rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white sm:block">
                + Buka Kasir POS
              </div>
            </div>

            {/* Stat cards */}
            <div className="grid grid-cols-3 gap-3">
              {stats.map((s) => (
                <div key={s.label} className="rounded-lg border border-border bg-card p-3">
                  <div className="text-[11px] text-muted-foreground">{s.label}</div>
                  <div className="mt-0.5 text-sm font-bold text-foreground sm:text-base font-mono">{s.value}</div>
                  <div className="mt-0.5 hidden text-[10px] text-muted-foreground lg:block">{s.sub}</div>
                </div>
              ))}
            </div>

            <div className="grid gap-3 lg:grid-cols-5">
              {/* Transactions table */}
              <div className="rounded-lg border border-border bg-card p-3 lg:col-span-3">
                <div className="mb-2 text-xs font-semibold text-foreground">Transaksi Kasir &amp; Pengiriman Terbaru</div>
                <div className="space-y-1.5">
                  {rows.map((r) => (
                    <div
                      key={r.no}
                      className="flex items-center justify-between gap-2 rounded-md border border-border/60 bg-background px-2.5 py-1.5"
                    >
                      <div className="min-w-0">
                        <div className="font-mono text-[11px] font-medium text-foreground">{r.no}</div>
                        <div className="truncate text-[10px] text-muted-foreground">{r.desc}</div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="hidden text-[11px] font-medium font-mono text-zinc-700 sm:block">{r.amount}</span>
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
                <div className="mb-2 text-xs font-semibold text-foreground">Tren Omzet Mingguan</div>
                <div className="flex h-28 items-end gap-1.5">
                  {bars.map((h, i) => (
                    <div
                      key={i}
                      className="flex-1 rounded-t-sm bg-primary/70"
                      style={{ height: `${h}%` }}
                    />
                  ))}
                </div>
                <div className="mt-2 flex justify-between text-[9px] text-muted-foreground font-mono">
                  {["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min", "Hari ini"].map((m) => (
                    <span key={m}>{m}</span>
                  ))}
                </div>
                <div className="mt-3 rounded-md bg-success/10 px-2.5 py-1.5 text-[10px] font-medium text-success">
                  ▲ Omzet stabil, stok terkendali
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Floating POS success card */}
      <div className="absolute -right-4 -top-8 hidden w-64 rounded-xl border border-border bg-card p-4 shadow-popover lg:block">
        <div className="flex items-center gap-2">
          <Store className="h-4 w-4 text-emerald-600" />
          <span className="text-xs font-semibold text-foreground">Transaksi Kasir Berhasil</span>
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
          POS-2026-0042 · QRIS Lunas
          <br />
          <span className="font-mono font-medium text-foreground">Rp 450.000</span> · Stok otomatis berkurang
        </p>
        <p className="mt-2 text-[10px] text-muted-foreground">Kasir Toko 1 · Baru saja</p>
      </div>

      {/* Floating DO card */}
      <div className="absolute -bottom-8 -left-4 hidden w-64 rounded-xl border border-border bg-card p-4 shadow-popover lg:block">
        <div className="flex items-center gap-2">
          <Truck className="h-4 w-4 text-primary" />
          <span className="text-xs font-semibold text-foreground">Surat Jalan Siap Kirim</span>
        </div>
        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
          DO-2026-0018 ke Toko Sinar Jaya telah dicetak, alokasi armada kurir selesai.
        </p>
      </div>
    </div>
  )
}

function StatsBand() {
  const stats = [
    { value: "Multi-Gudang", label: "pencatatan mutasi fisik antar gudang, rak, dan transit" },
    { value: "Kasir POS", label: "cetak nota thermal 58mm/80mm, QRIS, dan hitung kembalian" },
    { value: "Omnichannel", label: "impor dan pemetaan pesanan marketplace ke stok fisik" },
    { value: "13 Modul", label: "alur barang masuk, stok, kasir, hingga kirim dalam satu alur" },
  ]
  return (
    <section className="py-12 sm:py-14">
      <div className={`${container} grid grid-cols-2 gap-x-8 gap-y-8 sm:gap-y-10 lg:grid-cols-4`}>
        {stats.map((s) => (
          <div key={s.value}>
            <div className="text-2xl font-bold tracking-tight text-primary sm:text-3xl font-mono">{s.value}</div>
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
      title: "Stok tercatat beda dengan fisik",
      desc: "Buku catatan menunjukkan ada barang, tapi rak gudang kosong. Pembeli kecewa, pesanan terpaksa dibatalkan.",
    },
    {
      icon: Hourglass,
      title: "Antrean kasir menumpuk",
      desc: "Kasir lama mencari nama produk atau salah menghitung kembalian uang. Antrean panjang saat jam ramai toko.",
    },
    {
      icon: CalendarClock,
      title: "Overselling di toko online",
      desc: "Barang laku di toko fisik tapi stok di marketplace belum diubah. Penjual kena penalti pembatalan pesanan.",
    },
    {
      icon: FileSpreadsheet,
      title: "Surat jalan & opname manual",
      desc: "Surat jalan tulis tangan mudah hilang atau salah nomor armada. Audit stok opname harus tutup toko berhari-hari.",
    },
  ]
  return (
    <section className="bg-card py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Tantangan Operasional</p>
          <h2 className={`${sectionTitle} mt-3`}>Kelola stok dan kasir manual itu rawan selisih.</h2>
          <p className={`${sectionLead} mt-4`}>
            Waktu Anda habis untuk mencari barang di gudang atau mencocokkan nota manual. Tayooli merapikan
            alur dari penerimaan barang, kasir toko, hingga pengiriman.
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

function FeaturesSection() {
  return (
    <section id="fitur" className="scroll-mt-20 border-t border-border bg-muted/40 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>13 Modul Terpadu</p>
          <h2 className={`${sectionTitle} mt-3`}>
            Dari kasir toko, gudang, hingga surat jalan — semua terhubung.
          </h2>
          <p className={`${sectionLead} mt-4`}>
            Tayooli dirancang khusus untuk toko ritel, grosir, dan distributor di Indonesia yang butuh
            sistem cepat tanpa menu enterprise yang membingungkan.
          </p>
        </div>

        <div className="mt-14 space-y-16 sm:space-y-20">
          <FeatureRow
            eyebrowLabel="Kasir Cepat & Struk"
            title="Kasir POS toko dengan barcode scanner"
            desc="Layani transaksi pelanggan dengan cepat. Scan barcode kemasan, hitung kembalian otomatis, dan cetak struk thermal 58mm atau 80mm."
            bullets={[
              "Mendukung scan barcode via kamera HP/laptop dan scanner USB/Bluetooth",
              "Pilihan pembayaran fleksibel: Tunai, QRIS, dan Kartu Debit",
              "Saldo stok gudang langsung terpotong saat transaksi selesai",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-600/10 text-emerald-600">
                    <Store className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Point of Sale (POS)</span>
                  <span className="ml-auto rounded-full bg-emerald-50 text-emerald-700 px-2 py-0.5 text-[10px] font-medium">
                    Kasir Aktif
                  </span>
                </div>
                <div className="mt-4 space-y-2">
                  {[
                    { item: "Kopi Susu Gula Aren 250ml", qty: "2 pcs", val: "Rp 36.000" },
                    { item: "Minyak Goreng Sawit 2L", qty: "1 pouch", val: "Rp 34.000" },
                    { item: "Gula Pasir Kristal 1kg", qty: "2 pack", val: "Rp 29.000" },
                  ].map((f) => (
                    <div key={f.item} className="flex items-center justify-between rounded-md bg-background px-3 py-2">
                      <div className="min-w-0 pr-2">
                        <div className="text-xs font-medium text-foreground truncate">{f.item}</div>
                        <div className="text-[10px] text-muted-foreground">{f.qty}</div>
                      </div>
                      <span className="font-mono text-xs font-semibold text-foreground shrink-0">{f.val}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-3 flex items-center justify-between rounded-md bg-emerald-50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-900/30 px-3 py-2">
                  <span className="text-xs font-medium text-emerald-800 dark:text-emerald-200">Total Belanja (QRIS)</span>
                  <span className="font-mono text-xs font-bold text-emerald-700 dark:text-emerald-300">Rp 99.000</span>
                </div>
              </div>
            }
          />

          <FeatureRow
            eyebrowLabel="Logistik & Distribusi"
            title="Kelola multi-gudang dan terbitkan Surat Jalan (DO)"
            desc="Kendalikan pergerakan barang antar gudang cabang. Terbitkan dokumen Surat Jalan resmi berstandar ekspedisi Indonesia dalam format cetak rapi."
            bullets={[
              "Monitoring stok unit fisik per rak dan lokasi gudang",
              "Alur transfer stok cabang: Draft → Pending → In Transit → Received",
              "Cetak Surat Jalan Delivery Order lengkap nomor polisi dan nama kurir",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Truck className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Surat Jalan (DO)</span>
                </div>
                <div className="mt-4 space-y-2">
                  {[
                    { label: "Nomor DO", val: "DO-2026-0018" },
                    { label: "Tujuan Pengiriman", val: "Toko Sinar Jaya, Bekasi" },
                    { label: "Armada / Sopir", val: "B 9876 KDA (Sulaeman)" },
                    { label: "Total Muatan", val: "120 Karton" },
                  ].map((s) => (
                    <div key={s.label} className="flex items-center justify-between rounded-md bg-background px-3 py-2">
                      <span className="text-xs text-muted-foreground">{s.label}</span>
                      <span className="font-mono text-xs font-medium text-foreground">{s.val}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-3 flex items-center gap-2 rounded-md bg-primary/5 px-3 py-2">
                  <Check className="h-3.5 w-3.5 text-primary" />
                  <span className="text-[11px] text-muted-foreground">Siap cetak format resmi tanda terima</span>
                </div>
              </div>
            }
            reversed
          />

          <FeatureRow
            eyebrowLabel="Akurasi Inventori"
            title="Stock Opname berkala & pencatatan barang rusak"
            desc="Audit stok fisik tanpa perlu menghentikan penjualan toko. Rekonsiliasi selisih hitung otomatis, pisahkan barang cacat atau kadaluwarsa ke gudang scrap."
            bullets={[
              "Pencatatan opname fisik cepat per kategori atau per lorong rak",
              "Penyesuaian selisih otomatis tercatat di buku besar mutasi",
              "Pencatatan afkir/scrap agar stok yang rusak tidak ikut terjual",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <ClipboardCheck className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Audit Stock Opname</span>
                </div>
                <div className="mt-4 space-y-2">
                  <div className="rounded-md bg-background px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-muted-foreground">SKU: MAS-KOP-001</div>
                    <div className="mt-0.5 flex items-center justify-between">
                      <span className="text-xs text-foreground">Kopi Susu Gula Aren</span>
                      <span className="font-mono text-xs font-medium text-foreground">Sistem: 50 · Fisik: 48 (-2)</span>
                    </div>
                  </div>
                  <div className="rounded-md bg-background px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wide text-muted-foreground">SKU: MAS-OIL-002</div>
                    <div className="mt-0.5 flex items-center justify-between">
                      <span className="text-xs text-foreground">Minyak Goreng 2L</span>
                      <span className="font-mono text-xs font-medium text-foreground">Sistem: 30 · Fisik: 30 (Cocok)</span>
                    </div>
                  </div>
                </div>
                <div className="mt-3 flex items-center gap-2 rounded-md bg-success/5 px-3 py-2">
                  <Check className="h-3.5 w-3.5 text-success" />
                  <span className="text-[11px] text-muted-foreground">Selisih 2 botol dialihkan ke catatan scrap rusak</span>
                </div>
              </div>
            }
          />

          <FeatureRow
            eyebrowLabel="Omnichannel Ritel"
            title="Sinkronisasi stok dengan toko online"
            desc="Cegah overselling saat produk laku bersamaan di toko fisik dan marketplace. Saldo stok selalu termutakhirkan secara konsisten."
            bullets={[
              "Mendukung alokasi stok untuk Tokopedia, Shopee, TikTok, dan Lazada",
              "Peringatan stok menipis otomatis sebelum persediaan kosong",
              "Riwayat mutasi keluar masuk barang transparan dan rapi",
            ]}
            visual={
              <div className="rounded-xl border border-border bg-card p-5 shadow-card">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <ShoppingBag className="h-4 w-4" />
                  </div>
                  <span className="text-xs font-semibold text-foreground">Saluran Marketplace</span>
                </div>
                <div className="mt-4 space-y-2">
                  {[
                    { name: "Tokopedia", status: "Terhubung", color: "bg-success/10 text-success" },
                    { name: "Shopee", status: "Terhubung", color: "bg-success/10 text-success" },
                    { name: "TikTok Shop", status: "Aktif", color: "bg-primary/10 text-primary" },
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
      title: "Input master produk & gudang",
      desc: "Daftarkan SKU produk, harga modal, harga jual ritel, dan tentukan gudang atau lokasi penyimpanan fisik Anda.",
    },
    {
      n: "02",
      title: "Jalankan kasir POS & mutasi barang",
      desc: "Mulai transaksi belanja di toko fisik dengan scanner barcode dan terbitkan Surat Jalan resmi saat mengirim barang.",
    },
    {
      n: "03",
      title: "Pantau omzet & stok real-time",
      desc: "Stok gudang otomatis terpotong, laporan omzet harian langsung terangkum, dan alarm stok menipis siap mengingatkan Anda.",
    },
  ]
  return (
    <section id="cara-kerja" className="scroll-mt-20 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Cara Kerja</p>
          <h2 className={`${sectionTitle} mt-3`}>Mudah digunakan dalam tiga langkah.</h2>
          <p className={`${sectionLead} mt-4`}>
            Tidak butuh pelatihan berminggu-minggu. Tim kasir dan staf gudang Anda bisa langsung
            mengoperasikannya pada hari pertama.
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

function OperationalScenarios() {
  const items = [
    {
      title: "Toko Ritel & Grosir Fisik",
      desc: "Kasir melayani transaksi cepat dengan barcode scanner, menerima pembayaran tunai atau QRIS, mencetak nota belanja thermal, dan memotong stok toko secara langsung.",
      role: "Fokus: Kecepatan kasir & akurasi kas",
    },
    {
      title: "Gudang & Distribusi Barang",
      desc: "Menerima barang masuk dari pemasok, memisahkan barang rusak ke lokasi scrap, transfer antar cabang, dan menerbitkan Surat Jalan (DO) resmi pengiriman.",
      role: "Fokus: Kontrol fisik & dokumen jalan",
    },
    {
      title: "Penjualan Toko & Marketplace",
      desc: "Mengimpor pesanan penjualan multi-platform, memetakan SKU marketplace ke barang internal, dan audit fisik berkala lewat modul Stock Opname.",
      role: "Fokus: Pencegahan selisih rak & stok minus",
    },
  ]
  return (
    <section className="py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>Skenario Penggunaan</p>
          <h2 className={`${sectionTitle} mt-3`}>Cocok untuk berbagai model operasional fisik.</h2>
        </div>
        <div className="mt-12 grid gap-6 md:grid-cols-3">
          {items.map((t) => (
            <div key={t.title} className="flex flex-col rounded-xl border border-border bg-card p-6 shadow-card">
              <h3 className="text-base font-semibold text-foreground">{t.title}</h3>
              <p className="mt-3 flex-1 text-sm leading-relaxed text-muted-foreground">
                {t.desc}
              </p>
              <div className="mt-5 border-t border-border pt-4 text-xs font-medium text-primary">
                {t.role}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function Faq() {
  const items = [
    {
      q: "Apakah kasir POS bisa digunakan dengan alat barcode scanner?",
      a: "Bisa. Anda dapat menggunakan kamera bawaan perangkat atau alat barcode scanner laser eksternal (USB/Bluetooth) tipe plug-and-play tanpa driver tambahan.",
    },
    {
      q: "Format printer apa yang didukung untuk cetak struk kasir?",
      a: "Kasir POS mendukung pencetakan ke printer thermal ukuran standar 58mm dan 80mm secara langsung dari browser.",
    },
    {
      q: "Apakah Surat Jalan (DO) bisa langsung dicetak?",
      a: "Ya. Setiap Surat Jalan yang diterbitkan sudah memiliki layout cetak standar bisnis Indonesia lengkap dengan kolom nomor polisi kendaraan, nama sopir, dan tanda terima barang.",
    },
    {
      q: "Bagaimana cara kerja transfer stok antar gudang?",
      a: "Transfer stok memiliki alur jelas: pengajuan (Draft), persetujuan staf, barang dalam perjalanan (In Transit), hingga diterima dan diverifikasi oleh gudang tujuan.",
    },
    {
      q: "Bisa dicoba tanpa kartu kredit?",
      a: "Tentu. 14 hari gratis tanpa kartu kredit. Anda juga dapat masuk ke akun demo pengujian menggunakan admin@test.com / password123.",
    },
    {
      q: "Apakah data antar toko atau cabang terpisah aman?",
      a: "Ya. Tayooli menggunakan arsitektur multi-tenant dengan keamanan Row-Level Security di database PostgreSQL sehingga data antar akun terlindungi secara ketat.",
    },
  ]
  return (
    <section id="faq" className="scroll-mt-20 py-16 sm:py-20">
      <div className={container}>
        <div className="max-w-2xl">
          <p className={eyebrow}>FAQ</p>
          <h2 className={`${sectionTitle} mt-3`}>Pertanyaan yang sering diajukan.</h2>
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
              Operasional toko &amp; gudang lebih teratur.
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-background/70 sm:text-base">
              Coba gratis 14 hari tanpa kartu kredit. Kelola transaksi kasir, stok fisik, dan pengiriman barang Anda dalam satu sistem yang rapi.
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
      title: "Modul",
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
      title: "Aplikasi",
      links: [
        { label: "Kasir POS", href: "/login" },
        { label: "Gudang (WMS)", href: "/login" },
        { label: "Surat Jalan", href: "/login" },
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
              Sistem manajemen operasional ritel, multi-gudang (WMS), dan kasir POS untuk bisnis Indonesia.
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
          <p className="text-xs text-muted-foreground">&copy; 2026 Tayooli. Sistem Manajemen Ritel &amp; Pergudangan.</p>
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <ShieldCheck className="h-3.5 w-3.5 text-primary" />
            Data terisolasi per tenant · Keamanan RLS PostgreSQL
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
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-600" />
              Sistem Operasional Toko Ritel &amp; Pergudangan WMS
            </span>
            <h1 className="mt-5 text-4xl font-bold leading-[1.1] tracking-tight text-foreground sm:text-5xl lg:text-5xl xl:text-[3.4rem]">
              Stok gudang rapi, kasir cepat, pengiriman terkontrol.
            </h1>
            <p className="mx-auto mt-5 max-w-2xl text-base leading-relaxed text-muted-foreground sm:text-lg">
              Tayooli menyatukan transaksi kasir toko fisik (POS), manajemen stok multi-gudang,
              surat jalan (DO), dan sinkronisasi marketplace dalam satu alur kerja yang mudah digunakan.
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
              Tanpa kartu kredit · Batalkan kapan saja · Akun demo pengujian:{" "}
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
      <OperationalScenarios />
      <PricingSection />
      <Faq />
      <FinalCta />
      <Footer />
    </div>
  )
}
