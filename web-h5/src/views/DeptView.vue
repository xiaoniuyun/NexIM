<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const chat = useChatStore()
const auth = useAuthStore()
const depts = ref([])
const selected = ref(0)
const members = ref([])
onMounted(async () => {
  // 外部联系人默认无法查看组织架构
  if (auth.user?.isExternal === true) {
    toast('外部联系人无法查看组织架构', 'err')
    router.replace('/contacts')
    return
  }
  try {
    const r = await API.depts(); depts.value = r.data || []
    await pick(0)
  } catch (e) { toast(e.message, 'err') }
})
async function pick(id) {
  selected.value = id
  try { const r = await API.deptMembers(id); members.value = r.data || [] } catch (e) { toast(e.message, 'err') }
}
function chatWith(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  router.push('/chat/' + encodeURIComponent(convId))
}
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('org') }}</div><div class="ch-sub">点击部门筛选成员</div></div>
  </div>
  <div class="h5-body" style="padding:0">
    <div class="qq-chips">
      <button class="qq-chip" :class="{ on: selected === 0 }" @click="pick(0)">全部</button>
      <button v-for="d in depts" :key="d.id" class="qq-chip" :class="{ on: selected === d.id }" @click="pick(d.id)">{{ d.name }}</button>
    </div>
    <div class="qq-sec">成员 · {{ members.length }} 人</div>
    <button v-for="u in members" :key="u.uid" class="qq-row" @click="chatWith(u)">
      <span class="qq-ava"><Avatar :name="u.nickname" /></span>
      <div class="qq-main">
        <div class="qq-title"><span class="t">{{ u.nickname }}</span></div>
        <div class="qq-sub">{{ u.deptName || '' }} · @{{ u.username }}</div>
      </div>
      <span class="qq-arrow">›</span>
    </button>
    <div v-if="!members.length" class="qq-empty">{{ t('empty') }}</div>
  </div>
</template>
<style scoped>
.qq-chips { display: flex; gap: 8px; overflow-x: auto; padding: 10px 16px 6px; background: var(--card); }
.qq-chip { padding: 5px 12px; border-radius: 99px; background: #f3f5f9; color: #4a556b; font-size: 12px; white-space: nowrap; flex: none; }
.qq-chip.on { background: var(--brand); color: #fff; }
.qq-sec { font-size: 12px; font-weight: 600; color: #8a95aa; padding: 8px 16px; background: var(--card); }
.qq-arrow { color: #c3cad9; font-size: 17px; flex: none; }
</style>
