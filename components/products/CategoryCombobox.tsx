"use client"

import React, { useEffect, useMemo, useRef, useState } from "react"
import { useCreateCategory, useProductCategories } from "@/hooks/useProductCategories"
import type { ProductCategory } from "@/lib/api"
import { cn } from "@/lib/utils"

export interface CategoryComboboxProps {
  value: string | null | undefined
  onChange: (categoryId: string | null, categoryName?: string) => void
  placeholder?: string
  disabled?: boolean
  allowClear?: boolean
  className?: string
  ariaLabel?: string
}

export function CategoryCombobox({
  value,
  onChange,
  placeholder = "Pilih kategori...",
  disabled = false,
  allowClear = true,
  className,
  ariaLabel = "Kategori Produk",
}: CategoryComboboxProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [highlightedIndex, setHighlightedIndex] = useState<number>(-1)

  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { data: categories = [], isLoading } = useProductCategories()
  const createMutation = useCreateCategory()

  // Selected category object
  const selectedCategory = useMemo(() => {
    if (!value) return null
    return categories.find((c) => c.id === value) ?? null
  }, [categories, value])

  // Filtered categories by search query
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return categories
    return categories.filter((c) => c.name.toLowerCase().includes(q))
  }, [categories, query])

  // Check if query is an exact match for an existing category
  const hasExactMatch = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return true
    return categories.some((c) => c.name.toLowerCase() === q)
  }, [categories, query])

  // Total selectable items (including clear option if allowed)
  const totalItems = useMemo(() => {
    return (allowClear ? 1 : 0) + filtered.length
  }, [allowClear, filtered.length])

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

  // Focus search input and reset highlight when opened
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 30)
      setCreateError(null)
      setHighlightedIndex(-1)
    } else {
      setQuery("")
      setCreateError(null)
      setHighlightedIndex(-1)
    }
  }, [open])

  const handleSelect = (cat: ProductCategory | null) => {
    if (!cat) {
      onChange(null, "")
    } else {
      onChange(cat.id, cat.name)
    }
    setOpen(false)
  }

  const handleCreateNew = async () => {
    const name = query.trim()
    if (!name || createMutation.isPending) return
    setCreateError(null)
    try {
      const res = await createMutation.mutateAsync(name)
      onChange(res.id, res.name)
      setOpen(false)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Gagal menambahkan kategori")
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      setOpen(false)
    } else if (e.key === "ArrowDown") {
      e.preventDefault()
      setHighlightedIndex((prev) => (prev < totalItems - 1 ? prev + 1 : 0))
    } else if (e.key === "ArrowUp") {
      e.preventDefault()
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : totalItems - 1))
    } else if (e.key === "Enter") {
      e.preventDefault()
      if (highlightedIndex >= 0) {
        if (allowClear && highlightedIndex === 0) {
          handleSelect(null)
        } else {
          const itemIdx = allowClear ? highlightedIndex - 1 : highlightedIndex
          if (filtered[itemIdx]) {
            handleSelect(filtered[itemIdx])
          }
        }
      } else if (!hasExactMatch && query.trim()) {
        handleCreateNew()
      }
    }
  }

  return (
    <div ref={containerRef} className={cn("relative text-left", className)}>
      <div className="relative w-full">
        <button
          type="button"
          disabled={disabled}
          onClick={() => !disabled && setOpen((o) => !o)}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label={
            selectedCategory
              ? `${ariaLabel}: ${selectedCategory.name}`
              : (ariaLabel ?? placeholder)
          }
          className={cn(
            "w-full flex items-center justify-between gap-2 px-3 py-2 pr-12 text-sm rounded-lg border bg-white text-zinc-900 transition-colors text-left",
            "border-zinc-300 hover:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-600",
            disabled && "opacity-50 cursor-not-allowed bg-zinc-50 text-zinc-500",
            open && "border-indigo-600 ring-2 ring-indigo-500/20"
          )}
        >
          <span className={cn("truncate", !selectedCategory && "text-zinc-400")}>
            {selectedCategory ? selectedCategory.name : placeholder}
          </span>
        </button>

        {/* Action icons outside the button to avoid nested interactive elements */}
        <div className="absolute right-2.5 top-1/2 -translate-y-1/2 flex items-center gap-1">
          {allowClear && selectedCategory && !disabled && (
            <button
              type="button"
              aria-label="Hapus pilihan kategori"
              onClick={(e) => {
                e.stopPropagation()
                handleSelect(null)
              }}
              className="text-zinc-400 hover:text-zinc-600 p-0.5 rounded cursor-pointer transition-colors"
            >
              ×
            </button>
          )}
          <span className="text-zinc-400 text-xs pointer-events-none">▼</span>
        </div>
      </div>

      {open && (
        <div
          role="listbox"
          aria-label={ariaLabel}
          className="absolute z-50 mt-1 w-full rounded-lg border border-zinc-200 bg-white shadow-lg overflow-hidden text-sm"
        >
          {/* Search box inside dropdown */}
          <div className="p-2 border-b border-zinc-100 bg-zinc-50">
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value)
                setHighlightedIndex(-1)
              }}
              onKeyDown={handleKeyDown}
              placeholder="Cari atau buat kategori..."
              aria-label="Cari kategori"
              className="w-full px-2.5 py-1.5 text-xs rounded border border-zinc-300 bg-white focus:outline-none focus:border-indigo-600"
            />
          </div>

          {/* List items */}
          <div className="max-h-60 overflow-y-auto divide-y divide-zinc-50">
            {allowClear && (
              <div
                role="option"
                aria-selected={!value}
                onClick={() => handleSelect(null)}
                className={cn(
                  "px-3 py-2 cursor-pointer text-xs text-zinc-500 hover:bg-zinc-50 flex items-center justify-between",
                  !value && "font-semibold text-indigo-600 bg-indigo-50/50",
                  highlightedIndex === 0 && "bg-zinc-100"
                )}
              >
                <span>-- Tanpa Kategori --</span>
              </div>
            )}

            {isLoading ? (
              <div className="p-3 text-center text-xs text-zinc-400">Memuat kategori...</div>
            ) : filtered.length === 0 && hasExactMatch ? (
              <div className="p-3 text-center text-xs text-zinc-400">Kategori tidak ditemukan</div>
            ) : (
              filtered.map((cat, idx) => {
                const itemGlobalIdx = allowClear ? idx + 1 : idx
                const isHighlighted = highlightedIndex === itemGlobalIdx
                return (
                  <div
                    key={cat.id}
                    role="option"
                    aria-selected={cat.id === value}
                    onClick={() => handleSelect(cat)}
                    className={cn(
                      "px-3 py-2 cursor-pointer flex items-center justify-between gap-2 transition-colors",
                      cat.id === value
                        ? "bg-indigo-50 text-indigo-900 font-semibold"
                        : isHighlighted
                        ? "bg-zinc-100 text-zinc-900"
                        : "hover:bg-zinc-50 text-zinc-800"
                    )}
                  >
                    <span className="truncate">{cat.name}</span>
                    {typeof cat.product_count === "number" && (
                      <span className="text-[11px] text-zinc-400 flex-shrink-0">
                        {cat.product_count} produk
                      </span>
                    )}
                  </div>
                )
              })
            )}

            {/* Inline creation button when query doesn't match an existing category */}
            {!hasExactMatch && query.trim().length > 0 && (
              <div className="p-2 bg-indigo-50/60 border-t border-indigo-100">
                <button
                  type="button"
                  disabled={createMutation.isPending}
                  onClick={handleCreateNew}
                  className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-indigo-700 bg-white hover:bg-indigo-50 border border-indigo-200 rounded-md shadow-sm transition-colors disabled:opacity-50"
                >
                  {createMutation.isPending ? (
                    "Menambahkan..."
                  ) : (
                    <>
                      <span>+ Buat Kategori:</span>
                      <strong className="truncate max-w-[140px]">"{query.trim()}"</strong>
                    </>
                  )}
                </button>
                {createError && (
                  <p className="mt-1 text-[11px] text-rose-600 text-center">{createError}</p>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
