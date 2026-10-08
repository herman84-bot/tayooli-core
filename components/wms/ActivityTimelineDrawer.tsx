"use client"

import React from "react"
import {
  X,
  History,
  UserCheck,
  ShieldCheck,
  Package,
  Truck,
  CheckCircle2,
  Clock,
  ArrowRight,
  FileText,
  Boxes,
  Info
} from "lucide-react"
import { useAuditTrail } from "@/hooks/useWMS"
import { AuditTrailEntry } from "@/lib/api"

export interface ActivityTimelineDrawerProps {
  isOpen: boolean
  onClose: () => void
  title: string
  subtitle?: string
  entityType?: string
  entityId?: string
  actors?: {
    created_by_name?: string | null
    created_at?: string | null
    confirmed_by_name?: string | null
    confirmed_at?: string | null
    approved_by_name?: string | null
    approved_at?: string | null
    packed_by_name?: string | null
    packed_at?: string | null
    dispatched_by_name?: string | null
    dispatched_at?: string | null
    executed_by_name?: string | null
    executed_at?: string | null
  }
  metadata?: {
    status?: string | null
    reference_type?: string | null
    reference_number?: string | null
    product_name?: string | null
    sku?: string | null
    source_location?: string | null
    dest_location?: string | null
    quantity?: number | string | null
    batch_number?: string | null
  }
}

