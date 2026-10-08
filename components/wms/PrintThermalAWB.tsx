"use client"

// FE-08: Printable 100x150 mm thermal shipping label (AWB sticker).
// Standard 4x6 inch format for logistics dispatches (Sentry-WMS §1.3, OCA §1.3).

import React from "react"
import { Printer, X, Truck, Package, QrCode } from "lucide-react"
import { DeliveryOrder, DeliveryOrderItem } from "@/lib/api"

interface PrintThermalAWBProps {
  order: DeliveryOrder
  items: DeliveryOrderItem[]
  onClose: () => void
}

export function PrintThermalAWB({ order, items, onClose }: PrintThermalAWBProps) {
  const handlePrint = () => {
    window.print()
  }

  // Calculate total units
  const totalUnits = items.reduce((sum, it) => sum + Number(it.quantity || 0), 0)

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto print:p-0 print:bg-white print:static">
      <div className="bg-white text-zinc-900 rounded-xl shadow-2xl max-w-md w-full p-6 print:p-0 print:shadow-none print:max-w-none print:w-[100mm] print:h-[150mm]">
        {/* Print Controls (Hidden on Print) */}
        <div className="flex items-center justify-between border-b pb-4 mb-4 print:hidden">
          <div className="flex items-center gap-2">
            <Truck className="w-5 h-5 text-indigo-600" />
            <h2 className="text-base font-bold text-zinc-800">Label Thermal AWB (100x150 mm)</h2>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handlePrint}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium text-xs transition shadow-sm"
            >
              <Printer className="w-4 h-4" />
              Cetak Thermal
            </button>
            <button
              onClick={onClose}
              className="p-1.5 text-zinc-400 hover:text-zinc-600 rounded-lg hover:bg-zinc-100 transition"
              aria-label="Tutup"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 100mm x 150mm Thermal Sticker Canvas */}
        <div className="border-2 border-zinc-900 rounded p-3 text-zinc-900 bg-white font-sans text-xs flex flex-col justify-between h-[520px] print:h-full print:border-2 print:border-black print:rounded-none">
          {/* Top Section: Courier & Service */}
          <div>
            <div className="flex items-center justify-between border-b-2 border-zinc-900 pb-2">
              <div>
                <span className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block">
                  EKSPEDISI / KURIR
                </span>
                <span className="text-xl font-black uppercase tracking-tight text-zinc-900">
                  {order.expedition_name || "ARMADA GUDANG"}
                </span>
              </div>
              <div className="text-right">
                <span className="inline-block px-2 py-0.5 border-2 border-zinc-900 font-black text-xs uppercase">
                  {order.order_type || "REGULER"}
                </span>
              </div>
            </div>

            {/* Barcode & Tracking Number Representation */}
            <div className="my-3 text-center border-b-2 border-zinc-900 pb-3">
              {/* Simulated 1D Code 128 barcode bars */}
              <div className="flex justify-center items-center gap-[2px] h-12 px-4 mb-1 overflow-hidden">
                {Array.from({ length: 48 }).map((_, i) => (
                  <div
                    key={i}
                    className={`h-full bg-black ${
                      i % 7 === 0
                        ? "w-[4px]"
                        : i % 4 === 0
                        ? "w-[3px]"
                        : i % 3 === 0
                        ? "w-[2px]"
                        : "w-[1px]"
                    }`}
                  />
                ))}
              </div>
              <p className="font-mono font-black text-sm tracking-widest text-zinc-900">
                {order.tracking_number || order.do_number}
              </p>
              <p className="font-mono text-[10px] text-zinc-600">No. DO: {order.do_number}</p>
            </div>

            {/* Destination / Recipient (Big & Clear) */}
            <div className="border-b-2 border-zinc-900 pb-2 mb-2">
              <span className="text-[9px] font-bold text-zinc-500 uppercase block">PENERIMA:</span>
              <p className="text-sm font-black text-zinc-900 leading-snug">
                {order.customer_name || order.recipient_name || "PELANGGAN TAYOOLI"}
              </p>
              <p className="text-[11px] text-zinc-700 leading-tight mt-0.5">
                {order.recipient_name && order.customer_name
                  ? `U.P: ${order.recipient_name}`
                  : "Alamat tujuan pengiriman terdaftar"}
              </p>
            </div>

            {/* Sender */}
            <div className="border-b-2 border-zinc-900 pb-2 mb-2 text-[10px]">
              <span className="font-bold text-zinc-500 uppercase block">PENGIRIM:</span>
              <p className="font-bold text-zinc-900">TAYOOLI DISTRIBUTION WMS</p>
              <p className="text-zinc-600">Hub Logistik & Pergudangan Terpadu</p>
            </div>

            {/* Package Dimensions & Weight */}
            <div className="grid grid-cols-3 gap-1 border-b-2 border-zinc-900 pb-2 mb-2 text-center text-[10px]">
              <div className="border-r border-zinc-300">
                <span className="text-zinc-500 block">BERAT</span>
                <span className="font-black text-zinc-900 text-xs">
                  {order.package_weight_kg ? `${order.package_weight_kg} kg` : "1.0 kg"}
                </span>
              </div>
              <div className="border-r border-zinc-300">
                <span className="text-zinc-500 block">DIMENSI (PxLxT)</span>
                <span className="font-bold text-zinc-900 text-[11px]">
                  {order.package_length_cm || 0}x{order.package_width_cm || 0}x
                  {order.package_height_cm || 0} cm
                </span>
              </div>
              <div>
                <span className="text-zinc-500 block">KEMASAN</span>
                <span className="font-bold text-zinc-900 text-[11px] uppercase">
                  {order.packaging_type || "KARTON"}
                </span>
              </div>
            </div>

            {/* Item Manifest Preview */}
            <div className="text-[10px] space-y-1">
              <div className="flex justify-between font-bold border-b border-zinc-300 pb-0.5">
                <span>Ringkasan Isi Paket ({items.length} SKU)</span>
                <span>Total: {totalUnits} unit</span>
              </div>
              <div className="max-h-24 overflow-hidden divide-y divide-zinc-200">
                {items.slice(0, 4).map((it, idx) => (
                  <div key={it.id || idx} className="flex justify-between py-0.5">
                    <span className="truncate max-w-[190px]">
                      {it.product_name || it.product_sku}
                      {it.is_free_item && (
                        <span className="ml-1 text-[9px] font-bold text-emerald-700 bg-emerald-50 px-1 rounded">
                          BONUS
                        </span>
                      )}
                    </span>
                    <span className="font-bold font-mono">x{it.quantity}</span>
                  </div>
                ))}
                {items.length > 4 && (
                  <p className="text-center text-[9px] text-zinc-500 pt-0.5 italic">
                    +{items.length - 4} item lainnya dalam paket
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Bottom Footer */}
          <div className="pt-2 border-t-2 border-zinc-900 flex justify-between items-center text-[9px] text-zinc-500">
            <span>Disiapkan: {new Date().toLocaleDateString("id-ID")}</span>
            <span className="font-bold text-zinc-900">TAYOOLI WMS SPRINT 3</span>
          </div>
        </div>
      </div>
    </div>
  )
}
