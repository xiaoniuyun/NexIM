<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const demoRows = [
  { createdAt: '2026-11-15T09:41:22', convId: '产品评审群', fromUid: '李工', type: '文件', content: '02-技术架构文档.md（文件消息）', tag: '正常' },
  { createdAt: '2026-11-15T09:32:08', convId: '张伟 ↔ 林晓', fromUid: '张伟', type: '文本', content: '那我们十点先对齐一下接口', tag: '正常' },
  { createdAt: '2026-11-15T09:18:45', convId: '外部协作群', fromUid: '访客_3021', type: '文本', content: '[已脱敏] 含敏感词内容，已按策略拦截', tag: '命中' },
  { createdAt: '2026-11-15T08:57:13', convId: '前端研发组', fromUid: '赵琳', type: '文本', content: '动态路由已经跑通，菜单按权限码过滤了', tag: '正常' },
  { createdAt: '2026-11-15T08:12:30', convId: '系统通知', fromUid: '系统', type: '通知', content: '安全提醒：检测到异地登录尝试', tag: '正常' }
]
const f = ref({ keyword: '', from: '', convType: '', date: '' })
const rows = ref([])
const modal = ref(null)
onMounted(async () => {
  try {
    const r = await API.messageAudit()
    const real = (r.data?.list || []).map(m => ({ createdAt: m.createdAt, convId: m.convId, fromUid: m.fromUid, type: '文本', content: m.content, tag: m.status === 2 ? '已撤回' : '正常' }))
    rows.value = real.length ? real : demoRows
  } catch (e) { rows.value = demoRows }
})
const filtered = computed(() => rows.value.filter(r => {
  const q = f.value.keyword.trim().toLowerCase()
  return (!q || (r.content || '').toLowerCase().includes(q)) &&
    (!f.value.from || String(r.fromUid).includes(f.value.from.trim())) &&
    (!f.value.convType || (f.value.convType === '单聊' ? !String(r.convId).includes('群') : f.value.convType === '群聊' ? String(r.convId).includes('群') : r.type === '通知')) &&
    (!f.value.date || new Date(r.createdAt).toLocaleDateString() === new Date(f.value.date).toLocaleDateString())
}))
const fmt = ts => {
  const d = new Date(ts); return isNaN(d) ? ts : d.toLocaleString()
}
function exportResult() {
  modal.value.open({
    title: '导出审计结果 · MFA 二次验证',
    body: '<p>导出消息属于高敏操作，请输入管理员 MFA 验证码。</p><div class="mfa-code">123456</div><div class="field"><label>验证码</label><input id="mfa-code" placeholder="输入 6 位验证码"></div>',
    okText: '验证并导出',
    onOk: () => {
      if (document.getElementById('mfa-code').value.trim() !== '123456') { toast('验证码错误', 'err'); return false }
      toast(`已导出 ${filtered.value.length} 条审计结果（演示）`)
    }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">消息审计只读，导出需 MFA 二次验证；内容展示按脱敏规则处理。<span class="code" style="margin-left:8px">FR-ADMIN-06 / FR-SEC-03</span></div>
    <div class="card">
      <h3>审计检索</h3>
      <div class="toolbar" style="margin-top:12px">
        <input v-model="f.keyword" placeholder="消息关键字" style="width:200px">
        <input v-model="f.from" placeholder="发送人 UID / 姓名" style="width:170px">
        <select v-model="f.convType"><option value="">全部会话类型</option><option>单聊</option><option>群聊</option><option>系统通知</option></select>
        <input type="date" v-model="f.date">
        <button class="btn primary" @click="toast('已按条件检索')">检索</button>
      </div>
    </div>
    <div class="card">
      <div class="card-head"><h3>检索结果 · {{ filtered.length }} 条</h3><button class="btn" @click="exportResult">导出结果</button></div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>会话</th><th>发送人</th><th>类型</th><th>内容（脱敏）</th><th>检测结果</th></tr></thead>
        <tbody>
          <tr v-for="(r, i) in filtered" :key="i">
            <td>{{ fmt(r.createdAt) }}</td><td>{{ r.convId }}</td><td>{{ r.fromUid }}</td>
            <td><span class="state info">{{ r.type }}</span></td><td>{{ r.content }}</td>
            <td><span class="state" :class="r.tag === '正常' ? 'ok' : 'warn'">{{ r.tag }}</span></td>
          </tr>
          <tr v-if="!filtered.length"><td colspan="6" class="loading">未找到匹配的消息</td></tr>
        </tbody>
      </table>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
