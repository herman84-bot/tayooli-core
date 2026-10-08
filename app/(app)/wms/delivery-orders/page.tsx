"use client"

// Legacy route; next.config.ts redirects it to /wms/arus-barang?mode=keluar.
import DeliveryOrdersPanel from "@/components/wms/DeliveryOrdersPanel"

export default function DeliveryOrdersPage() {
  return <DeliveryOrdersPanel />
}
