<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'
import { API, fileUrl } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import LineIcon from '../components/LineIcon.vue'
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
const showCallMenu = ref(false)
const panelPage = ref(0)
const msgList = ref(null)
const modal = ref(null)
const emojis = ['😀','😄','😂','😊','😍','🤔','😎','👍','👏','🙏','🎉','❤️','🔥','✅','💡','🚀','📌','😅','🤝','😴','🙌']

// 聊天工具栏：分页宫格（对齐设计图）
const gridPages = [
  [
    { key: 'call-audio', label: '语音通话', icon: 'phone' },
    { key: 'call-video', label: '视频通话', icon: 'video' },
    { key: 'location', label: '位置', icon: 'location' },
    { key: 'file', label: '文件', icon: 'folder' },
    { key: 'redpacket', label: '红包', icon: 'redpacket' },
    { key: 'transfer', label: '转账', icon: 'transfer' },
    { key: 'favorite', label: '收藏', icon: 'star' },
    { key: 'screenshare', label: '屏幕分享', icon: 'screenshare' }
  ],
  [
    { key: 'image', label: '相册', icon: 'image' },
    { key: 'video-file', label: '视频', icon: 'video' },
    { key: 'voice', label: '语音消息', icon: 'mic' },
    { key: 'mention', label: '提及', icon: 'at' },
    { key: 'emoji', label: '表情', icon: 'smile' }
  ]
]
const gridItems = computed(() => gridPages[panelPage.value])
function gridAction(key) {
  showAttach.value = false
  switch (key) {
    case 'call-audio': startCall('audio'); return
    case 'call-video': startCall('video'); return
    case 'file': attach('file'); return
    case 'image': attach('image'); return
    case 'video-file': attach('video'); return
    case 'voice': toggleVoiceMode(); return
    case 'mention': openMention(); return
    case 'emoji': toggleEmoji(); return
    case 'favorite': router.push('/favorites'); return
    default: toast('暂不支持该功能', 'err')
  }
}

// 附件上传
const filePick = ref(null)
const pickKind = ref('file')
const uploading = ref(false)

// 语音录制
const recording = ref(false)
const recSeconds = ref(0)
let recorder = null
let cancelRecording = false
let recChunks = []
let recStream = null
let recTimer = null
let recStartedAt = 0

// 消息操作 / 转发
const showActions = ref(false)
const actionMsg = ref(null)
const showForward = ref(false)
const forwardIds = ref([])
const forwardNote = ref('')

// 会话设置 / 搜索 / 群成员
const showSettings = ref(false)
const showSearch = ref(false)
const searchKeyword = ref('')
const searchResults = ref([])
const searching = ref(false)
const showMembers = ref(false)
const showInvite = ref(false)
const members = ref([])
const inviteIds = ref([])
const friendList = ref([])
const mentionList = ref([])

