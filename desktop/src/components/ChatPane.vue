<script setup>
import { computed, ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import Avatar from '@/shared/components/Avatar.vue'
import LineIcon from '@/shared/components/LineIcon.vue'
import { useAuthStore } from '@/shared/stores/auth'
import { useChatStore } from '@/shared/stores/chat'
import { useCallStore } from '@/shared/stores/call'
import { API, fileUrl } from '@/shared/api'
import { toast } from '@/shared/toast'
import Modal from './Modal.vue'
import Icon from './Icon.vue'
import ForwardDialog from './ForwardDialog.vue'
import { desk, bytes, fullTime, uploadFiles, sendScreenshot, togglePin, toggleMute, pinnedSet, mutedSet } from '../composables/useDesktop'

const auth = useAuthStore()
const chat = useChatStore()
const call = useCallStore()

const text = ref('')
const listEl = ref(null)
const showEmoji = ref(false)
const showAttach = ref(false)
const showMore = ref(false)
const showMembers = ref(false)
const showInvite = ref(false)
const showSearch = ref(false)
const showMention = ref(false)
const mentionList = ref([])
const members = ref([])
const groupInfo = ref(null)
const friends = ref([])
const inviteIds = ref([])
const searchKw = ref('')
const searchResults = ref([])
const forwardMsg = ref(null)
const dragOver = ref(false)
const fileInput = ref(null)
const pickKind = ref('file')
const pinned = ref(pinnedSet())
const muted = ref(mutedSet())
const emojis = ['😀', '😄', '😂', '😊', '😍', '🤔', '😎', '👍', '👏', '🙏', '🎉', '❤️', '🔥', '✅', '💡', '🚀', '📌', '😅', '🤝', '😴', '🙌']

// 语音录制
const recording = ref(false)
const recSeconds = ref(0)
let recorder = null
let recChunks = []
let recStream = null
let recTimer = null
let recStartedAt = 0

const conv = computed(() => chat.convs.find(c => c.convId === chat.currentConv) || null)
const isGroup = computed(() => conv.value?.convType === 2)
const title = computed(() => conv.value?.peerName || '选择一个会话')
const msgs = computed(() => chat.messages)
const canCall = computed(() => conv.value && conv.value.convType === 1)
const isOwner = computed(() => isGroup.value && groupInfo.value?.ownerUid === auth.user?.uid)
const lastMineId = computed(() => {
  for (let i = msgs.value.length - 1; i >= 0; i--) if (msgs.value[i].mine) return msgs.value[i].msgId
  return ''
})
const peerRead = computed(() => !!(conv.value && chat.readMap[conv.value.convId]))

function scrollBottom(smooth = false) {
  nextTick(() => {
    const el = listEl.value
    if (el) el.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'auto' })
  })
}
watch(() => msgs.value.length, () => scrollBottom())
watch(() => chat.currentConv, () => { exitVoiceMode(); showSearch.value = false; searchResults.value = []; scrollBottom() })

function send() {
  const v = text.value.trim()
  if (!v) return
  if (!chat.currentConv) return toast('请先选择会话', 'err')
  chat.sendText(v)
  text.value = ''
  showEmoji.value = false
  scrollBottom(true)
}
function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() }
}

function pick(kind) {
  pickKind.value = kind
  showAttach.value = false
  if (!fileInput.value) return
  fileInput.value.accept = kind === 'image' ? 'image/*' : kind === 'video' ? 'video/*' : ''
  fileInput.value.click()
}
async function onPick(e) {
  const files = [...(e.target.files || [])]
  e.target.value = ''
  if (!files.length) return
  const ok = await uploadFiles(files, chat.currentConv, chat)
  if (ok) { toast(`已发送 ${ok} 个文件`); scrollBottom(true) }
}
async function onDrop(e) {
  dragOver.value = false
  const files = [...(e.dataTransfer?.files || [])]
  if (!files.length) return
  const ok = await uploadFiles(files, chat.currentConv, chat)
  if (ok) { toast(`已发送 ${ok} 个文件`); scrollBottom(true) }
}

function openFile(m) {
  if (!m.extra?.fileId) return
  window.open(fileUrl(m.extra.fileId), '_blank')
}

