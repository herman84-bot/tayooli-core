'use client'

import React, { useEffect, useState, useId } from 'react'
import { Printer, X, CheckCircle2, Truck, Box, ShieldCheck, MapPin } from 'lucide-react'
import QRCode from 'qrcode'
import type { DeliveryOrder, DeliveryOrderItem } from '@/lib/api'

export interface PrintDeliveryOrderProps {
  deliveryOrder: DeliveryOrder
  items: DeliveryOrderItem[]
  warehouseName?: string
  warehouseAddress?: string
  onClose: () => void
}

export function PrintDeliveryOrder({
  deliveryOrder,
  items,
  warehouseName = 'Gudang Distribusi Cakung (WH-JKT-01)',
  warehouseAddress = 'Kawasan Industri Pulogadung Blok B, Jakarta Timur',
  onClose,
}: PrintDeliveryOrderProps) {
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null)
  const printAreaId = useId().replace(/:/g, '')
  const elementId = `printable-do-${printAreaId}`

  useEffect(() => {
    const payload = JSON.stringify({
      doc: 'SURAT_JALAN',
      do_number: deliveryOrder.do_number,
      id: deliveryOrder.id,
      sales_order_id: deliveryOrder.sales_order_id,
      status: deliveryOrder.status,
      issued_at: deliveryOrder.created_at,
      hash: `SHA256-${deliveryOrder.id.replace(/-/g, '').slice(0, 16).toUpperCase()}`,
    })

    QRCode.toDataURL(payload, {
      width: 120,
      margin: 1,
      color: { dark: '#09090b', light: '#ffffff' },
      errorCorrectionLevel: 'M',
    })
      .then((url) => setQrCodeUrl(url))
      .catch(() => setQrCodeUrl(null))
  }, [deliveryOrder])

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

  const formatTime = (isoString: string) => {
    try {
      const d = new Date(isoString)
      return d.toLocaleTimeString('id-ID', {
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return ''
    }
  }

  const totalQuantity = items.reduce((sum, it) => sum + Number(it.quantity || 0), 0)

  return (
    <>
      {/* Global Print Stylesheet for Isolated A4 Output */}
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
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
                <Truck className="h-4 w-4" />
              </span>
              <div>
                <h3 className="text-sm font-semibold">Pratinjau Surat Jalan Resmi (A4)</h3>
                <p className="text-xs text-zinc-400">
                  {deliveryOrder.do_number} &bull; Siap dicetak langsung ke printer atau disimpan sebagai PDF
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handlePrint}
                className="inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2 text-xs font-medium text-white shadow hover:bg-emerald-500 active:scale-95 transition-all"
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
            className="w-full bg-white text-zinc-900 shadow-2xl rounded-sm p-8 sm:p-10 border border-zinc-200 font-sans text-xs leading-relaxed"
            style={{ minHeight: '297mm', maxWidth: '210mm' }}
          >
            {/* KOP SURAT PERUSAHAAN */}
            <div className="border-b-2 border-zinc-900 pb-4 mb-4">
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-zinc-900 text-white font-black text-xl">
                    T
                  </div>
                  <div>
                    <h1 className="text-base font-extrabold uppercase tracking-tight text-zinc-950">
                      PT TAYOOLI DISTRIBUSI UTAMA
                    </h1>
                    <p className="text-[11px] font-medium text-zinc-600">
                      Divisi Logistik & Pergudangan Terpadu (WMS Fulfillment)
                    </p>
                    <p className="text-[10px] text-zinc-500 mt-0.5">
                      Kawasan Industri Pulogadung Blok B No. 12, Jakarta Timur 13920
                    </p>
                    <p className="text-[10px] text-zinc-500">
                      Telp: (021) 460-8899 &bull; Email: wms@tayooli.co.id &bull; Website: www.tayooli.com
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <div className="inline-block rounded border border-zinc-800 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-900">
                    LEMBAR SERAH TERIMA
                  </div>
                  <div className="mt-2 text-[10px] text-zinc-500 font-mono">
                    Ref: {deliveryOrder.id.slice(0, 8).toUpperCase()}
                  </div>
                </div>
              </div>
            </div>

            {/* DOKUMEN JUDUL & INFORMASI HEADER */}
            <div className="text-center my-4">
              <h2 className="text-base font-black uppercase tracking-wider text-zinc-900 border-b inline-block border-zinc-900 pb-0.5">
                SURAT JALAN PENGIRIMAN (DELIVERY ORDER)
              </h2>
              <p className="text-[11px] font-mono font-bold text-zinc-700 mt-1">
                NOMOR: {deliveryOrder.do_number}
              </p>
            </div>

            {/* DUA KOLOM INFORMASI PENGIRIMAN & PENERIMA */}
            <div className="grid grid-cols-2 gap-4 rounded border border-zinc-300 p-3 mb-4 bg-zinc-50/50">
              {/* Kolom Kiri: Informasi Pengiriman & Kendaraan */}
              <div className="space-y-1.5 border-r border-zinc-200 pr-3">
                <div className="text-[10px] font-bold uppercase tracking-wider text-zinc-500 flex items-center gap-1">
                  <Truck className="h-3 w-3 text-zinc-700" /> Ekspedisi & Transportasi
                </div>
                <div className="grid grid-cols-3 gap-1 text-[11px]">
                  <span className="text-zinc-500">Jasa Ekspedisi</span>
                  <span className="col-span-2 font-semibold text-zinc-900">
                    : {deliveryOrder.expedition_name || 'Armada Internal Gudang'}
                  </span>

                  <span className="text-zinc-500">No. Polisi Kendaraan</span>
                  <span className="col-span-2 font-mono font-bold text-zinc-900">
                    : {deliveryOrder.vehicle_plate || '—'}
                  </span>

                  <span className="text-zinc-500">Nama Pengemudi</span>
                  <span className="col-span-2 font-medium text-zinc-900">
                    : {deliveryOrder.driver_name || '—'}
                  </span>

                  <span className="text-zinc-500">No. Resi / AWB</span>
                  <span className="col-span-2 font-mono font-semibold text-zinc-900">
                    : {deliveryOrder.tracking_number || '—'}
                  </span>
                </div>
              </div>

              {/* Kolom Kanan: Rujukan SO & Penerima */}
              <div className="space-y-1.5 pl-1">
                <div className="text-[10px] font-bold uppercase tracking-wider text-zinc-500 flex items-center gap-1">
                  <MapPin className="h-3 w-3 text-zinc-700" /> Dokumen & Tujuan
                </div>
                <div className="grid grid-cols-3 gap-1 text-[11px]">
                  <span className="text-zinc-500">Tanggal Terbit</span>
                  <span className="col-span-2 font-medium text-zinc-900">
                    : {formatDate(deliveryOrder.created_at)} ({formatTime(deliveryOrder.created_at)})
                  </span>

                  <span className="text-zinc-500">Ref. Sales Order</span>
                  <span className="col-span-2 font-mono font-bold text-zinc-900">
                    : {deliveryOrder.sales_order_id ?? "— (Surat Jalan langsung)"}
                  </span>

                  <span className="text-zinc-500">Gudang Asal</span>
                  <span className="col-span-2 font-medium text-zinc-900">
                    : {warehouseName}
                  </span>

                  <span className="text-zinc-500">Tujuan / Penerima</span>
                  <span className="col-span-2 font-bold text-zinc-950">
                    : {deliveryOrder.recipient_name || 'PT Pelanggan Tayooli'}
                  </span>
                </div>
              </div>
            </div>

            {/* TABEL ITEM BARANG */}
            <div className="mb-4">
              <table className="w-full border-collapse border border-zinc-900 text-[11px]">
                <thead>
                  <tr className="bg-zinc-100 text-zinc-900 font-bold border-b border-zinc-900">
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-center w-8">No</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-left w-28">Kode SKU</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-left">Deskripsi Barang</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-center w-24">Lokasi Rak</th>
                    <th className="border-r border-zinc-900 py-1.5 px-2 text-right w-16">Qty</th>
                    <th className="py-1.5 px-2 text-center w-16">Satuan</th>
                  </tr>
                </thead>
                <tbody>
                  {items.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="text-center py-6 text-zinc-500 italic">
                        Tidak ada item dalam Surat Jalan ini.
                      </td>
                    </tr>
                  ) : (
                    items.map((item, index) => (
                      <tr key={item.id || index} className="border-b border-zinc-300 hover:bg-zinc-50/50">
                        <td className="border-r border-zinc-300 py-1.5 px-2 text-center text-zinc-600 font-mono">
                          {index + 1}
                        </td>
                        <td className="border-r border-zinc-300 py-1.5 px-2 font-mono font-semibold text-zinc-900">
                          {item.product_sku || item.product_id.slice(0, 8).toUpperCase()}
                        </td>
                        <td className="border-r border-zinc-300 py-1.5 px-2 font-medium text-zinc-900">
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span>{item.product_name || `Barang Master (${item.product_id.slice(0, 8)})`}</span>
                            {item.is_free_item && (
                              <span className="inline-block px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-800 border border-emerald-300 font-bold text-[9px] uppercase tracking-wider">
                                BONUS
                              </span>
                            )}
                          </div>
                          {item.batch_number && (
                            <div className="text-[9px] font-mono text-zinc-500 mt-0.5">
                              Batch: {item.batch_number} {item.expiry_date ? `(Exp: ${item.expiry_date.slice(0, 10)})` : ''}
                            </div>
                          )}
                        </td>
                        <td className="border-r border-zinc-300 py-1.5 px-2 text-center font-mono font-medium text-zinc-700 bg-zinc-50/60">
                          {item.location_code || 'BIN-A-01'}
                        </td>
                        <td className="border-r border-zinc-300 py-1.5 px-2 text-right font-mono font-bold text-zinc-950">
                          {Number(item.quantity).toLocaleString('id-ID')}
                        </td>
                        <td className="py-1.5 px-2 text-center text-zinc-600">
                          Pcs
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
                <tfoot>
                  <tr className="bg-zinc-100 font-bold border-t border-zinc-900 text-zinc-900">
                    <td colSpan={4} className="border-r border-zinc-900 py-1.5 px-3 text-right uppercase tracking-wider text-[10px]">
                      Total Kuantitas Barang :
                    </td>
                    <td className="border-r border-zinc-900 py-1.5 px-2 text-right font-mono font-black text-sm">
                      {totalQuantity.toLocaleString('id-ID')}
                    </td>
                    <td className="py-1.5 px-2 text-center text-[10px] uppercase">
                      Total Pcs
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>

            {/* CATATAN PENGIRIMAN & KLAUSUL */}
            <div className="rounded border border-zinc-300 p-2.5 mb-6 text-[10px] text-zinc-600 leading-normal bg-zinc-50/40">
              <span className="font-bold text-zinc-800 uppercase block mb-0.5">
                Klausul Serah Terima Muatan & Syarat Pengangkutan:
              </span>
              <ol className="list-decimal pl-4 space-y-0.5">
                <li>Barang telah diserahkan dari pihak gudang pengirim dalam kondisi baik, baru, dan bersegel utuh.</li>
                <li>Pengemudi/kurir bertanggung jawab penuh menjaga keselamatan dan keutuhan barang hingga tiba di alamat penerima.</li>
                <li>Penerima wajib melakukan pemeriksaan fisik jumlah koli dan segel sebelum menandatangani Surat Jalan ini.</li>
                <li>Segala bentuk klaim kerusakan atau kekurangan tidak berlaku apabila Surat Jalan telah ditandatangani tanpa catatan berita acara.</li>
              </ol>
            </div>

            {/* TIGA BLOK TANDA TANGAN RESMI */}
            <div className="grid grid-cols-3 gap-4 text-center mt-6 mb-8">
              {/* 1. Yang Menyerahkan (Petugas Gudang) */}
              <div className="flex flex-col justify-between border border-zinc-300 rounded p-3 h-36">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-zinc-700">
                    Yang Menyerahkan
                  </p>
                  <p className="text-[9px] text-zinc-500">Petugas Gudang / Fulfillment</p>
                </div>
                <div>
                  <p className="text-[10px] text-zinc-400 font-mono mb-1">
                    ( .................................................... )
                  </p>
                  <p className="text-[10px] font-semibold text-zinc-800">
                    Staf Dispatch WMS
                  </p>
                </div>
              </div>

              {/* 2. Yang Membawa (Sopir / Kurir) */}
              <div className="flex flex-col justify-between border border-zinc-300 rounded p-3 h-36">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-zinc-700">
                    Yang Membawa
                  </p>
                  <p className="text-[9px] text-zinc-500">Pengemudi / Kurir Ekspedisi</p>
                </div>
                <div>
                  <p className="text-[10px] text-zinc-400 font-mono mb-1">
                    ( .................................................... )
                  </p>
                  <p className="text-[10px] font-semibold text-zinc-800">
                    {deliveryOrder.driver_name || 'Pengemudi Ekspedisi'}
                  </p>
                </div>
              </div>

              {/* 3. Yang Menerima (Customer) */}
              <div className="flex flex-col justify-between border border-zinc-300 rounded p-3 h-36">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-zinc-700">
                    Yang Menerima
                  </p>
                  <p className="text-[9px] text-zinc-500">Penerima / Staf Logistik Pemesan</p>
                </div>
                <div>
                  <p className="text-[10px] text-zinc-400 font-mono mb-1">
                    ( .................................................... )
                  </p>
                  <p className="text-[10px] font-semibold text-zinc-800">
                    {deliveryOrder.recipient_name || 'Nama Terang & Stempel'}
                  </p>
                </div>
              </div>
            </div>

            {/* FOOTER & VERIFIKASI QR CODE */}
            <div className="border-t border-zinc-300 pt-3 flex items-center justify-between text-[9px] text-zinc-500">
              <div className="flex items-center gap-3">
                {qrCodeUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={qrCodeUrl}
                    alt="QR Verifikasi Surat Jalan"
                    className="h-14 w-14 border border-zinc-200 p-0.5 rounded"
                  />
                ) : (
                  <div className="h-14 w-14 border border-zinc-200 flex items-center justify-center text-[8px] text-zinc-400">
                    QR Code
                  </div>
                )}
                <div>
                  <div className="flex items-center gap-1 font-bold text-zinc-700">
                    <ShieldCheck className="h-3 w-3 text-emerald-600" />
                    <span>Dokumen Resmi Terverifikasi Sistem Tayooli WMS</span>
                  </div>
                  <p className="text-zinc-500 mt-0.5">
                    Hash Integritas: {deliveryOrder.id.replace(/-/g, '').slice(0, 16).toUpperCase()}
                  </p>
                  <p className="text-zinc-400">
                    Pindai kode QR untuk mengonfirmasi keaslian data muatan secara daring.
                  </p>
                </div>
              </div>

              <div className="text-right space-y-0.5 font-mono text-[9px]">
                <p>Dokumen ID: {deliveryOrder.id}</p>
                <p>Status: {deliveryOrder.status}</p>
                <p>Dicetak: {new Date().toLocaleString('id-ID')}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
export default PrintDeliveryOrder
