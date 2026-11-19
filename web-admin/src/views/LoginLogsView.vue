<script setup>
import { ref, onMounted } from 'vue'
import { toast } from '../toast'
import { API } from '../api'
import ModalBox from '../components/ModalBox.vue'

const kw = ref('')
const result = ref('')
const risk = ref('')
const date = ref('')
const modal = ref(null)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = 20

function params() {
  return { kw: kw.value.trim(), result: result.value, risk: risk.value, page: page.value, size }
}
async function load(reset = false) {
  if (reset) page.value = 1
  try {
    const r = await API.loginLogs(params())
    let list = r.data?.list || []
    if (date.value) list = list.filter(x => new Date(x.createdAt).toISOString().slice(0, 10) === date.value)
    rows.value = list
    total.value = r.data?.total || 0
  } catch (e) { toast(e.message, 'err') }
}
onMounted(() => load())
function fmt(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
const riskText = r => r.risk || '正常'
const riskCls = r => !r.risk ? 'info' : (r.risk === '异常登录' ? 'danger' : 'warn')
function ban(r) {
  modal.value.open({
    title: '封禁 IP', body: `<p>确认封禁 IP「${r.ip}」30 分钟？封禁期间该 IP 的所有登录请求将被拒绝。</p><div class="muted">账号：${r.account} · 风险：${riskText(r)} · 操作写入审计</div>`,
    okText: '确认封禁', danger: true,
    onOk: async () => {
      try { await API.banIP(r.ip, 30); toast(`IP ${r.ip} 已封禁 30 分钟`); load() } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
async function exportCsv() {
  try { await API.exportLoginLogs(params()); toast('导出已完成') } catch (e) { toast(e.message, 'err') }
}
const pages = () => Math.max(1, Math.ceil(total.value / size))
function go(p) { page.value = Math.min(pages(), Math.max(1, p)); load() }
</script>
<template>
  <div>
    <div class="proto-hint">登录日志全量留存，高频失败自动标记异常，异常 IP 可一键封禁并写入审计。<span class="code" style="margin-left:8px">FR-ADMIN-09 / FR-AUTH-06</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="kw" placeholder="账号 / IP / 设备" style="width:190px" @keyup.enter="load(true)">
        <select v-model="result" @change="load(true)"><option value="">全部结果</option><option value="success">成功</option><option value="fail">失败</option></select>
        <select v-model="risk" @change="load(true)"><option value="">全部风险</option><option value="异常登录">异常登录</option></select>
        <input type="date" v-model="date" @change="load(true)">
        <button class="btn" @click="load(true)">查询</button>
        <span class="sp"></span>
        <button class="btn" @click="exportCsv">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>账号</th><th>来源 IP</th><th>平台 / 设备</th><th>结果</th><th>风险</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="r in rows" :key="r.id">
            <td>{{ fmt(r.createdAt) }}</td><td>{{ r.account }}</td><td><span class="code">{{ r.ip }}</span></td>
            <td>{{ r.platform }}<span class="muted"> · {{ r.deviceId }}</span></td>
            <td><span class="state" :class="r.result === 'success' ? 'ok' : 'danger'">{{ r.result === 'success' ? '成功' : '失败' }}</span></td>
            <td><span class="state" :class="riskCls(r)">{{ riskText(r) }}</span></td>
            <td><button v-if="r.risk" class="btn sm danger" @click="ban(r)">封禁 IP</button><span v-else>—</span></td>
          </tr>
          <tr v-if="!rows.length"><td colspan="7" class="loading">未找到匹配的登录记录</td></tr>
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