const IMG_EXT_RE = /\.(png|jpe?g|gif|webp|bmp|heic|heif)(\?.*)?$/i
const VIDEO_EXT_RE = /\.(mp4|mov|m4v|webm|avi|mkv)(\?.*)?$/i
// 早期移动端上传缺失类型被记为 file，按扩展名回退为图片显示
function isImgMsg(m) {
  if (m.type === 'image') return true
  if (m.type !== 'file') return false
  return IMG_EXT_RE.test(m.extra?.name || m.content || '')
}
// 被记为 file 的视频同样按视频渲染，保证各端都能直接播放
function isVideoMsg(m) {
  if (m.type === 'video') return true
  if (m.type !== 'file') return false
  return VIDEO_EXT_RE.test(m.extra?.name || m.content || '')
}
const videoRates = ref({})
function videoRate(m) { return videoRates.value[m.msgId] || 1 }
function cycleVideoRate(e, m) {
  const rates = [0.5, 1, 1.5, 2]
  const next = rates[(rates.indexOf(videoRate(m)) + 1) % rates.length]
  videoRates.value = { ...videoRates.value, [m.msgId]: next }
  const el = e.target.closest('.msg-video-wrap')?.querySelector('video')
  if (el) el.playbackRate = next
}
function videoFullscreen(e) {
  const el = e.target.closest('.msg-video-wrap')?.querySelector('video')
  if (!el) return
  if (document.fullscreenElement) document.exitFullscreen?.()
  else el.requestFullscreen?.()
}
// 无封面的视频：元数据就绪后跳到 1/4 处渲染一帧，避免片头黑帧
function onVideoMeta(e) {
  const v = e.target
  if (!v || v.dataset.seeked) return
  v.dataset.seeked = '1'
  if (v.poster) return
  const d = v.duration
  if (!isFinite(d) || d <= 4) return
  try { v.currentTime = d / 4 } catch (_) {}
}
function fmtDuration(ms) {
  const total = Math.max(0, Math.round((ms || 0) / 1000))
  const h = Math.floor(total / 3600)
  const m = String(Math.floor((total % 3600) / 60)).padStart(2, '0')
  const s = String(total % 60).padStart(2, '0')
  return h > 0 ? `${h}:${m}:${s}` : `${m}:${s}`
}

// ===== 语音消息（内联按住说话） =====
const voiceMode = ref(false)
const willCancel = ref(false)
let cancelRecording = false
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
    return toast('当前环境不支持录音', 'err')
  }
  try {
    cancelRecording = false
    startPending = true
    recStream = await navigator.mediaDevices.getUserMedia({ audio: true })
    // 授权/设备就绪期间用户已松手或退出语音模式：放弃本次录音
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
  if (blob.size < 800) return toast('录音时间太短', 'err')
  const ext = mime.indexOf('mp4') >= 0 ? 'm4a' : 'webm'
  const file = new File([blob], 'voice_' + Date.now() + '.' + ext, { type: mime })
  if (!chat.currentConv) return toast('请先选择会话', 'err')
  const fd = new FormData()
  fd.append('file', file)
  fd.append('convId', chat.currentConv)
  try {
    const r = await API.fileUpload(fd)
    const rec = r.data || {}
    chat.sendMedia('voice', rec.name || file.name, {
      fileId: rec.fileId, name: rec.name || file.name,
      size: rec.size || blob.size, mime, duration: secs
    })
    toast('语音已发送')
  } catch (err) {
    toast(err.message || '语音发送失败', 'err')
  }
}
function insertEmoji(e) { text.value += e }

async function copyMsg(m) {
  try { await navigator.clipboard.writeText(m.content || ''); toast('已复制') } catch (_) { toast('复制失败', 'err') }
}
async function favMsg(m) {
  try { await API.favorite(m.msgId); toast('已收藏') } catch (e) { toast(e.message, 'err') }
}
async function recallMsg(m) {
  if (!confirm('确认撤回该消息？')) return
  try {
    await API.revoke(m.msgId)
    m.status = 2
    chat.loadConversations()
    toast('已撤回')
  } catch (e) { toast(e.message, 'err') }
}
function delMsg(m) {
  chat.removeLocal(m.msgId)
  toast('已在本端删除')
}

