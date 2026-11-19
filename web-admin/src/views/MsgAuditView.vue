<script setup>
import { ref, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const f = ref({ kw: '', type: '', status: '', date: '' })
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const modal = ref(null)
const typeText = { 1: '文本', 2: '图片', 3: '系统', 4: '语音', 5: '文件', 6: '视频' }

function params() {
  return { kw: f.value.kw.trim(), type: f.value.type, status: f.value.status, page: page.value, size }
}
async function load(reset = false) {
  if (reset) page.value = 1
  try {
    const r = await API.messageAudit({ ...params(), size })
    let list = r.data?.list || []
    if (f.value.date) list = list.filter(m => new Date(m.createdAt).toISOString().slice(0, 10) === f.value.date)
    rows.value = list
    total.value = r.data?.total || 0
  } catch (e) { toast(e.message, 'err') }
}
onMounted(() => load())
const fmt = ts => ts ? new Date(ts).toLocaleString() : '—'
function statusText(m) { return m.status === 2 ? '已撤回' : m.status === 3 ? '已删除' : '正常' }

function exportResult() {
  modal.value.open({
    title: '导出审计结果 · MFA 二次验证',
    body: '<p>导出消息属于高敏操作，请输入管理员 MFA 验证码。</p><div class="mfa-code">123456</div><div class="field"><label>验证码</label><input id="mfa-code" placeholder="输入 6 位验证码"></div>',
    okText: '验证并导出',
    onOk: async () => {
      if (document.getElementById('mfa-code').value.trim() !== '123456') { toast('验证码错误', 'err'); return false }
      try {
        await API.exportMessages(params())
        toast('导出已完成')
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
const pages = () => Math.max(1, Math.ceil(total.value / size))
function go(p) { page.value = Math.min(pages(), Math.max(1, p)); load() }
</script>
<template>
  <div>
    <div class="proto-hint">消息审计只读，导出需 MFA 二次验证；内容按脱敏规则展示。<span class="code" style="margin-left:8px">FR-ADMIN-06 / FR-SEC-03</span></div>
    <div class="card">
      <h3>审计检索</h3>
      <div class="toolbar" style="margin-top:12px">
        <input v-model="f.kw" placeholder="消息 / 会话 / 发送人" style="width:200px" @keyup.enter="load(true)">
        <select v-model="f.type" @change="load(true)"><option value="">全部类型</option><option value="1">文本</option><option value="2">图片</option><option value="5">文件</option><option value="6">视频</option><option value="3">系统</option></select>
        <select v-model="f.status" @change="load(true)"><option value="">全部状态</option><option value="1">正常</option><option value="2">已撤回</option></select>
        <input type="date" v-model="f.date" @change="load(true)">
        <button class="btn primary" @click="load(true)">检索</button>
      </div>
    </div>
    <div class="card">
      <div class="card-head"><h3>检索结果 · {{ total }} 条</h3><button class="btn" @click="exportResult">导出结果</button></div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>消息ID</th><th>会话</th><th>发送人</th><th>接收方</th><th>类型</th><th>内容</th><th>状态</th></tr></thead>
        <tbody>
          <tr v-for="m in rows" :key="m.msgId">
            <td>{{ fmt(m.createdAt) }}</td><td><span class="code">{{ m.msgId }}</span></td>
            <td>{{ m.convId }}</td><td>{{ m.fromUid }}</td><td>{{ m.toId }}</td>
            <td><span class="state info">{{ typeText[m.type] || '消息' }}</span></td>
            <td style="max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ m.content }}</td>
            <td><span class="state" :class="m.status === 1 ? 'ok' : 'warn'">{{ statusText(m) }}</span></td>
          </tr>
          <tr v-if="!rows.length"><td colspan="8" class="loading">未找到匹配的消息</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ total }} 条
        <button :disabled="page <= 1" @click="go(page - 1)">‹</button><button class="on">{{ page }}</button>
        <button :disabled="page >= pages()" @click="go(page + 1)">›</button>
        <span class="muted" style="margin-left:8px">/ {{ pages() }} 页</span>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
