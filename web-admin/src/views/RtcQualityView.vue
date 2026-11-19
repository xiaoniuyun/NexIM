<script setup>
import { ref, computed, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

const data = ref({})
const loading = ref(true)

const kpis = computed(() => {
  const d = data.value
  return [
    { label: '当前并发通话', value: d.active ?? 0, desc: `近 24h 通话 ${d.total ?? 0} · 强制挂断 ${d.forceEnded ?? 0}`, icon: '📞' },
    { label: '接通率', value: `${(d.answerRate ?? 0).toFixed(1)}%`, desc: `已接通 ${d.answered ?? 0} · 未接 ${d.missed ?? 0}`, icon: '✅' },
    { label: '平均通话时长', value: fmtDur(d.avgDuration || 0), desc: `未接 ${d.missed ?? 0} 通`, icon: '⏱️' },
    { label: '平均 MOS', value: d.sampled ? (d.avgMos ?? 0).toFixed(2) : '—', desc: d.sampled ? `采样 ${d.sampled} 通` : '暂无客户端质量采样', icon: '📊' }
  ]
})
const mos = computed(() => {
  const buckets = data.value.mosBuckets || []
  const sum = buckets.reduce((s, b) => s + (b.value || 0), 0)
  return buckets.map(b => ({ label: b.label, pct: sum ? Math.round(b.value * 100 / sum) : 0, value: b.value }))
})
const alarms = computed(() => data.value.alarms || [])

function fmtDur(sec) {
  if (!sec) return '00:00'
  return `${String(Math.floor(sec / 60)).padStart(2, '0')}:${String(sec % 60).padStart(2, '0')}`
}
function fmtTime(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
async function load() {
  loading.value = true
  try { const r = await API.rtcQuality(); data.value = r.data || {} } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}
onMounted(load)
function detail(a) {
  toast(`${a.type}：${a.detail}`)
}
</script>
<template>
  <div>
    <div class="proto-hint">质量数据来自客户端 WebRTC 统计上报（MOS/丢包/RTT/抖动），异常自动聚合为告警。<span class="code" style="margin-left:8px">FR-RTC-11 / NFR-RTC-01~04</span>
      <button class="btn sm" style="float:right" @click="load">刷新</button>
    </div>
    <div class="grid4">
      <div v-for="k in kpis" :key="k.label" class="card kpi">
        <div class="k-t">{{ k.label }}</div><div class="k-v">{{ k.value }}</div><div class="k-d">{{ k.desc }}</div>
        <div class="k-ico">{{ k.icon }}</div>
      </div>
    </div>
    <div class="grid2">
      <div class="card">
        <h3>MOS 分布</h3>
        <div class="csub">按通话结束时客户端自评统计（近 24 小时）</div>
        <div v-if="!data.sampled" class="loading" style="padding:20px 0">暂无质量采样，通话结束后将自动上报</div>
        <template v-else>
          <div v-for="b in mos" :key="b.label" class="bar-row">
            <span class="bl">{{ b.label }}</span>
            <div class="bar-track"><div class="bar-fill" :style="{ width: b.pct + '%' }"></div></div>
            <span class="muted">{{ b.pct }}%</span>
          </div>
          <div class="legend">
            <span>采样 {{ data.sampled }} 通</span>
            <span>平均丢包 {{ (data.avgLoss || 0).toFixed(1) }}%</span>
            <span>平均 RTT {{ data.avgRtt || 0 }}ms</span>
            <span>抖动 {{ data.avgJitter || 0 }}ms</span>
          </div>
        </template>
      </div>
      <div class="card">
        <h3>通话告警</h3>
        <div class="csub">近 24 小时音视频相关告警</div>
        <div v-for="a in alarms" :key="a.callId + a.type" class="timeline-item" @click="detail(a)">
          <i class="tl-dot" :style="{ background: a.level === 'high' ? '#ef4b5b' : a.level === 'mid' ? '#f59e0b' : '#12b886' }"></i>
          <div style="flex:1">{{ a.type }}：{{ a.detail }}</div>
          <span class="tl-time">{{ fmtTime(a.createdAt) }}</span>
        </div>
        <div v-if="!alarms.length" class="loading" style="padding:20px 0">暂无告警</div>
      </div>
    </div>
  </div>
</template>
