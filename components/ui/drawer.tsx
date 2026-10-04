"use client"

import { X } from "lucide-react"
import { createContext, useContext, useEffect } from "react"

interface DrawerContextType {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const DrawerContext = createContext<DrawerContextType | null>(null)

export function DrawerProvider({ children, open, onOpenChange }: {
  children: React.ReactNode
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <DrawerContext.Provider value={{ open, onOpenChange }}>
      {children}
    </DrawerContext.Provider>
  )
}

function useDrawer() {
  const ctx = useContext(DrawerContext)
  if (!ctx) throw new Error("Drawer components must be used within DrawerProvider")
  return ctx
}

export function Drawer({ children }: { children: React.ReactNode }) {
  const { open, onOpenChange } = useDrawer()

  useEffect(() => {
    if (!open) return
    const handleEscape = (e: KeyboardEvent) => { if (e.key === "Escape") onOpenChange(false) }
    document.addEventListener("keydown", handleEscape)
    document.body.style.overflow = "hidden"
    return () => { document.removeEventListener("keydown", handleEscape); document.body.style.overflow = "" }
  }, [open, onOpenChange])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="fixed inset-0 bg-black/40 backdrop-blur-sm" onClick={() => onOpenChange(false)} />
      <div className="relative z-50 w-full max-w-2xl bg-white shadow-2xl flex flex-col border-l border-gray-200">
        {children}
      </div>
    </div>
  )
}

export function DrawerHeader({ title, icon }: { title: string; icon?: React.ReactNode }) {
  const { onOpenChange } = useDrawer()
  return (
    <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 bg-gray-50">
      <div className="flex items-center gap-3">
        {icon && <span className="text-gray-500">{icon}</span>}
        <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
      </div>
      <button onClick={() => onOpenChange(false)} className="p-2 rounded-md hover:bg-gray-200 transition-colors">
        <X className="w-5 h-5 text-gray-500" />
      </button>
    </div>
  )
}

export function DrawerContent({ children }: { children: React.ReactNode }) {
  return <div className="flex-1 overflow-y-auto p-6 bg-white">{children}</div>
}

export function DrawerFooter({ children }: { children: React.ReactNode }) {
  return <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3">{children}</div>
}
