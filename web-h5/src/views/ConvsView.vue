<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'
import { API } from '../api'
import { t } from '../i18n'
import Avatar from '../components/Avatar.vue'
import LineIcon from '../components/LineIcon.vue'
import { toast } from '../toast'

const router = useRouter()
const chat = useChatStore()
const auth = useAuthStore()
const keyword = ref('')
const showNew = ref(false)
const friendList = ref([])
const friendKeyword = ref('')
const menuConv = ref(null)

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  let list = chat.convs || []
  if (q) list = list.filter(c => (c.peerName || c.peerId || '').toLowerCase().includes(q) || (c.lastMsg || '').toLowerCase().includes(q))
  return [...list].sort((a, b) => (b.pinned || 0) - (a.pinned || 0) || (b.lastMsgAt || 0) - (a.lastMsgAt || 0))
})

let convTimer = null
onMounted(() => {
  chat.loadConversations()
  convTimer = setInterval(() => { if (!document.hidden) chat.loadConversations() }, 5000)
})
onUnmounted(() => { if (convTimer) clearInterval(convTimer) })

let pressTimer = null
let pressFired = false
function pressStart(c) {
  pressFired = false
  clearTimeout(pressTimer)
  pressTimer = setTimeout(() => { pressFired = true; menuConv.value = c }, 450)
}
function pressEnd() { clearTimeout(pressTimer) }
function openMenu(c) { clearTimeout(pressTimer); pressFired = false; menuConv.value = c }

function openConv(c) {
  if (pressFired) { pressFired = false; return }
  router.push('/chat/' + encodeURIComponent(c.convId))
}
async function openNew() {
  showNew.value = true
  try { const r = await API.friends(); friendList.value = r.data || [] } catch (e) { toast(e.message, 'err') }
}
function pickFriend(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  showNew.value = false
  router.push('/chat/' + encodeURIComponent(convId))
}
function fmtTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  const pad = n => String(n).padStart(2, '0')
  const y = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
  if (d.toDateString() === now.toDateString()) return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (d.toDateString() === y.toDateString()) return '昨天'
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}/${d.getDate()}`
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}
const friendFiltered = computed(() => friendList.value.filter(u => !friendKeyword.value || (u.nickname || '').includes(friendKeyword.value)))

async function togglePin(c) {
  const next = c.pinned ? 0 : 1
  try {
    await API.convSettings(c.convId, { pinned: next })
    c.pinned = next
    menuConv.value = null
    await chat.loadConversations()
    toast(t('pinSuccess'))
  } catch (e) { toast(e.message, 'err') }
}
async function toggleMute(c) {
  const next = c.muted ? 0 : 1
  try {
    await API.convSettings(c.convId, { muted: next })
    c.muted = next
    menuConv.value = null
    await chat.loadConversations()
    toast(t('pinSuccess'))
  } catch (e) { toast(e.message, 'err') }
}
async function markRead(c) {
  try {
    await API.markRead(c.convId)
    c.unread = 0
    menuConv.value = null
    await chat.loadConversations()
  } catch (e) { toast(e.message, 'err') }
}
async function removeConv(c) {
  if (!window.confirm(`删除与「${c.peerName || c.peerId}」的会话？聊天记录将不再显示。`)) return
  try {
    await API.deleteConv(c.convId)
    chat.convs = chat.convs.filter(x => x.convId !== c.convId)
    menuConv.value = null
    toast(t('pinSuccess'))
  } catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div class="cs-page">
    <div class="cs-head">
      <button class="cs-user" @click="router.push('/mine')">
        <Avatar :name="auth.user?.nickname || 'U'" size="lg" />
        <div class="cs-id">
          <div class="cs-name">{{ auth.user?.nickname || t('msg') }}</div>
          <div class="cs-status" :class="{ off: !chat.connected }">
            <i class="dot"></i>{{ chat.connected ? '在线 · 4G' : '连接中…' }}
          </div>
        </div>
      </button>
      <button class="cs-plus" :title="t('addFriend')" @click="openNew">＋</button>
    </div>
    <div class="cs-search">
      <LineIcon name="search" :size="19" />
      <input v-model="keyword" :placeholder="t('search')">
    </div>
    <div class="cs-sheet">
      <div v-if="filtered.length" class="cs-list">
        <button v-for="c in filtered" :key="c.convId" class="cs-row"
          @click="openConv(c)" @contextmenu.prevent="openMenu(c)"
          @touchstart="pressStart(c)" @touchend="pressEnd" @touchmove="pressEnd">
          <span class="cs-ava">
            <Avatar :name="c.peerName || c.peerId" />
            <i v-if="c.unread" class="cs-badge">{{ c.unread > 99 ? '99+' : c.unread }}</i>
          </span>
          <div class="cs-main">
            <div class="cs-title">
              <span class="t">{{ c.peerName || c.peerId }}</span>
              <span v-if="c.pinned" class="cs-tag">{{ t('pin') }}</span>
            </div>
            <div class="cs-sub">{{ c.muted ? `[${t('muted')}] ` : '' }}{{ c.lastMsg || '暂无消息' }}</div>
          </div>
          <div class="cs-right">
            <span class="cs-time">{{ fmtTime(c.lastMsgAt) }}</span>
          </div>
        </button>
      </div>
      <div v-else class="cs-empty">{{ t('noConv') }}</div>
    </div>
  </div>

  <div v-if="showNew" class="mask" @click.self="showNew = false">
    <div class="modal">
      <div class="mh">{{ t('createGroup') }} / {{ t('addFriend') }}</div>
      <div class="mb">
        <div class="h5-search"><LineIcon name="search" :size="16" /><input v-model="friendKeyword" :placeholder="t('inputSearchUser')"></div>
        <button v-for="u in friendFiltered" :key="u.uid" class="qentry" @click="pickFriend(u)">
          <Avatar :name="u.nickname" size="sm" />
          <div><div class="qe-t">{{ u.nickname }}</div><div class="qe-s">@{{ u.username }} · {{ u.external ? '企业外部用户' : (u.deptName || '') }}</div></div>
        </button>
        <div v-if="!friendFiltered.length" class="loading">{{ t('noFriend') }}</div>
      </div>
      <div class="mf"><button class="btn-ghost" @click="showNew = false">{{ t('cancel') }}</button></div>
    </div>
  </div>

  <div v-if="menuConv" class="mask sheet-mask" @click.self="menuConv = null">
    <div class="sheet">
      <div class="sheet-title">{{ menuConv.peerName || menuConv.peerId }}</div>
      <button class="sheet-item" @click="togglePin(menuConv)">{{ menuConv.pinned ? t('unpin') : t('pin') }}</button>
      <button class="sheet-item" @click="toggleMute(menuConv)">{{ menuConv.muted ? t('unmute') : t('mute') }}</button>
      <button class="sheet-item" @click="markRead(menuConv)">{{ t('markRead') }}</button>
      <button class="sheet-item danger" @click="removeConv(menuConv)">{{ t('del') }}</button>
      <button class="sheet-item cancel" @click="menuConv = null">{{ t('cancel') }}</button>
    </div>
  </div>
</template>
