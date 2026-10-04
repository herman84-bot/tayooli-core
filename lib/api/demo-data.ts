/**
 * Demo mock data for when the Go backend is unreachable.
 * Only served in development mode — never in production.
 *
 * Each key is the relative API path (e.g. "/dashboard/summary").
 * The value is the JSON response body to return.
 *
 * List endpoints return paginated envelopes: { data, total, page, per_page }.
 */

const TENANT = "550e8400-e29b-41d4-a716-446655440000"

const DEMO_INVOICES = [
  { id: "a0000000-0000-4000-8000-000000000001", tenant_id: TENANT, vendor_id: "11111111-1111-4111-8111-111111111101", vendor_name: "PT Nusantara Niaga", invoice_number: "INV-2026-0142", amount: "45000000.00", currency: "IDR", status: "pending_review", ai_confidence_score: 0.92, anomaly_score: 12, anomaly_detected: false, due_date: "2026-09-15", created_at: "2026-08-25T08:00:00Z", updated_at: "2026-08-25T08:00:00Z" },
  { id: "a0000000-0000-4000-8000-000000000002", tenant_id: TENANT, vendor_id: "11111111-1111-4111-8111-111111111102", vendor_name: "CV Karya Mandiri", invoice_number: "INV-2026-0141", amount: "8750000.00", currency: "IDR", status: "pending", ai_confidence_score: null, anomaly_score: 0, anomaly_detected: false, due_date: "2026-09-10", created_at: "2026-08-24T09:30:00Z", updated_at: "2026-08-24T09:30:00Z" },
  { id: "a0000000-0000-4000-8000-000000000003", tenant_id: TENANT, vendor_id: "11111111-1111-4111-8111-111111111103", vendor_name: "PT Maju Jaya", invoice_number: "INV-2026-0140", amount: "24500000.00", currency: "IDR", status: "approved", ai_confidence_score: 0.98, anomaly_score: 5, anomaly_detected: false, due_date: "2026-09-05", created_at: "2026-08-23T14:15:00Z", updated_at: "2026-08-24T10:00:00Z" },
  { id: "a0000000-0000-4000-8000-000000000004", tenant_id: TENANT, vendor_id: "11111111-1111-4111-8111-111111111104", vendor_name: "Toko Berkah", invoice_number: "INV-2026-0139", amount: "12300000.00", currency: "IDR", status: "rejected", ai_confidence_score: 0.45, anomaly_score: 72, anomaly_detected: true, due_date: "2026-08-30", created_at: "2026-08-22T11:00:00Z", updated_at: "2026-08-23T16:30:00Z" },
  { id: "a0000000-0000-4000-8000-000000000005", tenant_id: TENANT, vendor_id: "11111111-1111-4111-8111-111111111105", vendor_name: "PT IndoLogistik", invoice_number: "INV-2026-0138", amount: "67000000.00", currency: "IDR", status: "approved", ai_confidence_score: 0.95, anomaly_score: 8, anomaly_detected: false, due_date: "2026-09-20", created_at: "2026-08-21T08:45:00Z", updated_at: "2026-08-22T09:00:00Z" },
]

const DEMO_VENDORS = [
  {
    id: "11111111-1111-4111-8111-111111111101",
    tenant_id: TENANT,
    name: "PT Nusantara Niaga",
    email: "finance@nusantara.co.id",
    phone: "021-5551234",
    address: "Jl. Jenderal Sudirman No. 45, Jakarta",
    bank_account: "1234567890",
    bank_name: "BCA",
    tax_id: "01.234.567.8-012.000",
    status: "active" as const,
    avg_rating: "4.5",
    rating_count: 10,
    is_active: true,
    created_at: "2026-01-15T00:00:00Z",
    updated_at: "2026-01-15T00:00:00Z",
  },
  {
    id: "11111111-1111-4111-8111-111111111102",
    tenant_id: TENANT,
    name: "CV Karya Mandiri",
    email: "admin@karyamandiri.co.id",
    phone: "021-5555678",
    address: "Jl. Gatot Subroto No. 12, Jakarta",
    bank_account: "0987654321",
    bank_name: "Mandiri",
    tax_id: "02.345.678.9-013.000",
    status: "active" as const,
    avg_rating: "4.2",
    rating_count: 8,
    is_active: true,
    created_at: "2026-02-10T00:00:00Z",
    updated_at: "2026-02-10T00:00:00Z",
  },
  {
    id: "11111111-1111-4111-8111-111111111103",
    tenant_id: TENANT,
    name: "PT Maju Jaya",
    email: "finance@majujaya.com",
    phone: "021-5559012",
    address: "Jl. MH Thamrin No. 8, Jakarta",
    bank_account: "1122334455",
    bank_name: "BNI",
    tax_id: "03.456.789.0-014.000",
    status: "active" as const,
    avg_rating: "4.0",
    rating_count: 15,
    is_active: true,
    created_at: "2026-03-05T00:00:00Z",
    updated_at: "2026-03-05T00:00:00Z",
  },
  {
    id: "11111111-1111-4111-8111-111111111104",
    tenant_id: TENANT,
    name: "Toko Berkah",
    email: "toko@berkah.com",
    phone: "021-5553456",
    address: "Jl. Hayam Wuruk No. 22, Jakarta",
    bank_account: "5566778899",
    bank_name: "BRI",
    tax_id: "04.567.890.1-015.000",
    status: "active" as const,
    avg_rating: "3.8",
    rating_count: 5,
    is_active: true,
    created_at: "2026-04-20T00:00:00Z",
    updated_at: "2026-04-20T00:00:00Z",
  },
  {
    id: "11111111-1111-4111-8111-111111111105",
    tenant_id: TENANT,
    name: "PT IndoLogistik",
    email: "ap@indologistik.co.id",
    phone: "021-5557890",
    address: "Jl. Rasuna Said Blok X-5, Jakarta",
    bank_account: "9988776655",
    bank_name: "BCA",
    tax_id: "05.678.901.2-016.000",
    status: "active" as const,
    avg_rating: "4.7",
    rating_count: 22,
    is_active: true,
    created_at: "2026-01-20T00:00:00Z",
    updated_at: "2026-01-20T00:00:00Z",
  },
]

const DEMO_WAREHOUSES = [
  {
    id: "wh-001",
    tenant_id: TENANT,
    code: "WH-JKT-01",
    name: "Gudang Distribusi Cakung",
    address: "Kawasan Industri Pulogadung Blok B, Jakarta Timur",
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "wh-002",
    tenant_id: TENANT,
    code: "WH-SBY-01",
    name: "Gudang Regional Rungkut",
    address: "Kawasan Industri SIER, Surabaya",
    is_active: true,
    created_at: "2026-01-15T00:00:00Z",
    updated_at: "2026-01-15T00:00:00Z",
  },
  {
    id: "wh-003",
    tenant_id: TENANT,
    code: "WH-BDG-01",
    name: "Fasilitas Logistik Cimahi",
    address: "Jl. Industri Cimahi No. 18, Bandung",
    is_active: true,
    created_at: "2026-02-01T00:00:00Z",
    updated_at: "2026-02-01T00:00:00Z",
  },
]

