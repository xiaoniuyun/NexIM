<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { useCallStore } from '../stores/call'
import { useAuthStore } from '../stores/auth'
import { toast } from '../toast'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const call = useCallStore()
const auth = useAuthStore()

const convId = computed(() => decodeURIComponent(route.params.convId || ''))
const isVideo = computed(() => route.query.type === 'video' || call.callType === 'video')
const conv = computed(() => chat.convs.find(c => c.convId === convId.value) || {})
const name = computed(() => call.peerName || conv.value.peerName || conv.value.peerId || '对方')

const localEl = ref(null)
const remoteEl = ref(null)
let endedTimer = null

const fmt = s => `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`

const stateText = computed(() => {
  if (call.state === 'calling') return '正在呼叫…'
  if (call.state === 'ringing') return '等待接听…'
  if (call.state === 'connecting') return '正在连接…'
  if (call.state === 'active') return `通话中 ${fmt(call.duration)}`
  if (call.state === 'ended') return call.notice || '通话已结束'
  return ''
})

// 会话列表里找不到时，从 conv_u_x_u_y 解析对端
function resolveConv() {
  const found = chat.convs.find(c => c.convId === convId.value)
  if (found && found.peerId) return found
  const me = auth.user?.uid || ''
  const parts = convId.value.replace(/^conv_/, '').split('_')
  const peer = parts.find(p => p && p !== me) || ''
  return { convId: convId.value, peerId: peer, peerName: call.peerName || peer }
}

onMounted(async () => {
  call.init(chat)
  await nextTick()
  call.bindElements(localEl.value, remoteEl.value)
  if (convId.value.startsWith('conv_g_')) {
    toast('群聊暂不支持音视频通话', 'err')
    router.back()
    return
  }
  // 从聊天页进入 -> 发起呼叫；从来电浮层接听进入 -> 已在呼叫流程中
  if (call.state === 'idle') {
    const target = resolveConv()
    if (!target.peerId) {
      toast('缺少对方信息，无法发起通话', 'err')
      router.back()
      return
    }
    call.startCall(target, route.query.type === 'video' ? 'video' : 'audio')
  }
})

watch(() => call.state, s => {
  if (s === 'ended' && !endedTimer) {
    endedTimer = setTimeout(() => router.back(), 1200)
  }
})

onUnmounted(() => {
  if (endedTimer) clearTimeout(endedTimer)
  if (call.state === 'calling' || call.state === 'ringing' || call.state === 'connecting' || call.state === 'active') {
    call.hangup()
  } else {
    call.reset()
  }
})
</script>

<template>
  <div class="vcall" :class="{ video: isVideo }">
    <div class="vc-head">
      <span>{{ isVideo ? '🎬' : '📞' }} {{ name }}</span>
      <span class="sp"></span>
      <button @click="call.hangup()">－</button>
    </div>

    <div class="vc-body">
      <template v-if="isVideo">
        <video ref="remoteEl" class="vc-remote-video" autoplay playsinline></video>
        <div v-if="!call.hasRemoteVideo" class="vc-remote">{{ (name || '?').slice(0, 1) }}</div>
        <video ref="localEl" class="vc-local-video" autoplay playsinline muted></video>
      </template>
      <div v-else class="vc-remote">{{ (name || '?').slice(0, 1) }}</div>

      <div class="vc-caption">
        <div class="vc-name">{{ name }}</div>
        <div class="vc-state">{{ stateText }}</div>
        <div class="vc-tip">{{ isVideo ? 'WebRTC P2P · 视频 720p · 音频 Opus' : 'WebRTC P2P · 音频 Opus 48kHz' }}</div>
      </div>
    </div>

    <div class="vc-ctrl">
      <button class="vc-btn" :class="{ off: call.muted }" @click="call.toggleMute()">{{ call.muted ? '🔇' : '🎤' }}</button>
      <button v-if="isVideo" class="vc-btn" :class="{ off: !call.camOn }" @click="call.toggleCam()">{{ call.camOn ? '📹' : '📷' }}</button>
      <button v-if="isVideo" class="vc-btn small" @click="call.switchCamera()">🔄</button>
      <button class="vc-btn hang" @click="call.hangup()">📴</button>
    </div>

    <div class="vc-quality">
      <span>🔒 {{ call.callType === 'video' ? '端到端加密媒体' : 'DTLS-SRTP' }}</span>
      <span>{{ call.state === 'active' ? 'P2P 已连接' : '信令已就绪' }}</span>
    </div>
  </div>
</template>
