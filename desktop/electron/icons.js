// 无第三方图形依赖，直接生成应用图标 / 托盘图标 PNG。
const zlib = require('zlib')

let crcTable = null
function crc32(buf) {
  if (!crcTable) {
    crcTable = []
    for (let n = 0; n < 256; n++) {
      let c = n
      for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
      crcTable[n] = c >>> 0
    }
  }
  let c = 0xffffffff
  for (const b of buf) c = crcTable[(c ^ b) & 0xff] ^ (c >>> 8)
  return (c ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
  const len = Buffer.alloc(4)
  len.writeUInt32BE(data.length, 0)
  const t = Buffer.from(type, 'ascii')
  const crc = Buffer.alloc(4)
  crc.writeUInt32BE(crc32(Buffer.concat([t, data])), 0)
  return Buffer.concat([len, t, data, crc])
}

function encodePng(width, height, rgba) {
  const stride = width * 4 + 1
  const raw = Buffer.alloc(stride * height)
  for (let y = 0; y < height; y++) {
    raw[y * stride] = 0
    rgba.copy(raw, y * stride + 1, y * width * 4, (y + 1) * width * 4)
  }
  const ihdr = Buffer.alloc(13)
  ihdr.writeUInt32BE(width, 0)
  ihdr.writeUInt32BE(height, 4)
  ihdr[8] = 8
  ihdr[9] = 6
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk('IHDR', ihdr),
    chunk('IDAT', zlib.deflateSync(raw, { level: 9 })),
    chunk('IEND', Buffer.alloc(0))
  ])
}

function distToSegment(px, py, x1, y1, x2, y2) {
  const dx = x2 - x1
  const dy = y2 - y1
  const len2 = dx * dx + dy * dy || 1
  let t = ((px - x1) * dx + (py - y1) * dy) / len2
  t = Math.max(0, Math.min(1, t))
  const cx = x1 + t * dx
  const cy = y1 + t * dy
  return Math.hypot(px - cx, py - cy)
}

function inRoundedRect(x, y, w, h, r) {
  const cx = Math.max(r, Math.min(w - r, x))
  const cy = Math.max(r, Math.min(h - r, y))
  return Math.hypot(x - cx, y - cy) <= r
}

// 应用图标：蓝色圆角方块 + 白色 N
function appIcon(size) {
  const buf = Buffer.alloc(size * size * 4)
  const s = size
  const r = s * 0.22
  const nw = Math.max(1.2, s * 0.085)
  for (let y = 0; y < s; y++) {
    for (let x = 0; x < s; x++) {
      const i = (y * s + x) * 4
      if (!inRoundedRect(x + 0.5, y + 0.5, s, s, r)) continue
      // 垂直渐变
      const t = y / s
      let cr = Math.round(91 + (63 - 91) * t)
      let cg = Math.round(124 + (90 - 124) * t)
      let cb = Math.round(255 + (240 - 255) * t)
      const nx = x / s
      const ny = y / s
      const d = Math.min(
        distToSegment(nx, ny, 0.33, 0.29, 0.33, 0.71),
        distToSegment(nx, ny, 0.67, 0.29, 0.67, 0.71),
        distToSegment(nx, ny, 0.33, 0.29, 0.67, 0.71)
      )
      const edge = nw / 2 / s * 1
      const a = Math.max(0, Math.min(1, (edge - d) / (1.5 / s)))
      if (a > 0) {
        cr = Math.round(cr + (255 - cr) * a)
        cg = Math.round(cg + (255 - cg) * a)
        cb = Math.round(cb + (255 - cb) * a)
      }
      buf[i] = cr
      buf[i + 1] = cg
      buf[i + 2] = cb
      buf[i + 3] = 255
    }
  }
  return encodePng(s, s, buf)
}

// 未读红点（Windows 任务栏 overlay）
function dotIcon(size, color = [239, 68, 68]) {
  const buf = Buffer.alloc(size * size * 4)
  const c = (size - 1) / 2
  const rad = size * 0.46
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const i = (y * size + x) * 4
      const a = Math.max(0, Math.min(1, rad - Math.hypot(x - c, y - c) + 0.5))
      if (a <= 0) continue
      buf[i] = color[0]
      buf[i + 1] = color[1]
      buf[i + 2] = color[2]
      buf[i + 3] = Math.round(255 * a)
    }
  }
  return encodePng(size, size, buf)
}

module.exports = { appIcon, dotIcon }
