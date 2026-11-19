import { API } from '@h5/api'
import { toast } from '@h5/toast'

export const desk = typeof window !== 'undefined' ? window.desktop : null

const PIN_KEY = 'desk_pinned'
const MUTE_KEY = 'desk_muted'

function readSet(key) {
  try { return new Set(JSON.parse(localStorage.getItem(key) || '[]')) } catch (_) { return new Set() }
}
function writeSet(key, set) { localStorage.setItem(key, JSON.stringify([...set])) }

export function pinnedSet() { return readSet(PIN_KEY) }
export function mutedSet() { return readSet(MUTE_KEY) }
export function togglePin(convId) {
  const s = pinnedSet()
  s.has(convId) ? s.delete(convId) : s.add(convId)
  writeSet(PIN_KEY, s)
  return s.has(convId)
}
export function toggleMute(convId) {
  const s = mutedSet()
  s.has(convId) ? s.delete(convId) : s.add(convId)
  writeSet(MUTE_KEY, s)
  return s.has(convId)
}

// 本地消息缓存：冷启动先渲染历史，再由服务端增量覆盖（不含附件二进制）
const CACHE_PREFIX = 'desk_msg_'
export function readCache(convId) {
  if (!convId) return []
  try {
    const list = JSON.parse(localStorage.getItem(CACHE_PREFIX + convId) || '[]')
    return Array.isArray(list) ? list : []
  } catch (_) { return [] }
}
export function writeCache(convId, list) {
  if (!convId) return
  try {
    const safe = (list || []).slice(-120).map(m => ({
      msgId: m.msgId, clientMsgId: m.clientMsgId, convId: m.convId, seq: m.seq,
      fromUid: m.fromUid, fromName: m.fromName, type: m.type, content: m.content,
      extra: m.extra, createdAt: m.createdAt, status: m.status, mine: m.mine
    }))
    localStorage.setItem(CACHE_PREFIX + convId, JSON.stringify(safe))
  } catch (_) {}
}

export function bytes(n) {
  const v = Number(n) || 0
  if (v < 1024) return v + ' B'
  if (v < 1024 * 1024) return (v / 1024).toFixed(1) + ' KB'
  if (v < 1024 * 1024 * 1024) return (v / 1024 / 1024).toFixed(1) + ' MB'
  return (v / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}

export function timeText(ts) {
  if (!ts) return ''
  const d = new Date(Number(ts))
  if (Number.isNaN(d.getTime())) return ''
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  const y = new Date(now); y.setDate(now.getDate() - 1)
  if (d.toDateString() === y.toDateString()) return '昨天'
  return `${d.getMonth() + 1}/${d.getDate()}`
}

export function fullTime(ts) {
  if (!ts) return ''
  const d = new Date(Number(ts))
  return `${d.getMonth() + 1}-${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export function preview(m) {
  if (!m) return ''
  const t = m.type
  if (t === 'image') return '[图片]'
  if (t === 'video') return '[视频]'
  if (t === 'file') return '[文件] ' + (m.extra?.name || '')
  if (t === 'voice') return '[语音]'
  if (t === 'system') return m.content || ''
  return m.content || ''
}

export async function uploadFiles(files, convId, chat) {
  if (!convId) { toast('请先选择会话', 'err'); return 0 }
  let ok = 0
  for (const file of files || []) {
    if (!file) continue
    if (file.size > 100 * 1024 * 1024) { toast(`${file.name} 超过 100MB 上限`, 'err'); continue }
    const fd = new FormData()
    fd.append('file', file)
    fd.append('convId', convId)
    try {
      const r = await API.fileUpload(fd)
      const rec = r.data || {}
      const mime = rec.mimeType || file.type || ''
      const type = mime.startsWith('image/') ? 'image' : mime.startsWith('video/') ? 'video' : 'file'
      chat.sendMedia(type, rec.name || file.name, {
        fileId: rec.fileId, name: rec.name || file.name,
        size: rec.size || file.size, mime
      })
      ok++
    } catch (e) {
      toast(e.message || `${file.name} 上传失败`, 'err')
    }
  }
  return ok
}

export async function sendScreenshot(chat) {
  if (!desk) return
  try {
    const shot = await desk.captureScreen()
    const b64 = await desk.readFile(shot.path)
    const bin = atob(b64)
    const arr = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i)
    const file = new File([arr], shot.name || 'screenshot.png', { type: 'image/png' })
    const ok = await uploadFiles([file], chat.currentConv, chat)
    if (ok) toast('截图已发送')
  } catch (e) {
    toast(e.message || '截图发送失败', 'err')
  }
}
