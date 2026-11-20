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
  <div class="qq-body">
    <div class="qq-sec">待处理 · {{ pending.length }}</div>
    <div v-for="a in pending" :key="a.id" class="qq-row">
      <span class="qq-ava"><Avatar :name="nameOf(a)" /></span>
      <div class="qq-main">
        <div class="qq-title"><span class="t">{{ nameOf(a) }}</span></div>
        <div class="qq-sub">{{ a.message || '申请添加你为好友' }}</div>
      </div>
      <button class="qq-act ok" @click="handle(a, true)">通过</button>
      <button class="qq-act" @click="handle(a, false)">拒绝</button>
    </div>
    <div v-if="!pending.length" class="qq-empty">暂无待处理申请</div>
    <div class="qq-sec">已处理</div>
    <div v-for="a in done" :key="a.id" class="qq-row">
      <span class="qq-ava"><Avatar :name="nameOf(a)" /></span>
      <div class="qq-main">
        <div class="qq-title"><span class="t">{{ nameOf(a) }}</span></div>
        <div class="qq-sub">{{ a.message || '' }}</div>
      </div>
      <span class="qq-state" :class="a.status === 1 ? 'ok' : ''">{{ a.status === 1 ? '已同意' : '已拒绝' }}</span>
    </div>
  </div>
</template>
<style scoped>
.qq-sec { font-size: 12px; font-weight: 600; color: #8a95aa; padding: 10px 16px 6px; }
.qq-act { flex: none; font-size: 12px; font-weight: 600; padding: 6px 12px; border-radius: 8px; background: #f3f5f9; color: #4a556b; }
.qq-act.ok { background: var(--brand); color: #fff; }
.qq-state { flex: none; font-size: 12px; color: #9aa5b8; }
.qq-state.ok { color: #25b389; }
</style>
