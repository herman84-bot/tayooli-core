'use client'

import React, { useEffect, useState, useId } from 'react'
import { Printer, X, Receipt, Building2, CreditCard, ShieldCheck } from 'lucide-react'
import QRCode from 'qrcode'
import { formatCurrency, terbilang } from '@/lib/currency'
import type { SalesInvoice } from '@/hooks/useSalesInvoices'

export interface InvoiceLineItem {
  id?: string
  description: string
  quantity: number
  unitPrice: number
  discount?: number
  subtotal: number
}

export interface PrintSalesInvoiceProps {
  invoice: SalesInvoice
  customerName?: string
  customerAddress?: string
  customerTaxId?: string
  customerPhone?: string
  deliveryOrderRef?: string
  items?: InvoiceLineItem[]
  onClose: () => void
}

export function PrintSalesInvoice({
  invoice,
  customerName = 'PT Nusantara Retail Makmur',
  customerAddress = 'Jl. Gatot Subroto Kav. 52, Kel. Kuningan Barat, Jakarta Selatan 12710',
  customerTaxId = '02.456.789.0-034.000',
  customerPhone = '(021) 529-0123',
  deliveryOrderRef,
  items,
  onClose,
}: PrintSalesInvoiceProps) {
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null)
  const printAreaId = useId().replace(/:/g, '')
  const elementId = `printable-inv-${printAreaId}`

  // Default breakdown calculation if no line items are provided:
  // We calculate DPP (Dasar Pengenaan Pajak) and PPN 11% so total equals invoice.amount
  const totalAmount = Number(invoice.amount || 0)
  const dppAmount = Math.round((totalAmount / 1.11) * 100) / 100
  const ppnAmount = Math.round((totalAmount - dppAmount) * 100) / 100
  const shippingFee = 0

  const lineItems: InvoiceLineItem[] =
    items && items.length > 0
      ? items
      : [
          {
            description: `Pasokan Produk & Pemenuhan Pesanan Sales Order (${invoice.orderId ? invoice.orderId.slice(0, 8).toUpperCase() : 'SO-RETAIL-01'})`,
            quantity: 1,
            unitPrice: dppAmount,
            discount: 0,
            subtotal: dppAmount,
          },
        ]

  const isPaid = invoice.status === 'PAID'
  const isCancelled = invoice.status === 'CANCELLED'

  useEffect(() => {
    const payload = JSON.stringify({
      doc: 'FAKTUR_PENJUALAN',
      invoice_number: invoice.invoiceNumber,
      id: invoice.id,
      amount: invoice.amount,
      status: invoice.status,
      due_date: invoice.dueDate,
      hash: `TAX-VERIFY-${invoice.id.replace(/-/g, '').slice(0, 16).toUpperCase()}`,
    })

    QRCode.toDataURL(payload, {
      width: 120,
      margin: 1,
      color: { dark: '#09090b', light: '#ffffff' },
      errorCorrectionLevel: 'M',
    })
      .then((url) => setQrCodeUrl(url))
      .catch(() => setQrCodeUrl(null))
  }, [invoice])

  const handlePrint = () => {
    if (typeof window !== 'undefined') {
      window.print()
    }
  }

  const formatDate = (isoString: string) => {
    try {
      const d = new Date(isoString)
      return d.toLocaleDateString('id-ID', {
        day: '2-digit',
        month: 'long',
        year: 'numeric',
      })
    } catch {
      return isoString
    }
  }

  return (
    <>
      {/* Global Print Stylesheet for Pure A4 Output */}
      <style jsx global>{`
        @media print {
          body * {
            visibility: hidden !important;
          }
          #${elementId},
          #${elementId} * {
            visibility: visible !important;
          }
          #${elementId} {
            position: absolute !important;
            left: 0 !important;
            top: 0 !important;
            width: 100% !important;
            max-width: 100% !important;
            margin: 0 !important;
            padding: 10mm 12mm !important;
            background: #ffffff !important;
            color: #09090b !important;
            box-shadow: none !important;
            border: none !important;
          }
          .no-print {
            display: none !important;
          }
          @page {
            size: A4 portrait;
            margin: 8mm;
          }
        }
      `}</style>

      {/* Modal Backdrop Container */}
      <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-zinc-950/80 p-4 backdrop-blur-sm sm:p-6 lg:p-8 no-print">
        <div className="flex w-full max-w-4xl flex-col items-center gap-4">
          {/* Action Control Bar */}
          <div className="flex w-full items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900/90 px-4 py-3 text-white shadow-xl backdrop-blur-md">
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
                <Receipt className="h-4 w-4" />
              </span>
              <div>
                <h3 className="text-sm font-semibold">Pratinjau Faktur Penjualan (A4)</h3>
                <p className="text-xs text-zinc-400">
                  {invoice.invoiceNumber} &bull; Standar Komersial Resmi & Terbilang Rupiah
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handlePrint}
                className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-xs font-medium text-white shadow hover:bg-blue-500 active:scale-95 transition-all"
              >
                <Printer className="h-4 w-4" />
                Cetak / Print PDF
              </button>
              <button
                type="button"
                onClick={onClose}
                className="inline-flex items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 p-2 text-zinc-300 hover:bg-zinc-700 hover:text-white transition"
                title="Tutup Pratinjau"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Printable Document Sheet (A4 Standard Format) */}
          <div
            id={elementId}
            className="relative w-full bg-white text-zinc-900 shadow-2xl rounded-sm p-8 sm:p-10 border border-zinc-200 font-sans text-xs leading-relaxed"
            style={{ minHeight: '297mm', maxWidth: '210mm' }}
          >
            {/* WATERMARK STAMP PELUNASAN */}
            <div className="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden opacity-10">
              <div
                className={`text-8xl font-black uppercase tracking-widest -rotate-24 select-none border-8 px-12 py-4 rounded-3xl ${
                  isPaid
                    ? 'border-emerald-600 text-emerald-600'
                    : isCancelled
                    ? 'border-zinc-500 text-zinc-500'
                    : 'border-red-600 text-red-600'
                }`}
              >
                {isPaid ? 'LUNAS / PAID' : isCancelled ? 'BATAL' : 'BELUM LUNAS'}
              </div>
            </div>

            {/* KOP SURAT PERUSAHAAN */}
            <div className="border-b-2 border-zinc-900 pb-4 mb-4">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-700 text-white font-black text-xl">
                    T
                  </div>
                  <div>
                    <h1 className="text-base font-extrabold uppercase tracking-tight text-zinc-950">
                      PT TAYOOLI DISTRIBUSI UTAMA
                    </h1>
                    <p className="text-[11px] font-medium text-zinc-600">
                      General Trading, Logistics & B2B Distribution Center
                    </p>
                    <p className="text-[10px] text-zinc-500 mt-0.5">
                      Kawasan Industri Pulogadung Blok B No. 12, Jakarta Timur 13920
                    </p>
                    <p className="text-[10px] text-zinc-500">
                      NPWP: 01.345.678.9-012.000 &bull; Telp: (021) 460-8899 &bull; Email: billing@tayooli.co.id
                    </p>
                  </div>
                </div>

                <div className="text-right">
                  <div className="inline-block rounded border border-zinc-900 bg-zinc-900 px-3 py-1 text-[11px] font-black uppercase tracking-widest text-white">
                    FAKTUR PENJUALAN
                  </div>
                  <div className="mt-2 text-[10px] font-mono font-bold text-zinc-700">
                    SALES INVOICE
                  </div>
                </div>
              </div>
            </div>

            {/* DUA KOLOM IDENTITAS: PELANGGAN & DETAIL FAKTUR */}
            <div className="grid grid-cols-2 gap-6 mb-5">
              {/* Kolom Kiri: Ditujukan Kepada (Customer) */}
              <div className="rounded border border-zinc-300 p-3 bg-zinc-50/40 space-y-1">
                <span className="text-[10px] font-bold uppercase tracking-wider text-zinc-500 flex items-center gap-1">
                  <Building2 className="h-3 w-3 text-zinc-700" /> Ditujukan Kepada (Customer):
                </span>
                <p className="text-[12px] font-bold text-zinc-950">{customerName}</p>
                <p className="text-[10px] text-zinc-600">{customerAddress}</p>
                <div className="pt-1 text-[10px] text-zinc-500 grid grid-cols-3 gap-0.5 font-mono">
                  <span>NPWP</span>
                  <span className="col-span-2 text-zinc-900">: {customerTaxId}</span>
                  <span>Telepon</span>
                  <span className="col-span-2 text-zinc-900">: {customerPhone}</span>
                </div>
              </div>

              {/* Kolom Kanan: Rincian Faktur & Referensi */}
              <div className="rounded border border-zinc-300 p-3 bg-zinc-50/40 space-y-1">
                <span className="text-[10px] font-bold uppercase tracking-wider text-zinc-500 flex items-center gap-1">
                  <Receipt className="h-3 w-3 text-zinc-700" /> Informasi Dokumen Penagihan:
                </span>
                <div className="grid grid-cols-3 gap-1 text-[11px] pt-0.5">
                  <span className="text-zinc-500">Nomor Invoice</span>
                  <span className="col-span-2 font-mono font-bold text-zinc-950">
                    : {invoice.invoiceNumber}
                  </span>

                  <span className="text-zinc-500">Tanggal Terbit</span>
                  <span className="col-span-2 font-medium text-zinc-900">
                    : {formatDate(invoice.createdAt)}
                  </span>

                  <span className="text-zinc-500 font-semibold text-rose-700">Jatuh Tempo</span>
                  <span className="col-span-2 font-semibold text-rose-700">
                    : {formatDate(invoice.dueDate)}
                  </span>

                  <span className="text-zinc-500">Ref. Sales Order</span>
                  <span className="col-span-2 font-mono text-zinc-900">
                    : {invoice.orderId ? invoice.orderId.slice(0, 13) : '—'}
                  </span>

                  <span className="text-zinc-500">Ref. Surat Jalan</span>
                  <span className="col-span-2 font-mono text-zinc-900">
                    : {deliveryOrderRef || (invoice.orderId ? `DO-${invoice.orderId.slice(0, 8).toUpperCase()}` : 'DO-DIRECT-01')}
                  </span>
                </div>
              </div>
            </div>

            {/* TABEL ITEM FAKTUR */}
            <div className="mb-4">
              <table className="w-full border-collapse border border-zinc-900 text-[11px]">
                <thead>
                  <tr className="bg-zinc-100 text-zinc-900 font-bold border-b border-zinc-900">
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-center w-8">No</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-left">Deskripsi Barang / Jasa</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-right w-14">Qty</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-right w-28">Harga Satuan</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-right w-20">Diskon</th>
                    <th className="py-1.5 px-2 text-right w-32">Subtotal (Rp)</th>
                  </tr>
                </thead>
                <tbody>
                  {lineItems.map((item, index) => (
                    <tr key={item.id || index} className="border-b border-zinc-300">
                      <td className="border-r border-zinc-300 py-2 px-2 text-center text-zinc-600 font-mono">
                        {index + 1}
                      </td>
                      <td className="border-r border-zinc-300 py-2 px-2 font-medium text-zinc-900">
                        {item.description}
                      </td>
                      <td className="border-r border-zinc-300 py-2 px-2 text-right font-mono text-zinc-900">
                        {item.quantity}
                      </td>
                      <td className="border-r border-zinc-300 py-2 px-2 text-right font-mono text-zinc-900">
                        {formatCurrency(item.unitPrice)}
                      </td>
                      <td className="border-r border-zinc-300 py-2 px-2 text-right font-mono text-zinc-500">
                        {item.discount ? formatCurrency(item.discount) : '—'}
                      </td>
                      <td className="py-2 px-2 text-right font-mono font-semibold text-zinc-950">
                        {formatCurrency(item.subtotal)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* RINGKASAN KEUANGAN & TERBILANG */}
            <div className="grid grid-cols-12 gap-4 mb-4">
              {/* Kolom Kiri: Terbilang & Instruksi Pembayaran (7 Kolom) */}
              <div className="col-span-7 space-y-3">
                {/* Kotak Terbilang Resmi */}
                <div className="rounded border border-zinc-300 p-2.5 bg-zinc-50/60">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-zinc-600 block mb-0.5">
                    Terbilang Resmi:
                  </span>
                  <p className="text-[11px] font-serif font-bold italic text-zinc-900">
                    &ldquo;{terbilang(totalAmount)}&rdquo;
                  </p>
                </div>

                {/* Detail Rekening Bank Pembayaran */}
                <div className="rounded border border-zinc-300 p-2.5 bg-zinc-50/30 text-[10px]">
                  <span className="font-bold text-zinc-800 uppercase flex items-center gap-1 mb-1">
                    <CreditCard className="h-3 w-3 text-zinc-700" /> Instruksi Pembayaran / Transfer Bank:
                  </span>
                  <div className="grid grid-cols-2 gap-2 mt-1">
                    <div className="border border-zinc-200 rounded p-1.5 bg-white">
                      <p className="font-bold text-zinc-900">Bank Central Asia (BCA)</p>
                      <p className="font-mono font-bold text-blue-700">883-091-2300</p>
                      <p className="text-[9px] text-zinc-500">a.n PT Tayooli Distribusi Utama</p>
                    </div>
                    <div className="border border-zinc-200 rounded p-1.5 bg-white">
                      <p className="font-bold text-zinc-900">Bank Mandiri</p>
                      <p className="font-mono font-bold text-blue-700">120-00-9988771-2</p>
                      <p className="text-[9px] text-zinc-500">a.n PT Tayooli Distribusi Utama</p>
                    </div>
                  </div>
                  <p className="text-[9px] text-zinc-500 mt-1.5 italic">
                    * Harap sertakan nomor invoice pada berita transfer dan kirimkan bukti bayar ke billing@tayooli.co.id.
                  </p>
                </div>
              </div>

              {/* Kolom Kanan: Rincian Angka Subtotal, PPN, Total (5 Kolom) */}
              <div className="col-span-5 rounded border border-zinc-900 p-3 bg-zinc-50/50">
                <div className="space-y-1.5 text-[11px]">
                  <div className="flex justify-between text-zinc-600">
                    <span>Dasar Pengenaan Pajak (DPP)</span>
                    <span className="font-mono font-medium text-zinc-900">{formatCurrency(dppAmount)}</span>
                  </div>
                  <div className="flex justify-between text-zinc-600">
                    <span>Diskon Pembelian</span>
                    <span className="font-mono text-zinc-500">Rp 0</span>
                  </div>
                  <div className="flex justify-between text-zinc-600 border-b border-zinc-300 pb-1.5">
                    <span>PPN (11%)</span>
                    <span className="font-mono font-medium text-zinc-900">{formatCurrency(ppnAmount)}</span>
                  </div>
                  <div className="flex justify-between text-zinc-600 pt-0.5">
                    <span>Biaya Pengiriman</span>
                    <span className="font-mono text-zinc-700">{formatCurrency(shippingFee)}</span>
                  </div>
                  <div className="flex justify-between items-center border-t-2 border-zinc-900 pt-2 mt-2">
                    <span className="font-black uppercase tracking-wider text-xs text-zinc-950">TOTAL TAGIHAN</span>
                    <span className="font-mono font-black text-sm text-blue-900">
                      {formatCurrency(totalAmount)}
                    </span>
                  </div>
                </div>

                <div className="mt-3 text-center">
                  <span
                    className={`inline-block w-full py-1 rounded text-[10px] font-black uppercase tracking-widest ${
                      isPaid
                        ? 'bg-emerald-100 text-emerald-800 border border-emerald-300'
                        : isCancelled
                        ? 'bg-zinc-200 text-zinc-700 border border-zinc-300'
                        : 'bg-amber-100 text-amber-900 border border-amber-300'
                    }`}
                  >
                    STATUS: {invoice.status}
                  </span>
                </div>
              </div>
            </div>

            {/* TANDA TANGAN OTORISASI PERUSAHAAN */}
            <div className="grid grid-cols-2 gap-8 items-end mt-6 mb-6">
              <div className="text-[10px] text-zinc-500 space-y-1">
                <p className="font-bold text-zinc-700 uppercase">Catatan Penting:</p>
                <p>1. Faktur ini sah dan diproses secara komputerisasi oleh Tayooli ERP Engine.</p>
                <p>2. Keterlambatan pelunasan dapat dikenakan denda administratif sesuai perjanjian dagang.</p>
              </div>

              <div className="text-center">
                <p className="text-[10px] text-zinc-600 mb-1">
                  Jakarta, {formatDate(invoice.createdAt)}
                </p>
                <p className="text-[10px] font-bold uppercase tracking-wider text-zinc-900">
                  PT TAYOOLI DISTRIBUSI UTAMA
                </p>
                <div className="h-20 flex items-center justify-center">
                  <div className="rounded border border-dashed border-zinc-300 px-4 py-2 text-[9px] text-zinc-400 font-mono">
                    [Tanda Tangan Digital & Stempel Finance]
                  </div>
                </div>
                <p className="text-[11px] font-bold text-zinc-950 underline underline-offset-4">
                  Finance & Accounting Manager
                </p>
                <p className="text-[9px] text-zinc-500">Divisi Keuangan & Pajak</p>
              </div>
            </div>

            {/* FOOTER & VERIFIKASI QR CODE */}
            <div className="border-t border-zinc-300 pt-3 flex items-center justify-between text-[9px] text-zinc-500">
              <div className="flex items-center gap-3">
                {qrCodeUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={qrCodeUrl}
                    alt="QR Verifikasi Faktur"
                    className="h-14 w-14 border border-zinc-200 p-0.5 rounded"
                  />
                ) : (
                  <div className="h-14 w-14 border border-zinc-200 flex items-center justify-center text-[8px] text-zinc-400">
                    QR Code
                  </div>
                )}
                <div>
                  <div className="flex items-center gap-1 font-bold text-zinc-700">
                    <ShieldCheck className="h-3 w-3 text-blue-600" />
                    <span>Faktur Penjualan Komersial Sah & Terverifikasi</span>
                  </div>
                  <p className="text-zinc-500 mt-0.5">
                    Hash Sertifikat: {invoice.id.replace(/-/g, '').slice(0, 16).toUpperCase()}
                  </p>
                  <p className="text-zinc-400">
                    Verifikasi integritas faktur melalui QR Code di atas.
                  </p>
                </div>
              </div>

              <div className="text-right space-y-0.5 font-mono text-[9px]">
                <p>Dokumen ID: {invoice.id}</p>
                <p>Invoice No: {invoice.invoiceNumber}</p>
                <p>Dicetak: {new Date().toLocaleString('id-ID')}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
export default PrintSalesInvoice
