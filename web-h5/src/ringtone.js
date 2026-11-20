// 通话铃声：WebAudio 实时合成，无需音频文件。
// - 去电回铃音：450Hz，响 1s 停 4s
// - 来电铃声：440+480Hz 双音，响 2s 停 2s，并触发振动（支持的浏览器）
let ctx = null
let timer = null
let mode = ''
let vibrateTimer = null

function ensureCtx() {
  const AC = window.AudioContext || window.webkitAudioContext
  if (!AC) return null
  if (!ctx) ctx = new AC()
  if (ctx.state === 'suspended') ctx.resume().catch(() => {})
  return ctx
}

// 浏览器要求用户交互后才能出声：首次点击时解锁音频上下文
if (typeof document !== 'undefined') {
  document.addEventListener('pointerdown', () => { try { ensureCtx() } catch (_) {} }, { once: true })
}

function stopNodes() {
  if (timer) { clearInterval(timer); timer = null }
  if (ctx) {
    try { ctx.close() } catch (_) {}
    ctx = null
  }
}

function vibrate(pattern) {
  try {
    if (navigator.vibrate) navigator.vibrate(pattern)
  } catch (_) {}
}

function start(freqs, onSec, offSec, name) {
  stop()
  const c = ensureCtx()
  if (!c) return
  mode = name
  const loop = onSec + offSec
  const schedule = () => {
    if (!ctx) return
    const t0 = ctx.currentTime + 0.03
    freqs.forEach(f => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = f
      const peak = 0.16 / freqs.length
      gain.gain.setValueAtTime(0, t0)
      gain.gain.linearRampToValueAtTime(peak, t0 + 0.02)
      gain.gain.setValueAtTime(peak, t0 + onSec - 0.04)
      gain.gain.linearRampToValueAtTime(0, t0 + onSec)
      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(t0)
      osc.stop(t0 + onSec)
    })
  }
  schedule()
  timer = setInterval(schedule, loop * 1000)
  if (name === 'incoming') {
    const beat = () => vibrate([500, 400, 500])
    beat()
    vibrateTimer = setInterval(beat, 2000)
  }
}

/** 去电等待对方接听 */
export function startRingback() {
  if (mode) return
  start([450], 1, 4, 'outgoing')
}

/** 来电提醒 */
export function startRingtone() {
  if (mode === 'incoming') return
  start([440, 480], 2, 2, 'incoming')
}

/** 停止一切铃声与振动 */
export function stopRinging() {
  mode = ''
  if (vibrateTimer) { clearInterval(vibrateTimer); vibrateTimer = null }
  try { if (navigator.vibrate) navigator.vibrate(0) } catch (_) {}
  stopNodes()
}
