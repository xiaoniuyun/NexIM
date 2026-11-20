import { defineStore } from 'pinia'
import { ref } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

// 1对1 WebRTC 通话：信令复用 WebSocket（CALL_*），媒体走 P2P + STUN/TURN
export const useCallStore = defineStore('call', () => {
  const state = ref('idle') // idle | calling | ringing | connecting | active | ended
  const callId = ref('')
  const convId = ref('')
  const peerUid = ref('')
  const peerName = ref('')
  const callType = ref('audio')
  const direction = ref('outgoing') // outgoing | incoming
  const muted = ref(false)
  const camOn = ref(true)
  const duration = ref(0)
  const notice = ref('')
  const hasRemoteVideo = ref(false)

  let pc = null
  let localStream = null
  let remoteStream = null
  let chatRef = null
  let pendingOffer = ''
  let timer = null
  let iceServers = null
  let localEl = null
  let remoteEl = null
  let facing = 'user'

  function init(chat) {
    if (!chat) return
    chatRef = chat
    chat.onSignal(handleSignal)
  }

  function bindElements(localVideo, remoteVideo) {
    localEl = localVideo || null
    remoteEl = remoteVideo || null
    attachLocal()
    attachRemote()
  }

  function attachLocal() {
    if (localEl && localStream) {
      localEl.srcObject = localStream
      localEl.play().catch(() => {})
    }
  }

  function attachRemote() {
    if (remoteEl && remoteStream) {
      remoteEl.srcObject = remoteStream
      remoteEl.play().catch(() => {})
    }
  }

  async function ensureIce() {
    if (iceServers) return
    try {
      const r = await API.rtcConfig()
      iceServers = (r.data && r.data.iceServers) || []
    } catch (_) {
      iceServers = []
    }
    if (!iceServers.length) {
      // 兜底使用与页面同源的自建 coturn
      iceServers = [{ urls: ['stun:' + (location.hostname || '127.0.0.1') + ':3478'] }]
    }
  }

  async function openMedia(type) {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      throw new Error('当前环境不支持音视频采集（需 HTTPS 或 localhost）')
    }
    const constraints = type === 'video'
      ? {
          audio: { echoCancellation: true, noiseSuppression: true },
          video: { width: { ideal: 1280 }, height: { ideal: 720 }, facingMode: facing }
        }
      : { audio: { echoCancellation: true, noiseSuppression: true }, video: false }
    localStream = await navigator.mediaDevices.getUserMedia(constraints)
    attachLocal()
  }

  function createPC() {
    pc = new RTCPeerConnection({ iceServers: iceServers || [] })
    if (localStream) localStream.getTracks().forEach(t => pc.addTrack(t, localStream))
    remoteStream = new MediaStream()
    pc.ontrack = ev => {
      if (ev.streams && ev.streams[0]) {
        ev.streams[0].getTracks().forEach(t => {
          if (!remoteStream.getTracks().some(x => x.id === t.id)) remoteStream.addTrack(t)
        })
      } else if (ev.track) {
        remoteStream.addTrack(ev.track)
      }
      hasRemoteVideo.value = remoteStream.getVideoTracks().length > 0
      attachRemote()
      if (state.value === 'connecting' || state.value === 'calling') startTimer()
    }
    pc.onicecandidate = ev => {
      if (ev.candidate && chatRef) {
        chatRef.sendFrame('CALL_ICE', {
          callId: callId.value, toUid: peerUid.value, candidate: ev.candidate.toJSON()
        })
      }
    }
    pc.onconnectionstatechange = () => {
      if (!pc) return
      if (pc.connectionState === 'connected') {
        startTimer()
      } else if (pc.connectionState === 'failed') {
        finish('连接失败，可能需要 TURN 中继')
      }
    }
  }

  // 等待 ICE 收集（最多 1.5s），把候选尽量并入 SDP 首发
  function waitIce(timeout = 1500) {
    return new Promise(resolve => {
      if (!pc || pc.iceGatheringState === 'complete') return resolve()
      let done = false
      const end = () => {
        if (done) return
        done = true
        if (pc) pc.removeEventListener('icegatheringstatechange', onChange)
        resolve()
      }
      const onChange = () => { if (pc && pc.iceGatheringState === 'complete') end() }
      pc.addEventListener('icegatheringstatechange', onChange)
      setTimeout(end, timeout)
    })
  }

  async function startCall(conv, type = 'audio') {
    if (state.value !== 'idle') return
    if (!conv || !conv.peerId) {
      toast('缺少对方信息，无法发起通话', 'err')
      return
    }
    reset()
    convId.value = conv.convId
    peerUid.value = conv.peerId
    peerName.value = conv.peerName || conv.peerId
    callType.value = type === 'video' ? 'video' : 'audio'
    direction.value = 'outgoing'
    callId.value = 'call_' + Date.now() + '_' + Math.random().toString(36).slice(2, 6)
    state.value = 'calling'
    notice.value = '正在呼叫…'
    try {
      await openMedia(callType.value)
      await ensureIce()
      createPC()
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      await waitIce()
      chatRef?.sendFrame('CALL_INVITE', {
        callId: callId.value, convId: convId.value, toUid: peerUid.value,
        callType: callType.value, sdp: (pc.localDescription || offer).sdp
      })
    } catch (e) {
      finish(e.message || '呼叫失败')
    }
  }

  async function accept() {
    if (state.value !== 'ringing') return
    state.value = 'connecting'
    notice.value = '正在连接…'
    try {
      await openMedia(callType.value)
      await ensureIce()
      createPC()
      if (pendingOffer) await pc.setRemoteDescription({ type: 'offer', sdp: pendingOffer })
      const answer = await pc.createAnswer()
      await pc.setLocalDescription(answer)
      await waitIce()
      chatRef?.sendFrame('CALL_ANSWER', {
        callId: callId.value, toUid: peerUid.value, accept: true,
        sdp: (pc.localDescription || answer).sdp
      })
    } catch (e) {
      chatRef?.sendFrame('CALL_ANSWER', { callId: callId.value, toUid: peerUid.value, accept: false })
      finish(e.message || '接听失败')
    }
  }

  function reject() {
    if (callId.value && peerUid.value) {
      chatRef?.sendFrame('CALL_ANSWER', { callId: callId.value, toUid: peerUid.value, accept: false })
    }
    finish('已拒绝')
  }

  function hangup() {
    if (callId.value && peerUid.value) {
      chatRef?.sendFrame('CALL_HANGUP', { callId: callId.value, toUid: peerUid.value })
    }
    finish('通话已结束')
  }

  async function handleSignal(f) {
    const p = f.payload || {}
    switch (f.cmd) {
      case 'CALL_INVITE_PUSH': {
        // 通话中收到新呼叫：直接回拒（忙线）
        if (state.value !== 'idle') {
          chatRef?.sendFrame('CALL_ANSWER', { callId: p.callId, toUid: p.fromUid, accept: false })
          return
        }
        reset()
        callId.value = p.callId || ''
        convId.value = p.convId || (chatRef ? chatRef.ensureConv(p.fromUid, p.fromName) : '')
        peerUid.value = p.fromUid || ''
        peerName.value = p.fromName || p.fromUid
        callType.value = p.callType === 'video' ? 'video' : 'audio'
        direction.value = 'incoming'
        pendingOffer = p.sdp || ''
        notice.value = callType.value === 'video' ? '邀请你视频通话' : '邀请你语音通话'
        state.value = 'ringing'
        break
      }
      case 'CALL_ANSWER_PUSH': {
        if (p.callId !== callId.value) return
        if (!p.accept) { finish('对方已拒绝'); return }
        state.value = 'connecting'
        notice.value = '正在连接…'
        try {
          if (pc && p.sdp) await pc.setRemoteDescription({ type: 'answer', sdp: p.sdp })
        } catch (_) {
          finish('连接建立失败')
        }
        break
      }
      case 'CALL_ICE_PUSH': {
        if (p.callId !== callId.value || !pc || !p.candidate) return
        try { await pc.addIceCandidate(p.candidate) } catch (_) {}
        break
      }
      case 'CALL_HANGUP_PUSH': {
        if (p.callId !== callId.value) return
        finish(direction.value === 'incoming' ? '对方已取消' : '对方已挂断')
        break
      }
      case 'CALL_FORCE_END_PUSH': {
        if (p.callId && p.callId !== callId.value) return
        finish(p.reason || '管理员已结束通话')
        break
      }
      case 'CALL_ACK': {
        if (f.code && f.code !== 0) finish(f.message || '呼叫失败')
        break
      }
    }
  }

  function startTimer() {
    if (timer) return
    state.value = 'active'
    notice.value = '通话中'
    timer = setInterval(() => { duration.value++ }, 1000)
  }

  function stopTimer() {
    if (timer) { clearInterval(timer); timer = null }
  }

  function toggleMute() {
    if (!localStream) return
    muted.value = !muted.value
    localStream.getAudioTracks().forEach(t => { t.enabled = !muted.value })
  }

  function toggleCam() {
    if (!localStream) return
    const tracks = localStream.getVideoTracks()
    if (!tracks.length) return
    camOn.value = !camOn.value
    tracks.forEach(t => { t.enabled = camOn.value })
  }

  async function switchCamera() {
    if (callType.value !== 'video' || !localStream || !pc) return
    facing = facing === 'user' ? 'environment' : 'user'
    try {
      const ns = await navigator.mediaDevices.getUserMedia({ video: { facingMode: facing }, audio: false })
      const track = ns.getVideoTracks()[0]
      if (!track) return
      const sender = pc.getSenders().find(s => s.track && s.track.kind === 'video')
      if (sender) await sender.replaceTrack(track)
      const old = localStream.getVideoTracks()[0]
      if (old) { localStream.removeTrack(old); old.stop() }
      localStream.addTrack(track)
      attachLocal()
    } catch (_) {
      toast('切换摄像头失败', 'err')
    }
  }

  function stopMedia() {
    if (pc) {
      try {
        pc.onicecandidate = null
        pc.ontrack = null
        pc.onconnectionstatechange = null
        pc.close()
      } catch (_) {}
      pc = null
    }
    if (localStream) {
      localStream.getTracks().forEach(t => t.stop())
      localStream = null
    }
    remoteStream = null
    hasRemoteVideo.value = false
    if (localEl) localEl.srcObject = null
    if (remoteEl) remoteEl.srcObject = null
  }

  // 通话结束前采集 WebRTC 质量并上报，供管理端质量监控使用
  function reportQuality() {
    const id = callId.value
    const conn = pc
    if (!id || !conn || duration.value <= 0) return Promise.resolve()
    return conn.getStats().then(report => {
      let packetsLost = 0, packetsReceived = 0, jitter = 0, rtt = 0
      report.forEach(s => {
        if (s.type === 'inbound-rtp') {
          packetsLost += s.packetsLost || 0
          packetsReceived += s.packetsReceived || 0
          jitter = Math.max(jitter, Math.round((s.jitter || 0) * 1000))
        }
        if (s.type === 'candidate-pair' && (s.selected || s.state === 'succeeded')) {
          rtt = Math.max(rtt, Math.round((s.currentRoundTripTime || 0) * 1000))
        }
      })
      const total = packetsLost + packetsReceived
      const loss = total > 0 ? Math.min(100, (packetsLost / total) * 100) : 0
      let mos = 4.5 - loss * 0.25 - (rtt > 300 ? 0.6 : 0) - (jitter > 50 ? 0.4 : 0)
      mos = Math.max(1, Math.min(5, Number(mos.toFixed(2))))
      API.reportQuality(id, { mos, packetLoss: Number(loss.toFixed(2)), rtt, jitter }).catch(() => {})
    }).catch(() => {})
  }

  function finish(msg) {
    stopTimer()
    state.value = 'ended'
    notice.value = msg || '通话已结束'
    // 先采集 getStats 再关闭 PeerConnection，否则质量数据会丢失
    let stopped = false
    const teardown = () => {
      if (stopped) return
      stopped = true
      stopMedia()
      setTimeout(() => { if (state.value === 'ended') reset() }, 1600)
    }
    reportQuality().finally(teardown)
    setTimeout(teardown, 800)
  }

  function reset() {
    stopTimer()
    stopMedia()
    state.value = 'idle'
    callId.value = ''
    convId.value = ''
    peerUid.value = ''
    peerName.value = ''
    callType.value = 'audio'
    direction.value = 'outgoing'
    muted.value = false
    camOn.value = true
    duration.value = 0
    notice.value = ''
    pendingOffer = ''
    facing = 'user'
  }

  return {
    state, callId, convId, peerUid, peerName, callType, direction,
    muted, camOn, duration, notice, hasRemoteVideo,
    init, bindElements, startCall, accept, reject, hangup,
    toggleMute, toggleCam, switchCamera, reset
  }
})