async function toggleSearch() {
  showSearch.value = !showSearch.value
  searchResults.value = []
  searchKw.value = ''
  if (showSearch.value) nextTick(() => document.getElementById('chat-search-input')?.focus())
}
async function doSearch() {
  const kw = searchKw.value.trim()
  if (!kw) return
  try {
    const r = await API.searchMessages(kw, chat.currentConv)
    searchResults.value = r.data || []
  } catch (e) { toast(e.message, 'err') }
}
function jumpTo(msgId) {
  showSearch.value = false
  nextTick(() => {
    const el = listEl.value?.querySelector(`[data-mid="${msgId}"]`)
    if (!el) return toast('该消息不在当前加载范围内', 'err')
    el.scrollIntoView({ block: 'center', behavior: 'smooth' })
    el.classList.add('flash')
    setTimeout(() => el.classList.remove('flash'), 1600)
  })
}

async function openMembers() {
  if (!isGroup.value) return
  try {
    const [d, r] = await Promise.all([API.groupDetail(conv.value.peerId), API.groupMembers(conv.value.peerId)])
    groupInfo.value = d.data || null
    members.value = r.data || []
    showMembers.value = true
  } catch (e) { toast(e.message, 'err') }
}
async function kickGroupMember(m) {
  if (!confirm(`将「${m.nickname || m.uid}」移出群聊？`)) return
  try {
    await API.groupLeave(conv.value.peerId, m.uid)
    const r = await API.groupMembers(conv.value.peerId)
    members.value = r.data || []
    const d = await API.groupDetail(conv.value.peerId)
    groupInfo.value = d.data || groupInfo.value
    toast('已移出成员')
  } catch (e) { toast(e.message, 'err') }
}
async function dissolveCurrentGroup() {
  if (!confirm('解散后群成员将无法继续在群内发言，确认解散？')) return
  try {
    await API.groupDissolve(conv.value.peerId)
    showMembers.value = false
    chat.currentConv = ''
    await chat.loadConversations()
    toast('群聊已解散')
  } catch (e) { toast(e.message, 'err') }
}
async function openInvite() {
  try {
    const [mr, fr] = await Promise.all([API.groupMembers(conv.value.peerId), API.friends()])
    members.value = mr.data || []
    const ids = new Set(members.value.map(m => m.uid))
    friends.value = (fr.data || []).filter(f => !ids.has(f.uid))
    inviteIds.value = []
    showInvite.value = true
  } catch (e) { toast(e.message, 'err') }
}
async function doInvite() {
  if (!inviteIds.value.length) return toast('请选择成员', 'err')
  try {
    await API.groupInvite(conv.value.peerId, inviteIds.value)
    toast(`已邀请 ${inviteIds.value.length} 位成员`)
    showInvite.value = false
    chat.loadConversations()
  } catch (e) { toast(e.message, 'err') }
}
async function doPin() {
  togglePin(conv.value.convId); pinned.value = pinnedSet()
  showMore.value = false
  toast(pinned.value.has(conv.value.convId) ? '已置顶' : '已取消置顶')
}
function doMute() {
  toggleMute(conv.value.convId); muted.value = mutedSet()
  showMore.value = false
  toast(muted.value.has(conv.value.convId) ? '已开启免打扰' : '已关闭免打扰')
}
async function doRead() {
  try { await API.markRead(conv.value.convId); showMore.value = false; chat.loadConversations() } catch (e) { toast(e.message, 'err') }
}
async function leaveGroup() {
  if (!confirm('确认退出该群聊？')) return
  try {
    await API.groupLeave(conv.value.peerId, auth.user.uid)
    toast('已退出群聊')
    chat.currentConv = ''
    chat.loadConversations()
  } catch (e) { toast(e.message, 'err') }
}
function startCall(type) {
  if (!canCall.value) return toast('群聊暂不支持发起通话', 'err')
  call.startCall(conv.value, type)
}
async function toggleMention() {
  showAttach.value = false
  showEmoji.value = false
  showMention.value = !showMention.value
  if (!showMention.value) return
  if (isGroup.value) {
    try { const r = await API.groupMembers(conv.value.peerId); mentionList.value = r.data || [] } catch (_) { mentionList.value = [] }
  } else if (conv.value) {
    mentionList.value = [{ uid: conv.value.peerId, nickname: conv.value.peerName }]
  }
}
function mention(u) {
  text.value += '@' + (u.nickname || u.username || u.uid) + ' '
  showMention.value = false
}

function closePopovers(e) {
  if (!e.target.closest?.('.chat-tools')) showMore.value = false
}