let msgTimer = null
let convTimer = null
onMounted(async () => {
  chat.setViewing(true)
  window.addEventListener('keydown', onRecKey)
  window.addEventListener('keyup', onRecKeyUp)
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
  chat.setViewing(false)
  window.removeEventListener('keydown', onRecKey)
  window.removeEventListener('keyup', onRecKeyUp)
  stopRecord()
  releaseStream()
  if (msgTimer) clearInterval(msgTimer)
  if (convTimer) clearInterval(convTimer)
})
watch(convId, async v => {
  exitVoiceMode()
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
function fmtSize(n) {
  if (!n) return ''
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(1) + ' MB'
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
function toggleEmoji() {
  showAttach.value = false
  showEmoji.value = !showEmoji.value
}

async function openMention() {
  showMention.value = !showMention.value
  if (!showMention.value) return
  if (isGroup.value) {
    try { const r = await API.groupMembers(conv.value.peerId); mentionList.value = r.data || [] } catch (_) { mentionList.value = [] }
  } else {
    mentionList.value = [{ uid: conv.value.peerId, nickname: conv.value.peerName }]
  }
}
function mention(u) { text.value += '@' + (u.nickname || u.username || '') + ' '; showMention.value = false }

function attach(kind) {
  showAttach.value = false
  if (kind === 'emoji') { showEmoji.value = !showEmoji.value; return }
  if (kind === 'at') { openMention(); return }
  if (kind === 'call-audio') { startCall('audio'); return }
  pickKind.value = kind
  const input = filePick.value
  if (!input) return
  input.accept = kind === 'image' || kind === 'camera' ? 'image/*' : kind === 'video' ? 'video/*' : '*/*'
  if (kind === 'camera') input.setAttribute('capture', 'environment')
  else input.removeAttribute('capture')
  input.value = ''
  input.click()
}

async function onFilePick(e) {
  const file = e.target.files && e.target.files[0]
  if (!file) return
  if (file.size > 100 * 1024 * 1024) { toast('文件不能超过 100MB', 'err'); return }
  const fd = new FormData()
  fd.append('file', file)
  fd.append('convId', convId.value)
  uploading.value = true
  try {
    const r = await API.fileUpload(fd)
    const rec = r.data || {}
    const mime = rec.mimeType || file.type || ''
    const type = mime.startsWith('image/') ? 'image' : mime.startsWith('video/') ? 'video' : 'file'
    chat.sendMedia(type, rec.name || file.name, {
      fileId: rec.fileId, name: rec.name || file.name, size: rec.size || file.size, mime
    })
  } catch (err) {
    toast(err.message || '上传失败', 'err')
  } finally {
    uploading.value = false
  }
}

function openFile(m) {
  if (!m.extra?.fileId) return
  window.open(fileUrl(m.extra.fileId), '_blank')
}

// ===== 语音消息（内联按住说话） =====
const voiceMode = ref(false)
const willCancel = ref(false)
let startPending = false
let voiceStartY = 0
function toggleVoiceMode() {
  if (voiceMode.value) { exitVoiceMode(); return }
  showAttach.value = false
  showEmoji.value = false
  voiceMode.value = true
}
function exitVoiceMode() {
  if (recording.value || startPending) cancelRecord()
  willCancel.value = false
  voiceMode.value = false
}
function startVoiceHold(e) {
  if (e && e.currentTarget && e.pointerId != null) {
    try { e.currentTarget.setPointerCapture(e.pointerId) } catch (_) {}
  }
  voiceStartY = e?.clientY ?? 0
  willCancel.value = false
  if (!recording.value) toggleRecord()
}
function onVoiceMove(e) {
  if (!recording.value) return
  willCancel.value = voiceStartY - e.clientY > 60
}
function endVoiceHold() {
  if (!recording.value && !startPending) return
  if (willCancel.value) cancelRecord()
  else stopRecord()
  willCancel.value = false
}
async function toggleRecord() {
  if (recording.value) { stopRecord(); return }
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia || typeof MediaRecorder === 'undefined') {
    toast('当前浏览器不支持录音', 'err')
    return
  }
  try {
    cancelRecording = false
    startPending = true
    recStream = await navigator.mediaDevices.getUserMedia({ audio: true })
    // 授权期间用户已松手或退出语音模式：放弃本次录音
    if (!startPending || !voiceMode.value) {
      startPending = false
      releaseStream()
      return
    }
    startPending = false
    const mime = MediaRecorder.isTypeSupported('audio/webm;codecs=opus') ? 'audio/webm;codecs=opus'
      : MediaRecorder.isTypeSupported('audio/webm') ? 'audio/webm'
      : MediaRecorder.isTypeSupported('audio/mp4') ? 'audio/mp4' : ''
    recorder = new MediaRecorder(recStream, mime ? { mimeType: mime } : undefined)
    recChunks = []
    recorder.ondataavailable = e => { if (e.data && e.data.size) recChunks.push(e.data) }
    recorder.onstop = () => finishRecord()
    recStartedAt = Date.now()
    recSeconds.value = 0
    recorder.start()
    recording.value = true
    recTimer = setInterval(() => { recSeconds.value = Math.floor((Date.now() - recStartedAt) / 1000) }, 300)
  } catch (err) {
    toast('无法访问麦克风：' + (err.message || ''), 'err')
    releaseStream()
  }
}

function stopRecord() {
  startPending = false
  if (recTimer) { clearInterval(recTimer); recTimer = null }
  recording.value = false
  try { if (recorder && recorder.state !== 'inactive') recorder.stop() } catch (_) {}
}

function cancelRecord() {
  cancelRecording = true
  stopRecord()
}

function onRecKey(e) {
  if (!voiceMode.value) return
  if (e.key === 'Escape') {
    if (recording.value) cancelRecord()
    else voiceMode.value = false
    return
  }
  if (e.code === 'Space' && !recording.value) {
    const el = document.activeElement
    if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) return
    e.preventDefault()
    startVoiceHold()
  }
}
function onRecKeyUp(e) {
  if (e.code === 'Space' && (recording.value || startPending || willCancel.value)) endVoiceHold()
}

function releaseStream() {
  if (recStream) { recStream.getTracks().forEach(t => t.stop()); recStream = null }
}

async function finishRecord() {
  const secs = Math.max(1, Math.round((Date.now() - recStartedAt) / 1000))
  const mime = (recorder && recorder.mimeType) || 'audio/webm'
  const blob = new Blob(recChunks, { type: mime })
  recChunks = []
  recorder = null
  releaseStream()
  if (cancelRecording) { cancelRecording = false; return }
  if (blob.size < 800) { toast('录音时间太短', 'err'); return }
  const ext = mime.indexOf('mp4') >= 0 ? 'm4a' : 'webm'
  const file = new File([blob], 'voice_' + Date.now() + '.' + ext, { type: mime })
  const fd = new FormData()
  fd.append('file', file)
  fd.append('convId', convId.value)
  try {
    const r = await API.fileUpload(fd)
    const rec = r.data || {}
    chat.sendMedia('voice', rec.name || file.name, {
      fileId: rec.fileId, name: rec.name || file.name,
      size: rec.size || blob.size, mime, duration: secs
    })
  } catch (err) {
    toast(err.message || '语音发送失败', 'err')
  }
}

// ===== 消息操作 =====
function openActions(m) {
  if (m.status === 2) return
  actionMsg.value = m
  showActions.value = true
}
async function copyMsg() {
  const m = actionMsg.value
  try { await navigator.clipboard.writeText(m.content || '') ; toast(t('copied')) } catch (_) { toast('复制失败', 'err') }
  showActions.value = false
}
function startForward() {
  const m = actionMsg.value
  if (!m?.msgId || m.pending) { toast('消息发送中，暂不能转发', 'err'); return }
  showActions.value = false
  forwardIds.value = []
  forwardNote.value = ''
  showForward.value = true
}
async function doForward() {
  const m = actionMsg.value
  if (!m || !forwardIds.value.length) { toast('请选择转发目标', 'err'); return }
  try {
    const r = await API.forward(m.msgId, forwardIds.value, forwardNote.value.trim())
    const n = r.data?.forwarded ?? forwardIds.value.length
    toast(`已转发到 ${n} 个会话`)
    showForward.value = false
    chat.loadConversations()
  } catch (e) { toast(e.message, 'err') }
}
async function doFavorite() {
  const m = actionMsg.value
  if (!m?.msgId || m.pending) { toast('消息发送中，暂不能收藏', 'err'); return }
  try { await API.favorite(m.msgId); toast('已收藏') } catch (e) { toast(e.message, 'err') }
  showActions.value = false
}
function doRevoke() {
  const m = actionMsg.value
  showActions.value = false
  modal.value?.open({
    title: t('revoke'),
    body: '<p>撤回后各端将同步移除该消息。</p>',
    okText: t('revoke'), danger: true,
    onOk: async () => {
      try { await API.revoke(m.msgId) } catch (e) {}
      m.status = 2
      toast('消息已撤回')
    }
  })
}
function deleteLocal() {
  const m = actionMsg.value
  if (m?.msgId) chat.removeLocal(m.msgId)
  showActions.value = false
}

// ===== 会话设置 =====
async function toggleConvPin() {
  const next = conv.value.pinned ? 0 : 1
  try { await API.convSettings(convId.value, { pinned: next }); await chat.loadConversations(); toast(t('pinSuccess')) } catch (e) { toast(e.message, 'err') }
}
async function toggleConvMute() {
  const next = conv.value.muted ? 0 : 1
  try { await API.convSettings(convId.value, { muted: next }); await chat.loadConversations(); toast(t('pinSuccess')) } catch (e) { toast(e.message, 'err') }
}
async function markConvRead() {
  try { await API.markRead(convId.value); await chat.loadConversations(); toast(t('allRead')) } catch (e) { toast(e.message, 'err') }
}
function openSearchFromSettings() { showSettings.value = false; showSearch.value = true; searchResults.value = []; searchKeyword.value = '' }
async function doSearch() {
  const kw = searchKeyword.value.trim()
  if (!kw) return
  searching.value = true
  try {
    const r = await API.searchMessages(kw, convId.value)
    const label = { 1: '文本', 2: '图片', 3: '系统', 4: '语音', 5: '文件', 6: '视频' }
    searchResults.value = (r.data || []).map(x => ({ ...x, typeLabel: label[x.type] || '消息' }))
  } catch (e) { toast(e.message, 'err') } finally { searching.value = false }
}
function gotoResult(r) {
  showSearch.value = false
  nextTick(() => {
    const el = msgList.value?.querySelector(`[data-mid="${r.msgId}"]`)
    if (el) { el.scrollIntoView({ block: 'center' }); el.classList.add('flash'); setTimeout(() => el.classList.remove('flash'), 1200) }
    else toast('该消息不在当前加载范围内')
  })
}
async function openMembers() {
  try {
    const r = await API.groupMembers(conv.value.peerId)
    members.value = r.data || []
    showMembers.value = true
  } catch (e) { toast(e.message, 'err') }
}
async function openInvite() {
  try {
    const [mr, fr] = await Promise.all([API.groupMembers(conv.value.peerId), API.friends()])
    members.value = mr.data || []
    const memberIds = new Set(members.value.map(m => m.uid))
    friendList.value = (fr.data || []).filter(f => !memberIds.has(f.uid))
    inviteIds.value = []
    showInvite.value = true
  } catch (e) { toast(e.message, 'err') }
}
async function doInvite() {
  if (!inviteIds.value.length) { toast('请选择要邀请的成员', 'err'); return }
  try { await API.groupInvite(conv.value.peerId, inviteIds.value); toast(`已邀请 ${inviteIds.value.length} 位成员`); showInvite.value = false } catch (e) { toast(e.message, 'err') }
}
async function leaveGroup() {
  if (!window.confirm('退出后将从群聊列表中移除，确定退出？')) return
  try {
    await API.groupLeave(conv.value.peerId, auth.user?.uid)
    showSettings.value = false
    await chat.loadConversations()
    router.replace('/convs')
    toast('已退出群聊')
  } catch (e) { toast(e.message, 'err') }
}
function pickTarget(id) {
  const i = forwardIds.value.indexOf(id)
  if (i >= 0) forwardIds.value.splice(i, 1)
  else forwardIds.value.push(id)
}
</script>
<template>
  <div class="chat-head">
    <button class="ch-back" @click="router.back()"><LineIcon name="back" :size="24" :stroke="2.2" /></button>
    <button class="ch-info" @click="showSettings = true">
      <div class="ch-name">{{ name }}</div>
      <div class="ch-sub">
        <i class="dot"></i>{{ isGroup ? `${members.length || conv.memberCount || ''} 人群聊` : '在线 · WiFi' }}<span class="ch-arrow">›</span>
      </div>
    </button>
    <div class="sp"></div>
    <button v-if="!isGroup" class="ch-act" title="发起通话" @click="showCallMenu = true"><LineIcon name="phone" :size="23" /></button>
    <button class="ch-act" title="会话设置" @click="showSettings = true"><LineIcon name="menu" :size="23" /></button>
  </div>

  <div v-if="showCallMenu" class="mask sheet-mask" @click.self="showCallMenu = false">
    <div class="sheet">
      <div class="sheet-title">发起通话</div>
      <button class="sheet-item" @click="showCallMenu = false; startCall('audio')">语音通话</button>
      <button class="sheet-item" @click="showCallMenu = false; startCall('video')">视频通话</button>
      <button class="sheet-item cancel" @click="showCallMenu = false">{{ t('cancel') }}</button>
    </div>
  </div>

  <div ref="msgList" class="chat-msgs">
    <div class="time-sep">{{ new Date().toLocaleDateString() }}</div>
    <div v-for="(m, i) in chat.messages" :key="m.msgId || i" class="msg-row" :class="{ mine: m.mine }" :data-mid="m.msgId">
      <Avatar v-if="!m.mine" :name="m.fromName || '?'" :src="isGroup ? '' : conv.peerAvatar" size="xs" />
      <div class="m-wrap">
        <div v-if="!m.mine" class="msg-sender">{{ m.fromName }}</div>
        <div v-if="m.status === 2" class="bubble msg-status-revoked">消息已撤回</div>
        <template v-else>
          <div v-if="m.type === 'image'" class="bubble media">
            <img v-if="m.extra?.fileId" :src="fileUrl(m.extra.fileId)" class="msg-img" @click="openFile(m)">
            <div v-else class="img-ph">图片</div>
            <button class="media-more" @click.stop="openActions(m)">⋯</button>
          </div>
          <div v-else-if="m.type === 'video'" class="bubble media">
            <video v-if="m.extra?.fileId" :src="fileUrl(m.extra.fileId)" class="msg-video" controls></video>
            <div v-else class="video-ph"><span class="v-play">▶</span></div>
            <button class="media-more" @click.stop="openActions(m)">⋯</button>
          </div>
          <div v-else-if="m.type === 'file'" class="bubble file">
            <a v-if="m.extra?.fileId" :href="fileUrl(m.extra.fileId)" target="_blank" class="file-link">
              <LineIcon name="file" :size="22" /><div><div class="fname">{{ m.extra.name || m.content }}</div><div class="fsize">{{ fmtSize(m.extra.size) || '点击下载' }}</div></div>
            </a>
            <span v-else><LineIcon name="file" :size="22" /><div><div class="fname">{{ m.content }}</div><div class="fsize">点击下载</div></div></span>
            <button class="media-more" @click.stop="openActions(m)">⋯</button>
          </div>
          <div v-else-if="m.type === 'voice'" class="bubble voice">
            <audio v-if="m.extra?.fileId" :src="fileUrl(m.extra.fileId)" controls preload="none"></audio>
            <span v-else><LineIcon name="mic" :size="18" /> 语音 {{ m.extra?.duration ? m.extra.duration + '″' : '' }}</span>
            <button class="media-more" @click.stop="openActions(m)">⋯</button>
          </div>
          <div v-else class="bubble" @click="openActions(m)">{{ m.content }}</div>
        </template>
        <div class="msg-meta">
          <span>{{ fmtTime(m.createdAt || m.sendTime) }}</span>
          <span v-if="m.mine && m.status === 3" style="color:var(--danger)">发送失败</span>
          <span v-else-if="m.mine">{{ m.read ? '已读' : '已送达' }}</span>
        </div>
      </div>
      <Avatar v-if="m.mine" :name="auth.user?.nickname || '我'" :src="auth.user?.avatar" size="xs" />
    </div>
  </div>

  <div class="composer-wrap">
    <div v-if="showMention" class="mention-panel">
      <div class="sec-title">选择要提及的成员</div>
      <button v-for="u in mentionList" :key="u.uid" @click="mention(u)">@ {{ u.nickname || u.uid }}</button>
      <div v-if="!mentionList.length" class="loading">暂无成员</div>
    </div>
    <div v-if="showEmoji" class="emoji-panel">
      <button v-for="e in emojis" :key="e" @click="addEmoji(e)">{{ e }}</button>
    </div>
    <div class="c-sheet">
      <div v-if="recording || willCancel" class="voice-hud" :class="{ cancel: willCancel }">
        <div class="wave"><i v-for="n in 7" :key="n"></i></div>
        <span class="voice-timer">{{ recSeconds }}″</span>
        <span class="voice-tip">{{ willCancel ? '松开 取消' : '松开 发送 · 上滑 取消' }}</span>
      </div>
      <div class="c-input-row">
        <div
          v-if="voiceMode" class="voice-bar" :class="{ rec: recording, cancel: willCancel }"
          @pointerdown.prevent="startVoiceHold" @pointermove="onVoiceMove"
          @pointerup.prevent="endVoiceHold" @pointercancel="endVoiceHold"
        >
          <LineIcon name="mic" :size="19" />
          <span>{{ recording ? (willCancel ? '松开 取消' : '松开 发送') : '按住 说话' }}</span>
        </div>
        <input v-else v-model="text" :placeholder="t('inputMsg')" @keyup.enter="send">
      </div>
      <div class="c-tools">
        <button class="c-tool" :class="{ recording }" title="语音" @click="toggleVoiceMode"><LineIcon name="mic" :size="26" /></button>
        <button class="c-tool" :disabled="uploading" title="相册" @click="attach('image')"><LineIcon name="image" :size="26" /></button>
        <button class="c-tool" :disabled="uploading" title="拍摄" @click="attach('camera')"><LineIcon name="camera" :size="26" /></button>
        <button class="c-tool" title="贴纸" @click="toggleEmoji"><LineIcon name="sticker" :size="26" /></button>
        <button class="c-tool" title="表情" @click="toggleEmoji"><LineIcon name="smile" :size="26" /></button>
        <button class="c-close" :class="{ on: showAttach }" title="更多" @click="showEmoji = false; showAttach = !showAttach"><LineIcon name="close" :size="18" /></button>
      </div>
    </div>
    <div v-if="showAttach" class="c-panel">
      <div class="c-grid">
        <button v-for="it in gridItems" :key="it.key" class="c-cell" @click="gridAction(it.key)">
          <span class="c-tile"><LineIcon :name="it.icon" :size="26" /></span>
          <span>{{ it.label }}</span>
        </button>
      </div>
      <div class="c-dots">
        <i v-for="(_, i) in gridPages" :key="i" :class="{ on: panelPage === i }" @click="panelPage = i"></i>
      </div>
    </div>
  </div>
  <input ref="filePick" type="file" class="hidden-file" @change="onFilePick">

  <!-- 消息操作 -->
  <div v-if="showActions" class="mask sheet-mask" @click.self="showActions = false">
    <div class="sheet">
      <div class="sheet-title">{{ actionMsg?.content?.slice(0, 24) || '消息操作' }}</div>
      <button v-if="actionMsg?.type === 'text'" class="sheet-item" @click="copyMsg">{{ t('copy') }}</button>
      <button class="sheet-item" @click="startForward">{{ t('forward') }}</button>
      <button class="sheet-item" @click="doFavorite">{{ t('favorite') }}</button>
      <button v-if="actionMsg?.mine" class="sheet-item danger" @click="doRevoke">{{ t('revoke') }}</button>
      <button class="sheet-item danger" @click="deleteLocal">{{ t('deleteLocal') }}</button>
      <button class="sheet-item cancel" @click="showActions = false">{{ t('cancel') }}</button>
    </div>
  </div>

  <!-- 转发 -->
  <div v-if="showForward" class="mask" @click.self="showForward = false">
    <div class="modal">
      <div class="mh">{{ t('forwardTo') }}</div>
      <div class="mb">
        <div class="pick-list">
          <button v-for="c in chat.convs.filter(x => x.convId !== convId)" :key="c.convId" class="pick-row" @click="pickTarget(c.convId)">
            <Avatar :name="c.peerName || c.peerId" size="sm" />
            <span class="pick-name">{{ c.peerName || c.peerId }}</span>
            <span class="pick-check" :class="{ on: forwardIds.includes(c.convId) }">{{ forwardIds.includes(c.convId) ? '✓' : '' }}</span>
          </button>
        </div>
        <div class="field" style="margin-top:12px">
          <label>{{ t('forwardNote') }}</label>
          <input v-model="forwardNote" :placeholder="t('forwardNote')">
        </div>
      </div>
      <div class="mf">
        <button class="btn-ghost" @click="showForward = false">{{ t('cancel') }}</button>
        <button class="btn-primary" @click="doForward">{{ t('sendTo') }} ({{ forwardIds.length }})</button>
      </div>
    </div>
  </div>

  <!-- 搜索聊天记录 -->
  <div v-if="showSearch" class="mask" @click.self="showSearch = false">
    <div class="modal">
      <div class="mh">{{ t('searchInChat') }}</div>
      <div class="mb">
        <div class="h5-search" style="margin-bottom:10px"><LineIcon name="search" :size="16" /><input v-model="searchKeyword" :placeholder="t('searchInChat')" @keyup.enter="doSearch"></div>
        <div v-if="searching" class="loading">搜索中…</div>
        <button v-for="r in searchResults" :key="r.msgId" class="search-result" @click="gotoResult(r)">
          <div class="sr-top"><span>{{ r.fromName || r.fromUid }}</span><span class="sr-type">{{ r.typeLabel }}</span></div>
          <div class="sr-txt">{{ r.content }}</div>
          <div class="sr-time">{{ fmtTime(r.createdAt) }}</div>
        </button>
        <div v-if="!searching && searchKeyword && !searchResults.length" class="loading">{{ t('noResult') }}</div>
      </div>
      <div class="mf">
        <button class="btn-ghost" @click="showSearch = false">{{ t('cancel') }}</button>
        <button class="btn-primary" @click="doSearch">{{ t('search') }}</button>
      </div>
    </div>
  </div>

  <!-- 会话设置 -->
  <div v-if="showSettings" class="mask sheet-mask" @click.self="showSettings = false">
    <div class="sheet">
      <div class="sheet-title">{{ t('sessionSettings') }} · {{ name }}</div>
      <button class="sheet-item row" @click="toggleConvPin"><span>{{ conv.pinned ? t('unpin') : t('pin') }}</span><span>{{ conv.pinned ? '✓' : '' }}</span></button>
      <button class="sheet-item row" @click="toggleConvMute"><span>{{ conv.muted ? t('unmute') : t('mute') }}</span><span>{{ conv.muted ? '✓' : '' }}</span></button>
      <button class="sheet-item row" @click="markConvRead"><span>{{ t('markRead') }}</span></button>
      <button class="sheet-item row" @click="openSearchFromSettings"><span>{{ t('searchInChat') }}</span></button>
      <template v-if="isGroup">
        <button class="sheet-item row" @click="openMembers"><span>{{ t('members') }}</span><span>{{ members.length || conv.memberCount || '' }}</span></button>
        <button class="sheet-item row" @click="openInvite"><span>{{ t('invite') }}</span></button>
        <button class="sheet-item danger" @click="leaveGroup">{{ t('leaveGroup') }}</button>
      </template>
      <button class="sheet-item cancel" @click="showSettings = false">{{ t('cancel') }}</button>
    </div>
  </div>

  <!-- 群成员 -->
  <div v-if="showMembers" class="mask" @click.self="showMembers = false">
    <div class="modal">
      <div class="mh">{{ t('members') }} ({{ members.length }})</div>
      <div class="mb">
        <div v-for="m in members" :key="m.uid" class="member-row">
          <Avatar :name="m.nickname || m.uid" size="sm" />
          <span>{{ m.nickname || m.uid }}</span>
          <span v-if="m.role === 'owner'" class="tag-pin">群主</span>
        </div>
      </div>
      <div class="mf"><button class="btn-ghost" @click="showMembers = false">{{ t('confirm') }}</button></div>
    </div>
  </div>

  <!-- 邀请成员 -->
  <div v-if="showInvite" class="mask" @click.self="showInvite = false">
    <div class="modal">
      <div class="mh">{{ t('invite') }}</div>
      <div class="mb">
        <button v-for="f in friendList" :key="f.uid" class="pick-row" @click="pickTarget(f.uid)">
          <Avatar :name="f.nickname" size="sm" />
          <span class="pick-name">{{ f.nickname }}</span>
          <span class="pick-check" :class="{ on: inviteIds.includes(f.uid) }">{{ inviteIds.includes(f.uid) ? '✓' : '' }}</span>
        </button>
        <div v-if="!friendList.length" class="loading">{{ t('noFriend') }}</div>
      </div>
      <div class="mf">
        <button class="btn-ghost" @click="showInvite = false">{{ t('cancel') }}</button>
        <button class="btn-primary" @click="doInvite">{{ t('confirm') }}</button>
      </div>
    </div>
  </div>

  <ModalBox ref="modal" />
</template>
