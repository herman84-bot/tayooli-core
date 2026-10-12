"use client"

import React, { useEffect, useRef, useState } from "react"
import { api, type Product } from "@/lib/api"
import { cn } from "@/lib/utils"

export interface ProductComboboxProps {
  value: Product | null
  onSelect: (product: Product | null) => void
  placeholder?: string
  disabled?: boolean
  allowClear?: boolean
  className?: string
  /** Filter out specific products (e.g. already selected in other lines). */
  excludeIds?: string[]
  /** Accessible name for the trigger; defaults to the placeholder. */
  ariaLabel?: string
}

const DEBOUNCE_MS = 250

/**
 * productRef builds the minimal Product the combobox needs to display a
 * selection known only by id/name/sku (e.g. a saved line or a staging row),
 * without loading the full catalog. Returns null when id is empty.
 */
export function productRef(id: string, name?: string, sku?: string): Product | null {
  if (!id) return null
  return {
    id,
    name: name || "Produk",
    sku: sku || "",
    tenant_id: "",
    price: 0,
    created_at: "",
    updated_at: "",
  }
}

/**
 * ProductCombobox replaces static <select> with a searchable, paginated
 * product picker. Fetches from GET /products?q= so large catalogs (>1000 items)
 * remain fully discoverable without loading the entire DB into the browser.
 */
export function ProductCombobox({
  value,
  onSelect,
  placeholder = "Pilih produk...",
  disabled = false,
  allowClear = false,
  className,
  excludeIds,
  ariaLabel,
}: ProductComboboxProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<Product[]>([])
  const [loading, setLoading] = useState(false)
  const [highlightIdx, setHighlightIdx] = useState(0)

  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const activeReq = useRef(0)

  // Fetch products as query changes (debounced)
  useEffect(() => {
    if (!open) return
    const reqId = ++activeReq.current
    setLoading(true)

    const handle = setTimeout(async () => {
      try {
        const res = await api.products.list({
          q: query.trim() || undefined,
          per_page: 20,
          page: 1,
        })
        if (activeReq.current !== reqId) return
        const list = Array.isArray(res) ? res : Array.isArray(res?.data) ? res.data : []
        const filtered = excludeIds?.length ? list.filter((p) => !excludeIds.includes(p.id)) : list
        setResults(filtered)
        setHighlightIdx(0)
      } catch {
        if (activeReq.current === reqId) setResults([])
      } finally {
        if (activeReq.current === reqId) setLoading(false)
      }
    }, query ? DEBOUNCE_MS : 0)

    return () => clearTimeout(handle)
  }, [open, query, excludeIds])

  // Close on outside click
  useEffect(() => {
    if (!open) return
    const onDocClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener("mousedown", onDocClick)
    return () => document.removeEventListener("mousedown", onDocClick)
  }, [open])

  // Focus input when opened
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 30)
    } else {
      setQuery("")
    }
  }, [open])

  const handleSelect = (p: Product) => {
    onSelect(p)
    setOpen(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === "Enter" || e.key === "ArrowDown" || e.key === " ") {
        e.preventDefault()
        setOpen(true)
      }
      return
    }
    if (e.key === "Escape") {
      e.preventDefault()
      setOpen(false)
    } else if (e.key === "ArrowDown") {
      e.preventDefault()
      setHighlightIdx((i) => (results.length ? (i + 1) % results.length : 0))
    } else if (e.key === "ArrowUp") {
      e.preventDefault()
      setHighlightIdx((i) => (results.length ? (i - 1 + results.length) % results.length : 0))
    } else if (e.key === "Enter" && results[highlightIdx]) {
      e.preventDefault()
      handleSelect(results[highlightIdx])
    }
  }

  return (
    <div ref={containerRef} className={cn("relative text-left", className)} onKeyDown={handleKeyDown}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => !disabled && setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={value ? `${ariaLabel ?? placeholder}: ${value.name}` : (ariaLabel ?? placeholder)}
        className={cn(
          "w-full flex items-center justify-between gap-2 px-3 py-2 text-sm rounded-lg border bg-white text-zinc-900 transition-colors",
          "border-zinc-300 hover:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-600",
          disabled && "opacity-50 cursor-not-allowed bg-zinc-50 text-zinc-500",
          open && "border-indigo-600 ring-2 ring-indigo-500/20"
        )}
      >
        <span className={cn("truncate", !value && "text-zinc-400")}>
          {value ? (
            <span>
              <span className="font-medium text-zinc-900">{value.name}</span>
              {value.sku && <span className="ml-1.5 text-xs text-zinc-500 font-mono">[{value.sku}]</span>}
            </span>
          ) : (
            placeholder
          )}
        </span>
        <div className="flex items-center gap-1 flex-shrink-0">
          {allowClear && value && !disabled && (
            <span
              role="button"
              tabIndex={0}
              aria-label="Hapus pilihan produk"
              onClick={(e) => {
                e.stopPropagation()
                onSelect(null)
              }}
              className="text-zinc-400 hover:text-zinc-600 p-0.5"
            >
              ×
            </span>
          )}
          <span className="text-zinc-400 text-xs">▼</span>
        </div>
      </button>

      {open && (
        <div
          role="listbox"
          className="absolute z-50 mt-1 w-full rounded-lg border border-zinc-200 bg-white shadow-lg py-1 max-h-72 flex flex-col text-sm"
        >
          <div className="p-1.5 border-b border-zinc-100">
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Ketik nama atau SKU produk..."
              aria-label="Cari produk"
              className="w-full px-2.5 py-1.5 text-xs rounded border border-zinc-200 focus:outline-none focus:border-indigo-600 focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          <div className="overflow-y-auto flex-1 py-1">
            {loading && results.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-zinc-400">Mencari...</div>
            ) : results.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-zinc-400">
                {query ? "Tidak ada produk yang cocok" : "Ketik untuk mencari produk"}
              </div>
            ) : (
              results.map((p, idx) => (
                <div
                  key={p.id}
                  role="option"
                  aria-selected={value?.id === p.id}
                  onMouseEnter={() => setHighlightIdx(idx)}
                  onClick={() => handleSelect(p)}
                  className={cn(
                    "px-3 py-2 cursor-pointer flex items-center justify-between gap-2 transition-colors",
                    idx === highlightIdx ? "bg-indigo-50 text-indigo-900" : "hover:bg-zinc-50 text-zinc-800",
                    value?.id === p.id && "font-semibold"
                  )}
                >
                  <div className="truncate">
                    <div className="truncate">{p.name}</div>
                    <div className="text-xs text-zinc-400 font-mono">{p.sku}</div>
                  </div>
                  <div className="text-xs text-zinc-500 flex-shrink-0">
                    Rp{Number(p.price || 0).toLocaleString("id-ID")}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