function onVoiceKeyDown(e) {
  if (!voiceMode.value) return
  if (e.key === 'Escape') { e.preventDefault(); exitVoiceMode(); return }
  if (e.code === 'Space' && !recording.value) {
    const el = document.activeElement
    if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) return
    e.preventDefault()
    startVoiceHold()
  }
}
function onVoiceKeyUp(e) {
  if (e.code === 'Space' && (recording.value || startPending || willCancel.value)) {
    e.preventDefault()
    endVoiceHold()
  }
}

onMounted(() => {
  chat.setViewing(true)
  window.addEventListener('click', closePopovers)
  window.addEventListener('keydown', onVoiceKeyDown)
  window.addEventListener('keyup', onVoiceKeyUp)
  scrollBottom()
})
onBeforeUnmount(() => {
  chat.setViewing(false)
  stopRecord()
  releaseStream()
  window.removeEventListener('click', closePopovers)
  window.removeEventListener('keydown', onVoiceKeyDown)
  window.removeEventListener('keyup', onVoiceKeyUp)
})
</script>

<template>
  <div class="chat-pane" @dragover.prevent="dragOver = true" @drop.prevent="onDrop">
    <div v-if="!conv" class="chat-empty">
      <div class="empty-logo">N</div>
      <p>从左侧选择一个会话开始沟通</p>
      <p class="muted">Ctrl+K 可全局搜索会话与联系人</p>
    </div>

    <template v-else>
      <header class="chat-head">
        <div class="chat-title">
          <div class="ct-name">
            {{ title }}
            <i class="online-dot" :class="{ off: !chat.connected }"></i>
            <span v-if="isGroup" class="tag">群聊</span>
            <span v-if="muted.has(conv.convId)" class="tag off">免打扰</span>
          </div>
        </div>
        <div class="chat-tools">
          <button v-if="canCall" class="icon-btn" title="语音通话" @click="startCall('audio')"><Icon name="phone" :size="19" /></button>
          <button v-if="canCall" class="icon-btn" title="视频通话" @click="startCall('video')"><Icon name="video" :size="20" /></button>
          <button v-if="isGroup" class="icon-btn" title="群成员" @click="openMembers"><Icon name="users" :size="20" /></button>
          <button class="icon-btn" title="搜索聊天记录" @click="toggleSearch"><Icon name="search" :size="19" /></button>
          <button class="icon-btn" title="更多" @click="showMore = !showMore"><Icon name="more" :size="20" /></button>
          <div v-if="showMore" class="popover" @click.stop>
            <button @click="doPin">{{ pinned.has(conv.convId) ? '取消置顶' : '置顶会话' }}</button>
            <button @click="doMute">{{ muted.has(conv.convId) ? '关闭免打扰' : '消息免打扰' }}</button>
            <button @click="doRead">标记为已读</button>
            <button v-if="isGroup" @click="openInvite(); showMore = false">邀请成员</button>
            <button v-if="isGroup" class="danger" @click="leaveGroup(); showMore = false">退出群聊</button>
          </div>
        </div>
      </header>

      <div v-if="showSearch" class="search-bar">
        <input id="chat-search-input" v-model="searchKw" placeholder="搜索本会话消息，回车确认" @keyup.enter="doSearch">
        <button class="btn sm" @click="doSearch">搜索</button>
        <button class="btn sm ghost" @click="showSearch = false">关闭</button>
        <div v-if="searchResults.length" class="search-results">
          <div v-for="m in searchResults.slice(0, 20)" :key="m.msgId" class="search-row" @click="jumpTo(m.msgId)">
            <span class="sr-name">{{ m.fromName || m.fromUid }}</span>
            <span class="sr-text">{{ m.content }}</span>
            <span class="sr-time">{{ fullTime(m.createdAt) }}</span>
          </div>
        </div>
        <div v-else-if="searchKw" class="search-results"><div class="empty-tip">未找到匹配消息</div></div>
      </div>

      <div ref="listEl" class="msg-list scroll">
        <div
          v-for="m in msgs" :key="m.msgId" class="msg-row"
          :class="{ mine: m.mine, system: m.type === 'system' || m.type === 'call' }" :data-mid="m.msgId"
        >
          <template v-if="m.type === 'call'">
            <div class="sys-msg">
              <Icon :name="m.extra?.callType === 'video' ? 'video' : 'phone'" :size="14" />
              {{ m.content }}
            </div>
          </template>
          <template v-else-if="m.type === 'system'">
            <div class="sys-msg">{{ m.content }}</div>
          </template>
          <template v-else>
            <Avatar v-if="!m.mine" :name="m.fromName || m.fromUid" :src="isGroup ? '' : conv.peerAvatar" size="sm" />
            <div class="msg-body">
              <div v-if="!m.mine && isGroup" class="msg-from">{{ m.fromName || m.fromUid }}</div>
              <div class="bubble" :class="{ revoked: m.status === 2 }">
                <template v-if="m.status === 2">
                  <span class="muted">该消息已撤回</span>
                </template>
                <template v-else-if="isImgMsg(m)">
                  <img class="msg-img" :src="fileUrl(m.extra?.fileId)" alt="图片" @click="openFile(m)">
                </template>
                <template v-else-if="isVideoMsg(m)">
                  <div class="msg-video-wrap">
                    <video
                      class="msg-video" :src="fileUrl(m.extra?.fileId) + (m.extra?.coverFileId ? '' : '#t=0.1')"
                      :poster="m.extra?.coverFileId ? fileUrl(m.extra.coverFileId) : undefined"
                      controls preload="auto" @loadedmetadata="onVideoMeta($event)"
                    ></video>
                    <div class="video-bar">
                      <span v-if="m.extra?.durationMs" class="video-dur">{{ fmtDuration(m.extra.durationMs) }}</span>
                      <span class="video-ops">
                        <button class="video-op" @click="cycleVideoRate($event, m)">{{ videoRate(m) }}x</button>
                        <button class="video-op" @click="videoFullscreen($event)">全屏</button>
                      </span>
                    </div>
                  </div>
                </template>
                <template v-else-if="m.type === 'voice'">
                  <audio :src="fileUrl(m.extra?.fileId)" controls style="max-width:240px"></audio>
                </template>
                <template v-else-if="m.type === 'file'">
                  <div class="file-msg" @click="openFile(m)">
                    <span class="file-ico"><LineIcon name="file" :size="18" /></span>
                    <div>
                      <div class="file-name">{{ m.extra?.name || m.content || '文件' }}</div>
                      <div class="file-size">{{ bytes(m.extra?.size) }}</div>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="msg-text">{{ m.content }}</div>
                </template>
              </div>
              <div class="msg-meta">
                <span>{{ fullTime(m.createdAt) }}</span>
                <span v-if="m.pending" class="muted">· 发送中</span>
                <span v-if="m.status === 3" class="danger">· 发送失败</span>
                <span v-else-if="m.mine && m.msgId === lastMineId && peerRead && Number(m.createdAt) <= (chat.readMap[conv.convId] || 0)" class="state ok">· 已读</span>
              </div>
              <div class="msg-actions" v-if="m.status !== 2">
                <button @click="copyMsg(m)">复制</button>
                <button @click="forwardMsg = m">转发</button>
                <button @click="favMsg(m)">收藏</button>
                <button v-if="m.mine" @click="recallMsg(m)">撤回</button>
                <button class="danger" @click="delMsg(m)">删除</button>
              </div>
            </div>
            <Avatar v-if="m.mine" :name="auth.user?.nickname || '我'" :src="auth.user?.avatar" size="sm" />
          </template>
        </div>
        <div v-if="!msgs.length" class="empty-tip">暂无消息，发送第一条消息开始聊天</div>
      </div>

      <footer class="composer">
        <div class="composer-tools">
          <button class="tool-btn" title="表情" @click="showEmoji = !showEmoji"><Icon name="smile" :size="22" /></button>
          <button class="tool-btn" :disabled="!desk" title="截图并发送（Ctrl+Shift+A）" @click="sendScreenshot(chat)"><Icon name="scissors" :size="22" /></button>
          <button class="tool-btn" title="发送文件" @click="pick('file')"><Icon name="folder" :size="22" /></button>
          <button class="tool-btn" title="发送图片" @click="pick('image')"><Icon name="image" :size="22" /></button>
          <button class="tool-btn" title="发送视频" @click="pick('video')"><Icon name="video" :size="22" /></button>
          <button class="tool-btn" :class="{ recording, on: voiceMode }" title="语音输入（按住 Space）" @click="toggleVoiceMode"><Icon name="mic" :size="22" /></button>
          <button class="tool-btn" title="语音消息" @click="toggleVoiceMode"><Icon name="voice" :size="22" /></button>
          <button class="tool-btn" title="提醒成员" @click="toggleMention"><Icon name="addUser" :size="22" /></button>
          <span class="sp"></span>
          <button class="tool-btn" title="搜索聊天记录" @click="toggleSearch"><Icon name="clock" :size="22" /></button>
        </div>

        <div v-if="showEmoji" class="emoji-panel">
          <button v-for="e in emojis" :key="e" @click="insertEmoji(e)">{{ e }}</button>
        </div>
        <div v-if="showMention" class="mention-panel">
          <div v-for="u in mentionList" :key="u.uid" class="mention-row" @click="mention(u)">
            {{ u.nickname || u.username || u.uid }}
          </div>
          <div v-if="!mentionList.length" class="empty-tip">暂无可提醒成员</div>
        </div>

        <div v-if="recording" class="voice-hud" :class="{ cancel: willCancel }">
          <div class="wave"><i v-for="n in 7" :key="n"></i></div>
          <span class="voice-timer">{{ recSeconds }}″</span>
          <span class="voice-tip">{{ willCancel ? '松开 取消' : '松开 发送 · 上滑 取消' }}</span>
        </div>

        <div class="composer-body">
          <textarea
            v-if="!voiceMode" v-model="text" class="composer-input" rows="4"
            placeholder="输入消息…"
            @keydown="onKeydown"
          ></textarea>
          <div
            v-else class="voice-bar" :class="{ rec: recording, cancel: willCancel }"
            @pointerdown.prevent="startVoiceHold" @pointermove="onVoiceMove"
            @pointerup.prevent="endVoiceHold" @pointercancel="endVoiceHold"
          >
            <LineIcon name="mic" :size="18" />
            <span>{{ recording ? (willCancel ? '松开 取消' : '松开 发送') : '按住 说话' }}</span>
          </div>
          <div class="composer-foot" v-if="!voiceMode">
            <button class="send-btn" :disabled="!text.trim()" @click="send">
              <span>发送</span><i></i><Icon name="chevronDown" :size="14" />
            </button>
          </div>
        </div>
        <input ref="fileInput" type="file" multiple hidden @change="onPick">
      </footer>

      <div v-if="dragOver" class="drop-mask">松开鼠标发送文件</div>
    </template>

    <Modal v-if="showMembers" :title="groupInfo?.name || `群成员 · ${members.length} 人`" width="420px" @close="showMembers = false">
      <div class="sec-title">群公告</div>
      <div class="group-notice">{{ groupInfo?.notice || '暂无群公告' }}</div>
      <div class="sec-title">成员 · {{ members.length }} 人</div>
      <div class="member-list">
        <div v-for="m in members" :key="m.uid" class="member-row">
          <Avatar :name="m.nickname || m.uid" :src="m.avatar" size="sm" />
          <span style="flex:1">{{ m.nickname || m.uid }}</span>
          <span v-if="m.role === 'owner'" class="tag">群主</span>
          <button v-else-if="isOwner" class="btn sm danger" @click="kickGroupMember(m)">移出</button>
        </div>
      </div>
      <template #footer>
        <button v-if="isOwner" class="btn danger" @click="dissolveCurrentGroup">解散群聊</button>
        <button class="btn" @click="openInvite(); showMembers = false">邀请成员</button>
        <button class="btn" @click="showMembers = false">关闭</button>
      </template>
    </Modal>

    <Modal v-if="showInvite" title="邀请成员" width="420px" @close="showInvite = false">
      <div class="member-list">
        <label v-for="f in friends" :key="f.uid" class="member-row check">
          <input v-model="inviteIds" type="checkbox" :value="f.uid">
          <Avatar :name="f.nickname || f.uid" :src="f.avatar" size="sm" />
          <span>{{ f.nickname || f.uid }}</span>
        </label>
        <div v-if="!friends.length" class="empty-tip">暂无可邀请的好友</div>
      </div>
      <template #footer>
        <button class="btn" @click="showInvite = false">取消</button>
        <button class="btn primary" @click="doInvite">邀请</button>
      </template>
    </Modal>

    <ForwardDialog v-if="forwardMsg" :msg="forwardMsg" @close="forwardMsg = null" />
  </div>
</template>
