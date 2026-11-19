<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'
import { API } from '../api'
import { t } from '../i18n'
import Avatar from '../components/Avatar.vue'
import { toast } from '../toast'

const router = useRouter()
const chat = useChatStore()
const auth = useAuthStore()
const keyword = ref('')
const filter = ref('all')
const showNew = ref(false)
const friendList = ref([])
const friendKeyword = ref('')
const menuConv = ref(null)

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  let list = chat.convs || []
  if (filter.value === 'unread') list = list.filter(c => c.unread > 0)
  if (filter.value === 'group') list = list.filter(c => c.convType === 2)
  if (q) list = list.filter(c => (c.peerName || c.peerId || '').toLowerCase().includes(q) || (c.lastMsg || '').toLowerCase().includes(q))
  return [...list].sort((a, b) => (b.pinned || 0) - (a.pinned || 0) || (b.lastMsgAt || 0) - (a.lastMsgAt || 0))
})
const pinnedList = computed(() => filtered.value.filter(c => c.pinned))
const normalList = computed(() => filtered.value.filter(c => !c.pinned))

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
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
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
async function readAll() {
  try {
    await API.readAll()
    chat.convs.forEach(c => { c.unread = 0 })
    await chat.loadConversations()
    toast(t('allRead'))
  } catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div class="h5-head">
    <h2>{{ t('msg') }}</h2>
    <div class="sp"></div>
    <button class="icon-btn" @click="router.push('/mine')">
      <Avatar :name="auth.user?.nickname || 'U'" size="sm" />
    </button>
    <button class="icon-btn" @click="router.push('/friends')" :title="t('newFriends')">🔔<i class="hd-dot"></i></button>
    <button class="icon-btn" @click="openNew" :title="t('addFriend')">＋</button>
  </div>
  <div class="h5-body">
    <div class="h5-search">🔍<input v-model="keyword" :placeholder="t('search')"></div>
    <div class="chip-row">
      <button class="chip" :class="{on: filter==='all'}" @click="filter='all'">{{ t('all') }}</button>
      <button class="chip" :class="{on: filter==='unread'}" @click="filter='unread'">{{ t('unreadOnly') }}<b v-if="chat.unreadTotal">{{ chat.unreadTotal }}</b></button>
      <button class="chip" :class="{on: filter==='group'}" @click="filter='group'">{{ t('groupOnly') }}</button>
      <button class="chip ghost" @click="readAll">{{ t('readAll') }}</button>
    </div>
    <div v-if="filtered.length">
      <template v-if="pinnedList.length">
        <div class="sec-title">{{ t('pinned') }}</div>
        <button v-for="c in pinnedList" :key="c.convId" class="conv-item"
          @click="openConv(c)" @contextmenu.prevent="openMenu(c)"
          @touchstart="pressStart(c)" @touchend="pressEnd" @touchmove="pressEnd">
          <Avatar :name="c.peerName || c.peerId" />
          <div class="ci-main">
            <div class="ci-top">
              <span class="ci-name">{{ c.peerName || c.peerId }}</span>
              <span class="tag-pin">{{ t('pin') }}</span>
              <span class="ci-time">{{ fmtTime(c.lastMsgAt) }}</span>
            </div>
            <div class="ci-last">{{ c.muted ? `[${t('muted')}] ` : '' }}{{ c.lastMsg || '暂无消息' }}</div>
          </div>
          <span v-if="c.unread" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
        </button>
      </template>
      <template v-if="normalList.length">
        <div class="sec-title">{{ t('all') }}</div>
        <button v-for="c in normalList" :key="c.convId" class="conv-item"
          @click="openConv(c)" @contextmenu.prevent="openMenu(c)"
          @touchstart="pressStart(c)" @touchend="pressEnd" @touchmove="pressEnd">
          <Avatar :name="c.peerName || c.peerId" />
          <div class="ci-main">
            <div class="ci-top">
              <span class="ci-name">{{ c.peerName || c.peerId }}</span>
              <span class="ci-time">{{ fmtTime(c.lastMsgAt) }}</span>
            </div>
            <div class="ci-last">{{ c.muted ? `[${t('muted')}] ` : '' }}{{ c.lastMsg || '暂无消息' }}</div>
          </div>
          <span v-if="c.unread" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
        </button>
      </template>
      <div class="conv-hint">{{ t('longPressTip') }}</div>
    </div>
    <div v-else class="loading">{{ t('noConv') }}</div>
  </div>

  <div v-if="showNew" class="mask" @click.self="showNew = false">
    <div class="modal">
      <div class="mh">{{ t('createGroup') }} / {{ t('addFriend') }}</div>
      <div class="mb">
        <div class="h5-search">🔍<input v-model="friendKeyword" :placeholder="t('inputSearchUser')"></div>
        <button v-for="u in friendFiltered" :key="u.uid" class="qentry" @click="pickFriend(u)">
          <Avatar :name="u.nickname" size="sm" />
          <div><div class="qe-t">{{ u.nickname }}</div><div class="qe-s">@{{ u.username }} · {{ u.deptName || '' }}</div></div>
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
