// Generates PWA icons (pwa-192.png, pwa-512.png) programmatically using pngjs.
// Renders a rounded-square dark background with three glowing connector strokes
// resembling the Sub3API logo. No external native deps required.
const fs = require('fs')
const path = require('path')
const { PNG } = require('pngjs')

function lerp(a, b, t) {
  return a + (b - a) * t
}

// Simple sRGB → linear → sRGB for additive glow blending.
function addColor(dst, src, alpha) {
  for (let i = 0; i < 3; i++) {
    const s = src[i] / 255
    const d = dst[i] / 255
    const out = s * alpha + d * (1 - alpha)
    dst[i] = Math.round(Math.max(0, Math.min(255, out * 255)))
  }
}

function renderIcon(size, outPath) {
  const png = new PNG({ width: size, height: size })
  const cornerRadius = size * 0.22
  const inset = size * 0.03

  // Background gradient (navy -> indigo), diagonal.
  const bgTop = [16, 29, 76]   // #101d4c
  const bgBottom = [8, 21, 47] // #08152f

  // Brand gradient stops along stroke.
  const gradColors = [
    [103, 232, 249], // cyan #67e8f9
    [167, 139, 250], // violet #a78bfa
    [249, 168, 212], // pink #f9a8d4
  ]

  // Define the connector mark as a set of polyline segments in [0..1] space.
  // A flowing "S / layered routes" mark centered in the icon.
  const marks = [
    // Bottom-to-top curved flow
    [[0.30, 0.68], [0.38, 0.55], [0.50, 0.48], [0.62, 0.41], [0.70, 0.30]],
    // Parallel accent
    [[0.44, 0.74], [0.54, 0.62], [0.66, 0.55], [0.76, 0.46], [0.84, 0.38]],
    // Top short accent
    [[0.50, 0.24], [0.58, 0.30], [0.66, 0.36]],
  ]

  function insideRoundedRect(x, y) {
    // Coordinates in [0..1]
    const px = x * size
    const py = y * size
    const r = cornerRadius
    const x0 = inset
    const x1 = size - inset
    const y0 = inset
    const y1 = size - inset
    if (px < x0 || px > x1 || py < y0 || py > y1) return false
    const cx = Math.max(x0 + r, Math.min(px, x1 - r))
    const cy = Math.max(y0 + r, Math.min(py, y1 - r))
    const dx = px - cx
    const dy = py - cy
    return dx * dx + dy * dy <= r * r
  }

  function sampleStroke(x, y) {
    // Returns blend amount (0..1) from the connector strokes.
    let glow = 0
    const strokeWidth = 0.035 // fraction of icon
    for (const poly of marks) {
      for (let i = 0; i < poly.length - 1; i++) {
        const [ax, ay] = poly[i]
        const [bx, by] = poly[i + 1]
        // Point-to-segment distance.
        const abx = bx - ax
        const aby = by - ay
        const len2 = abx * abx + aby * aby
        let t = ((x - ax) * abx + (y - ay) * aby) / len2
        t = Math.max(0, Math.min(1, t))
        const cx = ax + t * abx
        const cy = ay + t * aby
        const dx = x - cx
        const dy = y - cy
        const dist = Math.sqrt(dx * dx + dy * dy)
        if (dist < strokeWidth) {
          const falloff = 1 - dist / strokeWidth
          glow = Math.max(glow, falloff)
        }
        // Soft outer glow
        if (dist < strokeWidth * 2.2) {
          const outer = (1 - dist / (strokeWidth * 2.2)) * 0.28
          glow = Math.max(glow, outer)
        }
      }
    }
    return Math.min(1, glow)
  }

  function gradientAt(t) {
    // t in [0..1] across stroke
    if (t <= 0.5) {
      const k = t / 0.5
      return [
        lerp(gradColors[0][0], gradColors[1][0], k),
        lerp(gradColors[0][1], gradColors[1][1], k),
        lerp(gradColors[0][2], gradColors[1][2], k),
      ]
    }
    const k = (t - 0.5) / 0.5
    return [
      lerp(gradColors[1][0], gradColors[2][0], k),
      lerp(gradColors[1][1], gradColors[2][1], k),
      lerp(gradColors[1][2], gradColors[2][2], k),
    ]
  }

  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const idx = (size * y + x) << 2
      const nx = x / size
      const ny = y / size

      if (!insideRoundedRect(nx, ny)) {
        png.data[idx] = 0
        png.data[idx + 1] = 0
        png.data[idx + 2] = 0
        png.data[idx + 3] = 0
        continue
      }

      // Background vertical gradient.
      const t = ny
      const bg = [
        lerp(bgTop[0], bgBottom[0], t),
        lerp(bgTop[1], bgBottom[1], t),
        lerp(bgTop[2], bgBottom[2], t),
      ]

      // Add stroke with gradient color along a coarse "progress" value.
      const glow = sampleStroke(nx, ny)
      const color = gradientAt((nx + ny) / 2)
      addColor(bg, color, glow)

      // Subtle vignette.
      const dx = nx - 0.5
      const dy = ny - 0.5
      const vignette = Math.min(1, Math.sqrt(dx * dx + dy * dy) * 1.5)
      for (let i = 0; i < 3; i++) {
        bg[i] = Math.round(bg[i] * (1 - vignette * 0.12))
      }

      png.data[idx] = bg[0]
      png.data[idx + 1] = bg[1]
      png.data[idx + 2] = bg[2]
      png.data[idx + 3] = 255
    }
  }

  fs.writeFileSync(outPath, PNG.sync.write(png))
}

const outDir = path.join(__dirname, '..', 'public')
fs.mkdirSync(outDir, { recursive: true })
renderIcon(192, path.join(outDir, 'pwa-192.png'))
renderIcon(512, path.join(outDir, 'pwa-512.png'))
console.log('Generated pwa-192.png and pwa-512.png')