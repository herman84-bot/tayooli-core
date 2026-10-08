"use client"

// FE-06: Printable Picking List ordered by shelf location (shelf_order / location_code ASC).
// Follows Sentry-WMS §1.2 & OCA §1.3.

import React from "react"
import { Printer, X, CheckSquare, Package, MapPin, Calendar, User, FileText } from "lucide-react"
import { PickingTaskDetail } from "@/lib/api"

interface PrintPickingListProps {
  detail: PickingTaskDetail
  onClose: () => void
}

export function PrintPickingList({ detail, onClose }: PrintPickingListProps) {
  const { task, items, delivery_order: order } = detail

  const handlePrint = () => {
    window.print()
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto print:p-0 print:bg-white print:static">
      <div className="bg-white text-zinc-900 rounded-xl shadow-2xl max-w-4xl w-full p-6 print:p-4 print:shadow-none print:max-w-none print:w-full">
        {/* Print Action Header (Hidden during print) */}
        <div className="flex items-center justify-between border-b pb-4 mb-6 print:hidden">
          <div className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-indigo-600" />
            <h2 className="text-lg font-bold text-zinc-800">Pratinjau Cetak Picking List</h2>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handlePrint}
              className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium text-sm transition shadow-sm"
            >
              <Printer className="w-4 h-4" />
              Cetak Dokumen
            </button>
            <button
              onClick={onClose}
              className="p-2 text-zinc-400 hover:text-zinc-600 rounded-lg hover:bg-zinc-100 transition"
              aria-label="Tutup"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Printable Document Content */}
        <div className="space-y-6 text-sm">
          {/* Header Title */}
          <div className="border-b-2 border-zinc-900 pb-4 flex justify-between items-start">
            <div>
              <h1 className="text-2xl font-black tracking-tight text-zinc-900 uppercase">
                DAFTAR AMBIL BARANG (PICKING LIST)
              </h1>
              <p className="text-xs text-zinc-500 font-mono mt-0.5">
                No. Task: {task.task_number || `PICK-${order.do_number}`}
              </p>
            </div>
            <div className="text-right">
              <span className="inline-block px-2.5 py-1 rounded bg-zinc-100 text-zinc-800 text-xs font-bold tracking-wider uppercase border border-zinc-300">
                {order.order_type || "DIRECT_DO"}
              </span>
              <p className="text-xs text-zinc-500 mt-1">
                Dicetak: {new Date().toLocaleString("id-ID")}
              </p>
            </div>
          </div>

          {/* Metadata Grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 bg-zinc-50 p-4 rounded-lg border border-zinc-200 print:bg-transparent print:border-zinc-300">
            <div>
              <span className="text-xs text-zinc-500 block">Surat Jalan (DO)</span>
              <span className="font-bold font-mono text-zinc-800">{order.do_number}</span>
            </div>
            <div>
              <span className="text-xs text-zinc-500 block">Pelanggan / Tujuan</span>
              <span className="font-semibold text-zinc-800">
                {order.customer_name || order.recipient_name || "-"}
              </span>
            </div>
            <div>
              <span className="text-xs text-zinc-500 block">Petugas Picker</span>
              <span className="font-semibold text-zinc-800">
                {task.picker_name || "Belum Ditugaskan"}
              </span>
            </div>
            <div>
              <span className="text-xs text-zinc-500 block">Status Pengambilan</span>
              <span className="font-bold text-indigo-700">{task.status}</span>
            </div>
          </div>

          {/* Items Table ordered by shelf_order */}
          <div className="overflow-hidden border border-zinc-300 rounded-lg">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="bg-zinc-100 border-b border-zinc-300 text-zinc-700 font-bold uppercase">
                  <th className="py-2.5 px-3 w-12 text-center">Cek</th>
                  <th className="py-2.5 px-3 w-12 text-center">Urut</th>
                  <th className="py-2.5 px-3">Lokasi Rak</th>
                  <th className="py-2.5 px-3">SKU & Nama Barang</th>
                  <th className="py-2.5 px-3">Batch / Kedaluwarsa</th>
                  <th className="py-2.5 px-3 text-right">Qty Minta</th>
                  <th className="py-2.5 px-3 text-right">Qty Ambil</th>
                  <th className="py-2.5 px-3 text-center">Catatan / Paraf</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {items && items.length > 0 ? (
                  items.map((item, idx) => (
                    <tr key={item.id} className="hover:bg-zinc-50 print:hover:bg-transparent">
                      <td className="py-3 px-3 text-center">
                        <div className="w-5 h-5 mx-auto border-2 border-zinc-400 rounded-sm"></div>
                      </td>
                      <td className="py-3 px-3 text-center font-mono font-bold text-zinc-600">
                        {item.shelf_order || idx + 1}
                      </td>
                      <td className="py-3 px-3">
                        <span className="inline-block px-2 py-0.5 rounded bg-amber-50 text-amber-900 border border-amber-200 font-mono font-bold text-xs">
                          {item.location_code || "RCK-DEFAULT"}
                        </span>
                      </td>
                      <td className="py-3 px-3">
                        <div className="font-bold text-zinc-800">{item.product_name}</div>
                        <div className="text-zinc-500 font-mono text-[11px]">{item.product_sku}</div>
                      </td>
                      <td className="py-3 px-3">
                        <div className="font-mono text-zinc-800">{item.batch_number || "-"}</div>
                        {item.expiry_date && (
                          <div className="text-zinc-500 text-[11px]">
                            Exp: {new Date(item.expiry_date).toLocaleDateString("id-ID")}
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-3 text-right font-bold text-zinc-800 text-sm">
                        {item.requested_qty}
                      </td>
                      <td className="py-3 px-3 text-right font-mono font-bold text-indigo-700">
                        {Number(item.picked_qty) > 0 ? item.picked_qty : "____"}
                      </td>
                      <td className="py-3 px-3 text-center text-zinc-400">
                        {item.status === "DAMAGED" ? (
                          <span className="text-rose-600 font-bold">RUSAK</span>
                        ) : (
                          "___________"
                        )}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={8} className="py-6 text-center text-zinc-500">
                      Tidak ada item dalam picking task ini.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Picking Guidelines & Signatures */}
          <div className="pt-4 border-t border-zinc-200 grid grid-cols-2 gap-8">
            <div className="text-xs text-zinc-600 space-y-1">
              <p className="font-bold text-zinc-800">Petunjuk Pengambilan:</p>
              <p>1. Ambil barang mengikuti urutan rak dari nomor terkecil ke terbesar.</p>
              <p>2. Pastikan nomor batch fisik dan tanggal kedaluwarsa cocok dengan dokumen.</p>
              <p>3. Jika barang rusak di rak, pisahkan dan laporkan segera ke Supervisor.</p>
            </div>

            <div className="grid grid-cols-2 gap-4 text-center">
              <div>
                <p className="text-xs text-zinc-500 mb-12">Petugas Picker,</p>
                <div className="border-b border-zinc-400 w-32 mx-auto"></div>
                <p className="text-xs font-semibold mt-1">
                  ( {task.picker_name || "...................."} )
                </p>
              </div>
              <div>
                <p className="text-xs text-zinc-500 mb-12">Supervisor Gudang,</p>
                <div className="border-b border-zinc-400 w-32 mx-auto"></div>
                <p className="text-xs font-semibold mt-1">( .................... )</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
