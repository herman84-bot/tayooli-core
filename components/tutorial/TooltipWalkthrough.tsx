'use client'

import { useEffect, useState } from 'react'
import { X } from 'lucide-react'

interface TooltipStep {
  target: string
  title: string
  description: string
}

const STEPS: TooltipStep[] = [
  {
    target: '[data-tutorial="dashboard-header"]',
    title: 'Dashboard Operasional',
    description: 'Ringkasan performa omzet kasir POS harian dan saldo fisik stok gudang.',
  },
  {
    target: '[data-tutorial="pos-button"]',
    title: 'Kasir POS',
    description: 'Buka mesin kasir untuk transaksi langsung, scan barcode, dan cetak struk belanja.',
  },
  {
    target: '[data-tutorial="sidebar"]',
    title: 'Menu Navigasi',
    description: 'Kelola master produk, stok multi-gudang, surat jalan, transfer, hingga opname.',
  },
  {
    target: '[data-tutorial="help-support"]',
    title: 'Pusat Bantuan',
    description: 'Butuh panduan operasional? Tanya asisten AI atau kirim tiket kendala teknis.',
  },
]

const STORAGE_KEY = 'tayooli_onboarding_completed'

export function TooltipWalkthrough() {
  const [isVisible, setIsVisible] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [spotlight, setSpotlight] = useState<{ x: number; y: number; w: number; h: number } | null>(null)

  useEffect(() => {
    const completed = localStorage.getItem(STORAGE_KEY)
    if (completed) return

    const timer = setTimeout(() => {
      setIsVisible(true)
    }, 1000)

    return () => clearTimeout(timer)
  }, [])

  // Update spotlight position when step changes
  useEffect(() => {
    if (!isVisible) return

    const updateSpotlight = () => {
      const step = STEPS[currentStep]
      if (!step) return

      const el = document.querySelector(step.target)
      if (el) {
        // Scroll element into view within main container
        const main = document.querySelector('main')
        if (main) {
          const rect = el.getBoundingClientRect()
          const mainRect = main.getBoundingClientRect()
          const relativeTop = rect.top - mainRect.top + main.scrollTop
          const targetScroll = relativeTop - (main.clientHeight / 2) + (el.clientHeight / 2)
          main.scrollTo({ top: Math.max(0, targetScroll), behavior: 'smooth' })
        }

        // Capture rect after a brief delay for scroll
        setTimeout(() => {
          const r = el.getBoundingClientRect()
          setSpotlight({ x: r.left - 4, y: r.top - 4, w: r.width + 8, h: r.height + 8 })
        }, 300)
      } else {
        setSpotlight(null)
      }
    }

    updateSpotlight()
  }, [isVisible, currentStep])

  const handleNext = () => {
    if (currentStep < STEPS.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      handleComplete()
    }
  }

  const handleComplete = () => {
    localStorage.setItem(STORAGE_KEY, 'true')
    setIsVisible(false)
  }

  const handleSkip = () => {
    localStorage.setItem(STORAGE_KEY, 'true')
    setIsVisible(false)
  }

  if (!isVisible) return null

  const step = STEPS[currentStep]
  if (!step) return null

  return (
    <>
      {/* Overlay with spotlight cutout */}
      <div className="fixed inset-0 z-50" aria-label="Tutorial overlay">
        {/* Dark background — use SVG for spotlight cutout */}
        <svg className="absolute inset-0 w-full h-full" aria-hidden="true">
          <defs>
            <mask id="spotlight-mask">
              <rect width="100%" height="100%" fill="white" />
              {spotlight && (
                <rect
                  x={spotlight.x}
                  y={spotlight.y}
                  width={spotlight.w}
                  height={spotlight.h}
                  rx={6}
                  fill="black"
                />
              )}
            </mask>
          </defs>
          <rect
            width="100%"
            height="100%"
            fill="rgba(0,0,0,0.35)"
            mask="url(#spotlight-mask)"
          />
        </svg>

        {/* Click anywhere to skip */}
        <div className="absolute inset-0" onClick={handleSkip} />
      </div>

      {/* Tooltip — always centered in viewport, above overlay */}
      <div
        className="fixed z-[60] w-72 bg-foreground text-background rounded-lg shadow-lg p-4"
        style={{
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
        }}
        role="dialog"
        aria-label={`Tutorial step ${currentStep + 1}`}
      >
        {/* Close button */}
        <button
          onClick={handleSkip}
          className="absolute top-2 right-2 p-1 rounded-md hover:bg-background/20 transition-colors"
          aria-label="Skip tutorial"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Content */}
        <div className="mb-3">
          <h3 className="text-sm font-semibold mb-1">{step.title}</h3>
          <p className="text-xs opacity-90 leading-relaxed">{step.description}</p>
        </div>

        {/* Progress dots + actions */}
        <div className="flex items-center justify-between">
          <div className="flex gap-1.5">
            {STEPS.map((_, i) => (
              <div
                key={i}
                className={`h-1.5 w-1.5 rounded-full transition-colors ${
                  i === currentStep ? 'bg-primary' : 'bg-background/30'
                }`}
              />
            ))}
          </div>

          <div className="flex gap-2">
            <button
              onClick={handleSkip}
              className="text-xs text-background/60 hover:text-background transition-colors"
            >
              Skip
            </button>
            <button
              onClick={handleNext}
              className="text-xs font-medium bg-primary text-primary-foreground px-3 py-1 rounded-md hover:bg-primary/90 transition-colors"
            >
              {currentStep < STEPS.length - 1 ? 'Lanjut' : 'Selesai'}
            </button>
          </div>
        </div>
      </div>
    </>
  )
}
