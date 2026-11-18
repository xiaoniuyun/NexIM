<script setup>
import { ref, computed, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

const data = ref(null)
const range = ref('7')
onMounted(async () => {
  try { data.value = (await API.dashboard()).data || {} } catch (e) { toast(e.message, 'err') }
})
function refresh() { API.dashboard().then(r => { data.value = r.data || {}; toast('指标已刷新') }).catch(e => toast(e.message, 'err')) }
function setRange(r) { range.value = r; toast('已切换至近 ' + r + ' 天趋势') }

const pts1 = computed(() => range.value === '30'
  ? '0,150 55,132 110,118 165,124 220,88 275,96 330,62 385,74 440,40 495,52'
  : '0,145 55,120 110,132 165,82 220,96 275,52 330,70 385,28 440,42 495,16')
const pts2 = computed(() => range.value === '30'
  ? '0,164 55,152 110,146 165,138 220,142 275,126 330,118 385,108 440,112 495,98'
  : '0,160 55,148 110,152 165,126 220,134 275,110 330,120 385,92 440,104 495,84')

const msgTypes = [['文本', 72], ['图片', 14], ['文件', 8], ['语音', 4], ['系统', 2]]
const clients = [['Windows', 34], ['H5 移动端', 32], ['macOS', 24], ['Linux', 10]]
const recentOps = [
  ['林晓', '更新角色 OPS_ADMIN 权限集合', '09:38'],
  ['王敏', '导出审计结果（320 条）', '09:21'],
  ['赵琳', '禁用账号 chenjing', '08:46'],
  ['未知', '登录失败 5 次触发锁定', '07:12']
]
const health = [
  ['Gateway 集群', '3 / 3 节点正常', 'ok'],
  ['消息到达率', '99.993%', 'ok'],
  ['Kafka 消费 lag', '1,204 条', 'warn'],
  ['待处理告警', '2 条 · 内容安全 / 异常登录', 'warn'],
  ['MySQL 主从', '正常 · 延迟 0.3s', 'ok'],
  ['对象存储', '可用率 100%', 'ok']
]
</script>
<template>
  <div>
    <div class="proto-hint">工作台指标来自消息链路埋点与聚合任务，支持近 7 / 30 天切换。<span class="code" style="margin-left:8px">FR-ADMIN-01</span></div>
    <div class="toolbar">
      <span class="sp"></span>
      <button class="btn sm" :class="range === '7' ? 'primary' : ''" @click="setRange('7')">近 7 天</button>
      <button class="btn sm" :class="range === '30' ? 'primary' : ''" @click="setRange('30')">近 30 天</button>
      <button class="btn sm" @click="refresh">⟳ 刷新</button>
    </div>
    <div v-if="!data" class="card loading">加载中...</div>
    <template v-else>
      <div class="grid4">
        <div class="card kpi"><div class="k-t">当前在线用户</div><div class="k-v">{{ data.onlineUsers }}</div><div class="k-d">较昨日 +12.4% · 峰值 9,180</div><div class="k-ico">🟢</div></div>
        <div class="card kpi"><div class="k-t">今日消息量</div><div class="k-v">{{ Number(data.todayMessages).toLocaleString() }}</div><div class="k-d">峰值 4,820 QPS · 群聊占 38%</div><div class="k-ico">💬</div></div>
        <div class="card kpi"><div class="k-t">活跃用户 DAU</div><div class="k-v">{{ data.dau }}</div><div class="k-d">较昨日 +5.1%</div><div class="k-ico">📈</div></div>
        <div class="card kpi"><div class="k-t">活跃群组</div><div class="k-v">{{ data.activeGroups }}</div><div class="k-d">今日有消息的会话数</div><div class="k-ico">👥</div></div>
      </div>
      <div class="grid2">
        <div class="card">
          <h3>消息量趋势</h3>
          <div class="csub">按小时聚合 · 单聊 / 群聊总量（{{ range === '7' ? '近 7 天' : '近 30 天' }}）</div>
          <div class="chart-wrap">
            <svg viewBox="0 0 500 200" width="100%" height="200" preserveAspectRatio="none">
              <line v-for="y in [40,80,120,160]" :key="y" x1="0" :y1="y" x2="500" :y2="y" stroke="#eef1f7" stroke-width="1"/>
              <defs><linearGradient id="ga" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#4f6ef7" stop-opacity=".28"/><stop offset="1" stop-color="#4f6ef7" stop-opacity="0"/></linearGradient></defs>
              <polygon :points="pts1 + ' 495,200 0,200'" fill="url(#ga)"/>
              <polyline :points="pts2" fill="none" stroke="#c8cfdf" stroke-width="2" stroke-dasharray="5 5"/>
              <polyline :points="pts1" fill="none" stroke="#4f6ef7" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="legend"><span><i style="background:#4f6ef7"></i>今日</span><span><i style="background:#c8cfdf"></i>昨日</span><span style="margin-left:auto">00:00 — 24:00</span></div>
        </div>
        <div class="card">
          <h3>消息类型分布</h3>
          <div class="csub">今日各类型消息占比</div>
          <div v-for="b in msgTypes" :key="b[0]" class="bar-row">
            <span class="bl">{{ b[0] }}</span><div class="bar-track"><div class="bar-fill" :style="{ width: b[1] + '%' }"></div></div><span class="muted">{{ b[1] }}%</span>
          </div>
        </div>
        <div class="card">
          <h3>客户端分布（登录分布）</h3>
          <div class="csub">按最后活跃端统计的登录占比</div>
          <div v-for="b in clients" :key="b[0]" class="bar-row">
            <span class="bl">{{ b[0] }}</span><div class="bar-track"><div class="bar-fill" :style="{ width: b[1] + '%' }"></div></div><span class="muted">{{ b[1] }}%</span>
          </div>
        </div>
      </div>
      <div class="grid2">
        <div class="card">
          <h3>最近后台操作</h3>
          <div class="csub">全部写操作自动记录审计日志</div>
          <div v-for="l in recentOps" :key="l[2]" class="timeline-item">
            <i class="tl-dot"></i>
            <div><b>{{ l[0] }}</b> · {{ l[1] }}</div>
            <span class="tl-time">{{ l[2] }}</span>
          </div>
        </div>
        <div class="card">
          <h3>系统健康度</h3>
          <div class="csub">核心链路实时状态</div>
          <div v-for="r in health" :key="r[0]" class="switch-row">
            <div>{{ r[0] }}<div class="sr-desc">{{ r[1] }}</div></div>
            <span class="state" :class="r[2]">{{ r[2] === 'ok' ? '正常' : '关注' }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
