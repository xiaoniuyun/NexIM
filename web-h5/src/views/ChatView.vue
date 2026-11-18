<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import ModalBox from '../components/ModalBox.vue'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const auth = useAuthStore()
const convId = computed(() => decodeURIComponent(route.params.convId || ''))
const conv = computed(() => chat.convs.find(c => c.convId === convId.value) || {})
const name = computed(() => conv.value.peerName || conv.value.peerId || '会话')

const text = ref('')
const showAttach = ref(false)
const showEmoji = ref(false)
const showMention = ref(false)
const msgList = ref(null)
const modal = ref(null)
const emojis = ['😀','😄','😂','😊','😍','🤔','😎','👍','👏','🙏','🎉','❤️','🔥','✅','💡','🚀','📌','😅','🤝','😴','🙌']

// WebSocket 之外的兜底同步：即使推送通道异常，也能在数秒内看到最新消息
let msgTimer = null
let convTimer = null
onMounted(async () => {
  await chat.openConv(convId.value)
  scrollBottom()
  msgTimer = setInterval(async () => {
    if (document.hidden) return
    const grew = await chat.syncMessages()
    if (grew) nextTick(scrollBottom)
  }, 3000)
  convTimer = setInterval(() => { if (!document.hidden) chat.loadConversations() }, 5000)
})
onUnmounted(() => {
  if (msgTimer) clearInterval(msgTimer)
  if (convTimer) clearInterval(convTimer)
})
// 路由参数变化时组件会被复用，必须重新加载会话，否则消息停留在上一个会话
watch(convId, async v => {
  if (!v || v === chat.currentConv) return
  await chat.openConv(v)
  nextTick(scrollBottom)
})
watch(() => chat.messages.length, () => nextTick(scrollBottom))

function scrollBottom() { if (msgList.value) msgList.value.scrollTop = msgList.value.scrollHeight }
function fmtTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
const isGroup = computed(() => conv.value.convType === 2)
function startCall(kind) {
  if (isGroup.value) { toast('群聊暂不支持音视频通话', 'err'); return }
  router.push('/call/' + encodeURIComponent(convId.value) + '?type=' + kind)
}
function send() {
  const v = text.value.trim()
  if (!v) return
  chat.sendText(v)
  text.value = ''
  showEmoji.value = false; showMention.value = false
}
function addEmoji(e) { text.value += e }
function mention(u) { text.value += '@' + (u.nickname || u.username) + ' '; showMention.value = false }
function attach(kind) {
  showAttach.value = false
  if (kind === 'emoji') { showEmoji.value = !showEmoji.value; return }
  if (kind === 'at') { showMention.value = !showMention.value; return }
  toast('已选择「' + kind + '」（演示）')
}
function longPress(m) {
  if (!m.mine || m.status === 2) return
  modal.value?.open({
    title: '消息操作',
    body: '<p>撤回后各端将同步移除该消息。</p>',
    okText: '撤回', danger: true,
    onOk: async () => {
      try { await API.revoke(m.msgId) } catch (e) {}
      m.status = 2
      toast('消息已撤回')
    }
  })
}
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div>
      <div class="ch-name">{{ name }}</div>
      <div class="ch-sub"><i class="dot"></i>{{ t('online') }}</div>
    </div>
    <div class="sp"></div>
    <button v-if="!isGroup" class="icon-btn" @click="startCall('audio')">📞</button>
    <button v-if="!isGroup" class="icon-btn" @click="startCall('video')">🎥</button>
    <button class="icon-btn" @click="modal?.open({ title:'会话设置', body:'<p>会话设置（演示）</p>' })">⋯</button>
  </div>

  <div ref="msgList" class="chat-msgs">
    <div class="time-sep">{{ new Date().toLocaleDateString() }}</div>
    <div v-for="(m, i) in chat.messages" :key="m.msgId || i" class="msg-row" :class="{ mine: m.mine }">
      <Avatar v-if="!m.mine" :name="m.fromName || '?'" size="xs" />
      <div class="m-wrap">
        <div v-if="!m.mine" class="msg-sender">{{ m.fromName }}</div>
        <div v-if="m.status === 2" class="bubble msg-status-revoked">消息已撤回</div>
        <template v-else>
          <div v-if="m.type === 'image'" class="bubble media"><div class="img-ph">图片</div></div>
          <div v-else-if="m.type === 'video'" class="bubble media"><div class="video-ph"><span class="v-play">▶</span></div></div>
          <div v-else-if="m.type === 'file'" class="bubble file">📄<div><div class="fname">{{ m.content }}</div><div class="fsize">点击下载</div></div></div>
          <div v-else class="bubble" @click="longPress(m)">{{ m.content }}</div>
        </template>
        <div class="msg-meta">
          <span>{{ fmtTime(m.createdAt || m.sendTime) }}</span>
          <span v-if="m.mine && m.status === 3" style="color:var(--danger)">发送失败</span>
          <span v-else-if="m.mine">{{ m.read ? '已读' : '已送达' }}</span>
        </div>
      </div>
      <Avatar v-if="m.mine" :name="auth.user?.nickname || '我'" size="xs" />
    </div>
  </div>

  <div class="composer-wrap">
    <div v-if="showMention" class="mention-panel">
      <div class="sec-title">选择要提及的成员</div>
      <button v-for="u in chat.convs" :key="u.convId" @click="mention({ nickname: u.peerName })">@ {{ u.peerName || u.peerId }}</button>
    </div>
    <div v-if="showEmoji" class="emoji-panel">
      <button v-for="e in emojis" :key="e" @click="addEmoji(e)">{{ e }}</button>
    </div>
    <div v-if="showAttach" class="attach-panel">
      <button @click="attach('image')"><span class="at-icon">🖼️</span>相册</button>
      <button @click="attach('video')"><span class="at-icon">🎬</span>视频</button>
      <button @click="attach('file')"><span class="at-icon">📎</span>文件</button>
      <button @click="attach('call-audio')"><span class="at-icon">📞</span>语音通话</button>
      <button @click="attach('emoji')"><span class="at-icon">😊</span>表情</button>
    </div>
    <div class="composer">
      <button class="icon-btn" @click="showAttach = !showAttach">＋</button>
      <button class="icon-btn" @click="showEmoji = !showEmoji">😊</button>
      <button class="icon-btn" @click="showMention = !showMention">@</button>
      <input v-model="text" :placeholder="t('inputMsg')" @keyup.enter="send">
      <button class="send-btn" @click="send">➤</button>
    </div>
  </div>
  <ModalBox ref="modal" />
</template>
