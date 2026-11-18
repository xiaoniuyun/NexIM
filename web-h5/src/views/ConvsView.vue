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
const showNew = ref(false)
const friendList = ref([])
const friendKeyword = ref('')

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  let list = chat.convs || []
  if (q) list = list.filter(c => (c.peerName || c.peerId || '').toLowerCase().includes(q) || (c.lastMsg || '').includes(q))
  return [...list].sort((a, b) => (b.pinned || 0) - (a.pinned || 0) || (b.lastMsgAt || 0) - (a.lastMsgAt || 0))
})

let convTimer = null
onMounted(() => {
  chat.loadConversations()
  convTimer = setInterval(() => { if (!document.hidden) chat.loadConversations() }, 5000)
})
onUnmounted(() => { if (convTimer) clearInterval(convTimer) })

function openConv(c) {
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
    <div v-if="filtered.length">
      <div v-if="filtered.some(c => c.pinned)" class="sec-title">置顶会话</div>
      <button v-for="c in filtered.filter(c => c.pinned)" :key="c.convId" class="conv-item" @click="openConv(c)">
        <Avatar :name="c.peerName || c.peerId" />
        <div class="ci-main">
          <div class="ci-top">
            <span class="ci-name">{{ c.peerName || c.peerId }}</span>
            <span v-if="c.pinned" class="tag-pin">置顶</span>
            <span class="ci-time">{{ fmtTime(c.lastMsgAt) }}</span>
          </div>
          <div class="ci-last">{{ c.lastMsg || '暂无消息' }}</div>
        </div>
        <span v-if="c.unread" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
      </button>
      <div v-if="filtered.some(c => !c.pinned)" class="sec-title">全部会话</div>
      <button v-for="c in filtered.filter(c => !c.pinned)" :key="c.convId" class="conv-item" @click="openConv(c)">
        <Avatar :name="c.peerName || c.peerId" />
        <div class="ci-main">
          <div class="ci-top">
            <span class="ci-name">{{ c.peerName || c.peerId }}</span>
            <span class="ci-time">{{ fmtTime(c.lastMsgAt) }}</span>
          </div>
          <div class="ci-last">{{ c.lastMsg || '暂无消息' }}</div>
        </div>
        <span v-if="c.unread" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
      </button>
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
</template>
