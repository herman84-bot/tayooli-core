"use client"

import { useState, useEffect } from "react"
import { useAIPermissions, useUpdateAIPermissions } from "@/hooks/useAIPermissions"
import { useCopilot } from "@/hooks/useCopilot"
import { Shield, ShieldAlert, Check, X, Loader2, AlertCircle } from "lucide-react"
import { cn } from "@/lib/utils"

export function AutonomySettingsModal() {
  const { isSettingsOpen, closeSettings, autonomyLevel: storeAutonomyLevel, setAutonomyLevel: setStoreAutonomyLevel } = useCopilot()
  const { data: perms, isLoading } = useAIPermissions()
  const updateMutation = useUpdateAIPermissions()

  const [autonomyLevel, setAutonomyLevel] = useState<"advisory" | "assisted" | "autopilot">(storeAutonomyLevel || "assisted")
  const [scopes, setScopes] = useState<string[]>([])
  const [emergencyStop, setEmergencyStop] = useState(false)

  useEffect(() => {
    if (perms) {
      // Prioritize explicit store preference if set, otherwise sync with backend permission
      const level = storeAutonomyLevel || perms.autonomy_level || "assisted"
      setAutonomyLevel(level)
      setScopes(perms.allowed_scopes || [])
      setEmergencyStop(perms.emergency_stop)
    }
  }, [perms, storeAutonomyLevel])

  if (!isSettingsOpen) return null

  const handleToggleScope = (scope: string) => {
    setScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    )
  }

  const handleSave = async () => {
    setStoreAutonomyLevel(autonomyLevel)
    await updateMutation.mutateAsync({
      autonomy_level: autonomyLevel,
      allowed_scopes: scopes,
      emergency_stop: emergencyStop,
    })
    closeSettings()
  }

  const handleToggleEmergencyStop = async () => {
    const nextState = !emergencyStop
    setEmergencyStop(nextState)
    await updateMutation.mutateAsync({
      emergency_stop: nextState,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4 animate-in fade-in duration-150">
      <div className="w-full max-w-md rounded-xl border border-border bg-card text-card-foreground shadow-lg overflow-hidden animate-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border bg-muted/30">
          <div className="flex items-center gap-2.5">
            <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
              <Shield className="h-4 w-4" />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-foreground">Pengaturan & Wewenang AI</h3>
              <p className="text-[11px] text-muted-foreground">Governance izin & batas kendali otonom Copilot</p>
            </div>
          </div>
          <button
            onClick={closeSettings}
            className="p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Body */}
        <div className="p-5 space-y-5 max-h-[75vh] overflow-y-auto text-xs">
          {isLoading ? (
            <div className="py-8 flex items-center justify-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin text-primary" />
              <span>Memuat preferensi keamanan...</span>
            </div>
          ) : (
            <>
              {/* Emergency Stop Switch */}
              <div className={cn(
                "p-3.5 rounded-xl border transition-colors",
                emergencyStop
                  ? "bg-rose-50/80 border-rose-300 dark:bg-rose-950/40 dark:border-rose-800"
                  : "bg-muted/40 border-border"
              )}>
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-start gap-2.5">
                    <ShieldAlert className={cn("h-4 w-4 mt-0.5", emergencyStop ? "text-rose-600" : "text-muted-foreground")} />
                    <div>
                      <div className="font-semibold text-foreground">Emergency Kill-Switch</div>
                      <div className="text-[11px] text-muted-foreground">
                        Matikan seketika seluruh akses eksekusi AI di seluruh tenant
                      </div>
                    </div>
                  </div>
                  <button
                    onClick={handleToggleEmergencyStop}
                    className={cn(
                      "px-3 py-1.5 rounded-md font-medium text-xs transition-colors cursor-pointer",
                      emergencyStop
                        ? "bg-rose-600 text-white hover:bg-rose-700"
                        : "bg-background border border-border text-foreground hover:bg-muted"
                    )}
                  >
                    {emergencyStop ? "Aktif (Matikan)" : "Aktifkan"}
                  </button>
                </div>
              </div>

              {/* Autonomy Level */}
              <div>
                <label className="block font-semibold text-foreground mb-2">Tingkat Otonomi AI</label>
                <div className="space-y-2">
                  {[
                    {
                      id: "advisory",
                      title: "Level 1: Advisory (Read-Only)",
                      desc: "Hanya baca data, menjawab pertanyaan. Dilarang melakukan mutasi.",
                    },
                    {
                      id: "assisted",
                      title: "Level 2: Assisted (Rekomendasi)",
                      desc: "AI menyusun saran tindakan, eksekusi tetap memerlukan konfirmasi Anda.",
                    },
                    {
                      id: "autopilot",
                      title: "Level 3: Autopilot Terbatas",
                      desc: "Aksi rutin risiko rendah otomatis jalan. Aksi finansial tetap butuh konfirmasi.",
                    },
                  ].map((lvl) => (
                    <label
                      key={lvl.id}
                      className={cn(
                        "flex items-start gap-2.5 p-2.5 rounded-lg border cursor-pointer transition-colors",
                        autonomyLevel === lvl.id
                          ? "border-primary bg-primary/5 text-foreground"
                          : "border-border hover:bg-muted/40 text-muted-foreground"
                      )}
                    >
                      <input
                        type="radio"
                        name="autonomyLevel"
                        value={lvl.id}
                        checked={autonomyLevel === lvl.id}
                        onChange={() => setAutonomyLevel(lvl.id as any)}
                        className="mt-0.5 text-primary focus:ring-primary"
                      />
                      <div>
                        <div className="font-medium text-foreground">{lvl.title}</div>
                        <div className="text-[11px] text-muted-foreground">{lvl.desc}</div>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              {/* Granular Scopes */}
              <div>
                <label className="block font-semibold text-foreground mb-2">Cakupan Izin Operasional</label>
                <div className="space-y-1.5">
                  {[
                    { id: "workspace.read", label: "Membaca data dan ringkasan workspace" },
                    { id: "workspace.profile_write", label: "Mengubah identitas & profil perusahaan" },
                    { id: "workspace.team_write", label: "Mengundang dan mengelola tim" },
                    { id: "workspace.master_write", label: "Menyimpan draf produk dan data operasional" },
                    { id: "workspace.payments_write", label: "Konfigurasi payment gateway (Kritis)" },
                  ].map((s) => (
                    <label key={s.id} className="flex items-center gap-2 p-1.5 rounded-md hover:bg-muted/30 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={scopes.includes(s.id)}
                        onChange={() => handleToggleScope(s.id)}
                        className="rounded-xs border-border text-primary focus:ring-primary"
                      />
                      <span className="text-foreground">{s.label}</span>
                    </label>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-border bg-muted/20 flex items-center justify-end gap-2">
          <button
            onClick={closeSettings}
            className="px-3.5 py-1.5 rounded-md border border-border bg-background text-foreground hover:bg-muted font-medium transition-colors text-xs"
          >
            Batal
          </button>
          <button
            onClick={handleSave}
            disabled={updateMutation.isPending}
            className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors shadow-xs text-xs cursor-pointer"
          >
            {updateMutation.isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            <span>Simpan Perubahan</span>
          </button>
        </div>
      </div>
    </div>
  )
}
