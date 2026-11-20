<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import LineIcon from '../components/LineIcon.vue'
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
  <div class="qq-body">
    <div class="qq-search"><LineIcon name="search" :size="16" /><input v-model="keyword" :placeholder="t('inputSearchUser')"></div>
    <div class="qq-sec">{{ keyword ? '搜索结果' : '推荐联系人' }}</div>
    <button v-for="u in result" :key="u.uid" class="qq-row" @click="apply(u)">
      <span class="qq-ava"><Avatar :name="u.nickname" /></span>
      <div class="qq-main">
        <div class="qq-title"><span class="t">{{ u.nickname }}</span><span v-if="u.external" class="qq-tag">外部</span></div>
        <div class="qq-sub">@{{ u.username }} · {{ u.external ? '企业外部用户' : (u.deptName || '企业内部用户') }}</div>
      </div>
      <span class="qq-add-btn">添加</span>
    </button>
    <div v-if="!result.length" class="qq-empty">{{ t('noFriend') }}</div>
  </div>
</template>
<style scoped>
.qq-sec { font-size: 12px; font-weight: 600; color: #8a95aa; padding: 8px 16px; }
.qq-add-btn { flex: none; font-size: 12px; font-weight: 600; color: var(--brand); background: var(--brand-soft); border-radius: 8px; padding: 5px 11px; }
</style>
