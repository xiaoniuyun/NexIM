<script setup>
import { computed, ref, onMounted } from 'vue'
import { API } from '@/shared/api'
import { useChatStore } from '@/shared/stores/chat'
import { toast } from '@/shared/toast'
import { fullTime } from '../composables/useDesktop'

const chat = useChatStore()
const calls = ref([])
const favorites = ref([])
const friends = ref([])

onMounted(async () => {
  try {
    const [c, f, fr] = await Promise.all([API.calls(30), API.favorites(), API.friends()])
    calls.value = c.data || []
    favorites.value = f.data || []
    friends.value = fr.data || []
  } catch (e) { toast(e.message, 'err') }
})

const answered = computed(() => calls.value.filter(c => c.status === 3 && c.duration > 0))
const avgDuration = computed(() => answered.value.length
  ? Math.round(answered.value.reduce((s, c) => s + c.duration, 0) / answered.value.length)
  : 0)
const quality = computed(() => {
  const q = calls.value.filter(c => c.mos > 0)
  return q.length ? (q.reduce((s, c) => s + c.mos, 0) / q.length).toFixed(2) : '—'
})
const topUnread = computed(() => [...chat.convs].filter(c => c.unread > 0).sort((a, b) => b.unread - a.unread).slice(0, 5))

function typeText(c) { return c.callType === 'video' ? '视频' : '语音' }
function statusText(c) {
  if (c.status === 4) return '未接/拒接'
  if (c.status === 3) return '已结束'
  if (c.status === 2) return '通话中'
  return '呼叫中'
}
function dur(c) {
  const s = c.duration || 0
  return `${Math.floor(s / 60)}分${s % 60}秒`
}
</script>

<template>
  <div class="pane">
    <header class="pane-head"><h2>统计</h2></header>
    <div class="pane-body scroll">
      <div class="kpi-grid">
        <div class="kpi"><span class="kpi-num">{{ chat.convs.length }}</span><span class="kpi-label">会话总数</span></div>
        <div class="kpi"><span class="kpi-num">{{ chat.unreadTotal }}</span><span class="kpi-label">未读消息</span></div>
        <div class="kpi"><span class="kpi-num">{{ friends.length }}</span><span class="kpi-label">联系人</span></div>
        <div class="kpi"><span class="kpi-num">{{ favorites.length }}</span><span class="kpi-label">收藏</span></div>
        <div class="kpi"><span class="kpi-num">{{ avgDuration }}s</span><span class="kpi-label">平均通话时长</span></div>
        <div class="kpi"><span class="kpi-num">{{ quality }}</span><span class="kpi-label">平均通话 MOS</span></div>
      </div>

      <div class="card">
        <div class="card-head"><h3>未读排行</h3></div>
        <div v-for="c in topUnread" :key="c.convId" class="rank-row">
          <span class="rank-name">{{ c.peerName || c.convId }}</span>
          <span class="rank-bar"><i :style="{ width: Math.min(100, c.unread * 10) + '%' }"></i></span>
          <span class="rank-num">{{ c.unread }}</span>
        </div>
        <div v-if="!topUnread.length" class="empty-tip">没有未读会话</div>
      </div>

      <div class="card">
        <div class="card-head"><h3>最近通话</h3><span class="muted">{{ calls.length }} 条</span></div>
        <table class="tbl">
          <thead><tr><th>时间</th><th>类型</th><th>状态</th><th>时长</th><th>MOS</th></tr></thead>
          <tbody>
            <tr v-for="c in calls" :key="c.callId">
              <td>{{ fullTime(c.createdAt) }}</td>
              <td>{{ typeText(c) }}</td>
              <td>{{ statusText(c) }}<span v-if="c.forceEnded" class="tag off">强制结束</span></td>
              <td>{{ c.duration ? dur(c) : '—' }}</td>
              <td>{{ c.mos ? c.mos.toFixed(2) : '—' }}</td>
            </tr>
            <tr v-if="!calls.length"><td colspan="5" class="empty-tip">暂无通话记录</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
