"use client"

import React, { useEffect, useRef } from "react"
import * as THREE from "three"

/**
 * TayooliAntigravityBackground
 *
 * Exact 1:1 reverse-engineered WebGL/Three.js GPGPU shader implementation
 * from Google Antigravity (`https://antigravity.google/oauth-callback`):
 * - Poisson-Disk Sampling (shape [500, 500], density 230)
 * - 256x256 FloatType Ping-Pong RenderTargets (`simMaterial` & `renderMaterial`)
 * - Ian McEwan Ashima Arts 2D/3D/4D Simplex Noise in GLSL
 * - Signed Distance Rounded Box (`sdRoundBox(uv, vec2(0.5, 0.2), vec4(0.25))`)
 * - PerspectiveCamera(40, aspect, 0.1, 1000) at z = 3.1, mesh scale (5, 5, 5)
 * - Exact oauth-callback parameters:
 *   ringWidth = 0.011, ringWidth2 = 0.107, ringDisplacement = 0.53,
 *   density = 230, particlesScale = 0.59
 */

const GLSL_NOISE = `
  // MATHS
  vec3 permute(vec3 x) { return mod(((x*34.0)+1.0)*x, 289.0); }
  vec4 permute(vec4 x){return mod(((x*34.0)+1.0)*x, 289.0);}
  float permute(float x){return floor(mod(((x*34.0)+1.0)*x, 289.0));}

  vec4 taylorInvSqrt(vec4 r){return 1.79284291400159 - 0.85373472095314 * r;}
  float taylorInvSqrt(float r){return 1.79284291400159 - 0.85373472095314 * r;}

  // SIMPLEX NOISES
  // Simplex 2D noise
  float snoise(vec2 v){
    const vec4 C = vec4(0.211324865405187, 0.366025403784439,
            -0.577350269189626, 0.024390243902439);
    vec2 i  = floor(v + dot(v, C.yy) );
    vec2 x0 = v -   i + dot(i, C.xx);
    vec2 i1;
    i1 = (x0.x > x0.y) ? vec2(1.0, 0.0) : vec2(0.0, 1.0);
    vec4 x12 = x0.xyxy + C.xxzz;
    x12.xy -= i1;
    i = mod(i, 289.0);
    vec3 p = permute( permute( i.y + vec3(0.0, i1.y, 1.0 ))
    + i.x + vec3(0.0, i1.x, 1.0 ));
    vec3 m = max(0.5 - vec3(dot(x0,x0), dot(x12.xy,x12.xy),
      dot(x12.zw,x12.zw)), 0.0);
    m = m*m ;
    m = m*m ;
    vec3 x = 2.0 * fract(p * C.www) - 1.0;
    vec3 h = abs(x) - 0.5;
    vec3 ox = floor(x + 0.5);
    vec3 a0 = x - ox;
    m *= 1.79284291400159 - 0.85373472095314 * ( a0*a0 + h*h );
    vec3 g;
    g.x  = a0.x  * x0.x  + h.x  * x0.y;
    g.yz = a0.yz * x12.xz + h.yz * x12.yw;
    return 130.0 * dot(m, g);
  }

  // Simplex 3D Noise by Ian McEwan, Ashima Arts
  float snoise(vec3 v){
    const vec2  C = vec2(1.0/6.0, 1.0/3.0) ;
    const vec4  D = vec4(0.0, 0.5, 1.0, 2.0);

    vec3 i  = floor(v + dot(v, C.yyy) );
    vec3 x0 =   v - i + dot(i, C.xxx) ;

    vec3 g = step(x0.yzx, x0.xyz);
    vec3 l = 1.0 - g;
    vec3 i1 = min( g.xyz, l.zxy );
    vec3 i2 = max( g.xyz, l.zxy );

    vec3 x1 = x0 - i1 + 1.0 * C.xxx;
    vec3 x2 = x0 - i2 + 2.0 * C.xxx;
    vec3 x3 = x0 - 1. + 3.0 * C.xxx;

    i = mod(i, 289.0 );
    vec4 p = permute( permute( permute(
              i.z + vec4(0.0, i1.z, i2.z, 1.0 ))
            + i.y + vec4(0.0, i1.y, i2.y, 1.0 ))
            + i.x + vec4(0.0, i1.x, i2.x, 1.0 ));

    float n_ = 1.0/7.0;
    vec3  ns = n_ * D.wyz - D.xzx;

    vec4 j = p - 49.0 * floor(p * ns.z *ns.z);

    vec4 x_ = floor(j * ns.z);
    vec4 y_ = floor(j - 7.0 * x_ );

    vec4 x = x_ *ns.x + ns.yyyy;
    vec4 y = y_ *ns.x + ns.yyyy;
    vec4 h = 1.0 - abs(x) - abs(y);

    vec4 b0 = vec4( x.xy, y.xy );
    vec4 b1 = vec4( x.zw, y.zw );

    vec4 s0 = floor(b0)*2.0 + 1.0;
    vec4 s1 = floor(b1)*2.0 + 1.0;
    vec4 sh = -step(h, vec4(0.0));

    vec4 a0 = b0.xzyw + s0.xzyw*sh.xxyy ;
    vec4 a1 = b1.xzyw + s1.xzyw*sh.zzww ;

    vec3 p0 = vec3(a0.xy,h.x);
    vec3 p1 = vec3(a0.zw,h.y);
    vec3 p2 = vec3(a1.xy,h.z);
    vec3 p3 = vec3(a1.zw,h.w);

    vec4 norm = taylorInvSqrt(vec4(dot(p0,p0), dot(p1,p1), dot(p2, p2), dot(p3,p3)));
    p0 *= norm.x;
    p1 *= norm.y;
    p2 *= norm.z;
    p3 *= norm.w;

    vec4 m = max(0.6 - vec4(dot(x0,x0), dot(x1,x1), dot(x2,x2), dot(x3,x3)), 0.0);
    m = m * m;
    return 42.0 * dot( m*m, vec4( dot(p0,x0), dot(p1,x1),
                                  dot(p2,x2), dot(p3,x3) ) );
  }
`

