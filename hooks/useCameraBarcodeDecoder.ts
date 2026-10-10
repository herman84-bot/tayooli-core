"use client"

import { useEffect, useRef, useState } from "react"

interface DetectedBarcode {
  rawValue: string
}
interface BarcodeDetectorLike {
  detect(source: CanvasImageSource): Promise<DetectedBarcode[]>
}
interface BarcodeDetectorCtor {
  new (opts?: { formats?: string[] }): BarcodeDetectorLike
  getSupportedFormats?: () => Promise<string[]>
}

const WANTED_FORMATS = [
  "ean_13",
  "ean_8",
  "upc_a",
  "upc_e",
  "code_128",
  "code_39",
  "code_93",
  "itf",
  "codabar",
  "qr_code",
  "data_matrix",
]

export function getBarcodeDetectorCtor(): BarcodeDetectorCtor | null {
  if (typeof window === "undefined") return null
  const ctor = (window as unknown as { BarcodeDetector?: BarcodeDetectorCtor }).BarcodeDetector
  return ctor ?? null
}

/**
 * Decodes barcodes from a live <video> element using the browser-native
 * Shape Detection API (BarcodeDetector). Supported on Chrome/Edge Android,
 * ChromeOS, and macOS Chrome; NOT on iOS Safari or Firefox. When unsupported,
 * `supported` is false so the UI can tell the user to use a USB scanner or
 * manual entry instead of pretending the camera scans.
 *
 * The same code is not re-emitted within `cooldownMs` to avoid duplicate posts
 * while the barcode stays in frame.
 */
export function useCameraBarcodeDecoder(
  videoRef: React.RefObject<HTMLVideoElement | null>,
  active: boolean,
  onDetect: (code: string) => void,
  cooldownMs = 2000
) {
  const [supported, setSupported] = useState<boolean | null>(() =>
    typeof window === "undefined" ? null : getBarcodeDetectorCtor() !== null
  )
  const onDetectRef = useRef(onDetect)
  useEffect(() => {
    onDetectRef.current = onDetect
  }, [onDetect])
  const lastRef = useRef<{ code: string; at: number }>({ code: "", at: 0 })


  useEffect(() => {
    const Ctor = getBarcodeDetectorCtor()
    if (!active || !Ctor) return

    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null
    let detector: BarcodeDetectorLike | null = null

    const init = async () => {
      try {
        const available = Ctor.getSupportedFormats ? await Ctor.getSupportedFormats() : WANTED_FORMATS
        const formats = WANTED_FORMATS.filter((f) => available.includes(f))
        detector = new Ctor(formats.length ? { formats } : undefined)
      } catch {
        setSupported(false)
        return
      }
      tick()
    }

    const tick = async () => {
      if (cancelled || !detector) return
      const video = videoRef.current
      if (video && video.readyState >= 2) {
        try {
          const found = await detector.detect(video)
          const code = found[0]?.rawValue?.trim()
          if (code) {
            const now = Date.now()
            const last = lastRef.current
            if (code !== last.code || now - last.at > cooldownMs) {
              lastRef.current = { code, at: now }
              onDetectRef.current(code)
            }
          }
        } catch {
          // transient decode failure on a frame — keep polling
        }
      }
      if (!cancelled) timer = setTimeout(tick, 250)
    }

    void init()
    return () => {
      cancelled = true
      if (timer) clearTimeout(timer)
    }
  }, [active, videoRef, cooldownMs])

  return { supported }
}
