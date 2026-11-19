<script setup>
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
import { useCallStore } from '@h5/stores/call'
import { API } from '@h5/api'
import { toast, toasts } from '@h5/toast'
import LoginView from './views/LoginView.vue'
import MainView from './views/MainView.vue'
import CallOverlay from './components/CallOverlay.vue'
import { desk, mutedSet, preview } from './composables/useDesktop'

const auth = useAuthStore()
const chat = useChatStore()
const call = useCallStore()
const offs = []
const lastSeen = new Map()

function boot() {
  chat.connect()
  call.init(chat)
}

onMounted(() => {
  if (auth.isLogin) boot()
  if (!desk) return

  offs.push(desk.onNotifyClick(({ convId }) => {
    window.dispatchEvent(new CustomEvent('nexim:open-conv', { detail: convId }))
  }))
  offs.push(desk.onReadAll(async () => {
    try {
      await API.readAll()
      chat.loadConversations()
      toast('已全部标记为已读')
    } catch (e) { toast(e.message, 'err') }
  }))
  offs.push(desk.onSearch(() => window.dispatchEvent(new CustomEvent('nexim:search'))))
  offs.push(desk.onScreenshotError(msg => toast(msg, 'err')))
  offs.push(desk.onUpdate(s => {
    if (s.status === 'available') toast(`发现新版本 ${s.version}，正在下载`)
    else if (s.status === 'downloaded') toast('新版本已就绪，重启后安装')
    else if (s.status === 'error') toast('检查更新失败：' + (s.message || '网络异常'), 'err')
  }))
})

onBeforeUnmount(() => offs.forEach(fn => fn && fn()))

watch(() => auth.isLogin, ok => { if (ok) boot(); else chat.reset() })
watch(() => chat.unreadTotal, n => desk?.setBadge(n), { immediate: true })

// 桌面通知：仅在其他会话有新消息、窗口未聚焦且未免打扰时弹出
watch(() => chat.convs, list => {
  for (const c of list || []) {
    const prev = lastSeen.get(c.convId) || 0
    const at = Number(c.lastMsgAt) || 0
    if (prev && at > prev && c.convId !== chat.currentConv && !mutedSet().has(c.convId) && !document.hasFocus()) {
      desk?.notify({
        title: c.peerName || c.name || '新消息',
        body: preview({ type: c.lastMsgType, content: c.lastMsg, extra: c.lastExtra }),
        convId: c.convId
      })
    }
    if (at) lastSeen.set(c.convId, at)
  }
}, { deep: true })
</script>

<template>
  <LoginView v-if="!auth.isLogin" />
  <MainView v-else />
  <CallOverlay v-if="auth.isLogin" />
  <div class="toast-host">
    <div v-for="t in toasts" :key="t.id" class="toast" :class="t.type">{{ t.msg }}</div>
  </div>
</template>
