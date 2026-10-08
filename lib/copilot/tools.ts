/**
 * Tool Registry & Schema Declarations for Tayooli Copilot
 * Uses standard JSON Schema definitions compatible with Google GenAI / Gemini function calling.
 */

export interface ToolDefinition {
  name: string
  description: string
  parameters: {
    type: string
    properties: Record<string, any>
    required?: string[]
  }
}

export const COPILOT_TOOLS: ToolDefinition[] = [
  {
    name: "update_company_profile",
    description: "Memperbarui data profil perusahaan atau PT dalam workspace (nama legal PT, alamat kantor pusat, NPWP/Tax ID, mata uang). GUNAKAN NILAI EXACT DARI USER. Dilarang menggunakan nilai default/placeholder.",
    parameters: {
      type: "object",
      properties: {
        company_name: {
          type: "string",
          description: "Nama legal baru perusahaan/PT (wajib exact dari user)",
        },
        address: {
          type: "string",
          description: "Alamat lengkap kantor pusat",
        },
        tax_id: {
          type: "string",
          description: "Nomor Pokok Wajib Pajak (NPWP)",
        },
        currency: {
          type: "string",
          enum: ["IDR", "USD"],
          description: "Mata uang utama operasional",
        },
      },
      required: ["company_name"],
    },
  },
  {
    name: "invite_team_member",
    description: "Mengundang pengguna baru ke dalam workspace dengan hak akses tertentu. GUNAKAN EMAIL EXACT DARI USER. Role harus eksplisit.",
    parameters: {
      type: "object",
      properties: {
        email: {
          type: "string",
          description: "Alamat email pengguna yang diundang (harus format email valid)",
        },
        role: {
          type: "string",
          enum: ["admin", "accountant", "approver", "member"],
          description: "Hak akses peran dalam workspace (default: member)",
        },
      },
      required: ["email", "role"],
    },
  },
  {
    name: "configure_payment_gateway",
    description: "Mengatur penyedia payment gateway aktif untuk workspace (Pakasir atau Midtrans) beserta status aktif. Tindakan berisiko finansial tinggi.",
    parameters: {
      type: "object",
      properties: {
        provider: {
          type: "string",
          enum: ["pakasir", "midtrans"],
          description: "Nama penyedia gateway",
        },
        is_active: {
          type: "boolean",
          description: "Status pengaktifan gateway",
        },
        api_key: {
          type: "string",
          description: "API Key atau Server Key resmi dari penyedia (opsional)",
        },
      },
      required: ["provider", "is_active"],
    },
  },
  {
    name: "create_vendor",
    description: "Mendaftarkan rekanan vendor baru ke dalam master data. GUNAKAN NILAI EXACT DARI USER (nama, email, telepon, alamat). Dilarang mengarang nama atau placeholder.",
    parameters: {
      type: "object",
      properties: {
        name: {
          type: "string",
          description: "Nama vendor atau PT pemasok (wajib exact dari user, min 2 karakter)",
        },
        email: {
          type: "string",
          description: "Email resmi kontak vendor",
        },
        phone: {
          type: "string",
          description: "Nomor telepon/WhatsApp vendor",
        },
        address: {
          type: "string",
          description: "Alamat lengkap kantor atau gudang vendor",
        },
        bank_name: {
          type: "string",
          description: "Nama bank untuk pembayaran (BCA, Mandiri, BRI, BNI)",
        },
        bank_account: {
          type: "string",
          description: "Nomor rekening bank vendor",
        },
        tax_id: {
          type: "string",
          description: "NPWP vendor",
        },
      },
      required: ["name"],
    },
  },
  {
    name: "create_customer",
    description: "Mendaftarkan pelanggan (customer) baru ke dalam master data O2C. GUNAKAN NILAI EXACT DARI USER.",
    parameters: {
      type: "object",
      properties: {
        name: {
          type: "string",
          description: "Nama lengkap pelanggan atau nama PT klien (wajib exact dari user)",
        },
        email: {
          type: "string",
          description: "Alamat email pelanggan",
        },
        phone: {
          type: "string",
          description: "Nomor kontak telepon/WhatsApp",
        },
        address: {
          type: "string",
          description: "Alamat penagihan atau pengiriman",
        },
      },
      required: ["name"],
    },
  },
  {
    name: "create_product",
    description: "Menambahkan produk baru ke katalog barang dengan SKU unik dan harga. GUNAKAN NILAI EXACT DARI USER.",
    parameters: {
      type: "object",
      properties: {
        sku: {
          type: "string",
          description: "Kode SKU unik produk (contoh: PRD-001)",
        },
        name: {
          type: "string",
          description: "Nama barang atau jasa",
        },
        price: {
          type: "number",
          description: "Harga jual produk dalam format angka murni tanpa simbol",
        },
        description: {
          type: "string",
          description: "Deskripsi singkat produk",
        },
      },
      required: ["sku", "name", "price"],
    },
  },
  {
    name: "create_purchase_invoice",
    description: "Membuat draf purchase invoice (tagihan vendor/pemasok) baru dengan nama vendor dan nominal. GUNAKAN NILAI EXACT DARI USER.",
    parameters: {
      type: "object",
      properties: {
        vendor_name: {
          type: "string",
          description: "Nama mitra vendor pemasok (wajib exact)",
        },
        amount: {
          type: "number",
          description: "Total nominal tagihan invoice",
        },
        currency: {
          type: "string",
          enum: ["IDR", "USD"],
          description: "Mata uang tagihan (default IDR)",
        },
        description: {
          type: "string",
          description: "Keterangan atau peruntukan tagihan invoice",
        },
      },
      required: ["vendor_name"],
    },
  },
  {
    name: "get_dashboard_summary",
    description: "Mengambil data ringkasan metrik keuangan dan operasional dashboard workspace.",
    parameters: {
      type: "object",
      properties: {
        metric: {
          type: "string",
          enum: ["overview", "cash_flow", "invoices"],
          description: "Kategori ringkasan yang diminta",
        },
        period: {
          type: "string",
          description: "Periode waktu (contoh: current_month, today)",
        },
      },
    },
  },
  {
    name: "navigate_to_module",
    description: "Membuka halaman atau modul tertentu dalam aplikasi Tayooli ERP jika user meminta membuka menu.",
    parameters: {
      type: "object",
      properties: {
        path: {
          type: "string",
          enum: [
            "/dashboard",
            "/dashboard/invoices",
            "/dashboard/purchase-orders",
            "/dashboard/goods-receipts",
            "/dashboard/payment-orders",
            "/dashboard/vendors",
            "/customers",
            "/sales-orders",
            "/sales-invoices",
            "/products",
            "/accounting/chart-of-accounts",
            "/accounting/journal-entries",
            "/settings",
            "/dashboard/payment-gateways",
            "/pos",
            "/wms",
            "/wms/arus-barang",
            "/wms/delivery-orders",
            "/wms/marketplace",
            "/wms/transfers",
            "/wms/opname",
            "/wms/scrap",
            "/wms/scanner",
          ],
          description: "Rute URL tujuan di dalam aplikasi",
        },
        module_name: {
          type: "string",
          description: "Nama modul yang dituju (misal: 'Pengaturan', 'Katalog Produk')",
        },
      },
      required: ["path", "module_name"],
    },
  },
]
