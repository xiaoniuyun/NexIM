<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const keyword = ref('')
const users = ref([])
onMounted(async () => {
  try { const r = await API.users(); users.value = (r.data || []).filter(u => u.uid !== auth.user?.uid) } catch (e) { toast(e.message, 'err') }
})
const result = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return users.value.slice(0, 12)
  return users.value.filter(u => (u.nickname || '').toLowerCase().includes(q) || (u.username || '').toLowerCase().includes(q))
})
async function apply(u) {
  try { await API.friendApply(u.uid, '你好，我是 ' + (auth.user?.nickname || '')); toast('好友申请已发送') }
  catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('addFriend') }}</div><div class="ch-sub">{{ t('confidential') }}</div></div>
  </div>
  <div class="h5-body">
    <div class="h5-search">🔍<input v-model="keyword" :placeholder="t('inputSearchUser')"></div>
    <div class="sec-title">{{ keyword ? '搜索结果' : '推荐联系人' }}</div>
    <div v-for="u in result" :key="u.uid" class="qentry">
      <Avatar :name="u.nickname" />
      <div style="flex:1"><div class="qe-t">{{ u.nickname }}</div><div class="qe-s">@{{ u.username }} · {{ u.deptName || '' }}</div></div>
      <button class="btn-sm" @click="apply(u)">添加</button>
    </div>
    <div v-if="!result.length" class="loading">{{ t('noFriend') }}</div>
  </div>
</template>
