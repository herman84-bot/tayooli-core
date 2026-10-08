"use client"

import React, { useRef, useState, useEffect, useCallback } from "react"
import { RotateCcw, Check, PenTool, AlertCircle } from "lucide-react"

export interface SignatureCanvasProps {
  onSave: (signatureSvgOrDataUrl: string) => void
  disabled?: boolean
  height?: number
  className?: string
}

interface Point {
  x: number
  y: number
}

export function SignatureCanvas({
  onSave,
  disabled = false,
  height = 180,
  className = "",
}: SignatureCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [isDrawing, setIsDrawing] = useState(false)
  const [strokes, setStrokes] = useState<Point[][]>([])
  const [currentStroke, setCurrentStroke] = useState<Point[]>([])
  const [hasContent, setHasContent] = useState(false)
  const [canvasWidth, setCanvasWidth] = useState(500)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  // Measure container width
  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect()
        if (rect.width > 0) {
          setCanvasWidth(Math.floor(rect.width))
        }
      }
    }
    updateWidth()
    window.addEventListener("resize", updateWidth)
    return () => window.removeEventListener("resize", updateWidth)
  }, [])

  // Redraw canvas from strokes
  const redrawCanvas = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext("2d")
    if (!ctx) return

    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.lineWidth = 2.5
    ctx.lineCap = "round"
    ctx.lineJoin = "round"
    ctx.strokeStyle = "#1E293B" // Slate 800

    const allStrokes = currentStroke.length > 0 ? [...strokes, currentStroke] : strokes

    for (const stroke of allStrokes) {
      if (stroke.length < 1) continue
      ctx.beginPath()
      ctx.moveTo(stroke[0].x, stroke[0].y)
      for (let i = 1; i < stroke.length; i++) {
        ctx.lineTo(stroke[i].x, stroke[i].y)
      }
      ctx.stroke()
    }
  }, [strokes, currentStroke])

  useEffect(() => {
    redrawCanvas()
  }, [redrawCanvas, canvasWidth, height])

  // Coordinate helper
  const getCanvasPoint = (
    e: React.MouseEvent<HTMLCanvasElement> | React.TouchEvent<HTMLCanvasElement>
  ): Point | null => {
    const canvas = canvasRef.current
    if (!canvas) return null
    const rect = canvas.getBoundingClientRect()

    let clientX = 0
    let clientY = 0

    if ("touches" in e) {
      if (e.touches.length === 0) return null
      clientX = e.touches[0].clientX
      clientY = e.touches[0].clientY
    } else {
      clientX = e.clientX
      clientY = e.clientY
    }

    const scaleX = canvas.width / rect.width
    const scaleY = canvas.height / rect.height

    return {
      x: (clientX - rect.left) * scaleX,
      y: (clientY - rect.top) * scaleY,
    }
  }

  // Mouse handlers
  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (disabled) return
    e.preventDefault()
    const pt = getCanvasPoint(e)
    if (!pt) return
    setIsDrawing(true)
    setCurrentStroke([pt])
    setErrorMsg(null)
  }

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!isDrawing || disabled) return
    e.preventDefault()
    const pt = getCanvasPoint(e)
    if (!pt) return
    setCurrentStroke((prev) => [...prev, pt])
    setHasContent(true)
  }

  const handleMouseUp = () => {
    if (!isDrawing || disabled) return
    setIsDrawing(false)
    if (currentStroke.length > 0) {
      setStrokes((prev) => [...prev, currentStroke])
      setCurrentStroke([])
      setHasContent(true)
    }
  }

  // Touch handlers
  const handleTouchStart = (e: React.TouchEvent<HTMLCanvasElement>) => {
    if (disabled) return
    e.preventDefault()
    const pt = getCanvasPoint(e)
    if (!pt) return
    setIsDrawing(true)
    setCurrentStroke([pt])
    setErrorMsg(null)
  }

  const handleTouchMove = (e: React.TouchEvent<HTMLCanvasElement>) => {
    if (!isDrawing || disabled) return
    e.preventDefault()
    const pt = getCanvasPoint(e)
    if (!pt) return
    setCurrentStroke((prev) => [...prev, pt])
    setHasContent(true)
  }

  const handleTouchEnd = (e: React.TouchEvent<HTMLCanvasElement>) => {
    if (!isDrawing || disabled) return
    e.preventDefault()
    setIsDrawing(false)
    if (currentStroke.length > 0) {
      setStrokes((prev) => [...prev, currentStroke])
      setCurrentStroke([])
      setHasContent(true)
    }
  }

  const handleClear = () => {
    if (disabled) return
    setStrokes([])
    setCurrentStroke([])
    setHasContent(false)
    setErrorMsg(null)
    const canvas = canvasRef.current
    if (canvas) {
      const ctx = canvas.getContext("2d")
      if (ctx) ctx.clearRect(0, 0, canvas.width, canvas.height)
    }
  }

  // Export strokes to SVG string
  const exportToSvg = (): string => {
    const allStrokes = [...strokes]
    if (allStrokes.length === 0) return ""

    let pathD = ""
    for (const stroke of allStrokes) {
      if (stroke.length === 0) continue
      pathD += `M ${stroke[0].x.toFixed(1)} ${stroke[0].y.toFixed(1)} `
      for (let i = 1; i < stroke.length; i++) {
        pathD += `L ${stroke[i].x.toFixed(1)} ${stroke[i].y.toFixed(1)} `
      }
    }

    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${canvasWidth} ${height}" width="${canvasWidth}" height="${height}"><path d="${pathD.trim()}" fill="none" stroke="#1e293b" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`
  }

  const handleSave = () => {
    if (disabled) return
    if (!hasContent || strokes.length === 0) {
      setErrorMsg("Tanda tangan belum dibuat. Silakan goreskan tanda tangan pada kanvas.")
      return
    }

    const svgString = exportToSvg()
    if (!svgString || svgString.trim().length === 0) {
      // Fallback to dataURL
      const canvas = canvasRef.current
      if (canvas) {
        onSave(canvas.toDataURL("image/png"))
      } else {
        setErrorMsg("Gagal menyimpan tanda tangan.")
      }
      return
    }

    onSave(svgString)
  }

  return (
    <div ref={containerRef} className={`w-full space-y-2 ${className}`}>
      <div className="relative border-2 border-dashed border-slate-300 rounded-xl bg-slate-50 overflow-hidden focus-within:border-blue-500 transition-colors">
        <canvas
          ref={canvasRef}
          width={canvasWidth}
          height={height}
          style={{ width: "100%", height: `${height}px`, touchAction: "none" }}
          className={`block bg-transparent cursor-crosshair ${
            disabled ? "opacity-50 cursor-not-allowed" : ""
          }`}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
          onTouchCancel={handleTouchEnd}
          aria-label="Kanvas Tanda Tangan Digital"
        />

        {!hasContent && (
          <div className="absolute inset-0 pointer-events-none flex flex-col items-center justify-center text-slate-400">
            <PenTool className="w-6 h-6 mb-1.5 opacity-60" />
            <span className="text-xs font-medium">Tanda tangani di sini dengan mouse atau jari</span>
          </div>
        )}

        <div className="absolute bottom-1 right-2 pointer-events-none text-[10px] text-slate-400 tracking-wider uppercase font-mono">
          Area TTD
        </div>
      </div>

      {errorMsg && (
        <div className="flex items-center gap-1.5 text-xs text-rose-600 bg-rose-50 px-3 py-1.5 rounded-lg border border-rose-200">
          <AlertCircle className="w-3.5 h-3.5 shrink-0" />
          <span>{errorMsg}</span>
        </div>
      )}

      <div className="flex items-center justify-between pt-1">
        <button
          type="button"
          onClick={handleClear}
          disabled={disabled || !hasContent}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-slate-200 text-slate-600 hover:bg-slate-100 disabled:opacity-40 disabled:cursor-not-allowed transition"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          Hapus / Ulangi
        </button>

        <button
          type="button"
          onClick={handleSave}
          disabled={disabled || !hasContent}
          className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed shadow-xs transition"
        >
          <Check className="w-3.5 h-3.5" />
          Terapkan Tanda Tangan
        </button>
      </div>
    </div>
  )
}
