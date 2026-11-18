<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'
import { useChatStore } from '../stores/chat'

const router = useRouter()
const chat = useChatStore()
const depts = ref([])
const selected = ref(0)
const members = ref([])
onMounted(async () => {
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
  <div class="h5-body" style="padding:10px">
    <div style="display:flex;gap:7px;flex-wrap:wrap;margin-bottom:10px">
      <button class="btn-sm" :class="{ on: selected === 0 }" @click="pick(0)">全部</button>
      <button v-for="d in depts" :key="d.id" class="btn-sm" :class="{ on: selected === d.id }" @click="pick(d.id)">{{ d.name }}</button>
    </div>
    <div class="sec-title">成员 · {{ members.length }} 人</div>
    <button v-for="u in members" :key="u.uid" class="qentry" @click="chatWith(u)">
      <Avatar :name="u.nickname" size="sm" />
      <div><div class="qe-t">{{ u.nickname }}</div><div class="qe-s">{{ u.deptName || '' }} · @{{ u.username }}</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <div v-if="!members.length" class="loading">{{ t('empty') }}</div>
  </div>
</template>
