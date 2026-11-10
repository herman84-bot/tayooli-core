"use client"

import React, { useEffect, useRef } from "react"

/**
 * TayooliAntigravityBackground
 * 
 * Reverse engineered from Google Antigravity (https://antigravity.google/oauth-callback):
 * - Poisson-disk sampled particulate lattice with organic Simplex/perlin displacement
 * - Dynamic interactive magnetic ring reacting to cursor / touch movement
 * - Fluid ring displacement physics with breathing oscillation and trailing inertia
 * - Rounded particulate pill/pillbox rendering with rotation aligned to tangential displacement
 * - Palette calibrated to Tayooli's Quiet Precision brand (teal-ink #1e6b58, emerald #3b816e, sage #7cc9ab)
 * - Battery-friendly: auto-pauses when tab is hidden, throttles offscreen via IntersectionObserver,
 *   and respects prefers-reduced-motion.
 */

// Simple 2D/3D Simplex-like Noise Generator for lightweight CPU canvas animation
class SimplexNoise {
  private p: Uint8Array
  private perm: Uint8Array
  private permMod12: Uint8Array

  constructor(seed = 42) {
    this.p = new Uint8Array(256)
    for (let i = 0; i < 256; i++) {
      this.p[i] = Math.floor(Math.abs(Math.sin(seed + i * 1.37) * 256)) % 256
    }
    this.perm = new Uint8Array(512)
    this.permMod12 = new Uint8Array(512)
    for (let i = 0; i < 512; i++) {
      this.perm[i] = this.p[i & 255]
      this.permMod12[i] = this.perm[i] % 12
    }
  }

  // 2D Simplex Noise
  noise2D(xin: number, yin: number): number {
    const F2 = 0.5 * (Math.sqrt(3.0) - 1.0)
    const G2 = (3.0 - Math.sqrt(3.0)) / 6.0

    let n0 = 0, n1 = 0, n2 = 0

    const s = (xin + yin) * F2
    const i = Math.floor(xin + s)
    const j = Math.floor(yin + s)
    const t = (i + j) * G2
    const X0 = i - t
    const Y0 = j - t
    const x0 = xin - X0
    const y0 = yin - Y0

    let i1 = 0, j1 = 0
    if (x0 > y0) {
      i1 = 1
      j1 = 0
    } else {
      i1 = 0
      j1 = 1
    }

    const x1 = x0 - i1 + G2
    const y1 = y0 - j1 + G2
    const x2 = x0 - 1.0 + 2.0 * G2
    const y2 = y0 - 1.0 + 2.0 * G2

    const ii = i & 255
    const jj = j & 255
    const gi0 = this.permMod12[ii + this.perm[jj]]
    const gi1 = this.permMod12[ii + i1 + this.perm[jj + j1]]
    const gi2 = this.permMod12[ii + 1 + this.perm[jj + 1]]

    const grad3 = [
      [1, 1], [-1, 1], [1, -1], [-1, -1],
      [1, 0], [-1, 0], [1, 0], [-1, 0],
      [0, 1], [0, -1], [0, 1], [0, -1]
    ]

    let t0 = 0.5 - x0 * x0 - y0 * y0
    if (t0 >= 0) {
      t0 *= t0
      n0 = t0 * t0 * (grad3[gi0][0] * x0 + grad3[gi0][1] * y0)
    }

    let t1 = 0.5 - x1 * x1 - y1 * y1
    if (t1 >= 0) {
      t1 *= t1
      n1 = t1 * t1 * (grad3[gi1][0] * x1 + grad3[gi1][1] * y1)
    }

    let t2 = 0.5 - x2 * x2 - y2 * y2
    if (t2 >= 0) {
      t2 *= t2
      n2 = t2 * t2 * (grad3[gi2][0] * x2 + grad3[gi2][1] * y2)
    }

    return 70.0 * (n0 + n1 + n2)
  }
}

interface Particle {
  baseX: number
  baseY: number
  seedA: number
  seedB: number
  sizeScale: number
}

