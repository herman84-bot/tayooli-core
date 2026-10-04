/**
 * Copilot Tool Execution Dispatcher
 * Dispatches verified tool calls from ActionPreviewCard to the corresponding
 * Go backend / Next.js API endpoints.
 */

export interface ExecutionResult {
  success: boolean
  message: string
  targetModule?: string
  data?: any
  undo?: {
    method: "DELETE" | "PATCH" | "POST"
    path: string
    label: string
  }
}

export async function executeToolCall(
  toolName: string,
  payload: Record<string, any>,
  onNavigate?: (path: string) => void
): Promise<ExecutionResult> {
  switch (toolName) {
    case "update_company_profile": {
      const res = await fetch("/api/v1/settings/profile", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal memperbarui profil perusahaan")
      }
      const data = await res.json()
      return {
        success: true,
        message: `Profil perusahaan berhasil diperbarui menjadi ${payload.company_name || "data baru"}.`,
        targetModule: "settings",
        data,
      }
    }

    case "invite_team_member":
    case "invite_member": {
      const res = await fetch("/api/v1/settings/team", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal mengundang anggota tim")
      }
      const data = await res.json()
      const memberId = data?.member?.id || data?.id
      return {
        success: true,
        message: `Undangan berhasil dikirim ke ${payload.email} dengan peran ${payload.role}.`,
        targetModule: "team-members",
        data,
        undo: memberId
          ? {
              method: "DELETE",
              path: `/api/v1/settings/team/${memberId}`,
              label: `Batalkan Undangan ${payload.email}`,
            }
          : undefined,
      }
    }

    case "configure_payment_gateway": {
      const res = await fetch("/api/v1/payments/configs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal memperbarui konfigurasi payment gateway")
      }
      const data = await res.json()
      return {
        success: true,
        message: `Payment gateway ${payload.provider} berhasil diatur (Aktif: ${payload.is_active ? "Ya" : "Tidak"}).`,
        targetModule: "payments",
        data,
      }
    }

    case "create_vendor": {
      const res = await fetch("/api/v1/vendors", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          name: payload.name,
          email: payload.email || "",
          phone: payload.phone || "",
          address: payload.address || "",
          bank_account: payload.bank_account || "",
          bank_name: payload.bank_name || "",
          tax_id: payload.tax_id || "",
        }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal membuat vendor baru")
      }
      const data = await res.json()
      const vendorId = data?.id || data?.data?.id
      return {
        success: true,
        message: `Vendor '${payload.name}' berhasil ditambahkan ke master data.`,
        targetModule: "vendors",
        data,
        undo: vendorId
          ? {
              method: "DELETE",
              path: `/api/v1/vendors/${vendorId}`,
              label: `Hapus Vendor '${payload.name}'`,
            }
          : undefined,
      }
    }

    case "create_customer": {
      const res = await fetch("/api/v1/customers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          name: payload.name,
          email: payload.email || "",
          phone: payload.phone || "",
          address: payload.address || "",
        }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal membuat pelanggan baru")
      }
      const data = await res.json()
      const custId = data?.id || data?.data?.id
      return {
        success: true,
        message: `Pelanggan '${payload.name}' berhasil didaftarkan.`,
        targetModule: "customers",
        data,
        undo: custId
          ? {
              method: "DELETE",
              path: `/api/v1/customers/${custId}`,
              label: `Hapus Pelanggan '${payload.name}'`,
            }
          : undefined,
      }
    }

    case "create_product": {
      const res = await fetch("/api/v1/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          sku: payload.sku,
          name: payload.name,
          price: Number(payload.price) || 0,
          description: payload.description || "",
        }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Gagal membuat produk baru")
      }
      const data = await res.json()
      const prodId = data?.id || data?.data?.id
      return {
        success: true,
        message: `Produk '${payload.name}' (${payload.sku}) berhasil dibuat.`,
        targetModule: "products",
        data,
        undo: prodId
          ? {
              method: "DELETE",
              path: `/api/v1/products/${prodId}`,
              label: `Hapus Produk '${payload.name}'`,
            }
          : undefined,
      }
    }

    case "create_purchase_invoice": {
      if (payload.vendor_id) {
        const res = await fetch("/api/v1/invoices", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            vendor_id: payload.vendor_id,
            amount: String(payload.amount || "0"),
            currency: payload.currency || "IDR",
            description: payload.description || `Invoice untuk ${payload.vendor_name || payload.vendor_id}`,
          }),
        })
        if (!res.ok) {
          const err = await res.json().catch(() => ({}))
          throw new Error(err?.error?.message || "Gagal membuat draf invoice")
        }
        const data = await res.json()
        return {
          success: true,
          message: `Draf purchase invoice untuk vendor '${payload.vendor_name || payload.vendor_id}' senilai Rp ${(Number(payload.amount) || 0).toLocaleString("id-ID")} berhasil dibuat.`,
          targetModule: "invoices",
          data,
        }
      }

      const queryParams = new URLSearchParams()
      if (payload.vendor_name) queryParams.set("vendor", payload.vendor_name)
      if (payload.amount) queryParams.set("amount", String(payload.amount))
      if (payload.currency) queryParams.set("currency", payload.currency)

      const targetPath = `/dashboard/invoices/new${queryParams.toString() ? `?${queryParams.toString()}` : ""}`
      if (onNavigate) {
        onNavigate(targetPath)
      }
      return {
        success: true,
        message: `Membuka formulir purchase invoice baru untuk vendor '${payload.vendor_name || "-"}' senilai Rp ${(Number(payload.amount) || 0).toLocaleString("id-ID")}...`,
        targetModule: "invoices",
      }
    }

    case "get_dashboard_summary": {
      try {
        const res = await fetch("/api/v1/dashboard/summary", {
          method: "GET",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
        })
        if (res.ok) {
          const data = await res.json()
          return {
            success: true,
            message: "Ringkasan metrik dashboard keuangan berhasil dimuat.",
            targetModule: "dashboard",
            data,
          }
        }
      } catch {
        // Fall back to navigation
      }
      if (onNavigate) {
        onNavigate("/dashboard")
      }
      return {
        success: true,
        message: "Membuka halaman ringkasan dashboard...",
        targetModule: "dashboard",
      }
    }

    case "navigate_to_module": {
      if (onNavigate && payload.path) {
        onNavigate(payload.path)
      }
      return {
        success: true,
        message: `Membuka modul ${payload.module_name || payload.path}...`,
      }
    }

    default:
      throw new Error(`Tool '${toolName}' tidak dikenali oleh sistem.`)
  }
}