// 1D Value Noise for autonomous cursor drift (identical to Antigravity's `kl` class)
class ValueNoise1D {
  private readonly MAX_VERTICES = 256
  private readonly MAX_VERTICES_MASK = 255
  private readonly r: number[] = []

  constructor() {
    for (let i = 0; i < this.MAX_VERTICES; ++i) {
      this.r.push(Math.random())
    }
  }

  getVal(x: number): number {
    const flo = Math.floor(x)
    const rem = x - flo
    const smooth = rem * rem * (3 - 2 * rem)
    const idxA = ((flo % this.MAX_VERTICES_MASK) + this.MAX_VERTICES_MASK) % this.MAX_VERTICES_MASK
    const idxB = (idxA + 1) % this.MAX_VERTICES_MASK
    return this.r[idxA] * (1 - smooth) + this.r[idxB] * smooth
  }
}

// Fast 2D Poisson Disk Sampling matching Antigravity's `PoissonDiskSampling`
function generatePoissonPoints(
  width: number,
  height: number,
  minDistance: number,
  maxDistance: number,
  maxTries: number
): [number, number][] {
  const cellSize = minDistance / Math.SQRT2
  const cols = Math.ceil(width / cellSize)
  const rows = Math.ceil(height / cellSize)
  const grid = new Int32Array(cols * rows).fill(-1)
  const samplePoints: [number, number][] = []
  const processList: [number, number][] = []
  const minDistSq = minDistance * minDistance
  const deltaDist = Math.max(0, maxDistance - minDistance)

  const addPoint = (pt: [number, number]) => {
    const idx = samplePoints.length
    samplePoints.push(pt)
    processList.push(pt)
    const gx = Math.floor(pt[0] / cellSize)
    const gy = Math.floor(pt[1] / cellSize)
    if (gx >= 0 && gx < cols && gy >= 0 && gy < rows) {
      grid[gy * cols + gx] = idx
    }
  }

  const inNeighbourhood = (pt: [number, number]): boolean => {
    const gx = Math.floor(pt[0] / cellSize)
    const gy = Math.floor(pt[1] / cellSize)
    const xStart = Math.max(0, gx - 2)
    const xEnd = Math.min(cols - 1, gx + 2)
    const yStart = Math.max(0, gy - 2)
    const yEnd = Math.min(rows - 1, gy + 2)

    for (let y = yStart; y <= yEnd; y++) {
      const rowOffset = y * cols
      for (let x = xStart; x <= xEnd; x++) {
        const ptIdx = grid[rowOffset + x]
        if (ptIdx !== -1) {
          const existing = samplePoints[ptIdx]
          const dx = pt[0] - existing[0]
          const dy = pt[1] - existing[1]
          if (dx * dx + dy * dy < minDistSq) {
            return true
          }
        }
      }
    }
    return false
  }

  addPoint([Math.random() * width, Math.random() * height])

  while (processList.length > 0) {
    const current = processList.shift()!
    for (let t = 0; t < maxTries; t++) {
      const radius = minDistance + deltaDist * Math.random()
      const angle = Math.random() * Math.PI * 2
      const nx = current[0] + Math.cos(angle) * radius
      const ny = current[1] + Math.sin(angle) * radius
      if (nx >= 0 && nx < width && ny >= 0 && ny < height) {
        const candidate: [number, number] = [nx, ny]
        if (!inNeighbourhood(candidate)) {
          addPoint(candidate)
        }
      }
    }
  }

  return samplePoints
}

