"use client"

import React, { useState, useMemo, useEffect, useRef } from "react"
import Link from "next/link"
import QRCode from "qrcode"
import {
  Store,
  Search,
  Barcode as BarcodeIcon,
  ShoppingCart,
  Plus,
  Minus,
  Trash2,
  CreditCard,
  QrCode,
  Banknote,
  CheckCircle2,
  Printer,
  RotateCcw,
  Sparkles,
  Layers,
  ArrowRight,
  X,
  Volume2,
  Tag,
  ShieldCheck,
  AlertCircle,
  History,
  Clock,
  User,
  PauseCircle,
  PlayCircle,
  Loader2,
  Percent,
  Receipt,
  Share2,
  FileText,
} from "lucide-react"
import { useBarcodeScanner } from "@/hooks/useBarcodeScanner"
import { useProducts } from "@/hooks/useProducts"
import { usePOSOrders, usePOSCheckout, usePOSCreatePayment, usePOSPaymentStatus, usePOSSimulatePayment } from "@/hooks/usePOS"
import { useWMSStock } from "@/hooks/useWMSLedger"
import type { POSOrder, POSPaymentCharge } from "@/lib/api"
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"
import type { ExportColumn } from "@/lib/export"

// Sale Mode
type SaleMode = "JUAL_PUTUS" | "KONSINYASI"

interface CartItem {
  id: string
  name: string
  sku: string
  barcode: string
  price: number
  quantity: number
  discount: number
  saleMode: SaleMode
  consignmentVendor?: string
  stock: number
}

interface ProductItem {
  id: string
  name: string
  sku: string
  barcode: string
  price: number
  stock: number
  category: string
  isConsignment?: boolean
  vendorName?: string
}



