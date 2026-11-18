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
const list = ref([])
onMounted(async () => {
  try { const r = await API.friendApplies(); list.value = r.data || [] } catch (e) { toast(e.message, 'err') }
})
const pending = computed(() => list.value.filter(x => x.status === 0))
const done = computed(() => list.value.filter(x => x.status !== 0))
async function handle(a, accept) {
  try {
    await API.friendHandle(a.id, accept)
    a.status = accept ? 1 : 2
    toast(accept ? '已同意好友申请' : '已拒绝')
  } catch (e) { toast(e.message, 'err') }
}
function nameOf(a) { return a.fromName || a.nickname || ('用户 ' + a.fromUid) }
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('newFriends') }}</div><div class="ch-sub">{{ pending.length }} 条待处理</div></div>
  </div>
  <div class="h5-body">
    <div class="sec-title">待处理 · {{ pending.length }}</div>
    <div v-for="a in pending" :key="a.id" class="qentry">
      <Avatar :name="nameOf(a)" />
      <div style="flex:1;min-width:0">
        <div class="qe-t">{{ nameOf(a) }}</div>
        <div class="qe-s" style="white-space:normal">{{ a.message || '申请添加你为好友' }}</div>
        <div class="qe-s">{{ new Date(a.createdAt).toLocaleString() }}</div>
      </div>
      <button class="btn-sm" @click="handle(a, true)">通过</button>
      <button class="btn-ghost" @click="handle(a, false)">拒绝</button>
    </div>
    <div v-if="!pending.length" class="loading">暂无待处理申请</div>
    <div class="sec-title">已处理</div>
    <div v-for="a in done" :key="a.id" class="qentry">
      <Avatar :name="nameOf(a)" />
      <div style="flex:1"><div class="qe-t">{{ nameOf(a) }}</div><div class="qe-s">{{ a.message || '' }}</div></div>
      <span class="state" :class="a.status === 1 ? 'ok' : 'off'">{{ a.status === 1 ? '已同意' : '已拒绝' }}</span>
    </div>
  </div>
</template>