const mapRange = (
  val: number,
  inMin: number,
  inMax: number,
  outMin: number,
  outMax: number
) => ((val - inMin) * (outMax - outMin)) / (inMax - inMin) + outMin

const clampDelta = (dt: number) => (!Number.isFinite(dt) || dt <= 0 ? 0 : Math.min(dt, 0.1))

export function TayooliAntigravityBackground() {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    // Guard for JSDOM / headless test environments without WebGL
    let renderer: THREE.WebGLRenderer
    const canvas = document.createElement("canvas")
    canvas.style.pointerEvents = "none"
    canvas.style.width = "100%"
    canvas.style.height = "100%"
    canvas.style.position = "absolute"
    canvas.style.top = "0"
    canvas.style.left = "0"

    try {
      renderer = new THREE.WebGLRenderer({
        canvas,
        antialias: true,
        alpha: true,
        powerPreference: "high-performance",
        preserveDrawingBuffer: false,
        stencil: false,
        precision: "highp",
      })
    } catch {
      return
    }

    container.appendChild(canvas)

    const gl = renderer.getContext()
    // Enable float render target support
    const hasFloatLinear =
      gl.getExtension("EXT_color_buffer_float") ||
      gl.getExtension("WEBGL_color_buffer_float") ||
      gl.getExtension("OES_texture_float")
    const hasHalfFloat =
      gl.getExtension("EXT_color_buffer_half_float") ||
      gl.getExtension("OES_texture_half_float")

    // Exact parameters from https://antigravity.google/oauth-callback
    const ringWidth = 0.011
    const ringWidth2 = 0.107
    const ringDisplacement = 0.53
    const density = 230
    const particlesScale = 0.59

    const prefersReducedMotion =
      typeof window !== "undefined" &&
      window.matchMedia &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches

    const isDark = document.documentElement.classList.contains("dark")
    const colorScheme = isDark ? 0 : 1

    // Exact Antigravity colorControls
    const color1 = isDark ? "#7189ff" : "#2c64ed"
    const color2 = isDark ? "#3074f9" : "#f84242"
    const color3 = isDark ? "#000000" : "#ffcf03"

    const pixelRatio = Math.min(window.devicePixelRatio || 1, 2)
    let width = container.offsetWidth || window.innerWidth || 1280
    let height = container.offsetHeight || window.innerHeight || 800

    renderer.setSize(width, height)
    renderer.setPixelRatio(pixelRatio)

    const scene = new THREE.Scene()
    const camera = new THREE.PerspectiveCamera(
      40,
      gl.drawingBufferWidth / Math.max(1, gl.drawingBufferHeight),
      0.1,
      1000
    )
    camera.position.z = 3.1

    // Raycasting setup for 3D mouse intersection
    const raycaster = new THREE.Raycaster()
    const mouse = new THREE.Vector2(0, 0)
    const intersectionPoint = new THREE.Vector3(0, 0, 0)
    let isIntersecting = false
    let hasMoved = false
    const cursorClient = { x: 0, y: 0 }

    const raycastPlane = new THREE.Mesh(
      new THREE.PlaneGeometry(12.5, 12.5),
      new THREE.MeshBasicMaterial({ visible: false, side: THREE.DoubleSide })
    )
    scene.add(raycastPlane)

    // 1. Generate Poisson-Disk Points
    const effectiveDensity = prefersReducedMotion ? Math.min(density, 110) : density
    const rawPoints = generatePoissonPoints(
      500,
      500,
      mapRange(effectiveDensity, 0, 300, 10, 2),
      mapRange(effectiveDensity, 0, 300, 11, 3),
      prefersReducedMotion ? 12 : 20
    )

    const pointsData: number[] = []
    for (let i = 0; i < rawPoints.length; i++) {
      pointsData.push(rawPoints[i][0] - 250, rawPoints[i][1] - 250)
    }
    const count = pointsData.length / 2

    // 2. Create 256x256 DataTexture
    const texSize = 256
    const texLength = texSize * texSize
    const posArray = new Float32Array(texLength * 4)
    for (let i = 0; i < count && i < texLength; i++) {
      const idx = i * 4
      posArray[idx + 0] = pointsData[i * 2 + 0] * (1 / 250)
      posArray[idx + 1] = pointsData[i * 2 + 1] * (1 / 250)
      posArray[idx + 2] = 0
      posArray[idx + 3] = 0
    }

    const posTex = new THREE.DataTexture(
      posArray,
      texSize,
      texSize,
      THREE.RGBAFormat,
      THREE.FloatType
    )
    posTex.needsUpdate = true

    const rtType = hasFloatLinear
      ? THREE.FloatType
      : hasHalfFloat
        ? THREE.HalfFloatType
        : THREE.FloatType

    const createRenderTarget = () =>
      new THREE.WebGLRenderTarget(texSize, texSize, {
        wrapS: THREE.ClampToEdgeWrapping,
        wrapT: THREE.ClampToEdgeWrapping,
        minFilter: THREE.NearestFilter,
        magFilter: THREE.NearestFilter,
        format: THREE.RGBAFormat,
        type: rtType,
        depthBuffer: false,
        stencilBuffer: false,
      })

    let rt1 = createRenderTarget()
    let rt2 = createRenderTarget()

    renderer.setRenderTarget(rt1)
    renderer.setClearColor(0x000000, 0)
    renderer.clear()
    renderer.setRenderTarget(rt2)
    renderer.setClearColor(0x000000, 0)
    renderer.clear()
    renderer.setRenderTarget(null)

    const noise1D = new ValueNoise1D()
    const ringPos = new THREE.Vector2(0, 0)
    const cursorPos = new THREE.Vector2(0, 0)
    let everRendered = false

    const computeParticleScale = () =>
      (renderer.domElement.width / pixelRatio / 2000) * particlesScale

    // 3. Simulation Pass (GPGPU Ping-Pong)
    const simScene = new THREE.Scene()
    const simCamera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1)
    const simMaterial = new THREE.ShaderMaterial({
      uniforms: {
        uPosition: { value: posTex },
        uPosRefs: { value: posTex },
        uRingPos: { value: new THREE.Vector2(0, 0) },
        uRingRadius: { value: 0.2 },
        uDeltaTime: { value: 0 },
        uRingWidth: { value: ringWidth },
        uRingWidth2: { value: ringWidth2 },
        uRingDisplacement: { value: ringDisplacement },
        uTime: { value: 0 },
      },
      vertexShader: `
        void main() {
          gl_Position = vec4(position, 1.0);
        }
      `,
      fragmentShader: `
        precision highp float;
        uniform sampler2D uPosition;
        uniform sampler2D uPosRefs;
        uniform vec2 uRingPos;
        uniform float uTime;
        uniform float uDeltaTime;
        uniform float uRingRadius;
        uniform float uRingWidth;
        uniform float uRingWidth2;
        uniform float uRingDisplacement;

        ${GLSL_NOISE}

        void main() {
          vec2 simTexCoords = gl_FragCoord.xy / vec2(${texSize.toFixed(1)}, ${texSize.toFixed(1)});
          vec4 pFrame = texture2D(uPosition, simTexCoords);

          float scale = pFrame.z;
          float velocity = pFrame.w;
          vec2 refPos = texture2D(uPosRefs, simTexCoords).xy;

          float time = uTime * .5;
          vec2 curentPos = refPos;

          vec2 pos = pFrame.xy;
          pos *= .8;

          float dist = distance(curentPos.xy, uRingPos);
          float noise0 = snoise(vec3(curentPos.xy * .2 + vec2(18.4924, 72.9744), time * 0.5));
          float dist1 = distance(curentPos.xy + (noise0 * .005), uRingPos);

          float t = smoothstep(uRingRadius - (uRingWidth * 2.), uRingRadius, dist) - smoothstep(uRingRadius, uRingRadius + uRingWidth, dist1);
          float t2 = smoothstep(uRingRadius - (uRingWidth2 * 2.), uRingRadius, dist) - smoothstep(uRingRadius, uRingRadius + uRingWidth2, dist1);
          float t3 = smoothstep(uRingRadius + uRingWidth2, uRingRadius, dist);

          t = pow(t, 2.);
          t2 = pow(t2, 3.);

          t += t2 * 3.;
          t += t3 * .4;
          t += snoise(vec3(curentPos.xy * 30. + vec2(11.4924, 12.9744), time * 0.5)) * t3 * .5;

          float nS = snoise(vec3(curentPos.xy * 2. + vec2(18.4924, 72.9744), time * 0.5));
          t += pow((nS + 1.5) * .5, 2.) * .6;

          // Mid scale noise
          float noise1 = snoise(vec3(curentPos.xy * 4. + vec2(88.494, 32.4397), time * 0.35));
          float noise2 = snoise(vec3(curentPos.xy * 4. + vec2(50.904, 120.947), time * 0.35));

          // Close scale noise
          float noise3 = snoise(vec3(curentPos.xy * 20. + vec2(18.4924, 72.9744), time * .5));
          float noise4 = snoise(vec3(curentPos.xy * 20. + vec2(50.904, 120.947), time * .5));

          vec2 disp = vec2(noise1, noise2) * .03;
          disp += vec2(noise3, noise4) * .005;

          // Sin wave
          disp.x += sin((refPos.x * 20.) + (time * 4.)) * .02 * clamp(dist, 0., 1.);
          disp.y += cos((refPos.y * 20.) + (time * 3.)) * .02 * clamp(dist, 0., 1.);

          pos -= (uRingPos - (curentPos + disp)) * pow(t2, .75) * uRingDisplacement;

          // Add scale
          float scaleDiff = t - scale;
          scaleDiff *= .2;
          scale += scaleDiff;

          // Final position
          vec2 finalPos = curentPos + disp + (pos * .25);

          velocity *= .5;
          velocity += scale * .25;

          gl_FragColor = vec4(finalPos, scale, velocity);
        }
      `,
    })

    const simQuad = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), simMaterial)
    simScene.add(simQuad)

    // 4. Render Points Mesh
    const geom = new THREE.BufferGeometry()
    const uvs = new Float32Array(count * 2)
    const positions = new Float32Array(count * 3)
    const seeds = new Float32Array(count * 4)

    for (let i = 0; i < count; i++) {
      const tx = i % texSize
      const ty = Math.floor(i / texSize)
      uvs[i * 2] = tx / texSize
      uvs[i * 2 + 1] = ty / texSize
      seeds[i * 4 + 0] = Math.random()
      seeds[i * 4 + 1] = Math.random()
      seeds[i * 4 + 2] = Math.random()
      seeds[i * 4 + 3] = Math.random()
    }

    geom.setAttribute("position", new THREE.BufferAttribute(positions, 3))
    geom.setAttribute("uv", new THREE.BufferAttribute(uvs, 2))
    geom.setAttribute("seeds", new THREE.BufferAttribute(seeds, 4))

    const renderMaterial = new THREE.ShaderMaterial({
      uniforms: {
        uPosition: { value: posTex },
        uTime: { value: 0 },
        uColor1: { value: new THREE.Color(color1) },
        uColor2: { value: new THREE.Color(color2) },
        uColor3: { value: new THREE.Color(color3) },
        uAlpha: { value: 1 },
        uRingPos: { value: new THREE.Vector2(0, 0) },
        uRez: {
          value: new THREE.Vector2(
            renderer.domElement.width,
            renderer.domElement.height
          ),
        },
        uParticleScale: { value: computeParticleScale() },
        uPixelRatio: { value: pixelRatio },
        uColorScheme: { value: colorScheme },
      },
      vertexShader: `
        precision highp float;
        attribute vec4 seeds;

        uniform sampler2D uPosition;
        uniform float uTime;
        uniform float uParticleScale;
        uniform float uPixelRatio;
        uniform int uColorScheme;

        varying vec4 vSeeds;
        varying float vVelocity;
        varying vec2 vLocalPos;
        varying vec2 vScreenPos;
        varying float vScale;

        void main() {
          vec4 pos = texture2D(uPosition, uv);
          vSeeds = seeds;

          vVelocity = pos.w;
          vScale = pos.z;
          vLocalPos = pos.xy;
          vec4 viewSpace = modelViewMatrix * vec4(vec3(pos.xy, 0.), 1.0);

          gl_Position = projectionMatrix * viewSpace;
          vScreenPos = gl_Position.xy;

          gl_PointSize = ((vScale * 7.) * (uPixelRatio * 0.5) * uParticleScale);
        }
      `,
      fragmentShader: `
        precision highp float;

        varying vec4 vSeeds;
        varying vec2 vScreenPos;
        varying vec2 vLocalPos;
        varying float vScale;
        varying float vVelocity;

        uniform vec3 uColor1;
        uniform vec3 uColor2;
        uniform vec3 uColor3;

        uniform vec2 uRingPos;
        uniform vec2 uRez;

        uniform float uAlpha;
        uniform float uTime;

        uniform int uColorScheme;

        ${GLSL_NOISE}

        #define PI 3.1415926535897932384626433832795

        float sdRoundBox( in vec2 p, in vec2 b, in vec4 r )
        {
          r.xy = (p.x>0.0)?r.xy : r.zw;
          r.x  = (p.y>0.0)?r.x  : r.y;
          vec2 q = abs(p)-b+r.x;
          return min(max(q.x,q.y),0.0) + length(max(q,0.0)) - r.x;
        }

        vec2 rotate(vec2 v, float a) {
          float s = sin(a);
          float c = cos(a);
          mat2 m = mat2(c, s, -s, c);
          return m * v;
        }

        void main() {
          float uBorderSize = 0.2;
          float ratio = uRez.x / uRez.y;

          float noiseAngle = snoise(vec3(vLocalPos * 10. + vec2(18.4924, 72.9744), uTime * .85));
          float noiseColor = snoise(vec3(vLocalPos * 2. + vec2(74.664, 91.556), uTime * .5));
          noiseColor = (noiseColor + 1.) * .5;

          float angle = atan(vLocalPos.y - uRingPos.y, vLocalPos.x - uRingPos.x);

          vec2 uv = gl_PointCoord.xy;
          uv -= vec2(0.5);
          uv.y *= -1.;
          uv = rotate(uv, -angle + (noiseAngle * .5));

          vec2 tuv = vScreenPos;
          tuv = rotate(tuv, uTime * 1.);
          tuv.y *= 1./ratio;
          tuv += .5;

          float h = 0.8;
          float progress = smoothstep(0., .75, pow(noiseColor, 2.));
          vec3 col = mix(mix(uColor1, uColor2, progress/h), mix(uColor2, uColor3, (progress - h)/(1.0 - h)), step(h, progress));
          vec3 color = col;

          float rounded = sdRoundBox(uv, vec2(0.5, 0.2), vec4(.25));
          rounded = smoothstep(.1, 0., rounded);

          float a = uAlpha * rounded * smoothstep(0.1, 0.2, vScale);

          if(a < 0.01){
            discard;
          }

          color = clamp(color, 0., 1.);
          color = mix(color, color * clamp(vVelocity, 0., 1.), float(uColorScheme));

          gl_FragColor = vec4(color, clamp(a, 0., 1.));

          #ifdef SRGB_TRANSFER
            gl_FragColor = sRGBTransferOETF( gl_FragColor );
          #endif
        }
      `,
      transparent: true,
      depthTest: false,
      depthWrite: false,
    })

    const pointsMesh = new THREE.Points(geom, renderMaterial)
    pointsMesh.position.set(0, 0, 0)
    pointsMesh.scale.set(5, 5, 5)
    scene.add(pointsMesh)

    // Mouse / Touch Interaction
    const onMouseMove = (e: MouseEvent) => {
      cursorClient.x = e.clientX
      cursorClient.y = e.clientY
      hasMoved = true
    }

    const onPointerDown = (e: PointerEvent) => {
      if (e.pointerType !== "mouse") {
        cursorClient.x = e.clientX
        cursorClient.y = e.clientY
        hasMoved = true
      }
    }

    window.addEventListener("mousemove", onMouseMove, { passive: true })
    window.addEventListener("pointerdown", onPointerDown, { passive: true })

    let rectDirty = true
    let canvasRect: DOMRect | null = null

    const onResize = () => {
      rectDirty = true
      width = container.offsetWidth || window.innerWidth
      height = container.offsetHeight || window.innerHeight
      renderer.setSize(width, height)
      camera.aspect = width / Math.max(1, height)
      camera.updateProjectionMatrix()
      renderMaterial.uniforms.uRez.value.set(
        renderer.domElement.width,
        renderer.domElement.height
      )
      renderMaterial.uniforms.uPixelRatio.value = pixelRatio
    }

    window.addEventListener("resize", onResize, { passive: true })

    const clock = new THREE.Clock()
    let sceneTime = 0
    let lastClockTime = 0
    let lastSimTime = 0
    let skipRayFrame = false
    let animFrameId: number | null = null
    let isVisible = true

    const animate = () => {
      if (!isVisible || document.hidden) {
        animFrameId = null
        return
      }
      animFrameId = requestAnimationFrame(animate)

      const elapsed = clock.getElapsedTime()
      const dt = clampDelta(elapsed - lastClockTime)
      lastClockTime = elapsed
      sceneTime += dt

      // Sync theme dynamically if .dark class toggles
      const nowDark = document.documentElement.classList.contains("dark")
      const targetScheme = nowDark ? 0 : 1
      if (renderMaterial.uniforms.uColorScheme.value !== targetScheme) {
        renderMaterial.uniforms.uColorScheme.value = targetScheme
        renderMaterial.uniforms.uColor1.value.set(nowDark ? "#7189ff" : "#2c64ed")
        renderMaterial.uniforms.uColor2.value.set(nowDark ? "#3074f9" : "#f84242")
        renderMaterial.uniforms.uColor3.value.set(nowDark ? "#000000" : "#ffcf03")
      }

      // Raycast cursor onto raycastPlane
      skipRayFrame = !skipRayFrame
      if (!skipRayFrame && hasMoved) {
        if (rectDirty || !canvasRect) {
          canvasRect = canvas.getBoundingClientRect()
          rectDirty = false
        }
        if (canvasRect.width > 0 && canvasRect.height > 0) {
          const nx = ((cursorClient.x - canvasRect.left) / canvasRect.width) * 2 - 1
          const ny = -((cursorClient.y - canvasRect.top) / canvasRect.height) * 2 + 1
          if (nx >= -1 && nx <= 1 && ny >= -1 && ny <= 1) {
            mouse.set(nx, ny)
            raycaster.setFromCamera(mouse, camera)
            const hits = raycaster.intersectObject(raycastPlane)
            if (hits.length > 0) {
              intersectionPoint.copy(hits[0].point)
              isIntersecting = true
            } else {
              isIntersecting = false
            }
          } else {
            isIntersecting = false
          }
        }
      }

      // Update simulation uniforms (exact Antigravity math)
      const simDt = clampDelta(elapsed - lastSimTime)
      lastSimTime = elapsed

      const nX = (noise1D.getVal(sceneTime * 0.66 + 94.234) - 0.5) * 2
      const nY = (noise1D.getVal(sceneTime * 0.75 + 21.028) - 0.5) * 2

      if (isIntersecting) {
        cursorPos.set(
          intersectionPoint.x * 0.175 + nX * 0.1,
          intersectionPoint.y * 0.175 + nY * 0.1
        )
        ringPos.set(
          ringPos.x + (cursorPos.x - ringPos.x) * 0.02,
          ringPos.y + (cursorPos.y - ringPos.y) * 0.02
        )
      } else {
        cursorPos.set(nX * 0.2, nY * 0.1)
        ringPos.set(
          ringPos.x + (cursorPos.x - ringPos.x) * 0.01,
          ringPos.y + (cursorPos.y - ringPos.y) * 0.01
        )
      }

      const pScale = computeParticleScale()

      simMaterial.uniforms.uPosition.value = everRendered ? rt1.texture : posTex
      simMaterial.uniforms.uTime.value = sceneTime
      simMaterial.uniforms.uDeltaTime.value = simDt
      simMaterial.uniforms.uRingRadius.value =
        0.175 + Math.sin(sceneTime * 1.0) * 0.03 + Math.cos(sceneTime * 3.0) * 0.02
      simMaterial.uniforms.uRingPos.value.copy(ringPos)
      simMaterial.uniforms.uRingWidth.value = ringWidth
      simMaterial.uniforms.uRingWidth2.value = ringWidth2
      simMaterial.uniforms.uRingDisplacement.value = ringDisplacement

      renderer.setRenderTarget(rt2)
      renderer.render(simScene, simCamera)
      renderer.setRenderTarget(null)

      renderMaterial.uniforms.uPosition.value = everRendered ? rt2.texture : posTex
      renderMaterial.uniforms.uTime.value = sceneTime
      renderMaterial.uniforms.uRingPos.value.copy(ringPos)
      renderMaterial.uniforms.uParticleScale.value = pScale

      renderer.autoClear = false
      renderer.clear()
      renderer.render(scene, camera)

      // Ping-pong swap
      const tmp = rt1
      rt1 = rt2
      rt2 = tmp
      everRendered = true
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          isVisible = entry.isIntersecting
          if (isVisible && !document.hidden && animFrameId === null) {
            clock.start()
            lastClockTime = 0
            lastSimTime = 0
            animFrameId = requestAnimationFrame(animate)
          } else if (!isVisible && animFrameId !== null) {
            cancelAnimationFrame(animFrameId)
            animFrameId = null
            clock.stop()
          }
        }
      },
      { rootMargin: "100px 0px", threshold: 0 }
    )
    observer.observe(container)

    const onVisibilityChange = () => {
      if (document.hidden) {
        if (animFrameId !== null) {
          cancelAnimationFrame(animFrameId)
          animFrameId = null
        }
        clock.stop()
      } else if (isVisible && animFrameId === null) {
        clock.start()
        lastClockTime = 0
        lastSimTime = 0
        animFrameId = requestAnimationFrame(animate)
      }
    }

    document.addEventListener("visibilitychange", onVisibilityChange)
    animFrameId = requestAnimationFrame(animate)

    return () => {
      if (animFrameId !== null) cancelAnimationFrame(animFrameId)
      observer.disconnect()
      document.removeEventListener("visibilitychange", onVisibilityChange)
      window.removeEventListener("mousemove", onMouseMove)
      window.removeEventListener("pointerdown", onPointerDown)
      window.removeEventListener("resize", onResize)
      geom.dispose()
      renderMaterial.dispose()
      simMaterial.dispose()
      simQuad.geometry.dispose()
      raycastPlane.geometry.dispose()
      ;(raycastPlane.material as THREE.Material).dispose()
      rt1.dispose()
      rt2.dispose()
      posTex.dispose()
      renderer.dispose()
      if (canvas.parentElement) {
        canvas.parentElement.removeChild(canvas)
      }
    }
  }, [])

  return (
    <div
      ref={containerRef}
      data-main-particles-component=""
      className="pointer-events-none fixed inset-0 z-0 overflow-hidden focus-visible:outline-none active:opacity-100 disabled:opacity-50"
      aria-hidden="true"
    />
  )
}
