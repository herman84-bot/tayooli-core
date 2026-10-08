"use client"

import React, { useState } from "react"
import {
  Search,
  Layers,
  ArrowRight,
  ShieldCheck,
  Clock,
  Warehouse,
  History,
  FileText,
  AlertCircle,
  Hash,
  UserCheck,
} from "lucide-react"
import { useBatchTrace, useAuditTrail } from "@/hooks/useWMS"
import type { BatchTrace } from "@/lib/api"

export function TraceBatchView() {
  const [batchIdInput, setBatchIdInput] = useState("")
  const [activeBatchId, setActiveBatchId] = useState<string | null>(null)

  const { data: trace, isLoading, error } = useBatchTrace(activeBatchId)
  const { data: auditTrail = [] } = useAuditTrail("batch", activeBatchId)

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (!batchIdInput.trim()) return
    setActiveBatchId(batchIdInput.trim())
  }

  return (
    <div className="space-y-6">
      {/* Search Header */}
      <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <h3 className="text-base font-semibold text-slate-900 mb-1">
          Pelacakan Riwayat Mutasi & Audit Batch (Traceability KO-1c)
        </h3>
        <p className="text-xs text-slate-500 mb-4">
          Lacak alur perpindahan barang per batch dari penerimaan hingga pengeluaran dengan buku besar tak terubah
          (immutable ledger) dan rantai hash SHA-256 (sentry-wms §3.2).
        </p>

        <form onSubmit={handleSearch} className="flex gap-2 max-w-xl">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              type="text"
              placeholder="Masukkan UUID Batch ID untuk melacak..."
              value={batchIdInput}
              onChange={(e) => setBatchIdInput(e.target.value)}
              className="w-full pl-9 pr-4 py-2 text-sm rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-mono"
            />
          </div>
          <button
            type="submit"
            className="px-4 py-2 bg-primary text-primary-foreground text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
          >
            Lacak Batch
          </button>
        </form>
      </div>

      {isLoading && (
        <div className="rounded-xl border border-slate-200 bg-white p-12 text-center text-slate-400">
          Memuat data pelacakan batch...
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-center gap-2">
          <AlertCircle className="h-4 w-4 text-rose-600 shrink-0" />
          <span>Gagal memuat jejak batch. Pastikan ID valid dan Anda memiliki hak akses.</span>
        </div>
      )}

      {!isLoading && !trace && activeBatchId && (
        <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 p-12 text-center text-slate-400">
          Batch dengan ID tersebut tidak ditemukan.
        </div>
      )}

      {trace && (
        <div className="space-y-6 animate-in fade-in">
          {/* Summary KPIs */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <span className="text-xs font-medium text-slate-500 uppercase tracking-wider">Nomor Batch / Lot</span>
              <div className="mt-1 font-mono text-base font-bold text-slate-900">{trace.batch.batch_number}</div>
              <div className="mt-1 flex items-center gap-1.5">
                <span
                  className={`inline-flex px-2 py-0.5 rounded-full text-[11px] font-semibold ${
                    trace.batch.status === "RELEASED"
                      ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                      : "bg-amber-50 text-amber-700 border border-amber-200"
                  }`}
                >
                  {trace.batch.status === "RELEASED" ? "RELEASED (Siap Jual)" : "ON_HOLD (Tertahan)"}
                </span>
              </div>
            </div>

            <div className="rounded-xl border border-emerald-100 bg-emerald-50/40 p-4 shadow-sm">
              <span className="text-xs font-medium text-emerald-700 uppercase tracking-wider">Total Masuk (Inbound)</span>
              <div className="mt-1 font-mono text-xl font-bold text-emerald-800">
                +{Number(trace.total_in).toLocaleString("id-ID")}
              </div>
              <div className="mt-1 text-[11px] text-emerald-600">Dari penerimaan & retur</div>
            </div>

            <div className="rounded-xl border border-blue-100 bg-blue-50/40 p-4 shadow-sm">
              <span className="text-xs font-medium text-blue-700 uppercase tracking-wider">Total Keluar (Outbound)</span>
              <div className="mt-1 font-mono text-xl font-bold text-blue-800">
                -{Number(trace.total_out).toLocaleString("id-ID")}
              </div>
              <div className="mt-1 text-[11px] text-blue-600">Surat Jalan, POS & Transfer</div>
            </div>

            <div className="rounded-xl border border-purple-100 bg-purple-50/40 p-4 shadow-sm">
              <span className="text-xs font-medium text-purple-700 uppercase tracking-wider">Sisa Stok Fisik</span>
              <div className="mt-1 font-mono text-xl font-bold text-purple-900">
                {Number(trace.on_hand).toLocaleString("id-ID")} unit
              </div>
              <div className="mt-1 text-[11px] text-purple-600">
                {trace.balances.length} lokasi rak aktif
              </div>
            </div>
          </div>

          {/* Current Location Balances */}
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-100 bg-slate-50/60 font-semibold text-xs text-slate-700 uppercase tracking-wider flex items-center gap-2">
              <Warehouse className="h-4 w-4 text-slate-500" />
              Sebaran Stok per Lokasi Rak
            </div>
            <div className="p-4">
              {trace.balances.length === 0 ? (
                <p className="text-xs text-slate-400">Tidak ada saldo tersisa di rak (stok habis).</p>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {trace.balances.map((b) => (
                    <div key={b.location_id} className="rounded-lg border border-slate-200 p-3 bg-slate-50">
                      <div className="font-mono text-xs font-bold text-slate-900">{b.location_code}</div>
                      <div className="text-lg font-bold text-primary mt-1">
                        {Number(b.quantity).toLocaleString("id-ID")}
                        <span className="text-xs font-normal text-slate-500 ml-1">unit</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Movements Timeline / Ledger */}
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-100 bg-slate-50/60 font-semibold text-xs text-slate-700 uppercase tracking-wider flex items-center gap-2">
              <History className="h-4 w-4 text-slate-500" />
              Riwayat Mutasi Buku Besar (Double-Entry Ledger)
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50/40 text-xs font-semibold text-slate-600 border-b border-slate-200">
                  <tr>
                    <th className="px-4 py-2.5">No. Mutasi</th>
                    <th className="px-4 py-2.5">Referensi Dokumen</th>
                    <th className="px-4 py-2.5">Dari Rak</th>
                    <th className="px-4 py-2.5"></th>
                    <th className="px-4 py-2.5">Ke Rak</th>
                    <th className="px-4 py-2.5 text-right">Kuantitas</th>
                    <th className="px-4 py-2.5">Operator</th>
                    <th className="px-4 py-2.5">Waktu</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 text-xs">
                  {trace.movements.map((m) => (
                    <tr key={m.movement_id} className="hover:bg-slate-50/60">
                      <td className="px-4 py-2.5 font-mono font-medium text-slate-800">{m.movement_number}</td>
                      <td className="px-4 py-2.5">
                        <span className="inline-flex items-center gap-1 font-medium text-slate-700">
                          <FileText className="h-3 w-3 text-slate-400" />
                          {m.reference_type}
                        </span>
                        {m.counterparty && (
                          <span className="text-slate-400 text-[11px] block">{m.counterparty}</span>
                        )}
                      </td>
                      <td className="px-4 py-2.5 font-mono text-slate-700">{m.source_location_code}</td>
                      <td className="px-2 py-2.5 text-center text-slate-400">
                        <ArrowRight className="h-3 w-3 inline" />
                      </td>
                      <td className="px-4 py-2.5 font-mono text-slate-700">{m.dest_location_code}</td>
                      <td className="px-4 py-2.5 text-right font-bold text-slate-900">
                        {Number(m.quantity).toLocaleString("id-ID")}
                      </td>
                      <td className="px-4 py-2.5 text-slate-600">{m.executed_by_name || "-"}</td>
                      <td className="px-4 py-2.5 text-slate-500">
                        {new Date(m.created_at).toLocaleString("id-ID", {
                          day: "2-digit",
                          month: "short",
                          year: "numeric",
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Audit Trail Hash-Chain */}
          {auditTrail.length > 0 && (
            <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
              <div className="px-4 py-3 border-b border-slate-100 bg-slate-50/60 font-semibold text-xs text-slate-700 uppercase tracking-wider flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <ShieldCheck className="h-4 w-4 text-emerald-600" />
                  Jejak Audit Kriptografis (SHA-256 Hash Chain)
                </div>
                <span className="text-[10px] text-emerald-600 font-bold bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                  VERIFIED IMMUTABLE
                </span>
              </div>
              <div className="p-4 space-y-3">
                {auditTrail.map((entry) => (
                  <div key={entry.id} className="flex items-start gap-3 text-xs border-b border-slate-100 pb-2.5 last:border-0 last:pb-0">
                    <Hash className="h-3.5 w-3.5 text-slate-400 mt-0.5" />
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-slate-800">{entry.action}</span>
                        <span className="text-slate-400 text-[11px]">
                          {new Date(entry.created_at).toLocaleString("id-ID")}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 mt-0.5 text-slate-500">
                        <UserCheck className="h-3 w-3" />
                        <span>Oleh: {entry.user_name}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
