<script setup>
import { ref, onMounted } from 'vue'
import { toast } from '../toast'
import { API } from '../api'
import ModalBox from '../components/ModalBox.vue'

const kw = ref('')
const typeF = ref('')
const resultF = ref('')
const modal = ref(null)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = 20

function fmtTime(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  const y = new Date(today.getTime() - 86400000).toDateString() === d.toDateString()
  const hm = `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
  if (sameDay) return '今天 ' + hm
  if (y) return '昨天 ' + hm
  return `${d.getMonth()+1}-${d.getDate()} ${hm}`
}
function fmtDur(sec) {
  if (!sec) return '00:00'
  return `${String(Math.floor(sec/60)).padStart(2,'0')}:${String(sec%60).padStart(2,'0')}`
}
function params() {
  return { kw: kw.value.trim(), type: typeF.value, result: resultF.value, page: page.value, size }
}
async function load(reset = false) {
  if (reset) page.value = 1
  try {
    const r = await API.calls(params())
    rows.value = r.data?.list || []
    total.value = r.data?.total || 0
  } catch (e) { toast(e.message, 'err') }
}
onMounted(() => load())
const pages = () => Math.max(1, Math.ceil(total.value / size))
function go(p) { page.value = Math.min(pages(), Math.max(1, p)); load() }

const qCls = q => q === '良好' ? 'ok' : (q === '未接通' || q === '未采样' ? 'off' : q === '强制挂断' || q === '较差' ? 'danger' : 'warn')

function playback(r) {
  modal.value.open({
    title: '录制回放 · MFA 二次验证',
    body: `<p>回放通话录制属于高敏操作，请输入管理员 MFA 验证码。</p><div class="mfa-code">123456</div>
      <div class="field"><label>验证码</label><input id="pb-code" placeholder="输入 6 位验证码"></div>
      <div class="muted">通话：${r.fromName} → ${r.peerName} · ${fmtTime(r.createdAt)} · 房间 ${r.roomId}</div>`,
    okText: '验证并回放',
    onOk: () => {
      const el = document.getElementById('pb-code')
      if (!el || el.value.trim() !== '123456') { toast('验证码错误', 'err'); return false }
      toast('该通话未开启云端录制，暂无回放文件', 'err')
    }
  })
}
function forceEnd(r) {
  modal.value.open({
    title: '强制挂断',
    body: `<p>确认强制结束通话「${r.fromName} → ${r.peerName}」？</p>
      <div class="muted">双方将收到 CALL_FORCE_END 信令，该记录标记为「强制挂断」，操作写入审计日志。</div>`,
    okText: '强制挂断', danger: true,
    onOk: async () => {
      try {
        await API.endCall(r.callId)
        toast('通话已强制挂断')
        load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
async function exportCsv() {
  try { await API.exportCalls(params()); toast('导出已完成') } catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div>
    <div class="proto-hint">通话记录来自 rtc 服务落库，强制挂断会下发 CALL_FORCE_END 信令并写入审计；质量数据由客户端 WebRTC 采样上报。<span class="code" style="margin-left:8px">FR-RTC-09/10/14</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="kw" placeholder="发起人 / 对端 / 房间号" style="width:210px" @keyup.enter="load(true)">
        <select v-model="typeF" @change="load(true)"><option value="">全部类型</option><option value="audio">语音</option><option value="video">视频</option></select>
        <select v-model="resultF" @change="load(true)"><option value="">全部结果</option><option value="answered">已接通</option><option value="missed">未接听</option><option value="force">强制挂断</option><option value="active">通话中</option></select>
        <button class="btn primary" @click="load(true)">检索</button>
        <span class="sp"></span>
        <button class="btn" @click="exportCsv">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>类型</th><th>发起人</th><th>对端 / 群</th><th>房间号</th><th>成员数</th><th>时长</th><th>模式</th><th>质量</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="r in rows" :key="r.callId">
            <td>{{ fmtTime(r.createdAt) }}</td><td><span class="state info">{{ r.typeLabel }}</span></td>
            <td>{{ r.fromName }}</td><td>{{ r.peerName }}</td><td><span class="code">{{ r.roomId }}</span></td>
            <td>{{ r.members }}</td><td>{{ fmtDur(r.duration) }}</td><td><span class="code">{{ r.mode }}</span></td>
            <td><span class="state" :class="qCls(r.quality)">{{ r.quality }}</span></td>
            <td style="white-space:nowrap">
              <button v-if="r.recorded" class="btn sm" @click="playback(r)">回放</button>
              <button v-if="r.result === 'active'" class="btn sm danger" @click="forceEnd(r)">强制挂断</button>
              <button v-else-if="r.forceEnded" class="btn sm" disabled>已强制挂断</button>
            </td>
          </tr>
          <tr v-if="!rows.length"><td colspan="10" class="loading">未找到匹配的通话记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ total }} 条
        <button :disabled="page <= 1" @click="go(page - 1)">‹</button>
        <button class="on">{{ page }}</button>
        <button :disabled="page >= pages()" @click="go(page + 1)">›</button>
        <span class="muted" style="margin-left:8px">/ {{ pages() }} 页</span>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