const DEMO_LOCATIONS = [
  {
    id: "loc-001",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    code: "BIN-A-01",
    name: "Rak Depan Kiri A1",
    barcode: "LOC-A01-BIN",
    type: "INTERNAL",
    is_pallet: false,
    max_capacity: 500,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "loc-002",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    code: "BIN-B-04",
    name: "Rak Minyak & Cairan B4",
    barcode: "LOC-B04-BIN",
    type: "INTERNAL",
    is_pallet: false,
    max_capacity: 800,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "loc-003",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    code: "PLT-JKT-09",
    name: "Pallet LPN Beras Premium",
    barcode: "PLT-0982-LPN",
    type: "INTERNAL",
    is_pallet: true,
    pallet_number: "PLT-0982",
    max_capacity: 1200,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "loc-004",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    code: "STAGE-OUT",
    name: "Area Staging Pengiriman",
    barcode: "LOC-STAGE-OUT",
    type: "TRANSIT",
    is_pallet: false,
    max_capacity: 2000,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
]

interface DemoTransfer {
  id: string
  tenant_id: string
  transfer_number: string
  from_warehouse_id: string
  to_warehouse_id: string
  status: string
  requested_by: string
  vehicle_plate?: string | null
  driver_name?: string | null
  dispatched_at?: string | null
  received_at?: string | null
  notes?: string | null
  created_at: string
  updated_at: string
}

let DEMO_TRANSFERS: DemoTransfer[] = [
  {
    id: "tr-001",
    tenant_id: TENANT,
    transfer_number: "TR-20260408-001",
    from_warehouse_id: "wh-001",
    to_warehouse_id: "wh-002",
    status: "IN_TRANSIT",
    requested_by: "Budi Santoso",
    vehicle_plate: "B 9876 KDA",
    driver_name: "Joko Susilo",
    dispatched_at: "2026-04-08T08:30:00Z",
    notes: "Pengiriman stok mingguan sembako",
    created_at: "2026-04-08T07:00:00Z",
    updated_at: "2026-04-08T08:30:00Z",
  },
  {
    id: "tr-002",
    tenant_id: TENANT,
    transfer_number: "TR-20260407-002",
    from_warehouse_id: "wh-001",
    to_warehouse_id: "wh-003",
    status: "APPROVED",
    requested_by: "Ahmad Staff",
    vehicle_plate: "D 1455 AC",
    driver_name: "Dedi Supriyadi",
    notes: "Siap dispatch menunggu armada",
    created_at: "2026-04-07T14:00:00Z",
    updated_at: "2026-04-07T15:00:00Z",
  },
  {
    id: "tr-003",
    tenant_id: TENANT,
    transfer_number: "TR-20260405-003",
    from_warehouse_id: "wh-002",
    to_warehouse_id: "wh-001",
    status: "RECEIVED",
    requested_by: "Siti Supervisor",
    vehicle_plate: "L 8821 UW",
    driver_name: "Bambang W",
    received_at: "2026-04-06T11:00:00Z",
    notes: "Selesai dibukukan",
    created_at: "2026-04-05T09:00:00Z",
    updated_at: "2026-04-06T11:00:00Z",
  },
]

const DEMO_OPNAMES = [
  {
    id: "opn-001",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    opname_number: "OPN-20260401-001",
    status: "IN_PROGRESS",
    conducted_by: "Budi Santoso",
    approved_by: null,
    notes: "Audit Siklus Bulanan Rak A & B - Gudang Cakung",
    created_at: "2026-04-01T08:00:00Z",
    updated_at: "2026-04-01T09:30:00Z",
  },
  {
    id: "opn-002",
    tenant_id: TENANT,
    warehouse_id: "wh-002",
    opname_number: "OPN-20260325-002",
    status: "COMPLETED",
    conducted_by: "Siti Aminah",
    approved_by: "Ahmad Fauzi",
    notes: "Stock Opname Akhir Q1 2026 - Rungkut",
    created_at: "2026-03-25T09:00:00Z",
    updated_at: "2026-03-25T16:00:00Z",
  },
  {
    id: "opn-003",
    tenant_id: TENANT,
    warehouse_id: "wh-001",
    opname_number: "OPN-20260408-003",
    status: "DRAFT",
    conducted_by: "Hendro Pratama",
    approved_by: null,
    notes: "Persiapan Stock Opname Mingguan Pallet Sembako",
    created_at: "2026-04-08T10:00:00Z",
    updated_at: "2026-04-08T10:00:00Z",
  },
]

const DEMO_OPNAME_ITEMS = [
  {
    id: "opi-001",
    opname_id: "opn-001",
    tenant_id: TENANT,
    product_id: "prod-001",
    location_id: "loc-001",
    system_qty: "50.0000",
    physical_qty: "46.0000",
    discrepancy_qty: "-4.0000",
    notes: "4 sak kemasan sobek dipisahkan ke karantina",
    created_at: "2026-04-01T08:30:00Z",
  },
  {
    id: "opi-002",
    opname_id: "opn-001",
    tenant_id: TENANT,
    product_id: "prod-002",
    location_id: "loc-002",
    system_qty: "82.0000",
    physical_qty: "82.0000",
    discrepancy_qty: "0.0000",
    notes: "Stok fisik sesuai sistem",
    created_at: "2026-04-01T08:45:00Z",
  },
  {
    id: "opi-003",
    opname_id: "opn-001",
    tenant_id: TENANT,
    product_id: "prod-003",
    location_id: "loc-001",
    system_qty: "115.0000",
    physical_qty: "120.0000",
    discrepancy_qty: "5.0000",
    notes: "Ditemukan 5 kg lebih di sisi belakang rak",
    created_at: "2026-04-01T09:15:00Z",
  },
  {
    id: "opi-004",
    opname_id: "opn-002",
    tenant_id: TENANT,
    product_id: "prod-001",
    location_id: "loc-001",
    system_qty: "40.0000",
    physical_qty: "40.0000",
    discrepancy_qty: "0.0000",
    notes: "Audit Q1 terverifikasi",
    created_at: "2026-03-25T11:00:00Z",
  },
]

const DEMO_SCRAPS = [
  {
    id: "scrap-001",
    tenant_id: TENANT,
    scrap_number: "SCRAP-20260402-001",
    warehouse_id: "wh-001",
    product_id: "prod-002",
    source_location_id: "loc-002",
    scrap_location_id: "loc-004",
    quantity: "3.0000",
    reason: "Kemasan jerigen bocor tertusuk pallet saat handling",
    reported_by: "Budi Santoso",
    created_at: "2026-04-02T10:15:00Z",
  },
  {
    id: "scrap-002",
    tenant_id: TENANT,
    scrap_number: "SCRAP-20260405-002",
    warehouse_id: "wh-001",
    product_id: "prod-001",
    source_location_id: "loc-001",
    scrap_location_id: "loc-004",
    quantity: "4.0000",
    reason: "Beras basah akibat rembesan air atap sisi timur gudang",
    reported_by: "Hendro Pratama",
    created_at: "2026-04-05T14:30:00Z",
  },
  {
    id: "scrap-003",
    tenant_id: TENANT,
    scrap_number: "SCRAP-20260320-003",
    warehouse_id: "wh-002",
    product_id: "prod-004",
    source_location_id: "loc-001",
    scrap_location_id: "loc-004",
    quantity: "6.0000",
    reason: "Botol pecah saat proses pemindahan antar lorong rak",
    reported_by: "Siti Aminah",
    created_at: "2026-03-20T16:00:00Z",
  },
]

let DEMO_PRODUCTS = [
  {
    id: "prod-001",
    tenant_id: TENANT,
    name: "Kopi Susu Gula Aren Botol 250ml",
    sku: "MAS-KOP-001",
    price: 25000,
    description: "Kopi susu siap minum kemasan botol PET 250ml",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "prod-002",
    tenant_id: TENANT,
    name: "Beras Putih Premium 5kg",
    sku: "MAS-RCE-001",
    price: 80000,
    description: "Beras medium pulen kemasan karung 5kg",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "prod-003",
    tenant_id: TENANT,
    name: "Minyak Goreng Sawit 2 Liter",
    sku: "MAS-OIL-002",
    price: 30000,
    description: "Minyak goreng sawit pouch refill 2L",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "prod-004",
    tenant_id: TENANT,
    name: "Gula Pasir Kristal 1kg",
    sku: "MAS-SGR-001",
    price: 17500,
    description: "Gula pasir tebu kristal 1kg",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "prod-005",
    tenant_id: TENANT,
    name: "Teh Tarik Instant Sachet (10 pcs)",
    sku: "MAS-TEA-001",
    price: 22000,
    description: "Teh tarik bubuk pouch isi 10 sachet",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
]

let DEMO_DELIVERY_ORDERS = [
  {
    id: "do-001",
    tenant_id: TENANT,
    sales_order_id: "so-001",
    warehouse_id: "wh-001",
    do_number: "DO/2026/09/0001",
    status: "SHIPPED",
    expedition_name: "JNE Trucking (JTR)",
    tracking_number: "JTR9823419087",
    driver_name: "Bambang Sudarsono",
    vehicle_plate: "B 9482 TKL",
    recipient_name: "PT Nusantara Retail Makmur",
    received_date: null,
    created_at: "2026-09-08T09:00:00Z",
    updated_at: "2026-09-08T10:30:00Z",
  },
  {
    id: "do-002",
    tenant_id: TENANT,
    sales_order_id: "so-002",
    warehouse_id: "wh-001",
    do_number: "DO/2026/09/0002",
    status: "DELIVERED",
    expedition_name: "SiCepat Cargo",
    tracking_number: "SC9918237746",
    driver_name: "Ahmad Rifa'i",
    vehicle_plate: "B 1234 SIK",
    recipient_name: "CV Berkah Mart Sejahtera",
    received_date: "2026-09-08T14:30:00Z",
    created_at: "2026-09-07T08:00:00Z",
    updated_at: "2026-09-08T14:30:00Z",
  },
  {
    id: "do-003",
    tenant_id: TENANT,
    sales_order_id: "so-003",
    warehouse_id: "wh-002",
    do_number: "DO/2026/09/0003",
    status: "DRAFT",
    expedition_name: "Armada Gudang Internal",
    tracking_number: "INT-SBY-049",
    driver_name: "Slamet Riyadi",
    vehicle_plate: "L 8812 XY",
    recipient_name: "Toko Sembako Abadi",
    received_date: null,
    created_at: "2026-09-09T07:15:00Z",
    updated_at: "2026-09-09T07:15:00Z",
  },
]

let DEMO_DELIVERY_ORDER_ITEMS = [
  {
    id: "doi-001",
    tenant_id: TENANT,
    delivery_order_id: "do-001",
    product_id: "prod-001",
    quantity: 120,
    location_id: "loc-001",
    product_name: "Kopi Susu Gula Aren Botol 250ml",
    product_sku: "MAS-KOP-001",
    location_code: "BIN-A-01",
    created_at: "2026-09-08T09:00:00Z",
  },
  {
    id: "doi-002",
    tenant_id: TENANT,
    delivery_order_id: "do-001",
    product_id: "prod-004",
    quantity: 50,
    location_id: "loc-002",
    product_name: "Gula Pasir Kristal 1kg",
    product_sku: "MAS-SGR-001",
    location_code: "BIN-A-02",
    created_at: "2026-09-08T09:00:00Z",
  },
  {
    id: "doi-003",
    tenant_id: TENANT,
    delivery_order_id: "do-002",
    product_id: "prod-002",
    quantity: 40,
    location_id: "loc-003",
    product_name: "Beras Putih Premium 5kg",
    product_sku: "MAS-RCE-001",
    location_code: "BIN-B-01",
    created_at: "2026-09-07T08:00:00Z",
  },
  {
    id: "doi-004",
    tenant_id: TENANT,
    delivery_order_id: "do-002",
    product_id: "prod-003",
    quantity: 60,
    location_id: "loc-004",
    product_name: "Minyak Goreng Sawit 2 Liter",
    product_sku: "MAS-OIL-002",
    location_code: "BIN-B-02",
    created_at: "2026-09-07T08:00:00Z",
  },
  {
    id: "doi-005",
    tenant_id: TENANT,
    delivery_order_id: "do-003",
    product_id: "prod-005",
    quantity: 80,
    location_id: "loc-005",
    product_name: "Teh Tarik Instant Sachet (10 pcs)",
    product_sku: "MAS-TEA-001",
    location_code: "BIN-C-01",
    created_at: "2026-09-09T07:15:00Z",
  },
]

let DEMO_MARKETPLACE_BATCHES = [
  {
    id: "batch-001",
    tenant_id: TENANT,
    batch_number: "BATCH-MKT-SHOPEE-20260909-01",
    channel: "SHOPEE",
    warehouse_id: "wh-001",
    warehouse_name: "Gudang Distribusi Cakung",
    file_name: "shopee_orders_sept_week1.xlsx",
    total_orders: 120,
    processed_orders: 118,
    failed_orders: 0,
    unmapped_skus: 2,
    status: "COMPLETED",
    uploaded_by: "Admin E-commerce",
    created_at: "2026-09-09T08:30:00Z",
  },
  {
    id: "batch-002",
    tenant_id: TENANT,
    batch_number: "BATCH-MKT-TKP-20260909-02",
    channel: "TOKOPEDIA",
    warehouse_id: "wh-001",
    warehouse_name: "Gudang Distribusi Cakung",
    file_name: "tokopedia_flashsale_orders.csv",
    total_orders: 85,
    processed_orders: 85,
    failed_orders: 0,
    unmapped_skus: 0,
    status: "COMPLETED",
    uploaded_by: "Admin E-commerce",
    created_at: "2026-09-09T10:15:00Z",
  },
  {
    id: "batch-003",
    tenant_id: TENANT,
    batch_number: "BATCH-MKT-TT-20260909-03",
    channel: "TIKTOK",
    warehouse_id: "wh-002",
    warehouse_name: "Gudang Regional Rungkut",
    file_name: "tiktok_live_batch_0909.csv",
    total_orders: 45,
    processed_orders: 42,
    failed_orders: 1,
    unmapped_skus: 2,
    status: "COMPLETED",
    uploaded_by: "Spv Gudang Surabaya",
    created_at: "2026-09-09T12:00:00Z",
  },
  {
    id: "batch-004",
    tenant_id: TENANT,
    batch_number: "BATCH-MKT-LAZ-20260908-01",
    channel: "LAZADA",
    warehouse_id: "wh-003",
    warehouse_name: "Fasilitas Logistik Cimahi",
    file_name: "lazada_megasale.xlsx",
    total_orders: 60,
    processed_orders: 58,
    failed_orders: 2,
    unmapped_skus: 0,
    status: "COMPLETED",
    uploaded_by: "Operator Cimahi",
    created_at: "2026-09-08T16:20:00Z",
  },
]

let DEMO_MARKETPLACE_ORDERS = [
  {
    id: "ord-shp-001",
    tenant_id: TENANT,
    batch_id: "batch-001",
    warehouse_id: "wh-001",
    warehouse_name: "Gudang Distribusi Cakung",
    channel: "SHOPEE",
    external_order_id: "240909SHP8901A",
    order_date: "2026-09-09T08:15:00Z",
    customer_name: "Budi Santoso",
    customer_phone: "081234567890",
    shipping_address: "Jl. Sudirman No. 45, Jakarta Pusat, DKI Jakarta",
    courier: "J&T Express",
    tracking_number: "JT123456789ID",
    total_amount: 150000,
    shipping_fee: 10000,
    marketplace_fee: 4500,
    net_amount: 155500,
    status: "COMPLETED",
    sales_order_id: "so-shp-001",
    created_at: "2026-09-09T08:30:00Z",
    items: [
      {
        id: "item-001-1",
        tenant_id: TENANT,
        order_id: "ord-shp-001",
        external_sku: "KOPISUSU-BOTOL-250ML",
        product_id: "prod-001",
        item_name: "Kopi Susu Gula Aren 250ml",
        quantity: 6,
        unit_price: 25000,
        subtotal: 150000,
        is_mapped: true,
        product_name: "Kopi Susu Gula Aren Botol 250ml",
        product_sku: "MAS-KOP-001",
        multiplier: 1,
      },
    ],
  },
  {
    id: "ord-tkp-002",
    tenant_id: TENANT,
    batch_id: "batch-002",
    warehouse_id: "wh-001",
    warehouse_name: "Gudang Distribusi Cakung",
    channel: "TOKOPEDIA",
    external_order_id: "TKP-20260909-9021",
    order_date: "2026-09-09T09:45:00Z",
    customer_name: "Siti Nurhaliza",
    customer_phone: "081987654321",
    shipping_address: "Perumahan Indah Blok C3 No. 12, Kelapa Gading, Jakarta Utara",
    courier: "SiCepat REG",
    tracking_number: "004012345678",
    total_amount: 240000,
    shipping_fee: 15000,
    marketplace_fee: 6000,
    net_amount: 249000,
    status: "COMPLETED",
    sales_order_id: "so-tkp-002",
    created_at: "2026-09-09T10:15:00Z",
    items: [
      {
        id: "item-002-1",
        tenant_id: TENANT,
        order_id: "ord-tkp-002",
        external_sku: "KOPISUSU-DUS-24",
        product_id: "prod-001",
        item_name: "Kopi Susu Dus (24 pcs)",
        quantity: 2,
        unit_price: 120000,
        subtotal: 240000,
        is_mapped: true,
        product_name: "Kopi Susu Gula Aren Botol 250ml",
        product_sku: "MAS-KOP-001",
        multiplier: 24,
      },
    ],
  },
  {
    id: "ord-tt-003",
    tenant_id: TENANT,
    batch_id: "batch-003",
    warehouse_id: "wh-002",
    warehouse_name: "Gudang Regional Rungkut",
    channel: "TIKTOK",
    external_order_id: "TT-5789123004",
    order_date: "2026-09-09T11:20:00Z",
    customer_name: "Rian Pratama",
    customer_phone: "081311223344",
    shipping_address: "Jl. Asia Afrika No. 8, Bandung, Jawa Barat",
    courier: "J&T Cargo",
    tracking_number: "JTC987654321",
    total_amount: 85000,
    shipping_fee: 0,
    marketplace_fee: 4250,
    net_amount: 80750,
    status: "UNMAPPED_SKU",
    sales_order_id: null,
    created_at: "2026-09-09T12:00:00Z",
    items: [
      {
        id: "item-003-1",
        tenant_id: TENANT,
        order_id: "ord-tt-003",
        external_sku: "TT-VIRAL-MATCHA-LATTE",
        product_id: null,
        item_name: "Matcha Latte Creamy 250ml (Viral TikTok)",
        quantity: 2,
        unit_price: 42500,
        subtotal: 85000,
        is_mapped: false,
      },
    ],
  },
  {
    id: "ord-shp-004",
    tenant_id: TENANT,
    batch_id: "batch-001",
    warehouse_id: "wh-001",
    warehouse_name: "Gudang Distribusi Cakung",
    channel: "SHOPEE",
    external_order_id: "240909SHP4421X",
    order_date: "2026-09-09T08:20:00Z",
    customer_name: "Dewi Sartika",
    customer_phone: "085678901234",
    shipping_address: "Jl. Diponegoro No. 88, Menteng, Jakarta Pusat",
    courier: "Shopee Xpress Standard",
    tracking_number: "SPXID029482710",
    total_amount: 180000,
    shipping_fee: 12000,
    marketplace_fee: 5400,
    net_amount: 186600,
    status: "STOCK_INSUFFICIENT",
    sales_order_id: null,
    created_at: "2026-09-09T08:30:00Z",
    items: [
      {
        id: "item-004-1",
        tenant_id: TENANT,
        order_id: "ord-shp-004",
        external_sku: "MINYAK-GORENG-2L",
        product_id: "prod-003",
        item_name: "Minyak Goreng Sawit Pouch 2L",
        quantity: 6,
        unit_price: 30000,
        subtotal: 180000,
        is_mapped: true,
        product_name: "Minyak Goreng Sawit 2 Liter",
        product_sku: "MAS-OIL-002",
        multiplier: 1,
      },
    ],
  },
  {
    id: "ord-laz-005",
    tenant_id: TENANT,
    batch_id: "batch-004",
    warehouse_id: "wh-003",
    warehouse_name: "Fasilitas Logistik Cimahi",
    channel: "LAZADA",
    external_order_id: "LAZ-889102-ID",
    order_date: "2026-09-08T15:30:00Z",
    customer_name: "Hendra Wijaya",
    customer_phone: "087812345678",
    shipping_address: "Komp. Cimahi Permai Blok D No. 5, Cimahi, Jawa Barat",
    courier: "Ninja Van",
    tracking_number: "NVNID00991823",
    total_amount: 320000,
    shipping_fee: 20000,
    marketplace_fee: 9600,
    net_amount: 330400,
    status: "COMPLETED",
    sales_order_id: "so-laz-005",
    created_at: "2026-09-08T16:20:00Z",
    items: [
      {
        id: "item-005-1",
        tenant_id: TENANT,
        order_id: "ord-laz-005",
        external_sku: "BERAS-PREM-5KG",
        product_id: "prod-002",
        item_name: "Beras Premium Pulen 5kg",
        quantity: 4,
        unit_price: 80000,
        subtotal: 320000,
        is_mapped: true,
        product_name: "Beras Putih Premium 5kg",
        product_sku: "MAS-RCE-001",
        multiplier: 1,
      },
    ],
  },
]

let DEMO_MARKETPLACE_SKU_MAPPINGS = [
  {
    id: "map-001",
    tenant_id: TENANT,
    channel_name: "SHOPEE",
    external_sku: "KOPISUSU-BOTOL-250ML",
    external_name: "Kopi Susu Gula Aren 250ml - Botol Siap Minum",
    product_id: "prod-001",
    product_name: "Kopi Susu Gula Aren Botol 250ml",
    product_sku: "MAS-KOP-001",
    multiplier: 1,
    mapping_type: "MARKETPLACE",
    created_at: "2026-08-01T10:00:00Z",
  },
  {
    id: "map-002",
    tenant_id: TENANT,
    channel_name: "TOKOPEDIA",
    external_sku: "KOPISUSU-DUS-24",
    external_name: "Kopi Susu Dus (Karton isi 24 pcs)",
    product_id: "prod-001",
    product_name: "Kopi Susu Gula Aren Botol 250ml",
    product_sku: "MAS-KOP-001",
    multiplier: 24,
    mapping_type: "MARKETPLACE",
    created_at: "2026-08-05T11:00:00Z",
  },
  {
    id: "map-003",
    tenant_id: TENANT,
    channel_name: "SHOPEE",
    external_sku: "MINYAK-GORENG-2L",
    external_name: "Minyak Goreng Sawit Pouch 2 Liter Refill",
    product_id: "prod-003",
    product_name: "Minyak Goreng Sawit 2 Liter",
    product_sku: "MAS-OIL-002",
    multiplier: 1,
    mapping_type: "MARKETPLACE",
    created_at: "2026-08-10T09:00:00Z",
  },
  {
    id: "map-004",
    tenant_id: TENANT,
    channel_name: "LAZADA",
    external_sku: "BERAS-PREM-5KG",
    external_name: "Beras Premium Pulen Karung 5kg",
    product_id: "prod-002",
    product_name: "Beras Putih Premium 5kg",
    product_sku: "MAS-RCE-001",
    multiplier: 1,
    mapping_type: "MARKETPLACE",
    created_at: "2026-08-12T14:30:00Z",
  },
  {
    id: "map-005",
    tenant_id: TENANT,
    channel_name: "TIKTOK",
    external_sku: "GULA-PASIR-1KG",
    external_name: "Gula Pasir Tebu Kristal Putih 1kg",
    product_id: "prod-004",
    product_name: "Gula Pasir Kristal 1kg",
    product_sku: "MAS-SGR-001",
    multiplier: 1,
    mapping_type: "MARKETPLACE",
    created_at: "2026-08-15T16:00:00Z",
  },
]

const DEMO_CUSTOMERS = [
  {
    id: "00000000-0000-0000-0001-000000000001",
    name: "PT Retail Sukses Mandiri",
    email: "procurement@retailsukses.co.id",
    phone: "021-88991234",
    address: "Jl. Sudirman Kav. 21, Jakarta Selatan",
    created_at: "2026-01-10T08:00:00Z",
    updated_at: "2026-01-10T08:00:00Z",
  },
  {
    id: "00000000-0000-0000-0001-000000000002",
    name: "Toko Kelontong Berkah Jaya",
    email: "berkahjaya@gmail.com",
    phone: "081234567890",
    address: "Jl. Pasar Baru No. 45, Bandung",
    created_at: "2026-02-15T09:30:00Z",
    updated_at: "2026-02-15T09:30:00Z",
  },
  {
    id: "00000000-0000-0000-0001-000000000003",
    name: "CV Mitra Abadi Sejahtera",
    email: "purchasing@mitraabadi.com",
    phone: "031-7788990",
    address: "Kawasan SIER Rungkut, Surabaya",
    created_at: "2026-03-01T10:00:00Z",
    updated_at: "2026-03-01T10:00:00Z",
  },
]

const DEMO_SALES_ORDERS = [
  {
    id: "00000000-0000-0000-0002-000000000001",
    customer_id: "00000000-0000-0000-0001-000000000001",
    customer_name: "PT Retail Sukses Mandiri",
    order_number: "SO-2026-0001",
    total_amount: 15500000,
    status: "CONFIRMED",
    created_at: "2026-08-10T09:00:00Z",
  },
  {
    id: "00000000-0000-0000-0002-000000000002",
    customer_id: "00000000-0000-0000-0001-000000000002",
    customer_name: "Toko Kelontong Berkah Jaya",
    order_number: "SO-2026-0002",
    total_amount: 4750000,
    status: "PENDING",
    created_at: "2026-08-12T14:30:00Z",
  },
  {
    id: "00000000-0000-0000-0002-000000000003",
    customer_id: "00000000-0000-0000-0001-000000000003",
    customer_name: "CV Mitra Abadi Sejahtera",
    order_number: "SO-2026-0003",
    total_amount: 32000000,
    status: "CONFIRMED",
    created_at: "2026-08-15T11:15:00Z",
  },
]

interface DemoSalesInvoice {
  id: string
  tenant_id: string
  sales_order_id: string | null
  invoice_number: string
  amount: string
  status: string
  due_date: string
  created_at: string
  updated_at: string
}

const DEMO_SALES_INVOICES: DemoSalesInvoice[] = [
  {
    id: "00000000-0000-0000-0003-000000000001",
    tenant_id: TENANT,
    sales_order_id: "00000000-0000-0000-0002-000000000001",
    invoice_number: "INV-SO-2026-0001",
    amount: "15500000",
    status: "PAID",
    due_date: "2026-09-10",
    created_at: "2026-08-10T10:00:00Z",
    updated_at: "2026-08-11T12:00:00Z",
  },
  {
    id: "00000000-0000-0000-0003-000000000002",
    tenant_id: TENANT,
    sales_order_id: "00000000-0000-0000-0002-000000000002",
    invoice_number: "INV-SO-2026-0002",
    amount: "4750000",
    status: "UNPAID",
    due_date: "2026-09-15",
    created_at: "2026-08-12T15:00:00Z",
    updated_at: "2026-08-12T15:00:00Z",
  },
]

const DEMO_ACCOUNTS = [
  {
    id: "acc-001",
    tenant_id: TENANT,
    code: "1001",
    name: "Kas & Bank (BCA Operasional)",
    type: "Asset",
    parent_id: null,
    balance: 850000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-002",
    tenant_id: TENANT,
    code: "1101",
    name: "Piutang Usaha (AR)",
    type: "Asset",
    parent_id: null,
    balance: 240000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-003",
    tenant_id: TENANT,
    code: "1201",
    name: "Persediaan Barang Dagang",
    type: "Asset",
    parent_id: null,
    balance: 420000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-004",
    tenant_id: TENANT,
    code: "2001",
    name: "Utang Usaha (AP)",
    type: "Liability",
    parent_id: null,
    balance: 310000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-005",
    tenant_id: TENANT,
    code: "3001",
    name: "Modal Disetor",
    type: "Equity",
    parent_id: null,
    balance: 1000000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-006",
    tenant_id: TENANT,
    code: "4001",
    name: "Pendapatan Penjualan",
    type: "Revenue",
    parent_id: null,
    balance: 650000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "acc-007",
    tenant_id: TENANT,
    code: "5001",
    name: "Beban Pokok Penjualan (HPP)",
    type: "Expense",
    parent_id: null,
    balance: 380000000,
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
]

const DEMO_JOURNAL_ENTRIES = [
  {
    id: "je-001",
    tenant_id: TENANT,
    date: "2026-08-25",
    reference_id: "INV-2026-0142",
    description: "Penerimaan Tagihan Logistik PT Nusantara Niaga",
    total_debit: 45000000,
    total_credit: 45000000,
    status: "posted" as const,
    lines: [
      { id: "jel-001", account_id: "acc-007", account_name: "Beban Pokok Penjualan (HPP)", debit: 45000000, credit: 0 },
      { id: "jel-002", account_id: "acc-004", account_name: "Utang Usaha (AP)", debit: 0, credit: 45000000 },
    ],
    created_at: "2026-08-25T08:30:00Z",
    updated_at: "2026-08-25T08:30:00Z",
  },
  {
    id: "je-002",
    tenant_id: TENANT,
    date: "2026-08-24",
    reference_id: "INV-SO-2026-0001",
    description: "Pelunasan Piutang PT Retail Sukses Mandiri",
    total_debit: 15500000,
    total_credit: 15500000,
    status: "posted" as const,
    lines: [
      { id: "jel-003", account_id: "acc-001", account_name: "Kas & Bank (BCA Operasional)", debit: 15500000, credit: 0 },
      { id: "jel-004", account_id: "acc-002", account_name: "Piutang Usaha (AR)", debit: 0, credit: 15500000 },
    ],
    created_at: "2026-08-24T10:15:00Z",
    updated_at: "2026-08-24T10:15:00Z",
  },
]

const DEMO_APPROVALS = [
  {
    id: "appr-001",
    tenant_id: TENANT,
    workflow_id: "wf-invoice-approval",
    target_type: "invoice",
    target_id: "a0000000-0000-4000-8000-000000000001",
    status: "pending" as const,
    current_step_index: 1,
    requested_by: "Admin Keuangan",
    created_at: "2026-08-25T09:00:00Z",
    updated_at: "2026-08-25T09:00:00Z",
  },
  {
    id: "appr-002",
    tenant_id: TENANT,
    workflow_id: "wf-invoice-approval",
    target_type: "invoice",
    target_id: "a0000000-0000-4000-8000-000000000003",
    status: "approved" as const,
    current_step_index: 2,
    requested_by: "Admin Keuangan",
    approved_by: "Finance Manager",
    approved_at: "2026-08-24T10:00:00Z",
    created_at: "2026-08-23T14:15:00Z",
    updated_at: "2026-08-24T10:00:00Z",
  },
]

const DEMO_PURCHASE_ORDERS = [
  {
    id: "44444444-4444-4444-8444-444444440001",
    tenant_id: TENANT,
    vendor_id: "11111111-1111-4111-8111-111111111101",
    po_number: "PO-2026-0089",
    amount: "45000000.00",
    qty: 500,
    currency: "IDR",
    status: "open" as const,
    created_at: "2026-08-20T08:00:00Z",
    updated_at: "2026-08-20T08:00:00Z",
  },
  {
    id: "44444444-4444-4444-8444-444444440002",
    tenant_id: TENANT,
    vendor_id: "11111111-1111-4111-8111-111111111102",
    po_number: "PO-2026-0088",
    amount: "8750000.00",
    qty: 150,
    currency: "IDR",
    status: "received" as const,
    created_at: "2026-08-18T09:30:00Z",
    updated_at: "2026-08-22T14:00:00Z",
  },
  {
    id: "44444444-4444-4444-8444-444444440003",
    tenant_id: TENANT,
    vendor_id: "11111111-1111-4111-8111-111111111103",
    po_number: "PO-2026-0087",
    amount: "24500000.00",
    qty: 300,
    currency: "IDR",
    status: "partially_received" as const,
    created_at: "2026-08-15T11:00:00Z",
    updated_at: "2026-08-19T10:30:00Z",
  },
]

const DEMO_GOODS_RECEIPTS = [
  {
    id: "55555555-5555-4555-8555-555555550001",
    tenant_id: TENANT,
    po_id: "44444444-4444-4444-8444-444444440002",
    vendor_id: "11111111-1111-4111-8111-111111111102",
    received_qty: 150,
    received_amount: "8750000.00",
    currency: "IDR",
    status: "accepted" as const,
    received_at: "2026-08-22T14:00:00Z",
    created_at: "2026-08-22T14:00:00Z",
  },
  {
    id: "55555555-5555-4555-8555-555555550002",
    tenant_id: TENANT,
    po_id: "44444444-4444-4444-8444-444444440003",
    vendor_id: "11111111-1111-4111-8111-111111111103",
    received_qty: 150,
    received_amount: "12250000.00",
    currency: "IDR",
    status: "accepted" as const,
    received_at: "2026-08-19T10:30:00Z",
    created_at: "2026-08-19T10:30:00Z",
  },
]

const DEMO_PAYMENT_ORDERS = [
  {
    id: "66666666-6666-4666-8666-666666660001",
    tenant_id: TENANT,
    invoice_id: "a0000000-0000-4000-8000-000000000001",
    amount: "24500000.00",
    currency: "IDR",
    payment_method: "bank_transfer",
    reference_number: "TRF-BCA-20260824-001",
    status: "paid" as const,
    notes: "Pelunasan invoice PT Maju Jaya",
    created_by: "99999999-9999-4999-8999-999999990001",
    approved_by: "99999999-9999-4999-8999-999999990002",
    paid_at: "2026-08-24T16:00:00Z",
    created_at: "2026-08-24T10:00:00Z",
    updated_at: "2026-08-24T16:00:00Z",
  },
  {
    id: "66666666-6666-4666-8666-666666660002",
    tenant_id: TENANT,
    invoice_id: "a0000000-0000-4000-8000-000000000001",
    amount: "67000000.00",
    currency: "IDR",
    payment_method: "bank_transfer",
    reference_number: null,
    status: "approved" as const,
    notes: "Siap bayar PT IndoLogistik",
    created_by: "99999999-9999-4999-8999-999999990001",
    approved_by: "99999999-9999-4999-8999-999999990002",
    paid_at: null,
    created_at: "2026-08-23T11:00:00Z",
    updated_at: "2026-08-24T09:00:00Z",
  },
]

const DEMO_PLANS = [
  { name: "Starter", monthly: 199000, annual: 159200, featured: false },
  { name: "Pro", monthly: 449000, annual: 359200, featured: true },
  { name: "Enterprise", monthly: 1299000, annual: 1039200, featured: false },
]

const DEMO_PLAN_LIMITS: Record<string, unknown> = {
  starter: { id: "lim-starter", plan: "Starter", max_users: 5, max_vendors: 100, max_invoices_per_month: 500, max_ocr_per_month: 200, multi_entity: false, api_access: false, created_at: "2026-01-01T00:00:00Z" },
  pro: { id: "lim-pro", plan: "Pro", max_users: 25, max_vendors: 500, max_invoices_per_month: 2000, max_ocr_per_month: 1000, multi_entity: true, api_access: true, created_at: "2026-01-01T00:00:00Z" },
  bisnis: { id: "lim-bisnis", plan: "Pro", max_users: 25, max_vendors: 500, max_invoices_per_month: 2000, max_ocr_per_month: 1000, multi_entity: true, api_access: true, created_at: "2026-01-01T00:00:00Z" },
  enterprise: { id: "lim-enterprise", plan: "Enterprise", max_users: 999, max_vendors: 9999, max_invoices_per_month: 99999, max_ocr_per_month: 99999, multi_entity: true, api_access: true, created_at: "2026-01-01T00:00:00Z" },
}

const DEMO_SUBSCRIPTION = {
  id: "sub-demo-001",
  tenant_id: TENANT,
  plan: "Enterprise",
  status: "active",
  trial_started_at: "2026-01-01T00:00:00Z",
  trial_ends_at: "2036-01-01T00:00:00Z",
  billing_period: "annual",
  current_period_start: "2026-01-01T00:00:00Z",
  current_period_end: "2036-01-01T00:00:00Z",
  payment_provider: "enterprise",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
}

const DEMO_SUBSCRIPTION_INVOICES = [
  {
    id: "sub-inv-001",
    tenant_id: TENANT,
    subscription_id: "sub-demo-001",
    invoice_number: "INV-SUB-2026-08",
    amount: 449000,
    currency: "IDR",
    status: "paid",
    period_start: "2026-08-01T00:00:00Z",
    period_end: "2026-09-01T00:00:00Z",
    payment_provider: "midtrans",
    due_date: "2026-08-05T00:00:00Z",
    paid_at: "2026-08-01T10:00:00Z",
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T10:00:00Z",
  },
]

const DEMO_TEAM_MEMBERS = [
  { id: "usr-001", email: "admin@test.com", full_name: "Admin Tayooli", role: "admin", created_at: "2026-01-01T00:00:00Z" },
  { id: "usr-002", email: "finance@test.com", full_name: "Finance Manager", role: "finance", created_at: "2026-01-15T00:00:00Z" },
  { id: "usr-003", email: "staff@test.com", full_name: "Gudang Operator", role: "warehouse", created_at: "2026-02-01T00:00:00Z" },
]

const DEMO_AI_PERMISSIONS = {
  id: "demo-perm",
  autonomy_level: "assisted",
  allowed_scopes: ["workspace.read", "workspace.profile_write"],
  emergency_stop: false,
}

const DEMO_USAGE = {
  users: 3,
  vendors: 5,
  invoices: 12,
  ocr: 45,
}

/** Paginate an array of items. */
function paginate<T>(items: T[], searchParams: URLSearchParams): { data: T[]; total: number; page: number; per_page: number } {
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10))
  const per_page = Math.min(100, Math.max(1, parseInt(searchParams.get("per_page") ?? "20", 10)))
  const q = searchParams.get("q")?.toLowerCase()

  let filtered = items
  if (q) {
    filtered = items.filter((item) => {
      const values = Object.values(item as Record<string, unknown>)
      return values.some((v) => typeof v === "string" && v.toLowerCase().includes(q))
    })
  }

  const start = (page - 1) * per_page
  return {
    data: filtered.slice(start, start + per_page),
    total: filtered.length,
    page,
    per_page,
  }
}

/** Returns demo data for a given API path + query string, or null if no demo data exists. */
export function getDemoResponse(
  path: string,
  search = "",
  body?: unknown,
  method: string | unknown = "GET"
): unknown | null {
  if (process.env.NODE_ENV === "production") return null

  // Support backwards compatibility if 3rd argument is method ("GET"|"POST"|"PUT"|"PATCH"|"DELETE")
  // and 4th argument is body:
  let effectiveMethod = typeof method === "string" ? method : "GET"
  let effectiveBody = body
  if (typeof body === "string" && ["GET", "POST", "PUT", "PATCH", "DELETE"].includes(body.toUpperCase())) {
    effectiveMethod = body.toUpperCase()
    effectiveBody = method
  }

  const normalizedPath = path.startsWith("/api/v1") ? path.replace("/api/v1", "") : path
  const params = new URLSearchParams(search)
  const bodyRecord = (effectiveBody && typeof effectiveBody === "object" ? effectiveBody : {}) as Record<string, unknown>

  // Handle mutations in demo mode
  if (effectiveMethod === "POST") {
    if (normalizedPath === "/products") {
      const newProduct = {
        id: `prod-${Date.now()}`,
        tenant_id: TENANT,
        name: (bodyRecord.name as string) || "Produk Baru",
        sku: (bodyRecord.sku as string) || `SKU-${Date.now().toString().slice(-4)}`,
        price: Number(bodyRecord.price) || 0,
        description: (bodyRecord.description as string) || "",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      DEMO_PRODUCTS.unshift(newProduct)
      return newProduct
    }
    if (normalizedPath === "/customers") {
      const newCust = {
        id: `00000000-0000-0000-0001-${Date.now().toString().slice(-12).padStart(12, "0")}`,
        name: (bodyRecord.name as string) || "Customer Baru",
        email: (bodyRecord.email as string) || "customer@example.com",
        phone: (bodyRecord.phone as string) || "",
        address: (bodyRecord.address as string) || "",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      DEMO_CUSTOMERS.unshift(newCust)
      return newCust
    }
    if (normalizedPath === "/sales-orders") {
      const cust = DEMO_CUSTOMERS.find((c) => c.id === bodyRecord.customer_id) || DEMO_CUSTOMERS[0]
      const newSo = {
        id: `00000000-0000-0000-0002-${Date.now().toString().slice(-12).padStart(12, "0")}`,
        customer_id: (bodyRecord.customer_id as string) || cust.id,
        customer_name: cust.name,
        order_number: (bodyRecord.order_number as string) || `SO-2026-${Math.floor(1000 + Math.random() * 9000)}`,
        total_amount: Number(bodyRecord.total_amount) || 0,
        status: "CONFIRMED",
        created_at: new Date().toISOString(),
      }
      DEMO_SALES_ORDERS.unshift(newSo)
      return newSo
    }
    if (normalizedPath === "/sales-invoices") {
      const newSi = {
        id: `00000000-0000-0000-0003-${Date.now().toString().slice(-12).padStart(12, "0")}`,
        tenant_id: TENANT,
        sales_order_id: (bodyRecord.sales_order_id as string) || null,
        invoice_number: (bodyRecord.invoice_number as string) || `INV-${Date.now().toString().slice(-6)}`,
        amount: String(bodyRecord.amount || "0"),
        status: "UNPAID",
        due_date: (bodyRecord.due_date as string) || new Date().toISOString().slice(0, 10),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      DEMO_SALES_INVOICES.unshift(newSi)
      return newSi
    }
    if (normalizedPath === "/subscription") {
      return {
        ...DEMO_SUBSCRIPTION,
        plan: (bodyRecord.plan as string) || "Pro",
        billing_period: (bodyRecord.period as "monthly" | "annual") || "monthly",
      }
    }
  }

  if (effectiveMethod === "PATCH") {
    if (normalizedPath === "/ai/permissions") {
      return {
        ...DEMO_AI_PERMISSIONS,
        ...(bodyRecord || {}),
      }
    }
    if (normalizedPath === "/subscription") {
      return { status: "active" }
    }
    if (normalizedPath.startsWith("/products/")) {
      const id = normalizedPath.split("/")[2]
      const idx = DEMO_PRODUCTS.findIndex((p) => p.id === id)
      if (idx !== -1) {
        DEMO_PRODUCTS[idx] = {
          ...DEMO_PRODUCTS[idx],
          name: (bodyRecord.name as string) || DEMO_PRODUCTS[idx].name,
          sku: (bodyRecord.sku as string) || DEMO_PRODUCTS[idx].sku,
          price: bodyRecord.price !== undefined ? Number(bodyRecord.price) : DEMO_PRODUCTS[idx].price,
          description: bodyRecord.description !== undefined ? (bodyRecord.description as string) : DEMO_PRODUCTS[idx].description,
          updated_at: new Date().toISOString(),
        }
        return DEMO_PRODUCTS[idx]
      }
    }
    if (normalizedPath === "/auth/forgot-password") {
      return { message: "link reset password telah dikirim" }
    }
    if (normalizedPath === "/auth/reset-password") {
      return { message: "password berhasil diubah" }
    }
  }

  if (effectiveMethod === "DELETE") {
    if (normalizedPath.startsWith("/products/")) {
      const id = normalizedPath.split("/")[2]
      DEMO_PRODUCTS = DEMO_PRODUCTS.filter((p) => p.id !== id)
      return { success: true }
    }
  }

  // Direct entity endpoints (array or single object)
  if (normalizedPath === "/customers") return DEMO_CUSTOMERS
  if (normalizedPath === "/sales-orders") return DEMO_SALES_ORDERS
  if (normalizedPath === "/sales-invoices") return DEMO_SALES_INVOICES
  if (normalizedPath === "/plans") return DEMO_PLANS
  if (normalizedPath === "/subscription") return DEMO_SUBSCRIPTION
  if (normalizedPath === "/subscription/invoices") return DEMO_SUBSCRIPTION_INVOICES
  if (normalizedPath === "/team-members") return DEMO_TEAM_MEMBERS
  if (normalizedPath === "/settings/team") return { members: DEMO_TEAM_MEMBERS }
  if (normalizedPath === "/ai/permissions") return DEMO_AI_PERMISSIONS
  if (normalizedPath === "/usage") return DEMO_USAGE

  if (normalizedPath.startsWith("/plans/") && normalizedPath.endsWith("/limits")) {
    const planKey = normalizedPath.split("/")[2]?.toLowerCase() || "pro"
    return DEMO_PLAN_LIMITS[planKey] || DEMO_PLAN_LIMITS.pro
  }

  if (normalizedPath.startsWith("/customers/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_CUSTOMERS.find((c) => c.id === id) || DEMO_CUSTOMERS[0]
  }
  if (normalizedPath.startsWith("/sales-orders/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_SALES_ORDERS.find((s) => s.id === id) || DEMO_SALES_ORDERS[0]
  }
  if (normalizedPath.startsWith("/sales-invoices/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_SALES_INVOICES.find((s) => s.id === id) || DEMO_SALES_INVOICES[0]
  }
  if (normalizedPath.startsWith("/purchase-orders/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_PURCHASE_ORDERS.find((p) => p.id === id) || DEMO_PURCHASE_ORDERS[0]
  }
  if (normalizedPath.startsWith("/goods-receipts/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_GOODS_RECEIPTS.find((g) => g.id === id) || DEMO_GOODS_RECEIPTS[0]
  }
  if (normalizedPath.startsWith("/payment-orders/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_PAYMENT_ORDERS.find((p) => p.id === id) || DEMO_PAYMENT_ORDERS[0]
  }
  if (normalizedPath.startsWith("/accounts/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_ACCOUNTS.find((a) => a.id === id) || DEMO_ACCOUNTS[0]
  }
  if (normalizedPath.startsWith("/journal-entries/")) {
    const id = normalizedPath.split("/")[2]
    return DEMO_JOURNAL_ENTRIES.find((j) => j.id === id) || DEMO_JOURNAL_ENTRIES[0]
  }
  if (normalizedPath.startsWith("/approvals/")) {
    const parts = normalizedPath.split("/")
    const id = parts[2]
    const action = parts[3]
    const found = DEMO_APPROVALS.find((a) => a.id === id) || DEMO_APPROVALS[0]
    if (action === "approve") {
      return { ...found, status: "approved", approved_at: new Date().toISOString() }
    }
    if (action === "reject") {
      return { ...found, status: "rejected", rejected_at: new Date().toISOString() }
    }
    return found
  }

  // Static endpoints
  const STATIC: Record<string, unknown> = {
    "/dashboard/summary": {
      invoices: { total: 24, pending: 8, approved: 12, rejected: 2, pending_review: 2, total_amount: "1250000000", approved_amount: "890000000" },
      payments: { total: 18, paid: 12, paid_amount: "780000000", pending_amount: "470000000" },
      vendors: { active: 15 },
      purchase_orders: { total: 31 },
      goods_receipts: { total: 22 },
      monthly_trend: [
        { month: "Mar 2026", invoice_count: 3, total_amount: "180000000" },
        { month: "Apr 2026", invoice_count: 5, total_amount: "320000000" },
        { month: "Mei 2026", invoice_count: 4, total_amount: "210000000" },
        { month: "Jun 2026", invoice_count: 6, total_amount: "290000000" },
        { month: "Jul 2026", invoice_count: 3, total_amount: "140000000" },
        { month: "Agu 2026", invoice_count: 3, total_amount: "110000000" },
      ],
      top_vendors: [
        { vendor_id: "11111111-1111-4111-8111-111111111101", vendor_name: "PT Nusantara Niaga", invoice_count: 8, total_amount: "450000000" },
        { vendor_id: "11111111-1111-4111-8111-111111111102", vendor_name: "CV Karya Mandiri", invoice_count: 6, total_amount: "280000000" },
        { vendor_id: "11111111-1111-4111-8111-111111111103", vendor_name: "PT Maju Jaya", invoice_count: 4, total_amount: "195000000" },
        { vendor_id: "11111111-1111-4111-8111-111111111104", vendor_name: "Toko Berkah", invoice_count: 3, total_amount: "125000000" },
        { vendor_id: "11111111-1111-4111-8111-111111111105", vendor_name: "PT IndoLogistik", invoice_count: 3, total_amount: "200000000" },
      ],
    },
  }

  if (normalizedPath in STATIC) return STATIC[normalizedPath]

  if (normalizedPath === "/wms/warehouses") {
    return { data: DEMO_WAREHOUSES }
  }
  if (normalizedPath === "/wms/locations") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return { data: DEMO_LOCATIONS.filter((l) => l.warehouse_id === whId) }
    }
    return { data: DEMO_LOCATIONS }
  }
  if (normalizedPath === "/wms/transfers") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return {
        data: DEMO_TRANSFERS.filter(
          (t) => t.from_warehouse_id === whId || t.to_warehouse_id === whId
        ),
      }
    }
    return { data: DEMO_TRANSFERS }
  }
  if (normalizedPath.startsWith("/wms/transfers/")) {
    const parts = normalizedPath.split("/")
    const transferId = parts[3]
    const action = parts[4]
    const idx = DEMO_TRANSFERS.findIndex((t) => t.id === transferId)
    if (action === "dispatch") {
      if (idx !== -1) {
        DEMO_TRANSFERS[idx] = {
          ...DEMO_TRANSFERS[idx],
          status: "IN_TRANSIT",
          dispatched_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        return DEMO_TRANSFERS[idx]
      }
      const updated = {
        ...DEMO_TRANSFERS[0],
        id: transferId || DEMO_TRANSFERS[0].id,
        status: "IN_TRANSIT",
        dispatched_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      DEMO_TRANSFERS.unshift(updated)
      return updated
    }
    if (action === "receive") {
      if (idx !== -1) {
        DEMO_TRANSFERS[idx] = {
          ...DEMO_TRANSFERS[idx],
          status: "RECEIVED",
          received_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        return DEMO_TRANSFERS[idx]
      }
      const updated = {
        ...DEMO_TRANSFERS[0],
        id: transferId || DEMO_TRANSFERS[0].id,
        status: "RECEIVED",
        received_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      DEMO_TRANSFERS.unshift(updated)
      return updated
    }
    if (idx !== -1) return DEMO_TRANSFERS[idx]
    return DEMO_TRANSFERS[0]
  }
  if (normalizedPath === "/wms/delivery-orders") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return { data: DEMO_DELIVERY_ORDERS.filter((d) => d.warehouse_id === whId) }
    }
    return { data: DEMO_DELIVERY_ORDERS }
  }
  if (normalizedPath.startsWith("/wms/delivery-orders/")) {
    const parts = normalizedPath.split("/")
    const doId = parts[3]
    if (parts[4] === "dispatch") {
      const idx = DEMO_DELIVERY_ORDERS.findIndex((d) => d.id === doId)
      if (idx !== -1) {
        DEMO_DELIVERY_ORDERS[idx] = {
          ...DEMO_DELIVERY_ORDERS[idx],
          status: "SHIPPED",
          updated_at: new Date().toISOString(),
        }
        return DEMO_DELIVERY_ORDERS[idx]
      }
      const updated = {
        ...DEMO_DELIVERY_ORDERS[0],
        id: doId || DEMO_DELIVERY_ORDERS[0].id,
        status: "SHIPPED",
        updated_at: new Date().toISOString(),
      }
      DEMO_DELIVERY_ORDERS.unshift(updated)
      return updated
    }
    const foundDo = DEMO_DELIVERY_ORDERS.find((d) => d.id === doId) || DEMO_DELIVERY_ORDERS[0]
    const items = DEMO_DELIVERY_ORDER_ITEMS.filter((i) => i.delivery_order_id === doId)
    return {
      delivery_order: foundDo,
      items: items.length > 0 ? items : DEMO_DELIVERY_ORDER_ITEMS.slice(0, 2),
    }
  }
  if (normalizedPath === "/wms/opnames") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return { data: DEMO_OPNAMES.filter((o) => o.warehouse_id === whId) }
    }
    return { data: DEMO_OPNAMES }
  }
  if (normalizedPath.startsWith("/wms/opnames/")) {
    const parts = normalizedPath.split("/")
    const opnameId = parts[3]
    const foundOpname = DEMO_OPNAMES.find((o) => o.id === opnameId) || {
      id: opnameId,
      tenant_id: TENANT,
      warehouse_id: "wh-001",
      opname_number: "OPN-SAMPLE",
      status: "IN_PROGRESS",
      conducted_by: "Operator Gudang",
      approved_by: null,
      notes: "Audit demo",
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    const items = DEMO_OPNAME_ITEMS.filter((i) => i.opname_id === opnameId)
    return {
      opname: foundOpname,
      items: items.length > 0 ? items : DEMO_OPNAME_ITEMS.slice(0, 2),
    }
  }
  if (normalizedPath === "/wms/scraps") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return { data: DEMO_SCRAPS.filter((s) => s.warehouse_id === whId) }
    }
    return { data: DEMO_SCRAPS }
  }
  if (normalizedPath === "/wms/barcodes/resolve") {
    const code = params.get("code") || "SKU-SAMPLE"
    return {
      product_id: "prod-demo-01",
      sku: code.startsWith("SKU-") ? code : `SKU-${code.slice(-6).toUpperCase()}`,
      name: `Produk Master [${code}]`,
      barcode: code,
      multiplier: 1,
      source: "BARCODE",
    }
  }
  if (normalizedPath === "/wms/marketplace/batches") {
    const whId = params.get("warehouse_id")
    if (whId) {
      return { data: DEMO_MARKETPLACE_BATCHES.filter((b) => b.warehouse_id === whId) }
    }
    return { data: DEMO_MARKETPLACE_BATCHES }
  }
  if (normalizedPath === "/wms/marketplace/orders") {
    let orders = [...DEMO_MARKETPLACE_ORDERS]
    const whId = params.get("warehouse_id")
    if (whId) orders = orders.filter((o) => o.warehouse_id === whId)
    const batchId = params.get("batch_id")
    if (batchId) orders = orders.filter((o) => o.batch_id === batchId)
    const status = params.get("status")
    if (status) orders = orders.filter((o) => o.status === status)
    const channel = params.get("channel")
    if (channel) orders = orders.filter((o) => o.channel === channel)
    return { data: orders }
  }
  if (normalizedPath.startsWith("/wms/marketplace/orders/")) {
    const orderId = normalizedPath.split("/")[4]
    const foundOrder = DEMO_MARKETPLACE_ORDERS.find((o) => o.id === orderId) || DEMO_MARKETPLACE_ORDERS[0]
    return { data: foundOrder }
  }
  if (normalizedPath === "/wms/marketplace/sku-mappings") {
    const channel = params.get("channel_name") || params.get("channel")
    if (channel && channel !== "ALL") {
      return { data: DEMO_MARKETPLACE_SKU_MAPPINGS.filter((m) => m.channel_name === channel) }
    }
    return { data: DEMO_MARKETPLACE_SKU_MAPPINGS }
  }
  if (normalizedPath === "/wms/marketplace/import") {
    interface RawOrderItem { external_sku?: string; item_name?: string; quantity?: number; unit_price?: number; subtotal?: number }
    interface RawOrder { external_order_id?: string; order_date?: string; customer_name?: string; customer_phone?: string; shipping_address?: string; courier?: string; tracking_number?: string; total_amount?: number; shipping_fee?: number; marketplace_fee?: number; items?: RawOrderItem[] }
    const rawOrders = Array.isArray(bodyRecord?.orders) ? (bodyRecord.orders as RawOrder[]) : null
    let ordersToReturn: typeof DEMO_MARKETPLACE_ORDERS
    let totalCount = 0

    const batchId = `batch-${Date.now()}`
    const warehouseId = (bodyRecord?.warehouse_id as string) || "wh-001"
    const channel = (bodyRecord?.channel as string) || "SHOPEE"
    const fileName = (bodyRecord?.file_name as string) || "import_orders.csv"

    if (rawOrders && rawOrders.length > 0) {
      totalCount = rawOrders.length
      ordersToReturn = rawOrders.map((o: RawOrder, idx: number) => ({
        id: `ord-imp-${Date.now()}-${idx + 1}`,
        tenant_id: TENANT,
        batch_id: batchId,
        warehouse_id: warehouseId,
        warehouse_name: "Gudang Distribusi Cakung",
        channel,
        external_order_id: o.external_order_id || `IMP-${Date.now()}-${idx + 1}`,
        order_date: o.order_date || new Date().toISOString(),
        customer_name: o.customer_name || "Pelanggan Marketplace",
        customer_phone: o.customer_phone || "-",
        shipping_address: o.shipping_address || "-",
        courier: o.courier || "J&T Express",
        tracking_number: o.tracking_number || `TRK${Date.now()}${idx}`,
        total_amount: Number(o.total_amount) || 150000,
        shipping_fee: Number(o.shipping_fee) || 0,
        marketplace_fee: Number(o.marketplace_fee) || 0,
        net_amount: (Number(o.total_amount) || 150000) - (Number(o.marketplace_fee) || 0),
        status: "COMPLETED",
        sales_order_id: `so-imp-${Date.now()}-${idx + 1}`,
        created_at: new Date().toISOString(),
        items: Array.isArray(o.items) && o.items.length > 0
          ? o.items.map((it: RawOrderItem, iIdx: number) => ({
              id: `item-imp-${Date.now()}-${idx + 1}-${iIdx + 1}`,
              tenant_id: TENANT,
              order_id: `ord-imp-${Date.now()}-${idx + 1}`,
              external_sku: it.external_sku || "SKU-IMP",
              product_id: "prod-001",
              item_name: it.item_name || "Item Marketplace",
              quantity: Number(it.quantity) || 1,
              unit_price: Number(it.unit_price) || 0,
              subtotal: Number(it.subtotal) || 0,
              is_mapped: true,
              product_name: "Produk Master",
              product_sku: "MAS-001",
              multiplier: 1,
            }))
          : [
              {
                id: `item-imp-${Date.now()}-${idx + 1}-1`,
                tenant_id: TENANT,
                order_id: `ord-imp-${Date.now()}-${idx + 1}`,
                external_sku: "SKU-IMP",
                product_id: "prod-001",
                item_name: "Item Marketplace",
                quantity: 1,
                unit_price: 150000,
                subtotal: 150000,
                is_mapped: true,
                product_name: "Produk Master",
                product_sku: "MAS-001",
                multiplier: 1,
              },
            ],
      }))
    } else {
      ordersToReturn = DEMO_MARKETPLACE_ORDERS.slice(0, 3).map((o) => ({
        ...o,
        id: `ord-imp-${Date.now()}-${o.id}`,
        batch_id: batchId,
      }))
      totalCount = ordersToReturn.length
    }

    const newBatch = {
      ...DEMO_MARKETPLACE_BATCHES[0],
      id: batchId,
      batch_number: `BATCH-MKT-${Date.now()}`,
      channel,
      warehouse_id: warehouseId,
      file_name: fileName,
      total_orders: totalCount,
      processed_orders: totalCount,
      failed_orders: 0,
      unmapped_skus: 0,
      status: "COMPLETED",
      created_at: new Date().toISOString(),
    }

    DEMO_MARKETPLACE_BATCHES.unshift(newBatch)
    DEMO_MARKETPLACE_ORDERS.unshift(...ordersToReturn)

    return {
      data: {
        batch: newBatch,
        orders: ordersToReturn,
      },
    }
  }

  // Paginated list endpoints
  const PAGINATED: Record<string, unknown[]> = {
    "/invoices": DEMO_INVOICES,
    "/vendors": DEMO_VENDORS,
    "/products": DEMO_PRODUCTS,
    "/accounts": DEMO_ACCOUNTS,
    "/journal-entries": DEMO_JOURNAL_ENTRIES,
    "/approvals": DEMO_APPROVALS,
    "/purchase-orders": DEMO_PURCHASE_ORDERS,
    "/goods-receipts": DEMO_GOODS_RECEIPTS,
    "/payment-orders": DEMO_PAYMENT_ORDERS,
  }

  if (normalizedPath in PAGINATED) return paginate(PAGINATED[normalizedPath], params)

  return null
}
