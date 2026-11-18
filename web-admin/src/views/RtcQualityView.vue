<script setup>
import { toast } from '../toast'
const kpis = [
  { label: '当前并发通话', value: '126', desc: '峰值 214 · 1v1 占比 68%', icon: '📞' },
  { label: '接通率', value: '94.8%', desc: '未接 41 · 拒接 27', icon: '✅' },
  { label: '平均通话时长', value: '07:24', desc: 'P95 21:08', icon: '⏱️' },
  { label: '平均 MOS', value: '4.21', desc: '低于 3.8 占比 2.1%', icon: '📊' }
]
const mos = [['4.5~5.0', 62], ['4.0~4.5', 26], ['3.5~4.0', 8], ['3.0~3.5', 3], ['低于 3.0', 1]]
const alarms = [
  ['10:12', '周涛 的通话 MOS 降至 3.4（TURN 中继高丢包）', 'warn', '待处理'],
  ['08:31', '外部客户通话连续 2 次未接', 'warn', '已通知客服'],
  ['昨天 22:40', 'SFU 节点 sfu-2 CPU 85%', 'warn', '已自动扩容'],
  ['昨天 19:05', '群组视频 9 路上行带宽不足，已降层', 'ok', '已自动处理']
]
</script>
<template>
  <div>
    <div class="proto-hint">质量数据来自客户端 RTC_QUALITY 周期上报，异常自动触发告警并推送运维。<span class="code" style="margin-left:8px">FR-RTC-11 / NFR-RTC-01~04</span></div>
    <div class="grid4">
      <div v-for="k in kpis" :key="k.label" class="card kpi">
        <div class="k-t">{{ k.label }}</div><div class="k-v">{{ k.value }}</div><div class="k-d">{{ k.desc }}</div>
        <div class="k-ico">{{ k.icon }}</div>
      </div>
    </div>
    <div class="grid2">
      <div class="card">
        <h3>MOS 分布</h3>
        <div class="csub">按通话结束时客户端自评 + 服务端探针综合计算</div>
        <div v-for="b in mos" :key="b[0]" class="bar-row">
          <span class="bl">{{ b[0] }}</span><div class="bar-track"><div class="bar-fill" :style="{ width: b[1] + '%' }"></div></div><span class="muted">{{ b[1] }}%</span>
        </div>
        <div class="legend"><span>丢包率 &lt; 1%：68%</span><span>卡顿率 0.6%</span><span>音频时延 P99 312ms</span></div>
      </div>
      <div class="card">
        <h3>通话告警</h3>
        <div class="csub">近 24 小时音视频相关告警</div>
        <div v-for="a in alarms" :key="a[0]" class="timeline-item" @click="toast('告警详情（演示）')">
          <i class="tl-dot" :style="{ background: a[2] === 'warn' ? '#f59e0b' : '#12b886' }"></i>
          <div style="flex:1">{{ a[1] }}</div>
          <span class="tl-time">{{ a[0] }} · {{ a[3] }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
