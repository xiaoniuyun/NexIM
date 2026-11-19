import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { API, wsUrl } from '../api'
import { toast } from '../toast'
import { useAuthStore } from './auth'

export const useChatStore = defineStore('chat', () => {
  const auth = useAuthStore()
  const socket = ref(null)
  const seq = ref(0)
  const convs = ref([])
  const messages = ref([])
  const currentConv = ref('')
  const connected = ref(false)
  // 对端已读回执：convId -> 最近一次已读时间
  const readMap = ref({})
  // 通话信令回调：由 call store 注册，避免 store 之间循环依赖
  let signalHandler = null
  // 连接存活检测：服务端每 30s 推送 HEARTBEAT，超过 75s 无任何帧判定连接假死
  let lastFrameAt = 0
  let watchdog = null
  function stopWatchdog() {
    if (watchdog) { clearInterval(watchdog); watchdog = null }
  }
  function startWatchdog() {
    stopWatchdog()
    watchdog = setInterval(() => {
      if (!socket.value || socket.value.readyState > 1) return
      if (Date.now() - lastFrameAt > 75000) {
        const dead = socket.value
        socket.value = null
        try { dead.close() } catch (_) {}
        connected.value = false
        connect()
      }
    }, 20000)
  }
  function onSignal(fn) { signalHandler = fn }

  const unreadTotal = computed(() => convs.value.reduce((s, c) => s + (c.unread || 0), 0))

  function parseExtra(v) {
    if (!v) return null
    if (typeof v === 'object') return v
    try { return JSON.parse(v) } catch (_) { return null }
  }

  const TYPE_MAP = { 1: 'text', 2: 'image', 3: 'system', 4: 'voice', 5: 'file', 6: 'video' }
  function normType(v) {
    return typeof v === 'number' ? (TYPE_MAP[v] || 'text') : (v || 'text')
  }

  function normalize(list) {
    return (list || []).map(m => ({
      ...m, type: normType(m.type), extra: parseExtra(m.extra),
      mine: m.fromUid === auth.user?.uid, pending: false
    }))
  }

  // 按 msgId / clientMsgId 合并，避免同一条消息重复上屏
  function upsertMessage(m) {
    const idx = messages.value.findIndex(x =>
      (m.msgId && x.msgId === m.msgId) ||
      (m.clientMsgId && x.msgId === m.clientMsgId)
    )
    if (idx >= 0) messages.value.splice(idx, 1, { ...messages.value[idx], ...m, pending: false })
    else messages.value.push(m)
  }

  // 轮询/重连时与服务端合并消息：服务端为准，本地未确认消息保留在末尾
  async function syncMessages() {
    const convId = currentConv.value
    if (!convId) return false
    try {
      const r = await API.messages(convId)
      if (currentConv.value !== convId) return false
      const list = normalize(r.data)
      const localOnly = messages.value.filter(m => m.pending || m.status === 3)
      for (const p of localOnly) {
        if (!list.some(m => m.msgId === p.msgId || (m.clientMsgId && m.clientMsgId === p.clientMsgId))) list.push(p)
      }
      list.sort((a, b) => (a.seq ?? Number.MAX_SAFE_INTEGER) - (b.seq ?? Number.MAX_SAFE_INTEGER))
      const grew = list.length !== messages.value.length
      messages.value = list
      return grew
    } catch (_) {
      return false
    }
  }

  let fetchToken = 0
  async function fetchMessages(convId) {
    const token = ++fetchToken
    try {
      const r = await API.messages(convId)
      // 快速切换会话时丢弃过期响应，避免旧会话消息覆盖新会话
      if (token !== fetchToken || currentConv.value !== convId) return
      // 保留尚未被服务端确认（含发送失败）的本地消息
      const server = normalize(r.data)
      const pending = messages.value.filter(m =>
        (m.pending || m.status === 3) && m.convId === convId &&
        !server.some(s => s.msgId === m.msgId || (s.clientMsgId && s.clientMsgId === m.clientMsgId))
      )
      messages.value = server.concat(pending)
    } catch (e) {
      if (token === fetchToken && !messages.value.length) messages.value = []
    }
  }

  function connect() {
    if (!auth.isLogin || (socket.value && socket.value.readyState <= 1)) return
    const ws = new WebSocket(wsUrl())
    socket.value = ws
    ws.onopen = () => {
      connected.value = true
      lastFrameAt = Date.now()
      startWatchdog()
      sendFrame('AUTH', {})
      // 建连/重连后补齐会话与当前会话消息，避免离线期间消息丢失
      loadConversations()
      if (currentConv.value) fetchMessages(currentConv.value)
    }
    ws.onmessage = e => {
      lastFrameAt = Date.now()
      let f; try { f = JSON.parse(e.data) } catch (_) { return }
      if (f.cmd === 'PUSH') {
        const p = f.payload || {}
        if (currentConv.value === p.convId) {
          upsertMessage({
            msgId: p.msgId, clientMsgId: p.clientMsgId, convId: p.convId, seq: p.seq,
            fromUid: p.fromUid, fromName: p.fromName, type: normType(p.type),
            content: p.content, extra: parseExtra(p.extra), createdAt: p.sendTime, status: 1,
            mine: p.fromUid === auth.user?.uid
          })
        }
        loadConversations()
      } else if (f.cmd === 'SEND_ACK') {
        const p = f.payload || {}
        if (f.code === 0 && p.clientMsgId) {
          applyAck(p.clientMsgId, p.msgId, p.seq, p.sendTime)
        } else if (f.code) {
          const m = [...messages.value].reverse().find(x => x.mine && x.pending)
          if (m) { m.status = 3; m.pending = false }
          toast(f.message || '消息发送失败', 'err')
        }
      } else if (f.cmd === 'REVOKE_PUSH') {
        const p = f.payload || {}
        const m = messages.value.find(x => x.msgId === p.msgId)
        if (m) m.status = 2
        loadConversations()
      } else if (f.cmd === 'NOTICE_PUSH' || f.cmd === 'PRESENCE_PUSH') {
        const p = f.payload || {}
        if (f.cmd === 'NOTICE_PUSH' && p.kind === 'read' && p.convId) {
          let at = Date.now()
          if (p.convId === currentConv.value) {
            const mine = messages.value.filter(m => m.mine)
            if (mine.length) at = Number(mine[mine.length - 1].createdAt) || at
          }
          readMap.value = { ...readMap.value, [p.convId]: at }
        }
        loadConversations()
      } else if (f.cmd && f.cmd.indexOf('CALL_') === 0) {
        if (signalHandler) signalHandler(f)
      } else if (f.cmd === 'KICK') {
        auth.clearAuth(); location.hash = '#/login'
      }
    }
    ws.onclose = () => {
      connected.value = false
      stopWatchdog()
      if (auth.isLogin) setTimeout(connect, 3000)
    }
  }
  function sendFrame(cmd, payload) {
    if (socket.value && socket.value.readyState === 1) {
      socket.value.send(JSON.stringify({ cmd, seqNo: ++seq.value, payload }))
    }
  }
  function disconnect() {
    stopWatchdog()
    if (socket.value) { socket.value.onclose = null; socket.value.close(); socket.value = null }
    connected.value = false
  }

  async function loadConversations() {
    try { const r = await API.conversations(); convs.value = r.data || [] } catch (e) {}
  }
  async function openConv(convId) {
    currentConv.value = convId
    await fetchMessages(convId)
    sendFrame('READ', { convId })
    API.markRead(convId).catch(() => {}).then(loadConversations)
  }
  function applyAck(clientMsgId, msgId, seq, sendTime) {
    const m = messages.value.find(x => x.msgId === clientMsgId)
    if (!m) return
    if (msgId) m.msgId = msgId
    if (seq) m.seq = seq
    if (sendTime) m.createdAt = sendTime
    m.status = 1
    m.pending = false
  }

  // HTTP 发送兜底：WebSocket 不可用或未在超时内确认时使用（clientMsgId 幂等，不会重复）
  async function httpSend(payload) {
    try {
      const r = await API.sendMessage(payload)
      const d = r.data || {}
      applyAck(payload.clientMsgId, d.msgId, d.seq, d.sendTime)
      loadConversations()
    } catch (e) {
      const m = messages.value.find(x => x.msgId === payload.clientMsgId)
      if (m) { m.status = 3; m.pending = false }
      toast(e.message || '消息发送失败', 'err')
    }
  }

  function sendPayload(type, content, extra) {
    const conv = convs.value.find(c => c.convId === currentConv.value)
    const clientMsgId = 'c_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7)
    const extraText = extra ? JSON.stringify(extra) : ''
    const payload = {
      clientMsgId, convId: currentConv.value,
      convType: conv?.convType === 2 ? 'group' : 'single',
      toUid: conv?.peerId || '', type, content, extra: extraText
    }
    messages.value.push({
      msgId: clientMsgId, clientMsgId, convId: currentConv.value, fromUid: auth.user?.uid,
      fromName: auth.user?.nickname, type, content, extra: extra || null,
      createdAt: Date.now(), status: 1, mine: true, pending: true
    })
    if (socket.value && socket.value.readyState === 1) {
      sendFrame('SEND', payload)
      // WS 未确认（断流/被代理拦截）时自动落 HTTP 通道补发
      setTimeout(() => {
        const m = messages.value.find(x => x.msgId === clientMsgId)
        if (m && m.pending) httpSend(payload)
      }, 1800)
    } else {
      httpSend(payload)
    }
    return clientMsgId
  }

  function sendText(text) {
    return sendPayload('text', text)
  }

  // 发送图片/视频/文件等富消息，extra 携带 fileId/name/size/mime
  function sendMedia(type, content, extra) {
    return sendPayload(type, content, extra)
  }

  // 本地删除一条消息（仅本端视图）
  function removeLocal(msgId) {
    messages.value = messages.value.filter(m => m.msgId !== msgId)
  }
  function ensureConv(peerId, peerName) {
    const convId = 'conv_' + [auth.user?.uid, peerId].sort().join('_')
    if (!convs.value.find(c => c.convId === convId)) {
      convs.value.unshift({ convId, convType: 1, peerId, peerName, unread: 0, lastMsg: '', lastMsgAt: Date.now() })
    }
    return convId
  }
  function reset() {
    convs.value = []; messages.value = []; currentConv.value = ''; readMap.value = {}
    disconnect()
  }

  return {
    socket, convs, messages, currentConv, connected, unreadTotal, readMap,
    connect, disconnect, sendFrame, loadConversations, openConv, sendText, sendMedia, removeLocal,
    syncMessages, ensureConv, reset, onSignal
  }
})
