<script setup>
import { computed, ref, onMounted } from 'vue'
import { API } from '@h5/api'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
import { useCallStore } from '@h5/stores/call'
import { toast } from '@h5/toast'
import { fullTime } from '../composables/useDesktop'

const auth = useAuthStore()
const chat = useChatStore()
const call = useCallStore()

const records = ref([])
const friends = ref([])
const loading = ref(true)
const filter = ref('all')
const keyword = ref('')

const list = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return records.value.filter(c => {
    if (filter.value === 'missed' && c.status !== 4) return false
    if (filter.value === 'video' && c.callType !== 'video') return false
    if (filter.value === 'audio' && c.callType !== 'audio') return false
    if (filter.value === 'out' && c.callerUid !== auth.user?.uid) return false
    if (filter.value === 'in' && c.calleeUid !== auth.user?.uid) return false
    if (q && !peerName(c).toLowerCase().includes(q)) return false
    return true
  })
})

const stats = computed(() => {
  const answered = records.value.filter(c => c.status === 3 && c.duration > 0)
  const missed = records.value.filter(c => c.status === 4)
  const avg = answered.length ? Math.round(answered.reduce((s, c) => s + c.duration, 0) / answered.length) : 0
  const mos = records.value.filter(c => c.mos > 0)
  return {
    total: records.value.length,
    missed: missed.length,
    avg,
    mos: mos.length ? (mos.reduce((s, c) => s + c.mos, 0) / mos.length).toFixed(2) : '—'
  }
})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const [c, f] = await Promise.all([API.calls(80), API.friends()])
    records.value = c.data || []
    friends.value = f.data || []
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}

function peerUid(c) { return c.callerUid === auth.user?.uid ? c.calleeUid : c.callerUid }
function peerName(c) {
  const conv = chat.convs.find(x => x.convId === c.convId)
  if (conv?.peerName) return conv.peerName
  const f = friends.value.find(x => x.uid === peerUid(c))
  return f?.nickname || peerUid(c) || '未知联系人'
}
function directionText(c) { return c.callerUid === auth.user?.uid ? '呼出' : '呼入' }
function statusText(c) {
  if (c.status === 4) return c.callerUid === auth.user?.uid ? '对方未接' : '未接来电'
  if (c.status === 3) return c.forceEnded ? '管理员结束' : '已结束'
  if (c.status === 2) return '通话中'
  return '呼叫中'
}
function dur(s) {
  const v = Number(s) || 0
  if (!v) return '—'
  const m = Math.floor(v / 60); const sec = v % 60
  return m ? `${m}分${sec}秒` : `${sec}秒`
}
function redial(c, type) {
  const convId = c.convId
  let conv = chat.convs.find(x => x.convId === convId)
  if (!conv) {
    const uid = peerUid(c)
    if (!uid) return toast('缺少对方信息，无法回拨', 'err')
    const id = chat.ensureConv(uid, peerName(c))
    conv = chat.convs.find(x => x.convId === id)
  }
  if (!conv?.peerId || conv.convType !== 1) return toast('仅支持回拨单聊语音/视频', 'err')
  call.startCall(conv, type)
}
function openConv(c) {
  if (c.convId && chat.convs.some(x => x.convId === c.convId)) {
    window.dispatchEvent(new CustomEvent('nexim:open-conv', { detail: c.convId }))
  }
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>通话</h2>
      <div class="tabs">
        <button :class="{ on: filter === 'all' }" @click="filter = 'all'">全部</button>
        <button :class="{ on: filter === 'out' }" @click="filter = 'out'">呼出</button>
        <button :class="{ on: filter === 'in' }" @click="filter = 'in'">呼入</button>
        <button :class="{ on: filter === 'missed' }" @click="filter = 'missed'">未接</button>
        <button :class="{ on: filter === 'video' }" @click="filter = 'video'">视频</button>
      </div>
      <input v-model="keyword" class="pane-search" placeholder="搜索联系人">
      <button class="btn sm" @click="load">刷新</button>
    </header>

    <div class="pane-body scroll">
      <div class="kpi-grid">
        <div class="kpi"><span class="kpi-num">{{ stats.total }}</span><span class="kpi-label">通话总数</span></div>
        <div class="kpi"><span class="kpi-num">{{ stats.missed }}</span><span class="kpi-label">未接来电</span></div>
        <div class="kpi"><span class="kpi-num">{{ stats.avg }}s</span><span class="kpi-label">平均时长</span></div>
        <div class="kpi"><span class="kpi-num">{{ stats.mos }}</span><span class="kpi-label">平均 MOS</span></div>
      </div>

      <div class="card">
        <table class="tbl">
          <thead><tr><th>联系人</th><th>方向</th><th>类型</th><th>结果</th><th>时长</th><th>时间</th><th></th></tr></thead>
          <tbody>
            <tr v-for="c in list" :key="c.callId">
              <td>
                <button class="link-btn" @click="openConv(c)">{{ peerName(c) }}</button>
              </td>
              <td>{{ directionText(c) }}</td>
              <td>{{ c.callType === 'video' ? '视频' : '语音' }}</td>
              <td>
                <span :class="c.status === 4 ? 'danger' : ''">{{ statusText(c) }}</span>
              </td>
              <td>{{ dur(c.duration) }}</td>
              <td class="muted">{{ fullTime(c.createdAt) }}</td>
              <td class="row-actions">
                <button class="btn sm" @click="redial(c, 'audio')">语音回拨</button>
                <button class="btn sm" @click="redial(c, 'video')">视频回拨</button>
              </td>
            </tr>
            <tr v-if="!list.length"><td colspan="7" class="empty-tip">{{ loading ? '加载中…' : '暂无通话记录' }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
