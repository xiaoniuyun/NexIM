<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'

const router = useRouter()
const users = ref([])
const picked = ref([])
const groupName = ref('')
const keyword = ref('')

onMounted(async () => {
  try {
    const u = await API.users()
    users.value = u.data || []
  } catch (e) { toast(e.message, 'err') }
})

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return q ? users.value.filter(u => (u.nickname || '').toLowerCase().includes(q) || (u.username || '').toLowerCase().includes(q)) : users.value
})
function toggle(uid) {
  const i = picked.value.indexOf(uid)
  if (i >= 0) picked.value.splice(i, 1)
  else picked.value.push(uid)
}
function pickedUsers() { return users.value.filter(u => picked.value.includes(u.uid)) }
async function create() {
  if (!groupName.value.trim()) { toast('请输入群名称', 'err'); return }
  try {
    const r = await API.groupCreate({ name: groupName.value.trim(), members: picked.value })
    toast('群聊创建成功')
    const gid = r.data?.groupId
    router.push('/groups')
  } catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">发起群聊</div><div class="ch-sub">已选 {{ picked.length }} 人</div></div>
    <div class="sp"></div>
    <button class="btn-sm" @click="create">完成</button>
  </div>
  <div class="h5-body" style="padding:12px">
    <div class="field" style="margin-bottom:12px"><input v-model="groupName" placeholder="输入群名称（选填，默认按成员生成）" style="width:100%"></div>
    <div class="picked-row">
      <span v-if="!picked.length" class="muted" style="padding:8px 0">勾选联系人后点击完成</span>
      <Avatar v-for="u in pickedUsers()" :key="u.uid" :name="u.nickname" size="sm" />
    </div>
    <div class="h5-search">🔍<input v-model="keyword" placeholder="搜索联系人"></div>
    <div class="sec-title">选择联系人 · {{ filtered.length }} 人</div>
    <label v-for="u in filtered" :key="u.uid" class="contact-pick">
      <input type="checkbox" :checked="picked.includes(u.uid)" @change="toggle(u.uid)">
      <Avatar :name="u.nickname" size="sm" />
      <div><div style="font-weight:600">{{ u.nickname }}</div><div style="font-size:10px;color:var(--muted)">{{ u.deptName || '@' + u.username }}</div></div>
    </label>
    <div v-if="!filtered.length" class="loading">暂无联系人</div>
  </div>
</template>
