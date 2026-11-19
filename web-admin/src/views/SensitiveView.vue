<script setup>
import { ref, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const words = ref([])
const input = ref('')
const strategy = ref('block')
const modal = ref(null)
const hits = ref([])
const hitTotal = ref(0)
const hitPage = ref(1)
const hitSize = 20
const strategies = [
  ['block', '拦截并提示发送方', '命中后消息不投递，向发送方提示内容违规'],
  ['mask', '替换为 * 后放行', '敏感片段打码为 *，消息正常投递'],
  ['alert', '仅告警留痕', '消息正常投递，仅记录命中日志并通知管理员']
]
const actionText = { block: '拦截', mask: '替换', alert: '告警' }

async function load() {
  try {
    const [w, s] = await Promise.all([API.sensitive(), API.sensitiveStrategy()])
    words.value = w.data || []
    strategy.value = s.data?.strategy || 'block'
    loadHits()
  } catch (e) { toast(e.message, 'err') }
}
async function loadHits() {
  try {
    const r = await API.sensitiveHits(hitPage.value, hitSize)
    hits.value = r.data?.list || []
    hitTotal.value = r.data?.total || 0
  } catch (e) { toast(e.message, 'err') }
}
onMounted(load)

async function add() {
  const w = input.value.trim(); if (!w) return
  try { await API.addSensitive(w); input.value = ''; toast(`敏感词已添加，词库 ${words.value.length + 1} 条并实时生效`); load() } catch (e) { toast(e.message, 'err') }
}
async function del(w) {
  modal.value.open({
    title: '删除敏感词', body: `<p>确认从词库移除「${w}」？</p>`, okText: '删除', danger: true,
    onOk: async () => {
      try { await API.deleteSensitive(w); toast('敏感词已移除并实时生效'); load() } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
async function setStrategy(s) {
  if (strategy.value === s[0]) return
  try { await API.setSensitiveStrategy(s[0]); strategy.value = s[0]; toast('命中策略已切换为：' + s[1]) }
  catch (e) { toast(e.message, 'err') }
}
function importWords() {
  modal.value.open({
    title: '导入词库',
    body: '<p>支持 .txt / .csv 文件，每行或逗号分隔一个敏感词。</p><div class="field"><label>选择文件</label><input type="file" id="sw-file" accept=".txt,.csv"></div><div class="muted">导入后立即生效并同步到所有节点。</div>',
    okText: '导入',
    onOk: async () => {
      const file = document.getElementById('sw-file')?.files?.[0]
      if (!file) { toast('请选择文件', 'err'); return false }
      try {
        const text = await file.text()
        const r = await API.bulkSensitive(text)
        toast(`词库导入成功，新增 ${r.data?.imported ?? 0} 条`)
        load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function fmt(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  return `${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
function clearHits() {
  modal.value.open({
    title: '清空命中记录', body: '<p>确认清空全部敏感词命中记录？该操作不可恢复。</p>', okText: '清空', danger: true,
    onOk: async () => {
      try { await API.clearSensitiveHits(); hitPage.value = 1; loadHits(); toast('命中记录已清空') } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function pageGo(p) {
  const pages = Math.max(1, Math.ceil(hitTotal.value / hitSize))
  hitPage.value = Math.min(pages, Math.max(1, p))
  loadHits()
}
</script>
<template>
  <div>
    <div class="proto-hint">敏感词按策略处理，词库写入 Redis 后实时生效，命中记录落库可审计。<span class="code" style="margin-left:8px">FR-MSG-22 / FR-ADMIN-07</span></div>
    <div class="card">
      <div class="card-head"><h3>敏感词库 · {{ words.length }} 条</h3><button class="btn sm" @click="importWords">📥 导入词库</button></div>
      <div class="toolbar"><input v-model="input" placeholder="输入敏感词" style="width:260px" @keyup.enter="add"><button class="btn primary" @click="add">添加敏感词</button></div>
      <div>
        <span v-for="w in words" :key="w" class="word" style="display:inline-flex;align-items:center;gap:6px">{{ w }}<button class="word-del" @click="del(w)">✕</button></span>
        <span v-if="!words.length" class="muted">暂无敏感词</span>
      </div>
    </div>
    <div class="card">
      <h3>命中策略</h3>
      <div class="csub">选择敏感词命中后的处理动作（三选一，实时生效）</div>
      <div v-for="s in strategies" :key="s[0]" class="strategy-item" :class="{ on: strategy === s[0] }" @click="setStrategy(s)">
        <div><b>{{ s[1] }}</b><div class="sr-desc">{{ s[2] }}</div></div>
        <span class="state" :class="strategy === s[0] ? 'ok' : 'off'">{{ strategy === s[0] ? '当前策略' : '选择' }}</span>
      </div>
    </div>
    <div class="card">
      <div class="card-head"><h3>最近命中记录</h3><button class="btn sm" @click="clearHits">清空</button></div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>发送人</th><th>会话</th><th>命中词</th><th>处理动作</th><th>内容</th></tr></thead>
        <tbody>
          <tr v-for="h in hits" :key="h.id">
            <td>{{ fmt(h.createdAt) }}</td><td>{{ h.uid }}</td><td><span class="code">{{ h.convId }}</span></td>
            <td>{{ h.word }}</td><td><span class="state info">{{ actionText[h.action] || h.action }}</span></td>
            <td class="muted" style="max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">{{ h.content }}</td>
          </tr>
          <tr v-if="!hits.length"><td colspan="6" class="loading">暂无命中记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ hitTotal }} 条
        <button :disabled="hitPage <= 1" @click="pageGo(hitPage - 1)">‹</button>
        <button class="on">{{ hitPage }}</button>
        <button :disabled="hitPage >= Math.ceil(hitTotal / hitSize) || !hitTotal" @click="pageGo(hitPage + 1)">›</button>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