export default function POSPage() {
  const { data: dbProducts = [] } = useProducts()
  const { data: wmsStock = [] } = useWMSStock()
  const { data: posOrders = [], refetch: refetchOrders } = usePOSOrders(30)
  const posCheckout = usePOSCheckout()

  // Map real stock by product_id
  const stockByProduct = useMemo(() => {
    const map = new Map<string, number>()
    wmsStock.forEach((s) => {
      const current = map.get(s.product_id) || 0
      map.set(s.product_id, current + Number(s.quantity))
    })
    return map
  }, [wmsStock])

  // Product catalog — ONLY the signed-in tenant's real products.
  // Never fall back to hardcoded/demo items or invented stock/price: a new or
  // empty tenant must see an empty catalog, not data that looks like another
  // tenant's (QA finding: "POS menampilkan data tenant lain").
  const catalog = useMemo<ProductItem[]>(() => {
    return (dbProducts ?? []).map((p) => ({
      id: p.id,
      name: p.name,
      sku: p.sku,
      barcode: (p as unknown as { barcode?: string }).barcode || "",
      price: Number(p.price) || 0,
      stock: stockByProduct.get(p.id) ?? 0,
      category: "Umum",
    }))
  }, [dbProducts, stockByProduct])

  // Current retail sales mode toggle
  const [globalSaleMode, setGlobalSaleMode] = useState<SaleMode>("JUAL_PUTUS")

  // Customer & Cart State
  const [customerName, setCustomerName] = useState<string>("Pelanggan Umum")
  const [cart, setCart] = useState<CartItem[]>([])
  const [discountPercent, setDiscountPercent] = useState<number>(0)
  const [applyTax, setApplyTax] = useState<boolean>(true)
  const [heldCarts, setHeldCarts] = useState<{ id: string; name: string; time: string; items: CartItem[] }[]>([])

  // Search & Filter
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedCategory, setSelectedCategory] = useState("Semua")
  const [scanBarInput, setScanBarInput] = useState("")
  const [scanError, setScanError] = useState<string | null>(null)

  // Modals & Payments
  const [showPaymentModal, setShowPaymentModal] = useState(false)
  const [showClearCartModal, setShowClearCartModal] = useState(false)
  const [showHistoryModal, setShowHistoryModal] = useState(false)
  const [exportingOrders, setExportingOrders] = useState(false)

  const posExportColumns: ExportColumn<POSOrder>[] = [
    { header: "No. Transaksi", value: (o) => o.order_number, width: 20 },
    {
      header: "Tanggal & Waktu",
      value: (o) =>
        new Date(o.created_at).toLocaleString("id-ID", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" }),
      width: 20,
    },
    { header: "Pelanggan", value: (o) => o.customer_name || "Walk-in", width: 22 },
    { header: "Metode Bayar", value: (o) => o.payment_method, width: 18 },
    { header: "Subtotal (Rp)", value: (o) => Math.round(Number(o.subtotal) || 0), width: 16, align: "right" },
    { header: "Pajak (Rp)", value: (o) => Math.round(Number(o.tax_amount) || 0), width: 14, align: "right" },
    { header: "Diskon (Rp)", value: (o) => Math.round(Number(o.discount_amount) || 0), width: 14, align: "right" },
    { header: "Total (Rp)", value: (o) => Math.round(Number(o.total_amount) || 0), width: 16, align: "right" },
    { header: "Status", value: (o) => o.status || "OK", width: 14 },
  ]
  const [showReconciliationModal, setShowReconciliationModal] = useState(false)
  const [receiptWidth, setReceiptWidth] = useState<"58mm" | "80mm">("80mm")
  const [checkoutError, setCheckoutError] = useState<string | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<"CASH" | "QRIS">("CASH")
  const [cashTendered, setCashTendered] = useState<number>(0)
  const [qrisDataUrl, setQrisDataUrl] = useState<string>("")
  // The live QRIS intent created by the backend (null until the cashier picks
  // QRIS). This is what the QR image encodes — never a fabricated payload.
  const [qrisCharge, setQrisCharge] = useState<POSPaymentCharge | null>(null)
  // Guards the auto-confirm effect so one settled payment triggers exactly one
  // checkout, even across re-renders.
  const autoConfirmRef = useRef<string | null>(null)
  const [isPaidSuccess, setIsPaidSuccess] = useState(false)
  const [receiptData, setReceiptData] = useState<{
    orderNumber: string
    date: string
    items: CartItem[]
    subtotal: number
    tax: number
    discount: number
    total: number
    paid: number
    change: number
    method: string
    saleMode: SaleMode
    salesOrderId?: string | null
    salesInvoiceId?: string | null
  } | null>(null)

  // Categories list
  const categories = useMemo(() => {
    const set = new Set<string>()
    set.add("Semua")
    catalog.forEach((p) => {
      if (p.category) set.add(p.category)
    })
    return Array.from(set)
  }, [catalog])

  // Filtered Products
  const filteredProducts = useMemo(() => {
    return catalog.filter((p) => {
      const matchCategory = selectedCategory === "Semua" || p.category === selectedCategory
      const matchSearch =
        searchQuery === "" ||
        p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        p.sku.toLowerCase().includes(searchQuery.toLowerCase()) ||
        p.barcode.includes(searchQuery)

      return matchCategory && matchSearch
    })
  }, [catalog, selectedCategory, searchQuery])

  // Add Item to Cart (or increment)
  const addToCart = (product: ProductItem) => {
    setCart((prev) => {
      const existingIndex = prev.findIndex((item) => item.id === product.id)
      if (existingIndex >= 0) {
        const copy = [...prev]
        copy[existingIndex].quantity += 1
        return copy
      } else {
        return [
          ...prev,
          {
            id: product.id,
            name: product.name,
            sku: product.sku,
            barcode: product.barcode,
            price: product.price,
            quantity: 1,
            discount: 0,
            saleMode: product.isConsignment ? "KONSINYASI" : globalSaleMode,
            consignmentVendor: product.vendorName,
            stock: product.stock,
          },
        ]
      }
    })
  }

  // Update item quantity in cart
  const updateQuantity = (id: string, delta: number) => {
    setCart((prev) => {
      return prev
        .map((item) => {
          if (item.id === id) {
            const newQty = item.quantity + delta
            return newQty > 0 ? { ...item, quantity: newQty } : null
          }
          return item
        })
        .filter(Boolean) as CartItem[]
    })
  }

  // Update item discount
  const updateItemDiscount = (id: string, discount: number) => {
    setCart((prev) =>
      prev.map((item) =>
        item.id === id ? { ...item, discount: Math.max(0, discount) } : item
      )
    )
  }

  // Hold current cart
  const handleHoldCart = () => {
    if (cart.length === 0) return
    const id = `HOLD-${Date.now()}`
    const time = new Date().toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" })
    setHeldCarts((prev) => [
      ...prev,
      {
        id,
        name: `${customerName} (${cart.length} item)`,
        time,
        items: [...cart],
      },
    ])
    setCart([])
  }

  // Resume held cart
  const handleResumeCart = (id: string) => {
    const target = heldCarts.find((c) => c.id === id)
    if (!target) return
    setCart(target.items)
    setHeldCarts((prev) => prev.filter((c) => c.id !== id))
  }

  // Reprint receipt from order history
  const handleReprintReceipt = (order: POSOrder) => {
    const receipt = {
      orderNumber: order.order_number,
      date: new Date(order.created_at).toLocaleString("id-ID"),
      items: (order.items || []).map((it) => ({
        id: it.product_id,
        name: it.product_name,
        sku: it.sku,
        barcode: "",
        price: Number(it.price),
        quantity: Number(it.quantity),
        discount: Number(it.discount),
        saleMode: (order.sale_mode as SaleMode) || "JUAL_PUTUS",
        stock: 0,
      })),
      subtotal: Number(order.subtotal),
      tax: Number(order.tax_amount),
      discount: Number(order.discount_amount),
      total: Number(order.total_amount),
      paid: Number(order.payment_amount),
      change: Number(order.change_amount),
      method: order.payment_method,
      saleMode: (order.sale_mode as SaleMode) || "JUAL_PUTUS",
      salesOrderId: order.sales_order_id,
      salesInvoiceId: order.sales_invoice_id,
    }
    setReceiptData(receipt)
    setIsPaidSuccess(true)
    setShowPaymentModal(true)
    setShowHistoryModal(false)
  }

  // Remove item from cart
  const removeFromCart = (id: string) => {
    setCart((prev) => prev.filter((item) => item.id !== id))
  }

  // Clear all cart (in-app modal confirmation)
  const clearCart = () => {
    if (cart.length > 0) {
      setShowClearCartModal(true)
    }
  }

  const confirmClearCart = () => {
    setCart([])
    setShowClearCartModal(false)
  }

  // Hardware Scanner Integration
  const handleHardwareScan = (barcode: string) => {
    const clean = barcode.trim()
    const found = catalog.find(
      (p) =>
        (p.barcode !== "" && p.barcode === clean) ||
        p.sku.toLowerCase() === clean.toLowerCase() ||
        p.name.toLowerCase().includes(clean.toLowerCase())
    )

    if (found) {
      setScanError(null)
      addToCart(found)
    } else {
      // Unknown code: do NOT invent a fake product (its non-UUID id would also
      // make checkout fail). Tell the cashier instead.
      setScanError(`Barang dengan kode "${clean}" tidak ditemukan di master produk.`)
    }
  }

  const { triggerScan } = useBarcodeScanner({
    onScan: handleHardwareScan,
    soundFeedback: true,
    hapticFeedback: true,
  })

  // Manual Scan Input Submit
  const handleScanBarSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!scanBarInput.trim()) return
    triggerScan(scanBarInput.trim())
    setScanBarInput("")
  }

  // Cart Calculations
  const subtotal = useMemo(() => {
    return cart.reduce((sum, item) => sum + (item.price * item.quantity - (item.discount || 0)), 0)
  }, [cart])

  const discountAmount = useMemo(() => {
    return Math.floor((subtotal * discountPercent) / 100)
  }, [subtotal, discountPercent])

  const taxAmount = useMemo(() => {
    if (!applyTax) return 0
    return Math.floor((subtotal - discountAmount) * 0.11)
  }, [subtotal, discountAmount, applyTax])

  const grandTotal = useMemo(() => {
    return Math.max(0, subtotal - discountAmount + taxAmount)
  }, [subtotal, discountAmount, taxAmount])

  // Change calculation
  const changeAmount = useMemo(() => {
    return Math.max(0, cashTendered - grandTotal)
  }, [cashTendered, grandTotal])

  // Open Payment Modal.
  // Cash opens instantly. QRIS first creates a payment intent on the backend
  // (real gateway QR, or demo mode when the tenant has no credentials) and
  // renders the QR the server returned — there is no fabricated payload.
  const createPayment = usePOSCreatePayment()
  const simulatePayment = usePOSSimulatePayment()

  const [activePaymentOrderId, setActivePaymentOrderId] = useState<string | null>(null)

  const paymentStatus = usePOSPaymentStatus(activePaymentOrderId, {
    enabled: paymentMethod === "QRIS" && activePaymentOrderId != null && !isPaidSuccess,
  })

  const handleOpenPayment = async () => {
    if (cart.length === 0) return
    setCheckoutError(null)
    setCashTendered(grandTotal)
    setIsPaidSuccess(false)
    setQrisCharge(null)
    setActivePaymentOrderId(null)
    setQrisDataUrl("")

    if (paymentMethod === "QRIS") {
      try {
        const charge = await createPayment.mutateAsync({ amount: grandTotal, method: "QRIS" })
        setQrisCharge(charge)
        setActivePaymentOrderId(charge.order_id)
        const payload = charge.qr_string || charge.payment_link || charge.order_id
        const url = await QRCode.toDataURL(payload, { width: 280, margin: 1 })
        setQrisDataUrl(url)
      } catch (err: any) {
        setCheckoutError(
          err?.message || "Gagal membuat QRIS. Coba lagi atau gunakan pembayaran tunai."
        )
      }
    }

    setShowPaymentModal(true)
  }

  // Auto-confirm: once the gateway (or demo simulation) settles the payment,
  // issue the receipt exactly once — the checkout claims the payment by its
  // order id, and the backend consumes it so it can never mint two sales.
  const handleConfirmPayment = async (settledOrderId?: string) => {
    const orderId = settledOrderId ?? qrisCharge?.order_id
    if (paymentMethod === "QRIS") {
      if (!orderId) {
        setCheckoutError("Pembayaran QRIS belum siap. Tunggu QR tampil lalu coba lagi.")
        return
      }
      if (autoConfirmRef.current === orderId) return
      autoConfirmRef.current = orderId
    }
    setCheckoutError(null)
    try {
      const res = await posCheckout.mutateAsync({
        items: cart.map((c) => ({
          product_id: c.id,
          qty: c.quantity,
          price: c.price,
          discount: c.discount || 0,
        })),
        payments: [
          {
            method: paymentMethod,
            amount: paymentMethod === "CASH" ? cashTendered : grandTotal,
          },
        ],
        payment_order_id: paymentMethod === "QRIS" ? orderId : undefined,
        tax: taxAmount,
        discount: discountAmount,
        sale_mode: globalSaleMode,
      })

      const receipt = {
        orderNumber: res.order_number,
        date: new Date(res.created_at).toLocaleString("id-ID"),
        items: [...cart],
        subtotal: Number(res.subtotal) || subtotal,
        tax: Number(res.tax) || taxAmount,
        discount: Number(res.discount) || discountAmount,
        total: Number(res.total) || grandTotal,
        paid: paymentMethod === "CASH" ? cashTendered : grandTotal,
        change: Number(res.change) || (paymentMethod === "CASH" ? changeAmount : 0),
        method: paymentMethod,
        saleMode: globalSaleMode,
        salesOrderId: res.sales_order_id,
        salesInvoiceId: res.sales_invoice_id,
      }
      setReceiptData(receipt)
      setIsPaidSuccess(true)
      setCart([])
      refetchOrders()
    } catch (err: any) {
      if (paymentMethod === "QRIS") {
        autoConfirmRef.current = null
      }
      setCheckoutError(
        err?.message || "Gagal memproses pembayaran POS. Silakan periksa koneksi atau ketersediaan stok barang."
      )
    }
  }

  // When the polled status first hits a terminal value, drive the outcome
  // without the cashier having to tap:
  //   completed/consumed → write the sale (one payment, one sale),
  //   expired/failed     → tell them the QR is no longer payable.
  useEffect(() => {
    const status = paymentStatus.data?.status
    const orderId = paymentStatus.data?.order_id
    if (!status || !orderId || showPaymentModal === false || isPaidSuccess) return
    if (status === "completed" || status === "consumed") {
      void handleConfirmPayment(orderId)
    } else if (status === "expired" || status === "failed") {
      setCheckoutError(
        status === "expired"
          ? "Waktu pembayaran QRIS habis. Buat QR baru."
          : "Pembayaran QRIS gagal. Coba lagi atau gunakan tunai."
      )
      autoConfirmRef.current = null
    }
    // handleConfirmPayment intentionally not in deps (captures fresh cart)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paymentStatus.data?.status, paymentStatus.data?.order_id, showPaymentModal, isPaidSuccess])

  // Kasir Z-Report Reconciliation Summary
  const reconciliationData = useMemo(() => {
    let cashTotal = 0
    let qrisTotal = 0
    let grandRevenue = 0
    let cashCount = 0
    let qrisCount = 0

    const todayKey = new Date().toDateString()
    const todayOrders = posOrders.filter(
      (o) => new Date(o.created_at).toDateString() === todayKey
    )

    todayOrders.forEach((o) => {
      const total = Number(o.total_amount) || 0
      grandRevenue += total
      const method = (o.payment_method || "").toUpperCase()
      if (method === "CASH") {
        cashTotal += total
        cashCount++
      } else {
        qrisTotal += total
        qrisCount++
      }
    })

    return {
      orders: todayOrders,
      totalOrders: todayOrders.length,
      cashTotal,
      cashCount,
      qrisTotal,
      qrisCount,
      grandRevenue,
    }
  }, [posOrders])

  // Share Receipt via WhatsApp
  const handleShareWhatsApp = () => {
    if (!receiptData) return
    const itemsList = receiptData.items
      .map((i) => `• ${i.name} (${i.quantity}x) = Rp ${(i.price * i.quantity).toLocaleString("id-ID")}`)
      .join("\n")
    const text =
      `*STRUK PEMBAYARAN TAYOOLI POS*\n` +
      `No. Transaksi: ${receiptData.orderNumber}\n` +
      `Waktu: ${receiptData.date}\n` +
      `Metode: ${receiptData.method}\n` +
      `--------------------------------\n` +
      `${itemsList}\n` +
      `--------------------------------\n` +
      `Subtotal: Rp ${receiptData.subtotal.toLocaleString("id-ID")}\n` +
      (receiptData.discount > 0 ? `Diskon: -Rp ${receiptData.discount.toLocaleString("id-ID")}\n` : "") +
      `PPN: Rp ${receiptData.tax.toLocaleString("id-ID")}\n` +
      `*TOTAL: Rp ${receiptData.total.toLocaleString("id-ID")}*\n` +
      `Bayar: Rp ${receiptData.paid.toLocaleString("id-ID")}\n` +
      `Kembali: Rp ${receiptData.change.toLocaleString("id-ID")}\n` +
      `--------------------------------\n` +
      `*Status: LUNAS*\n` +
      `Terima kasih atas kunjungan Anda!`
    window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, "_blank")
  }

  // Print Receipt
  const handlePrintReceipt = () => {
    window.print()
  }

  return (
    <div className="min-h-screen bg-[#F8FAFC] flex flex-col">
      {/* ── Top POS Kasir Header ── */}
      <div className="bg-white border-b border-[#E2E8F0] px-4 sm:px-6 py-3.5 sticky top-0 z-30 shadow-xs">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-[#EFF6FF] border border-[#BFDBFE] text-[#2563EB]">
              <Store className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-xl font-extrabold tracking-tight text-slate-900">
                  Point of Sale (POS Kasir)
                </h1>
                <span className="inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded bg-emerald-100 text-emerald-800">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  ONLINE
                </span>
              </div>
              <p className="text-xs text-slate-500">
                Mode Penjualan Odoo/OCA pattern & Integrasi Master Produk & Stok Real-Time
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            {/* Export Button */}
            <ExportButton onClick={() => setExportingOrders(true)} disabled={posOrders.length === 0} />

            {/* Riwayat Transaksi Button */}
            <button
              type="button"
              onClick={() => setShowHistoryModal(true)}
              className="px-3 py-2 rounded-xl text-xs font-bold border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 flex items-center gap-1.5 transition shadow-xs"
            >
              <History className="w-4 h-4 text-[#2563EB]" />
              <span>Riwayat Transaksi</span>
              {posOrders.length > 0 && (
                <span className="px-1.5 py-0.5 rounded-full bg-blue-100 text-blue-700 text-[10px]">
                  {posOrders.length}
                </span>
              )}
            </button>

            {/* Rekonsiliasi Kasir (Z-Report) Button */}
            <button
              type="button"
              onClick={() => setShowReconciliationModal(true)}
              className="px-3 py-2 rounded-xl text-xs font-bold border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 flex items-center gap-1.5 transition shadow-xs"
              title="Laporan Rekonsiliasi Kasir & Penutupan Kas"
            >
              <Receipt className="w-4 h-4 text-emerald-600" />
              <span>Rekonsiliasi Kas</span>
            </button>

            {/* Sales Mode Switcher (Jual Putus vs Konsinyasi) */}
            <div className="flex items-center gap-2 bg-slate-100 p-1 rounded-xl border border-slate-200">
              <button
                type="button"
                onClick={() => setGlobalSaleMode("JUAL_PUTUS")}
                className={`px-3 py-1.5 rounded-lg text-xs font-bold transition-all min-h-[38px] flex items-center gap-1.5 ${
                  globalSaleMode === "JUAL_PUTUS"
                    ? "bg-[#2563EB] text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900"
                }`}
              >
                <Store className="w-3.5 h-3.5" />
                <span>Jual Putus</span>
              </button>
              <button
                type="button"
                onClick={() => setGlobalSaleMode("KONSINYASI")}
                className={`px-3 py-1.5 rounded-lg text-xs font-bold transition-all min-h-[38px] flex items-center gap-1.5 ${
                  globalSaleMode === "KONSINYASI"
                    ? "bg-[#EA580C] text-white shadow-xs"
                    : "text-slate-600 hover:text-slate-900"
                }`}
              >
                <Tag className="w-3.5 h-3.5" />
                <span>Konsinyasi</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* ── Main Layout: Catalog (Left) + Cart Sidebar (Right) ── */}
      <div className="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* ── LEFT COLUMN: Product Catalog & Search (Col 7 or 8) ── */}
        <div className="lg:col-span-7 xl:col-span-8 flex flex-col space-y-4">
          {/* Quick Barcode Scan Input Bar */}
          <div className="bg-white p-3.5 rounded-xl border border-[#E2E8F0] shadow-xs space-y-2">
            <form onSubmit={handleScanBarSubmit} className="flex gap-2">
              <div className="relative flex-1">
                <BarcodeIcon className="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-[#2563EB]" />
                <input
                  type="text"
                  value={scanBarInput}
                  onChange={(e) => {
                    setScanBarInput(e.target.value)
                    setScanError(null)
                  }}
                  placeholder="Scan barcode dengan USB Scanner Gun atau ketik SKU..."
                  className="w-full pl-11 pr-4 min-h-[48px] text-sm bg-slate-50 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB] font-mono"
                />
              </div>
              <button
                type="submit"
                className="px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-xs"
              >
                + Masuk Keranjang
              </button>
            </form>
            {scanError && (
              <div className="px-3 py-2 rounded-lg bg-amber-50 border border-amber-200 text-amber-800 text-xs font-semibold flex items-start gap-2">
                <AlertCircle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                <span>{scanError}</span>
              </div>
            )}
          </div>

          {/* Search & Category Filter Chips */}
          <div className="space-y-2">
            <div className="relative">
              <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Cari nama barang atau SKU..."
                className="w-full pl-10 pr-4 min-h-[44px] text-sm bg-white border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
              />
            </div>

            {/* Category Chips */}
            <div className="flex items-center gap-1.5 overflow-x-auto pb-1">
              {categories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setSelectedCategory(cat)}
                  className={`px-3.5 py-2 text-xs font-semibold rounded-lg transition-all min-h-[40px] whitespace-nowrap ${
                    selectedCategory === cat
                      ? "bg-[#2563EB] text-white shadow-xs"
                      : "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50"
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          {catalog.length === 0 && (
            <div className="bg-white rounded-xl border border-dashed border-slate-300 p-8 text-center space-y-2">
              <Layers className="w-10 h-10 mx-auto text-slate-300" />
              <div className="text-sm font-bold text-slate-700">Belum ada produk</div>
              <p className="text-xs text-slate-500">
                Tambahkan barang di master produk terlebih dahulu agar bisa dijual di kasir.
              </p>
              <Link
                href="/products"
                className="inline-flex items-center gap-1.5 mt-2 px-4 py-2 rounded-lg bg-[#2563EB] text-white text-xs font-bold hover:bg-[#1D4ED8]"
              >
                Buka Master Produk
                <ArrowRight className="w-3.5 h-3.5" />
              </Link>
            </div>
          )}

          {/* Products Grid */}
          <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-4 gap-3">
            {filteredProducts.map((product) => {
              const inCartItem = cart.find((c) => c.id === product.id)
              return (
                <button
                  type="button"
                  key={product.id}
                  onClick={() => addToCart(product)}
                  className="bg-white rounded-xl border border-[#E2E8F0] hover:border-[#BFDBFE] hover:shadow-md p-3.5 text-left transition-all flex flex-col justify-between group active:scale-98 relative min-h-[140px]"
                >
                  <div>
                    {/* Consignment Tag */}
                    {product.isConsignment && (
                      <span className="inline-block text-[10px] font-bold px-1.5 py-0.5 rounded bg-amber-100 text-amber-800 mb-1 border border-amber-200">
                        KONSINYASI
                      </span>
                    )}
                    <h3 className="font-bold text-sm text-slate-900 group-hover:text-[#2563EB] line-clamp-2 leading-tight">
                      {product.name}
                    </h3>
                    <div className="font-mono text-[11px] text-slate-400 mt-1">
                      {product.sku}
                    </div>
                  </div>

                  <div className="mt-3 pt-2 border-t border-slate-100 flex items-end justify-between gap-3 min-w-0">
                    <div className="min-w-0">
                      <div className="text-xs text-slate-400">Harga</div>
                      <div className="font-mono font-extrabold text-sm sm:text-base text-slate-900 truncate">
                        Rp {product.price.toLocaleString("id-ID")}
                      </div>
                    </div>

                    <div className="flex flex-col items-end shrink-0">
                      <span
                        className={`text-[10px] font-semibold ${
                          product.stock < 20 ? "text-amber-600" : "text-emerald-600"
                        }`}
                      >
                        Stok: {product.stock}
                      </span>
                      {inCartItem && (
                        <span className="mt-1 px-2 py-0.5 text-xs font-bold rounded-full bg-[#EFF6FF] text-[#2563EB] border border-[#BFDBFE]">
                          {inCartItem.quantity}x
                        </span>
                      )}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        </div>

        {/* ── RIGHT COLUMN: Cart Panel & Checkout (Col 5 or 4) ── */}
        <div className="lg:col-span-5 xl:col-span-4 flex flex-col bg-white rounded-2xl border border-[#E2E8F0] shadow-sm overflow-hidden h-[calc(100vh-140px)] sticky top-20">
          {/* Cart Header */}
          <div className="p-3.5 border-b border-[#E2E8F0] space-y-2 bg-slate-50/70">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ShoppingCart className="w-5 h-5 text-[#2563EB]" />
                <h2 className="font-bold text-sm text-slate-900">
                  Keranjang Kasir ({cart.reduce((s, i) => s + i.quantity, 0)})
                </h2>
              </div>
              <div className="flex items-center gap-1.5">
                {cart.length > 0 && (
                  <button
                    type="button"
                    onClick={handleHoldCart}
                    title="Tahan transaksi saat ini"
                    className="text-xs font-semibold text-amber-700 bg-amber-50 hover:bg-amber-100 border border-amber-200 px-2 py-1 rounded-md flex items-center gap-1"
                  >
                    <PauseCircle className="w-3.5 h-3.5" />
                    <span>Tahan</span>
                  </button>
                )}
                {cart.length > 0 && (
                  <button
                    type="button"
                    onClick={clearCart}
                    className="text-xs font-semibold text-rose-600 hover:text-rose-800 px-1.5 py-1"
                  >
                    Kosongkan
                  </button>
                )}
              </div>
            </div>

            {/* Customer Input & Held Carts Selector */}
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <User className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  value={customerName}
                  onChange={(e) => setCustomerName(e.target.value)}
                  placeholder="Nama Pelanggan / Walk-in..."
                  className="w-full pl-8 pr-2 py-1 text-xs bg-white border border-slate-200 rounded-md focus:ring-1 focus:ring-[#2563EB]"
                />
              </div>

              {heldCarts.length > 0 && (
                <div className="relative">
                  <select
                    onChange={(e) => {
                      if (e.target.value) handleResumeCart(e.target.value)
                    }}
                    value=""
                    className="text-xs font-bold bg-amber-100 border border-amber-300 text-amber-900 rounded-md px-2 py-1 cursor-pointer"
                  >
                    <option value="" disabled>
                      Resume ({heldCarts.length})
                    </option>
                    {heldCarts.map((h) => (
                      <option key={h.id} value={h.id}>
                        {h.time} - {h.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>
          </div>

          {/* Cart Items List */}
          <div className="flex-1 overflow-y-auto p-4 space-y-3 divide-y divide-slate-100">
            {cart.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-center p-6 text-slate-400 space-y-3">
                <ShoppingCart className="w-12 h-12 stroke-[1.2]" />
                <div className="text-sm font-semibold text-slate-700">Keranjang Masih Kosong</div>
                <p className="text-xs max-w-xs">
                  Scan barcode barang atau klik produk di katalog sebelah kiri untuk memulai transaksi kasir.
                </p>
              </div>
            ) : (
              cart.map((item) => {
                const lineTotal = item.price * item.quantity - (item.discount || 0)
                return (
                  <div key={item.id} className="pt-3 first:pt-0 space-y-1.5">
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="font-semibold text-xs text-slate-900 line-clamp-1">
                            {item.name}
                          </span>
                          {item.saleMode === "KONSINYASI" && (
                            <span className="text-[9px] font-bold px-1.5 py-0.2 rounded bg-amber-100 text-amber-900 shrink-0">
                              KONSINYASI
                            </span>
                          )}
                        </div>
                        <div className="font-mono text-[11px] text-slate-500 mt-0.5">
                          @ Rp {item.price.toLocaleString("id-ID")}
                        </div>
                      </div>

                      {/* Quantity Controller & Delete */}
                      <div className="flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => updateQuantity(item.id, -1)}
                          className="p-1 rounded-md border border-slate-300 text-slate-600 hover:bg-slate-100 min-h-[32px] min-w-[32px] flex items-center justify-center"
                        >
                          <Minus className="w-3.5 h-3.5" />
                        </button>
                        <span className="w-7 text-center font-mono font-bold text-xs">
                          {item.quantity}
                        </span>
                        <button
                          type="button"
                          onClick={() => updateQuantity(item.id, 1)}
                          className="p-1 rounded-md border border-slate-300 text-slate-600 hover:bg-slate-100 min-h-[32px] min-w-[32px] flex items-center justify-center"
                        >
                          <Plus className="w-3.5 h-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => removeFromCart(item.id)}
                          className="p-1 rounded-md text-slate-400 hover:text-rose-600 hover:bg-rose-50 min-h-[32px] min-w-[32px] flex items-center justify-center ml-1"
                          title="Hapus baris"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>

                      {/* Total Line Price */}
                      <div className="font-mono font-bold text-xs text-slate-900 text-right w-20 shrink-0">
                        Rp {Math.max(0, lineTotal).toLocaleString("id-ID")}
                      </div>
                    </div>

                    {/* Per-item discount input */}
                    <div className="flex items-center justify-end gap-1.5 text-[11px] text-slate-500">
                      <Percent className="w-3 h-3 text-slate-400" />
                      <span>Diskon item (Rp):</span>
                      <input
                        type="number"
                        min="0"
                        value={item.discount || ""}
                        onChange={(e) => updateItemDiscount(item.id, parseInt(e.target.value) || 0)}
                        placeholder="0"
                        className="w-16 px-1.5 py-0.5 text-right font-mono text-xs border border-slate-200 rounded focus:ring-1 focus:ring-[#2563EB]"
                      />
                    </div>
                  </div>
                )
              })
            )}
          </div>

          {/* Cart Footer & Calculations */}
          <div className="p-4 bg-slate-50 border-t border-[#E2E8F0] space-y-3">
            {/* Discount & Tax controls */}
            <div className="flex items-center justify-between text-xs gap-3">
              <div className="flex items-center gap-2">
                <span className="text-slate-500">Diskon:</span>
                <select
                  value={discountPercent}
                  onChange={(e) => setDiscountPercent(parseInt(e.target.value))}
                  className="px-2 py-1 rounded border border-slate-300 bg-white font-semibold text-xs"
                >
                  <option value={0}>0%</option>
                  <option value={5}>5%</option>
                  <option value={10}>10%</option>
                  <option value={15}>15%</option>
                </select>
              </div>

              <label className="flex items-center gap-1.5 cursor-pointer text-slate-600">
                <input
                  type="checkbox"
                  checked={applyTax}
                  onChange={(e) => setApplyTax(e.target.checked)}
                  className="rounded text-[#2563EB]"
                />
                <span>PPN 11%</span>
              </label>
            </div>

            {/* Calculations Breakdown */}
            <div className="space-y-1.5 text-xs text-slate-600 pt-2 border-t border-slate-200">
              <div className="flex justify-between">
                <span>Subtotal:</span>
                <span className="font-mono">Rp {subtotal.toLocaleString("id-ID")}</span>
              </div>
              {discountAmount > 0 && (
                <div className="flex justify-between text-emerald-600 font-semibold">
                  <span>Potongan Diskon:</span>
                  <span className="font-mono">- Rp {discountAmount.toLocaleString("id-ID")}</span>
                </div>
              )}
              {applyTax && (
                <div className="flex justify-between">
                  <span>PPN 11%:</span>
                  <span className="font-mono">Rp {taxAmount.toLocaleString("id-ID")}</span>
                </div>
              )}
              <div className="flex justify-between text-sm sm:text-base font-extrabold text-slate-900 pt-2 border-t border-slate-200">
                <span>Total Belanja:</span>
                <span className="font-mono text-[#2563EB]">
                  Rp {grandTotal.toLocaleString("id-ID")}
                </span>
              </div>
            </div>

            {/* Pay Button (min 48px touch target) */}
            <button
              type="button"
              disabled={cart.length === 0}
              onClick={handleOpenPayment}
              className="w-full min-h-[48px] rounded-xl font-extrabold text-sm sm:text-base bg-[#EA580C] hover:bg-[#C2410C] active:scale-95 text-white flex items-center justify-center gap-2 transition-all shadow-md disabled:opacity-50 disabled:pointer-events-none"
            >
              <Banknote className="w-5 h-5" />
              <span>Bayar Sekarang (Rp {grandTotal.toLocaleString("id-ID")})</span>
            </button>
          </div>
        </div>
      </div>

      {/* ── MODAL: Payment (Cash / QRIS) ── */}
      {showPaymentModal && (
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150">
            {!isPaidSuccess ? (
              <>
                <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                  <h3 className="text-lg font-bold text-slate-900">Pembayaran Kasir</h3>
                  <button
                    onClick={() => setShowPaymentModal(false)}
                    className="p-2 text-slate-400 hover:text-slate-700 min-h-[44px] min-w-[44px] flex items-center justify-center"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>

                <div className="mt-4 p-3 bg-blue-50 rounded-xl text-center">
                  <div className="text-xs text-[#2563EB] uppercase font-bold tracking-wider">
                    Total Yang Harus Dibayar
                  </div>
                  <div className="text-2xl sm:text-3xl font-mono font-black text-slate-900 mt-1">
                    Rp {grandTotal.toLocaleString("id-ID")}
                  </div>
                </div>

                {/* Method Tabs */}
                <div className="mt-4 grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setPaymentMethod("CASH")}
                    className={`py-3 px-3 rounded-xl border flex items-center justify-center gap-2 font-bold text-sm min-h-[48px] transition-all ${
                      paymentMethod === "CASH"
                        ? "border-[#2563EB] bg-[#EFF6FF] text-[#2563EB]"
                        : "border-slate-200 text-slate-600 hover:bg-slate-50"
                    }`}
                  >
                    <Banknote className="w-4 h-4" />
                    <span>Uang Tunai (Cash)</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => setPaymentMethod("QRIS")}
                    className={`py-3 px-3 rounded-xl border flex items-center justify-center gap-2 font-bold text-sm min-h-[48px] transition-all ${
                      paymentMethod === "QRIS"
                        ? "border-[#2563EB] bg-[#EFF6FF] text-[#2563EB]"
                        : "border-slate-200 text-slate-600 hover:bg-slate-50"
                    }`}
                  >
                    <QrCode className="w-4 h-4" />
                    <span>QRIS Dinamis</span>
                  </button>
                </div>

                {/* CASH MODE */}
                {paymentMethod === "CASH" && (
                  <div className="mt-4 space-y-4">
                    <div>
                      <label className="block text-xs font-semibold text-slate-600 mb-1">
                        Uang Tunai Diterima (Rp)
                      </label>
                      <input
                        type="number"
                        min="0"
                        value={cashTendered || ""}
                        onChange={(e) => setCashTendered(parseInt(e.target.value) || 0)}
                        className="w-full px-3.5 py-2.5 min-h-[48px] text-lg font-mono font-bold border border-slate-300 rounded-xl focus:ring-2 focus:ring-[#2563EB]"
                      />
                    </div>

                    {/* Quick Denominations (min 48px touch target) */}
                    <div className="grid grid-cols-4 gap-2">
                      {[
                        { label: "Pas", val: grandTotal },
                        { label: "20 rb", val: 20000 },
                        { label: "50 rb", val: 50000 },
                        { label: "100 rb", val: 100000 },
                      ].map((den) => (
                        <button
                          key={den.label}
                          type="button"
                          onClick={() => setCashTendered(den.val)}
                          className="py-2 px-2 text-xs font-bold rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 min-h-[48px]"
                        >
                          {den.label}
                        </button>
                      ))}
                    </div>

                    {/* Change calculator display */}
                    <div
                      className={`p-4 rounded-xl border flex items-center justify-between ${
                        cashTendered >= grandTotal
                          ? "bg-emerald-50 border-emerald-200 text-emerald-900"
                          : "bg-rose-50 border-rose-200 text-rose-900"
                      }`}
                    >
                      <span className="text-xs font-semibold">
                        {cashTendered >= grandTotal ? "Kembalian (Change):" : "Uang Kurang:"}
                      </span>
                      <span className="font-mono font-black text-lg">
                        Rp {Math.abs(cashTendered - grandTotal).toLocaleString("id-ID")}
                      </span>
                    </div>

                    {checkoutError && (
                      <div className="p-3 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs flex items-start gap-2">
                        <AlertCircle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
                        <div className="flex-1 font-semibold">{checkoutError}</div>
                      </div>
                    )}

                    <button
                      type="button"
                      disabled={cashTendered < grandTotal || posCheckout.isPending}
                      onClick={() => handleConfirmPayment()}
                      className="w-full min-h-[48px] rounded-xl font-bold text-sm bg-emerald-600 hover:bg-emerald-700 text-white disabled:opacity-50 flex items-center justify-center gap-2"
                    >
                      {posCheckout.isPending ? (
                        <>
                          <Loader2 className="w-4 h-4 animate-spin" />
                          <span>Menyimpan Transaksi...</span>
                        </>
                      ) : (
                        <span>Konfirmasi Pembayaran Tunai</span>
                      )}
                    </button>
                  </div>
                )}

                {/* QRIS MODE */}
                {paymentMethod === "QRIS" && (
                  <div className="mt-4 text-center space-y-3">
                    <div className="bg-white p-3 rounded-xl border border-slate-200 inline-block shadow-sm">
                      {qrisDataUrl ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={qrisDataUrl}
                          alt="QRIS Dinamis"
                          className="w-56 h-56 mx-auto object-contain"
                        />
                      ) : (
                        <div className="w-56 h-56 flex items-center justify-center text-xs text-slate-400">
                          Membuat QRIS...
                        </div>
                      )}
                    </div>

                    {/* Live status: the server owns the truth, the cashier
                        just watches it. No manual "mark as paid". */}
                    <div
                      className={`p-3 rounded-xl border text-xs flex items-center gap-2 justify-center font-semibold ${
                        paymentStatus.data?.status === "completed" ||
                        paymentStatus.data?.status === "consumed"
                          ? "bg-emerald-50 border-emerald-200 text-emerald-800"
                          : paymentStatus.data?.status === "expired" ||
                              paymentStatus.data?.status === "failed"
                            ? "bg-rose-50 border-rose-200 text-rose-800"
                            : "bg-blue-50 border-blue-200 text-blue-800"
                      }`}
                    >
                      {(paymentStatus.data?.status === "completed" ||
                        paymentStatus.data?.status === "consumed") ? (
                        <>
                          <CheckCircle2 className="w-4 h-4" />
                          <span>Pembayaran diterima — menyimpan transaksi...</span>
                        </>
                      ) : paymentStatus.data?.status === "expired" ||
                        paymentStatus.data?.status === "failed" ? (
                        <>
                          <AlertCircle className="w-4 h-4" />
                          <span>QRIS tidak lagi bisa dibayar</span>
                        </>
                      ) : (
                        <>
                          <Loader2 className="w-4 h-4 animate-spin" />
                          <span>Menunggu pembayaran masuk...</span>
                        </>
                      )}
                    </div>

                    <p className="text-xs text-slate-500">
                      Scan dengan aplikasi BCA, Mandiri, GoPay, OVO, atau ShopeePay
                    </p>

                    {qrisCharge && (
                      <p className="text-[10px] text-slate-400 font-mono break-all">
                        {qrisCharge.order_id} · Berlaku s/d{" "}
                        {new Date(qrisCharge.expires_at).toLocaleTimeString("id-ID")}
                      </p>
                    )}

                    {/* Demo mode: no merchant account is connected yet, so the
                        cashier may simulate settlement. The backend refuses
                        this as soon as the tenant configures real credentials. */}
                    {qrisCharge?.demo && (
                      <div className="p-3 rounded-xl bg-amber-50 border border-amber-200 text-left space-y-2">
                        <div className="text-[11px] font-bold text-amber-900 flex items-center gap-1.5">
                          <ShieldCheck className="w-4 h-4" />
                          MODE DEMO — gateway pembayaran belum dikonfigurasi
                        </div>
                        <p className="text-[11px] text-amber-800">
                          Hubungkan akun Midtrans milik Anda di Pengaturan → Pembayaran agar QRIS
                          asli dan dana masuk langsung ke rekening Anda.
                        </p>
                        <button
                          type="button"
                          data-testid="pos-simulate-payment"
                          disabled={simulatePayment.isPending || posCheckout.isPending}
                          onClick={() => qrisCharge && simulatePayment.mutate(qrisCharge.order_id)}
                          className="w-full min-h-[44px] rounded-lg font-bold text-xs bg-amber-600 hover:bg-amber-700 text-white flex items-center justify-center gap-2 disabled:opacity-50"
                        >
                          {simulatePayment.isPending ? (
                            <>
                              <Loader2 className="w-4 h-4 animate-spin" />
                              <span>Menyimulasikan...</span>
                            </>
                          ) : (
                            <>
                              <Sparkles className="w-4 h-4" />
                              <span>Simulasikan Pembayaran (Demo)</span>
                            </>
                          )}
                        </button>
                      </div>
                    )}

                    {checkoutError && (
                      <div className="p-3 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs flex items-start gap-2 text-left">
                        <AlertCircle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
                        <div className="flex-1 font-semibold">{checkoutError}</div>
                      </div>
                    )}

                    <div className="flex gap-2">
                      <button
                        type="button"
                        disabled={paymentStatus.isFetching}
                        onClick={() => paymentStatus.refetch()}
                        className="flex-1 min-h-[44px] rounded-xl font-bold text-xs border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 flex items-center justify-center gap-2 disabled:opacity-50"
                      >
                        <RotateCcw className="w-4 h-4" />
                        <span>Cek Status</span>
                      </button>
                      <button
                        type="button"
                        disabled={createPayment.isPending}
                        onClick={handleOpenPayment}
                        className="flex-1 min-h-[44px] rounded-xl font-bold text-xs border border-slate-300 bg-white hover:bg-slate-50 text-slate-700 flex items-center justify-center gap-2 disabled:opacity-50"
                      >
                        <QrCode className="w-4 h-4" />
                        <span>Buat QR Baru</span>
                      </button>
                    </div>
                  </div>
                )}
              </>
            ) : (
              /* ── RECEIPT / STRUK VIEW ── */
              <div className="space-y-4">
                <div className="text-center pb-2 border-b border-slate-200">
                  <div className="w-10 h-10 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center mx-auto mb-1.5">
                    <CheckCircle2 className="w-6 h-6" />
                  </div>
                  <h3 className="text-base font-black text-slate-900">Pembayaran Berhasil!</h3>
                  <p className="text-xs text-slate-500">Transaksi telah dicatat & stok terpotong</p>
                </div>

                {/* Format Selector: 58mm vs 80mm */}
                <div className="flex items-center justify-between text-xs px-1">
                  <span className="text-slate-500 font-medium">Format Kertas Thermal:</span>
                  <div className="inline-flex rounded-lg border border-slate-200 bg-slate-100 p-0.5 text-xs">
                    <button
                      type="button"
                      onClick={() => setReceiptWidth("58mm")}
                      className={`px-2.5 py-1 rounded-md font-medium transition ${
                        receiptWidth === "58mm" ? "bg-white text-slate-900 shadow-xs font-bold" : "text-slate-600 hover:text-slate-900"
                      }`}
                    >
                      58mm (Mobile)
                    </button>
                    <button
                      type="button"
                      onClick={() => setReceiptWidth("80mm")}
                      className={`px-2.5 py-1 rounded-md font-medium transition ${
                        receiptWidth === "80mm" ? "bg-white text-slate-900 shadow-xs font-bold" : "text-slate-600 hover:text-slate-900"
                      }`}
                    >
                      80mm (Desktop)
                    </button>
                  </div>
                </div>

                {/* Thermal Receipt Box */}
                <div
                  id="thermal-receipt"
                  className={`p-4 bg-white rounded-xl border border-slate-300 font-mono text-xs text-slate-800 space-y-2 mx-auto shadow-xs ${
                    receiptWidth === "58mm" ? "max-w-[280px]" : "max-w-[360px]"
                  }`}
                >
                  <div className="text-center pb-2 border-b border-dashed border-slate-300">
                    <div className="font-extrabold text-sm">TAYOOLI RETAIL & WMS</div>
                    <div className="text-[10px] text-slate-500">Kav. Logistik Pergudangan No. 42</div>
                    <div className="text-[10px] text-slate-500">Telp: 021-555-1234</div>
                  </div>

                  <div className="flex justify-between text-[11px]">
                    <span>No: {receiptData?.orderNumber}</span>
                    <span>{receiptData?.date}</span>
                  </div>
                  <div className="text-[11px] text-slate-500">
                    Kasir: Budi | Mode: {receiptData?.saleMode}
                  </div>
                  {receiptData?.salesOrderId && (
                    <div className="text-[10px] text-slate-500 font-mono truncate">
                      Ref SO: {receiptData.salesOrderId}
                    </div>
                  )}
                  {receiptData?.salesInvoiceId && (
                    <div className="text-[10px] text-slate-500 font-mono truncate">
                      No. Referensi: {receiptData.salesInvoiceId}
                    </div>
                  )}

                  <div className="border-t border-dashed border-slate-300 pt-2 space-y-1">
                    {receiptData?.items.map((item, idx) => (
                      <div key={idx} className="flex justify-between text-[11px]">
                        <span className="truncate max-w-[170px]">
                          {item.name} x{item.quantity}
                        </span>
                        <span>Rp {(item.price * item.quantity).toLocaleString("id-ID")}</span>
                      </div>
                    ))}
                  </div>

                  <div className="border-t border-dashed border-slate-300 pt-2 space-y-1 text-[11px]">
                    <div className="flex justify-between">
                      <span>Subtotal:</span>
                      <span>Rp {receiptData?.subtotal.toLocaleString("id-ID")}</span>
                    </div>
                    {receiptData && receiptData.discount > 0 && (
                      <div className="flex justify-between">
                        <span>Diskon:</span>
                        <span>- Rp {receiptData.discount.toLocaleString("id-ID")}</span>
                      </div>
                    )}
                    <div className="flex justify-between">
                      <span>PPN:</span>
                      <span>Rp {receiptData?.tax.toLocaleString("id-ID")}</span>
                    </div>
                    <div className="flex justify-between font-extrabold text-xs pt-1 border-t border-slate-300">
                      <span>TOTAL:</span>
                      <span>Rp {receiptData?.total.toLocaleString("id-ID")}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Bayar ({receiptData?.method}):</span>
                      <span>Rp {receiptData?.paid.toLocaleString("id-ID")}</span>
                    </div>
                    <div className="flex justify-between font-bold">
                      <span>Kembali:</span>
                      <span>Rp {receiptData?.change.toLocaleString("id-ID")}</span>
                    </div>
                  </div>

                  <div className="text-center pt-2 border-t border-dashed border-slate-300 text-[10px] text-slate-500">
                    Terima kasih atas kunjungan Anda!
                    <br />
                    Barang yang sudah dibeli tidak dapat ditukar.
                  </div>
                </div>

                {/* Print, WhatsApp, and New Order Actions */}
                <div className="flex flex-col sm:flex-row gap-2 pt-1">
                  <button
                    type="button"
                    onClick={handlePrintReceipt}
                    className="flex-1 min-h-[44px] rounded-xl font-bold text-xs bg-slate-800 hover:bg-slate-900 text-white flex items-center justify-center gap-2 shadow-xs transition"
                    title="Cetak struk ke printer thermal"
                  >
                    <Printer className="w-4 h-4" />
                    <span>Cetak Struk ({receiptWidth})</span>
                  </button>

                  <button
                    type="button"
                    onClick={handleShareWhatsApp}
                    className="flex-1 min-h-[44px] rounded-xl font-bold text-xs bg-emerald-600 hover:bg-emerald-700 text-white flex items-center justify-center gap-2 shadow-xs transition"
                    title="Kirim rincian struk ke nomor WhatsApp pelanggan"
                  >
                    <Share2 className="w-4 h-4" />
                    <span>Kirim WhatsApp</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      setShowPaymentModal(false)
                      setIsPaidSuccess(false)
                    }}
                    className="min-h-[44px] px-4 rounded-xl font-bold text-xs bg-[#2563EB] hover:bg-[#1D4ED8] text-white flex items-center justify-center gap-2 shadow-xs transition"
                  >
                    <RotateCcw className="w-4 h-4" />
                    <span>Selesai</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ── In-App Confirmation Modal for Clear Cart (replaces confirm) ── */}
      {showClearCartModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-sm bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl bg-rose-50 text-rose-600 border border-rose-100">
                <Trash2 className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">Kosongkan Keranjang</h3>
                <p className="text-xs text-slate-500 mt-0.5">Semua item belanja akan dihapus</p>
              </div>
            </div>

            <p className="text-sm text-slate-600 leading-relaxed">
              Apakah Anda yakin ingin mengosongkan seluruh item di keranjang belanja kasir saat ini?
            </p>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setShowClearCartModal(false)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={confirmClearCart}
                className="px-4 py-2 rounded-lg bg-rose-600 text-white text-sm font-semibold hover:bg-rose-700 shadow-sm transition"
              >
                Kosongkan
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── MODAL: Riwayat Transaksi POS (Reprint Struk) ── */}
      {showHistoryModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4 max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <History className="w-5 h-5 text-[#2563EB]" />
                <h3 className="text-base font-bold text-slate-900">Riwayat Transaksi POS</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowHistoryModal(false)}
                className="p-1.5 text-slate-400 hover:text-slate-700 min-h-[36px] min-w-[36px] flex items-center justify-center rounded-lg hover:bg-slate-100"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto space-y-2">
              {posOrders.length === 0 ? (
                <div className="text-center py-12 text-slate-400 text-sm">
                  Belum ada transaksi tersimpan hari ini.
                </div>
              ) : (
                posOrders.map((order) => (
                  <div
                    key={order.id}
                    className="p-3.5 rounded-xl border border-slate-200 hover:border-blue-300 hover:bg-blue-50/30 transition flex items-center justify-between gap-3"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-xs text-slate-900">
                          {order.order_number}
                        </span>
                        <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-800">
                          {order.payment_method}
                        </span>
                      </div>
                      <div className="text-xs text-slate-500 mt-1 flex items-center gap-2">
                        <span>{order.customer_name || "Pelanggan Umum"}</span>
                        <span>•</span>
                        <span>{new Date(order.created_at).toLocaleString("id-ID")}</span>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <div className="text-right">
                        <div className="font-mono font-bold text-sm text-slate-900">
                          Rp {Number(order.total_amount).toLocaleString("id-ID")}
                        </div>
                        <div className="text-[10px] text-slate-400">
                          {order.items?.length ?? 0} item
                        </div>
                      </div>

                      <button
                        type="button"
                        onClick={() => handleReprintReceipt(order)}
                        className="px-3 py-1.5 rounded-lg text-xs font-semibold bg-[#2563EB] hover:bg-[#1D4ED8] text-white flex items-center gap-1.5 shadow-xs transition"
                      >
                        <Printer className="w-3.5 h-3.5" />
                        <span>Cetak Struk</span>
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>

            <div className="pt-2 border-t border-slate-100 flex justify-end">
              <button
                type="button"
                onClick={() => setShowHistoryModal(false)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── MODAL: Rekonsiliasi Kasir & Tutup Shift (Z-Report) ── */}
      {showReconciliationModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4 max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-emerald-50 text-emerald-600 rounded-lg">
                  <Receipt className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">Rekonsiliasi Kasir (Z-Report)</h3>
                  <p className="text-xs text-slate-500">
                    Laporan ringkasan kas, penjualan, dan penutupan shift kasir
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowReconciliationModal(false)}
                className="p-1.5 text-slate-400 hover:text-slate-700 rounded-lg hover:bg-slate-100"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div id="z-report" className="flex-1 overflow-y-auto space-y-4 pr-1">
              <div className="text-[11px] text-slate-500">
                Periode: hari ini ({new Date().toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" })})
              </div>
              {/* Summary Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div className="p-3.5 rounded-xl border border-blue-100 bg-blue-50/40">
                  <span className="text-[11px] font-semibold text-blue-700 uppercase tracking-wider">Total Omzet</span>
                  <div className="font-mono text-lg font-black text-slate-900 mt-1">
                    Rp {reconciliationData.grandRevenue.toLocaleString("id-ID")}
                  </div>
                  <span className="text-[10px] text-slate-500">{reconciliationData.totalOrders} total transaksi</span>
                </div>

                <div className="p-3.5 rounded-xl border border-emerald-100 bg-emerald-50/40">
                  <span className="text-[11px] font-semibold text-emerald-700 uppercase tracking-wider">Kas Fisik (CASH)</span>
                  <div className="font-mono text-lg font-black text-slate-900 mt-1">
                    Rp {reconciliationData.cashTotal.toLocaleString("id-ID")}
                  </div>
                  <span className="text-[10px] text-slate-500">{reconciliationData.cashCount} transaksi tunai di laci</span>
                </div>

                <div className="p-3.5 rounded-xl border border-indigo-100 bg-indigo-50/40">
                  <span className="text-[11px] font-semibold text-indigo-700 uppercase tracking-wider">QRIS / Midtrans</span>
                  <div className="font-mono text-lg font-black text-slate-900 mt-1">
                    Rp {reconciliationData.qrisTotal.toLocaleString("id-ID")}
                  </div>
                  <span className="text-[10px] text-slate-500">{reconciliationData.qrisCount} transaksi langsung ke bank</span>
                </div>
              </div>

              {/* Rekonsiliasi Audit Notice */}
              <div className="p-3 rounded-xl border border-slate-200 bg-slate-50 text-xs text-slate-600 space-y-1">
                <p className="font-semibold text-slate-800 flex items-center gap-1.5">
                  <ShieldCheck className="w-4 h-4 text-emerald-600" />
                  Instruksi Pencocokan Kasir:
                </p>
                <p className="text-[11px] leading-relaxed">
                  1. Hitung uang fisik di laci kasir: jumlahnya harus tepat <strong>Rp {reconciliationData.cashTotal.toLocaleString("id-ID")}</strong>.
                </p>
                <p className="text-[11px] leading-relaxed">
                  2. Dana non-tunai QRIS sebesar <strong>Rp {reconciliationData.qrisTotal.toLocaleString("id-ID")}</strong> telah disalurkan langsung oleh Midtrans ke rekening bank toko Anda.
                </p>
              </div>

              {/* Rincian Transaksi */}
              <div className="space-y-2">
                <h4 className="text-xs font-bold text-slate-800 uppercase tracking-wider">Daftar Transaksi Hari Ini</h4>
                <div className="border border-slate-200 rounded-xl overflow-hidden">
                  <table className="w-full text-xs text-left">
                    <thead className="bg-slate-50 border-b border-slate-200 text-[10px] text-slate-500 uppercase font-semibold">
                      <tr>
                        <th className="px-3 py-2">No. Order</th>
                        <th className="px-3 py-2">Metode</th>
                        <th className="px-3 py-2">Waktu</th>
                        <th className="px-3 py-2 text-right">Nominal</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 font-mono">
                      {reconciliationData.orders.length === 0 ? (
                        <tr>
                          <td colSpan={4} className="text-center py-6 text-slate-400 font-sans">
                            Belum ada transaksi hari ini
                          </td>
                        </tr>
                      ) : (
                        reconciliationData.orders.map((o) => (
                          <tr key={o.id} className="hover:bg-slate-50/50">
                            <td className="px-3 py-2 font-bold text-slate-800">{o.order_number}</td>
                            <td className="px-3 py-2">
                              <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                (o.payment_method || "").toUpperCase() === "CASH"
                                  ? "bg-emerald-100 text-emerald-800"
                                  : "bg-indigo-100 text-indigo-800"
                              }`}>
                                {o.payment_method}
                              </span>
                            </td>
                            <td className="px-3 py-2 text-slate-500 font-sans text-[11px]">
                              {new Date(o.created_at).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit" })}
                            </td>
                            <td className="px-3 py-2 text-right font-bold text-slate-900">
                              Rp {Number(o.total_amount).toLocaleString("id-ID")}
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            <div className="pt-3 border-t border-slate-100 flex items-center justify-between">
              <span className="text-[11px] text-slate-400">
                Laporan digenerate otomatis oleh Tayooli POS
              </span>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => window.print()}
                  className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-900 text-white text-xs font-bold flex items-center gap-1.5 transition shadow-xs"
                >
                  <Printer className="w-3.5 h-3.5" />
                  <span>Cetak Z-Report</span>
                </button>
                <button
                  type="button"
                  onClick={() => setShowReconciliationModal(false)}
                  className="px-4 py-2 rounded-xl border border-slate-200 text-slate-700 hover:bg-slate-50 text-xs font-semibold transition"
                >
                  Tutup
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Global CSS for Clean Thermal Printer Output */}
      <style
        dangerouslySetInnerHTML={{
          __html: `
            @media print {
              body * {
                visibility: hidden !important;
              }
              #thermal-receipt, #thermal-receipt *,
              #z-report, #z-report * {
                visibility: visible !important;
              }
              #z-report {
                position: absolute !important;
                left: 0 !important;
                top: 0 !important;
                width: 100% !important;
                overflow: visible !important;
                max-height: none !important;
                background: white !important;
              }
              #thermal-receipt {
                position: fixed !important;
                left: 0 !important;
                top: 0 !important;
                width: 100% !important;
                max-width: 80mm !important;
                margin: 0 !important;
                padding: 4mm !important;
                background: white !important;
                color: black !important;
                border: none !important;
                box-shadow: none !important;
              }
            }
          `,
        }}
      />

      {/* Export Modal */}
      <ExportModal<POSOrder>
        open={exportingOrders}
        onClose={() => setExportingOrders(false)}
        title="Riwayat Transaksi POS"
        filename="riwayat-pos"
        allRows={posOrders}
        visibleRows={posOrders}
        columns={posExportColumns}
        filters={[
          { type: "dateRange", id: "date", label: "Tanggal transaksi", getDate: (o) => o.created_at },
          {
            type: "select",
            id: "method",
            label: "Metode bayar",
            options: [
              { value: "CASH", label: "Tunai" },
              { value: "QRIS", label: "QRIS" },
            ],
            match: (o, v) => (o.payment_method || "").toUpperCase() === v,
          },
        ]}
      />
    </div>
  )
}
