<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import Avatar from '../components/Avatar.vue'
import { toast } from '../toast'
import { useChatStore } from '../stores/chat'

const router = useRouter()
const chat = useChatStore()
const keyword = ref('')
const friends = ref([])
const depts = ref([])
const applies = ref([])
onMounted(async () => {
  try {
    const [f, d, a] = await Promise.all([API.friends(), API.depts(), API.friendApplies()])
    friends.value = f.data || []
    depts.value = d.data || []
    applies.value = (a.data || []).filter(x => x.status === 0)
  } catch (e) { toast(e.message, 'err') }
})
const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return friends.value
  return friends.value.filter(u =>
    (u.nickname || '').toLowerCase().includes(q) ||
    (u.username || '').toLowerCase().includes(q) ||
    (u.deptName || '').toLowerCase().includes(q)
  )
})
function subOf(u) {
  return u.deptName ? `${u.deptName} · @${u.username}` : `· @${u.username}`
}
function chatWith(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  router.push('/chat/' + encodeURIComponent(convId))
}
</script>
<template>
  <div class="h5-head">
    <h2>{{ t('contacts') }}</h2><div class="sp"></div>
    <button class="icon-btn" title="发起群聊" @click="router.push('/create-group')">👥＋</button>
    <button class="icon-btn" title="添加朋友" @click="router.push('/add-friend')">＋</button>
  </div>
  <div class="h5-body">
    <div class="h5-search">🔍<input v-model="keyword" placeholder="搜索姓名 / 拼音 / 部门"></div>
    <button class="qentry" @click="router.push('/friends')">
      <span class="at-icon">👥</span>
      <div><div class="qe-t">{{ t('newFriends') }}</div><div class="qe-s">{{ applies.length }} 条待处理</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <button class="qentry" @click="router.push('/groups')">
      <span class="at-icon">💬</span>
      <div><div class="qe-t">{{ t('groupList') }}</div><div class="qe-s">我加入的群聊</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <button class="qentry" @click="router.push('/dept')">
      <span class="at-icon">🏢</span>
      <div><div class="qe-t">{{ t('org') }}</div><div class="qe-s">{{ depts.length }} 个部门</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <div class="sec-title">我的好友 · {{ filtered.length }} 人</div>
    <button v-for="u in filtered" :key="u.uid" class="qentry" @click="chatWith(u)">
      <Avatar :name="u.nickname" />
      <div><div class="qe-t">{{ u.nickname }}</div><div class="qe-s">{{ subOf(u) }}</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <div v-if="!filtered.length" class="loading">{{ t('noFriend') }}</div>
  </div>
</template>
<style scoped>
.h5-search { height: 40px; border-radius: 12px; }
.qentry { padding: 14px 13px; margin-bottom: 8px; border-radius: 14px; }
.qentry .at-icon { width: 40px; height: 40px; border-radius: 13px; background: #f4f3ff; font-size: 19px; }
.qentry .qe-t { font-size: 15px; font-weight: 700; }
.qentry .qe-s { font-size: 12.5px; margin-top: 4px; }
.qentry .qe-arrow { font-size: 16px; color: #c3cad9; }
.qentry :deep(.avatar) { font-size: 15px; }
.sec-title { font-size: 12px; font-weight: 600; color: #4a556b; padding: 10px 4px 8px; }
</style>