export function TayooliAntigravityBackground() {
  const containerRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const container = containerRef.current
    const canvas = canvasRef.current
    if (!container || !canvas) return

    const ctx = canvas.getContext("2d", { alpha: true })
    if (!ctx) return

    let animationFrameId: number | null = null
    let isVisible = true
    let width = 0
    let height = 0
    let dpr = 1

    const prefersReducedMotion =
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches

    const noise = new SimplexNoise(12345)

    // Cursor & Ring Position State (smooth lerp follow)
    const targetCursor = { x: 0.5, y: 0.5 }
    const currentCursor = { x: 0.5, y: 0.5 }
    const ringPos = { x: 0.5, y: 0.5 }
    let hasMoved = false

    // Particle distribution
    let particles: Particle[] = []

    const initParticles = () => {
      particles = []
      // Density grid based on container dimensions
      const cols = Math.min(Math.max(Math.floor(width / 32), 22), prefersReducedMotion ? 25 : 48)
      const rows = Math.min(Math.max(Math.floor(height / 32), 22), prefersReducedMotion ? 25 : 48)

      for (let c = 0; c < cols; c++) {
        for (let r = 0; r < rows; r++) {
          // Jitter grid like Poisson disk distribution
          const u = (c + 0.15 + Math.sin(c * 17.1 + r * 31.4) * 0.35) / cols
          const v = (r + 0.15 + Math.cos(c * 43.7 + r * 19.2) * 0.35) / rows
          particles.push({
            baseX: u,
            baseY: v,
            seedA: Math.sin(c * 91.3 + r * 13.7),
            seedB: Math.cos(c * 27.1 + r * 71.9),
            sizeScale: 0.8 + Math.abs(Math.sin(c * 12.3 + r * 5.7)) * 0.4
          })
        }
      }
    }

    const resize = () => {
      if (!container || !canvas) return
      width = container.offsetWidth || window.innerWidth
      height = container.offsetHeight || window.innerHeight
      dpr = Math.min(window.devicePixelRatio || 1, 2)
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      canvas.style.width = `${width}px`
      canvas.style.height = `${height}px`
      ctx.setTransform(1, 0, 0, 1, 0, 0)
      ctx.scale(dpr, dpr)
      initParticles()
    }

    resize()
    window.addEventListener("resize", resize, { passive: true })

    const handlePointerMove = (e: MouseEvent | TouchEvent) => {
      hasMoved = true
      const clientX = "touches" in e ? e.touches[0].clientX : e.clientX
      const clientY = "touches" in e ? e.touches[0].clientY : e.clientY
      const rect = container.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) {
        targetCursor.x = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width))
        targetCursor.y = Math.max(0, Math.min(1, (clientY - rect.top) / rect.height))
      }
    }

    window.addEventListener("mousemove", handlePointerMove, { passive: true })
    window.addEventListener("touchmove", handlePointerMove, { passive: true })

    let startTime = performance.now()

    // Render loop
    const render = (now: number) => {
      if (!isVisible) {
        animationFrameId = null
        return
      }

      animationFrameId = requestAnimationFrame(render)

      const time = (now - startTime) * 0.001

      // Smooth cursor follow
      if (hasMoved) {
        currentCursor.x += (targetCursor.x - currentCursor.x) * 0.06
        currentCursor.y += (targetCursor.y - currentCursor.y) * 0.06
      } else {
        // Subtle autonomous float if user hasn't moved yet
        currentCursor.x = 0.5 + Math.sin(time * 0.4) * 0.12
        currentCursor.y = 0.5 + Math.cos(time * 0.3) * 0.1
      }

      // Ring position lag (elastic trailing effect like Antigravity)
      ringPos.x += (currentCursor.x - ringPos.x) * 0.035
      ringPos.y += (currentCursor.y - ringPos.y) * 0.035

      // Oscillating ring radius
      const baseRingRadius = 0.16 + Math.sin(time * 1.2) * 0.03 + Math.cos(time * 2.8) * 0.015

      ctx.clearRect(0, 0, width, height)

      // Theme detection (Tayooli Quiet Precision color scale)
      const isDark = document.documentElement.classList.contains("dark")
      const color1 = isDark ? "rgba(77, 163, 136, 0.55)" : "rgba(30, 107, 88, 0.45)" // primary teal
      const color2 = isDark ? "rgba(124, 201, 171, 0.75)" : "rgba(59, 129, 110, 0.65)" // medium emerald
      const color3 = isDark ? "rgba(163, 230, 205, 0.3)" : "rgba(19, 70, 56, 0.25)" // deep / accent

      const aspect = width / (height || 1)

      // Render particles
      for (let i = 0; i < particles.length; i++) {
        const p = particles[i]

        // Normalized relative coordinates
        const px = p.baseX
        const py = p.baseY

        // Antigravity distance metric accounting for viewport aspect ratio
        const dx = (px - ringPos.x) * aspect
        const dy = py - ringPos.y
        const dist = Math.sqrt(dx * dx + dy * dy)

        // Noise wave perturbation
        const n1 = prefersReducedMotion ? 0 : noise.noise2D(px * 3.5 + time * 0.15, py * 3.5 + time * 0.15) * 0.025
        const n2 = prefersReducedMotion ? 0 : noise.noise2D(px * 12.0 - time * 0.2, py * 12.0 + time * 0.2) * 0.008

        // Ring displacement wave (Gaussian-like peak at ring radius)
        const ringDiff = dist - baseRingRadius
        const ringIntensity = Math.exp(-ringDiff * ringDiff / 0.012)
        const innerAttract = Math.max(0, 1 - dist / (baseRingRadius * 1.5))

        // Displacement away or toward ring
        const pushFactor = (ringIntensity * 0.065 - innerAttract * 0.02)
        const angle = Math.atan2(dy, dx)
        const dispX = Math.cos(angle) * pushFactor + n1
        const dispY = Math.sin(angle) * pushFactor + n2

        const screenX = (px + dispX) * width
        const screenY = (py + dispY) * height

        // Scale reaction
        const scaleVal = Math.max(0.4, (1 + ringIntensity * 1.4) * p.sizeScale)
        const pillWidth = 4.2 * scaleVal
        const pillHeight = 2.4 * scaleVal

        // Particle orientation (rotates along displacement field)
        const particleAngle = angle + n1 * 5

        // Dynamic color blend based on ring proximity & noise
        ctx.save()
        ctx.translate(screenX, screenY)
        ctx.rotate(particleAngle)

        const alpha = Math.min(0.85, Math.max(0.12, 0.25 + ringIntensity * 0.65))
        ctx.fillStyle = ringIntensity > 0.4 ? color2 : (dist < baseRingRadius ? color1 : color3)
        ctx.globalAlpha = alpha

        // Draw rounded capsule/pill (Antigravity's signature particle shape)
        const rx = pillWidth / 2
        const ry = pillHeight / 2
        ctx.beginPath()
        if (typeof ctx.roundRect === "function") {
          ctx.roundRect(-rx, -ry, pillWidth, pillHeight, ry)
        } else {
          ctx.rect(-rx, -ry, pillWidth, pillHeight)
        }
        ctx.fill()

        ctx.restore()
      }
    }

    // Observer for viewport visibility
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        isVisible = entry.isIntersecting && !document.hidden
        if (isVisible && !animationFrameId) {
          startTime = performance.now()
          animationFrameId = requestAnimationFrame(render)
        }
      }
    }, { rootMargin: "80px" })

    observer.observe(container)

    const handleVisibilityChange = () => {
      if (document.hidden) {
        isVisible = false
        if (animationFrameId) {
          cancelAnimationFrame(animationFrameId)
          animationFrameId = null
        }
      } else {
        isVisible = true
        if (!animationFrameId) {
          startTime = performance.now()
          animationFrameId = requestAnimationFrame(render)
        }
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange)

    // Initial start
    animationFrameId = requestAnimationFrame(render)

    return () => {
      if (animationFrameId) cancelAnimationFrame(animationFrameId)
      window.removeEventListener("resize", resize)
      window.removeEventListener("mousemove", handlePointerMove)
      window.removeEventListener("touchmove", handlePointerMove)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      observer.disconnect()
    }
  }, [])

  return (
    <div
      ref={containerRef}
      className="pointer-events-none fixed inset-0 z-0 overflow-hidden"
      aria-hidden="true"
    >
      <canvas
        ref={canvasRef}
        className="pointer-events-none absolute inset-0 block h-full w-full opacity-90 transition-opacity duration-700"
      />
    </div>
  )
}
