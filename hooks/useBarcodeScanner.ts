"use client"

import { useEffect, useRef, useCallback } from "react"

export interface UseBarcodeScannerOptions {
  /** Callback fired when a barcode is successfully detected */
  onScan: (code: string) => void
  /** Maximum interval between keystrokes to be considered hardware scanner (ms). Default: 30 */
  maxIntervalMs?: number
  /** Minimum barcode character length to prevent single-key false positives. Default: 3 */
  minChars?: number
  /** Whether the listener is active. Default: true */
  enabled?: boolean
  /** Whether to play simulated audio beep on scan. Default: true */
  soundFeedback?: boolean
  /** Whether to trigger haptic vibration on devices supporting navigator.vibrate. Default: true */
  hapticFeedback?: boolean
  /** If true, ignore keystrokes when an input/textarea has focus unless keystroke interval <= 30ms */
  captureInInputs?: boolean
}

/**
 * Play a web-audio synthetic feedback tone
 */
export function playScannerTone(type: "success" | "error" = "success") {
  if (typeof window === "undefined") return
  try {
    const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
    if (!AudioContextClass) return
    const ctx = new AudioContextClass()

    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)

    const now = ctx.currentTime

    if (type === "success") {
      // Pleasant dual-pitch confirmation beep
      osc.type = "sine"
      osc.frequency.setValueAtTime(1200, now)
      osc.frequency.exponentialRampToValueAtTime(1800, now + 0.08)
      gain.gain.setValueAtTime(0.2, now)
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.09)
      osc.start(now)
      osc.stop(now + 0.09)
    } else {
      // Low dual error buzz
      osc.type = "sawtooth"
      osc.frequency.setValueAtTime(280, now)
      gain.gain.setValueAtTime(0.25, now)
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.18)
      osc.start(now)
      osc.stop(now + 0.18)
    }

    setTimeout(() => {
      ctx.close().catch(() => {})
    }, 300)
  } catch {
    // Ignore audio context autoplay policy restrictions
  }
}

/**
 * Trigger mobile haptic vibration
 */
export function triggerHaptic(type: "success" | "error" = "success") {
  if (typeof window === "undefined" || !("vibrate" in navigator)) return
  try {
    if (type === "success") {
      navigator.vibrate?.([40, 30, 40])
    } else {
      navigator.vibrate?.([100, 50, 100])
    }
  } catch {
    // Ignore
  }
}

/**
 * Hook to listen for USB HID hardware barcode scanners (<=30ms keystroke buffer)
 * and provide manual/camera trigger functions with sound & haptics.
 */
export function useBarcodeScanner({
  onScan,
  maxIntervalMs = 30,
  minChars = 3,
  enabled = true,
  soundFeedback = true,
  hapticFeedback = true,
  captureInInputs = true,
}: UseBarcodeScannerOptions) {
  const bufferRef = useRef<string>("")
  const lastKeyTimeRef = useRef<number>(0)
  const isFastStreamRef = useRef<boolean>(true)
  const onScanRef = useRef(onScan)

  useEffect(() => {
    onScanRef.current = onScan
  }, [onScan])

  const notifyScan = useCallback(
    (code: string, isSuccess = true) => {
      const clean = code.trim()
      if (!clean) return

      if (soundFeedback) {
        playScannerTone(isSuccess ? "success" : "error")
      }
      if (hapticFeedback) {
        triggerHaptic(isSuccess ? "success" : "error")
      }
      onScanRef.current(clean)
    },
    [soundFeedback, hapticFeedback]
  )

  useEffect(() => {
    if (!enabled || typeof window === "undefined") return

    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore functional modifiers alone
      if (e.key === "Shift" || e.key === "Control" || e.key === "Alt" || e.key === "Meta") {
        return
      }

      const activeEl = document.activeElement
      const isInput =
        activeEl instanceof HTMLInputElement ||
        activeEl instanceof HTMLTextAreaElement ||
        (activeEl as HTMLElement)?.isContentEditable

      const currentTime = Date.now()
      const interval = currentTime - lastKeyTimeRef.current
      lastKeyTimeRef.current = currentTime

      // If interval exceeds threshold, reset buffer
      if (interval > maxIntervalMs) {
        bufferRef.current = ""
        isFastStreamRef.current = true
      } else {
        // Keystroke was within <= 30ms window (indicative of hardware scanner)
      }

      if (e.key === "Enter") {
        const barcode = bufferRef.current.trim()
        if (barcode.length >= minChars && isFastStreamRef.current) {
          // Hardware scanner termination
          e.preventDefault()
          e.stopPropagation()
          bufferRef.current = ""
          notifyScan(barcode, true)
        } else {
          bufferRef.current = ""
        }
        return
      }

      // Check if standard printable character
      if (e.key.length === 1) {
        // If typing in input, but interval is slow (> 30ms), don't treat as scanner buffer
        if (isInput && !captureInInputs) {
          return
        }

        if (interval > maxIntervalMs && bufferRef.current.length > 0) {
          isFastStreamRef.current = false
        }

        bufferRef.current += e.key
      }
    }

    window.addEventListener("keydown", handleKeyDown, { capture: true })
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true })
    }
  }, [enabled, maxIntervalMs, minChars, captureInInputs, notifyScan])

  return {
    triggerScan: notifyScan,
    playTone: playScannerTone,
    triggerHaptic,
  }
}