export function ActivityTimelineDrawer({
  isOpen,
  onClose,
  title,
  subtitle,
  entityType,
  entityId,
  actors,
  metadata,
}: ActivityTimelineDrawerProps) {
  const { data: auditLogs = [], isLoading } = useAuditTrail(
    entityType || null,
    entityId || null
  )

  if (!isOpen) return null

  // Synthesize events from actors if auditLogs is empty or partial
  const synthesizedEvents: {
    id: string
    title: string
    actor: string
    timestamp: string
    icon: React.ComponentType<{ className?: string }>
    color: string
    desc?: string
  }[] = []

  if (actors?.created_at || actors?.created_by_name) {
    synthesizedEvents.push({
      id: "ev-created",
      title: "Dokumen Dibuat (Draft / Created)",
      actor: actors.created_by_name || "Admin Gudang",
      timestamp: actors.created_at || new Date().toISOString(),
      icon: FileText,
      color: "bg-blue-100 text-blue-700 border-blue-300",
      desc: "Inisiasi dokumen transaksi ke dalam sistem WMS",
    })
  }

  if (actors?.confirmed_at || actors?.confirmed_by_name) {
    synthesizedEvents.push({
      id: "ev-confirmed",
      title: "Dikonfirmasi / Disetujui (Confirmed / Approved)",
      actor: actors.confirmed_by_name || actors.approved_by_name || "Supervisor WMS",
      timestamp: actors.confirmed_at || actors.approved_at || actors.created_at || new Date().toISOString(),
      icon: ShieldCheck,
      color: "bg-indigo-100 text-indigo-700 border-indigo-300",
      desc: "Persetujuan alokasi stok & rilis instruksi gudang",
    })
  } else if (actors?.approved_at || actors?.approved_by_name) {
    synthesizedEvents.push({
      id: "ev-approved",
      title: "Disetujui (Approved)",
      actor: actors.approved_by_name || "Supervisor WMS",
      timestamp: actors.approved_at || actors.created_at || new Date().toISOString(),
      icon: ShieldCheck,
      color: "bg-indigo-100 text-indigo-700 border-indigo-300",
      desc: "Persetujuan transaksi mutasi / rilis stok",
    })
  }

  if (actors?.packed_at || actors?.packed_by_name) {
    synthesizedEvents.push({
      id: "ev-packed",
      title: "Pemeriksaan Meja Kemas (Packed 100%)",
      actor: actors.packed_by_name || "Petugas Meja Kemas",
      timestamp: actors.packed_at || new Date().toISOString(),
      icon: Package,
      color: "bg-emerald-100 text-emerald-700 border-emerald-300",
      desc: "Verifikasi 100% scan barcode SKU dan segel kemasan fisik",
    })
  }

  if (actors?.dispatched_at || actors?.dispatched_by_name) {
    synthesizedEvents.push({
      id: "ev-dispatched",
      title: "Pengiriman / Dispatch Selesai",
      actor: actors.dispatched_by_name || "Petugas Dispatch / Kurir",
      timestamp: actors.dispatched_at || new Date().toISOString(),
      icon: Truck,
      color: "bg-purple-100 text-purple-700 border-purple-300",
      desc: "Barang keluar dari dermaga muat dan stok dikurangkan dari buku besar",
    })
  }

  if (actors?.executed_at || actors?.executed_by_name) {
    synthesizedEvents.push({
      id: "ev-executed",
      title: "Mutasi Fisik Dieksekusi (Executed)",
      actor: actors.executed_by_name || "Operator Gudang",
      timestamp: actors.executed_at || actors.created_at || new Date().toISOString(),
      icon: CheckCircle2,
      color: "bg-emerald-100 text-emerald-700 border-emerald-300",
      desc: "Perpindahan fisik antar lokasi rak tercatat di buku besar mutasi stok",
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-slate-900/40 backdrop-blur-xs transition-opacity animate-fade-in">
      {/* Click outside to close backdrop */}
      <div className="flex-1" onClick={onClose} />

      {/* Slide-over panel */}
      <aside className="w-full max-w-md bg-white h-full shadow-2xl border-l border-slate-200 flex flex-col z-10 animate-slide-left">
        {/* Header */}
        <div className="p-4 border-b border-slate-200 flex items-start justify-between bg-slate-50/70">
          <div className="space-y-0.5">
            <div className="flex items-center gap-1.5 text-xs text-indigo-600 font-semibold uppercase tracking-wider">
              <History className="h-3.5 w-3.5" />
              <span>Riwayat Aktivitas & Jejak Audit (CR-05b)</span>
            </div>
            <h2 className="text-base font-bold text-slate-900 font-mono">{title}</h2>
            {subtitle && <p className="text-xs text-slate-500">{subtitle}</p>}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition"
            aria-label="Tutup panel"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-5 space-y-6">
          {/* Metadata Summary Card */}
          {metadata && (
            <div className="bg-slate-50 rounded-xl p-3.5 border border-slate-200 space-y-2 text-xs">
              <div className="flex items-center justify-between pb-2 border-b border-slate-200">
                <span className="text-slate-500 font-medium">Status Dokumen</span>
                <span className="font-bold px-2 py-0.5 rounded-full bg-slate-200/80 text-slate-800 uppercase text-[10px]">
                  {metadata.status || "DONE"}
                </span>
              </div>

              {metadata.product_name && (
                <div className="flex justify-between items-center text-slate-700">
                  <span className="text-slate-500">Produk</span>
                  <span className="font-bold text-slate-900 truncate max-w-[200px]">
                    {metadata.product_name} {metadata.sku && `(${metadata.sku})`}
                  </span>
                </div>
              )}

              {metadata.quantity !== undefined && (
                <div className="flex justify-between items-center text-slate-700">
                  <span className="text-slate-500">Kuantitas</span>
                  <span className="font-mono font-bold text-slate-900">{metadata.quantity} unit</span>
                </div>
              )}

              {metadata.source_location && metadata.dest_location && (
                <div className="flex justify-between items-center text-slate-700">
                  <span className="text-slate-500">Rute Lokasi</span>
                  <span className="font-mono flex items-center gap-1 font-semibold text-slate-800">
                    <span>{metadata.source_location}</span>
                    <ArrowRight className="h-3 w-3 text-slate-400" />
                    <span>{metadata.dest_location}</span>
                  </span>
                </div>
              )}

              {metadata.batch_number && (
                <div className="flex justify-between items-center text-slate-700">
                  <span className="text-slate-500">Nomor Batch (FEFO)</span>
                  <span className="font-mono font-semibold text-indigo-700 bg-indigo-50 px-1.5 py-0.5 rounded text-[11px]">
                    {metadata.batch_number}
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Actor Audit Summary Box */}
          <div className="bg-white rounded-xl border border-slate-200 p-4 space-y-3 shadow-2xs">
            <h3 className="text-xs font-bold text-slate-900 uppercase tracking-wider flex items-center gap-1.5">
              <UserCheck className="h-4 w-4 text-emerald-600" />
              <span>Daftar Pelaku (Actor Audit)</span>
            </h3>

            <div className="grid grid-cols-1 gap-2.5 text-xs divide-y divide-slate-100">
              {/* Dibuat Oleh */}
              <div className="pt-1 flex items-center justify-between">
                <span className="text-slate-500">Dibuat oleh:</span>
                <span className="font-bold text-slate-800">{actors?.created_by_name || "Admin Sistem"}</span>
              </div>

              {/* Disetujui Oleh */}
              <div className="pt-2 flex items-center justify-between">
                <span className="text-slate-500">Disetujui oleh:</span>
                <span className="font-bold text-indigo-700">
                  {actors?.confirmed_by_name || actors?.approved_by_name || (
                    <span className="text-slate-400 font-normal italic">Tidak memerlukan persetujuan</span>
                  )}
                </span>
              </div>

              {/* Dikemas Oleh (jika ada) */}
              {actors?.packed_by_name && (
                <div className="pt-2 flex items-center justify-between">
                  <span className="text-slate-500">Dikemas oleh:</span>
                  <span className="font-bold text-emerald-700">{actors.packed_by_name}</span>
                </div>
              )}

              {/* Dikirim / Dieksekusi Oleh */}
              <div className="pt-2 flex items-center justify-between">
                <span className="text-slate-500">
                  {actors?.dispatched_by_name ? "Dikirim oleh:" : "Dieksekusi oleh:"}
                </span>
                <span className="font-bold text-blue-700">
                  {actors?.dispatched_by_name || actors?.executed_by_name || "Operator / Sistem"}
                </span>
              </div>
            </div>
          </div>

          {/* Activity Timeline List */}
          <div className="space-y-4">
            <h3 className="text-xs font-bold text-slate-900 uppercase tracking-wider flex items-center gap-1.5">
              <Clock className="h-4 w-4 text-slate-500" />
              <span>Kronologi Aktivitas (Timeline)</span>
            </h3>

            {isLoading ? (
              <div className="py-8 text-center text-xs text-slate-400">Memuat jejak audit...</div>
            ) : auditLogs.length > 0 ? (
              <ol className="relative border-l border-slate-200 ml-3 space-y-6">
                {auditLogs.map((log: AuditTrailEntry) => (
                  <li key={log.id} className="ml-5">
                    <span className="absolute -left-2.5 flex h-5 w-5 items-center justify-center rounded-full bg-indigo-100 text-indigo-600 ring-4 ring-white">
                      <UserCheck className="h-3 w-3" />
                    </span>
                    <div className="space-y-0.5">
                      <div className="flex items-center justify-between text-xs">
                        <span className="font-bold text-slate-900">{log.action}</span>
                        <time className="text-[10px] text-slate-400">
                          {new Date(log.created_at).toLocaleString("id-ID", {
                            day: "numeric",
                            month: "short",
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </time>
                      </div>
                      <div className="text-[11px] text-slate-600">
                        Oleh: <strong className="text-slate-800">{log.user_name || "Sistem"}</strong>
                      </div>
                      {log.details && (
                        <p className="text-[11px] text-slate-500 bg-slate-50 p-2 rounded-lg border border-slate-150 mt-1 font-mono">
                          {JSON.stringify(log.details)}
                        </p>
                      )}
                    </div>
                  </li>
                ))}
              </ol>
            ) : synthesizedEvents.length > 0 ? (
              <ol className="relative border-l border-slate-200 ml-3 space-y-6">
                {synthesizedEvents.map((ev) => {
                  const Icon = ev.icon
                  return (
                    <li key={ev.id} className="ml-5">
                      <span className={`absolute -left-2.5 flex h-5 w-5 items-center justify-center rounded-full border ring-4 ring-white ${ev.color}`}>
                        <Icon className="h-3 w-3" />
                      </span>
                      <div className="space-y-0.5">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-bold text-slate-900">{ev.title}</span>
                          <time className="text-[10px] text-slate-400">
                            {new Date(ev.timestamp).toLocaleString("id-ID", {
                              day: "numeric",
                              month: "short",
                              hour: "2-digit",
                              minute: "2-digit",
                            })}
                          </time>
                        </div>
                        <div className="text-[11px] text-slate-600">
                          Oleh: <strong className="text-slate-800">{ev.actor}</strong>
                        </div>
                        {ev.desc && <p className="text-[11px] text-slate-500">{ev.desc}</p>}
                      </div>
                    </li>
                  )
                })}
              </ol>
            ) : (
              <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 text-center text-xs text-slate-500">
                Belum ada rekaman riwayat untuk transaksi ini.
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-200 bg-slate-50 flex items-center justify-between text-[11px] text-slate-500">
          <span className="flex items-center gap-1">
            <Info className="h-3.5 w-3.5 text-slate-400" />
            <span>Audit trail terkunci (immutable)</span>
          </span>
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1.5 bg-white border border-slate-200 text-slate-700 font-medium rounded-lg hover:bg-slate-100 transition text-xs"
          >
            Tutup
          </button>
        </div>
      </aside>
    </div>
  )
}
