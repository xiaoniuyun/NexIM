<script setup>
import { computed, ref, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
import { useCallStore } from '@h5/stores/call'
import { API, fileUrl } from '@h5/api'
import { toast } from '@h5/toast'
import Modal from './Modal.vue'
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
watch(() => chat.currentConv, () => { showSearch.value = false; searchResults.value = []; scrollBottom() })

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

// ===== 语音消息 =====
async function toggleRecord() {
  if (recording.value) { stopRecord(); return }
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia || typeof MediaRecorder === 'undefined') {
    return toast('当前环境不支持录音', 'err')
  }
  try {
    recStream = await navigator.mediaDevices.getUserMedia({ audio: true })
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
  if (recTimer) { clearInterval(recTimer); recTimer = null }
  recording.value = false
  try { if (recorder && recorder.state !== 'inactive') recorder.stop() } catch (_) {}
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

onMounted(() => {
  chat.setViewing(true)
  window.addEventListener('click', closePopovers)
  scrollBottom()
})
onBeforeUnmount(() => {
  chat.setViewing(false)
  stopRecord()
  releaseStream()
  window.removeEventListener('click', closePopovers)
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
          <Avatar :name="title" :src="conv?.peerAvatar" size="md" />
          <div>
            <div class="ct-name">
              {{ title }}
              <span v-if="isGroup" class="tag">群聊</span>
              <span v-if="muted.has(conv.convId)" class="tag off">免打扰</span>
            </div>
            <div class="ct-sub">{{ chat.connected ? '已连接 · 消息实时同步' : '连接中断，正在重连…' }}</div>
          </div>
        </div>
        <div class="chat-tools">
          <button class="icon-btn" title="搜索聊天记录" @click="toggleSearch">⌕</button>
          <button v-if="canCall" class="icon-btn" title="语音通话" @click="startCall('audio')">☎</button>
          <button v-if="canCall" class="icon-btn" title="视频通话" @click="startCall('video')">▣</button>
          <button v-if="isGroup" class="icon-btn" title="群成员" @click="openMembers">☰</button>
          <button class="icon-btn" title="更多" @click="showMore = !showMore">⋯</button>
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
          :class="{ mine: m.mine, system: m.type === 'system' }" :data-mid="m.msgId"
        >
          <template v-if="m.type === 'system'">
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
                <template v-else-if="m.type === 'image'">
                  <img class="msg-img" :src="fileUrl(m.extra?.fileId)" alt="图片" @click="openFile(m)">
                </template>
                <template v-else-if="m.type === 'video'">
                  <video class="msg-video" :src="fileUrl(m.extra?.fileId)" controls></video>
                </template>
                <template v-else-if="m.type === 'voice'">
                  <audio :src="fileUrl(m.extra?.fileId)" controls style="max-width:240px"></audio>
                </template>
                <template v-else-if="m.type === 'file'">
                  <div class="file-msg" @click="openFile(m)">
                    <span class="file-ico">📄</span>
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
          <button class="icon-btn" title="表情" @click="showEmoji = !showEmoji; showAttach = false">☺</button>
          <button class="icon-btn" title="附件" @click="showAttach = !showAttach; showEmoji = false">📎</button>
          <button class="icon-btn" :class="{ recording }" :title="recording ? `结束录音 ${recSeconds}″` : '语音消息'" @click="toggleRecord">{{ recording ? '⏺' : '🎙' }}</button>
          <button class="icon-btn" :disabled="!desk" title="截图并发送（Ctrl+Shift+A）" @click="sendScreenshot(chat)">✂</button>
          <span class="sp"></span>
          <span class="muted">{{ chat.connected ? 'Enter 发送 · Shift+Enter 换行' : '离线：消息将在重连后补发' }}</span>
        </div>

        <div v-if="showEmoji" class="emoji-panel">
          <button v-for="e in emojis" :key="e" @click="insertEmoji(e)">{{ e }}</button>
        </div>
        <div v-if="showAttach" class="attach-menu">
          <button @click="pick('image')">图片</button>
          <button @click="pick('video')">视频</button>
          <button @click="pick('file')">文件</button>
          <button @click="showAttach = false; toggleRecord()">语音消息</button>
          <button @click="toggleMention">@ 提醒</button>
        </div>
        <div v-if="showMention" class="mention-panel">
          <div v-for="u in mentionList" :key="u.uid" class="mention-row" @click="mention(u)">
            {{ u.nickname || u.username || u.uid }}
          </div>
          <div v-if="!mentionList.length" class="empty-tip">暂无可提醒成员</div>
        </div>

        <textarea
          v-model="text" class="composer-input" rows="3"
          placeholder="输入消息…（Enter 发送，Shift+Enter 换行，可直接拖入文件）"
          @keydown="onKeydown"
        ></textarea>
        <div class="composer-foot">
          <span class="muted">支持 100MB 以内图片 / 视频 / 文件</span>
          <button class="btn primary" :disabled="!text.trim()" @click="send">发送</button>
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
