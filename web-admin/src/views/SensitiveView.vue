<script setup>
import { ref, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const words = ref([])
const input = ref('')
const version = ref('v8')
const strategy = ref('block')
const modal = ref(null)
const strategies = [
  ['block', '拦截并提示发送方', '命中后消息不投递，向发送方提示内容违规'],
  ['mask', '替换为 * 后放行', '敏感片段打码为 *，消息正常投递'],
  ['alert', '仅告警留痕', '消息正常投递，仅记录命中日志并通知管理员']
]
const hits = [
  { t: '09:18:45', from: '访客_3021', conv: '外部协作群', word: '违禁词A', action: '拦截', status: '已处理' },
  { t: '08:42:11', from: '外部联系人', conv: '张伟 ↔ 外部', word: '敏感词B', action: '替换', status: '已处理' },
  { t: '昨天 21:30', from: 'chenjing', conv: '市场部群', word: '广告词C', action: '告警', status: '待复核' },
  { t: '昨天 16:05', from: '访客_3021', conv: '外部协作群', word: '违禁词A', action: '拦截', status: '已处理' }
]
async function load() { try { words.value = (await API.sensitive()).data || [] } catch (e) { toast(e.message, 'err') } }
onMounted(load)
function bump() { version.value = 'v' + (parseInt(version.value.slice(1)) + 1) }
async function add() {
  const w = input.value.trim(); if (!w) return
  try { await API.addSensitive(w); input.value = ''; bump(); toast(`敏感词已添加，词库 ${version.value} 热更新`); load() } catch (e) { toast(e.message, 'err') }
}
async function del(w) {
  modal.value.open({
    title: '删除敏感词', body: `<p>确认从词库移除「${w}」？</p>`, okText: '删除', danger: true,
    onOk: () => { words.value = words.value.filter(x => x !== w); bump(); toast(`敏感词已移除，词库 ${version.value} 热更新`) }
  })
}
function setStrategy(s) { strategy.value = s; toast('命中策略已切换为：' + s[1]) }
function importWords() {
  modal.value.open({
    title: '导入词库',
    body: '<p>支持 .txt / .csv 文件，每行一个敏感词。</p><div class="field"><label>选择文件</label><input type="file" id="sw-file" accept=".txt,.csv"></div><div class="muted">导入后词库版本号自动升级并热更新至所有节点。</div>',
    okText: '导入',
    onOk: () => { if (!document.getElementById('sw-file').value) { toast('请选择文件', 'err'); return false } bump(); toast(`词库导入成功，版本 ${version.value} 热更新`) }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">敏感词按策略处理，词库支持热更新，无需重启服务。<span class="code" style="margin-left:8px">FR-MSG-22 / FR-ADMIN-07</span></div>
    <div class="card">
      <div class="card-head"><h3>敏感词库 · 版本 {{ version }}</h3><button class="btn sm" @click="importWords">📥 导入词库</button></div>
      <div class="toolbar"><input v-model="input" placeholder="输入敏感词" style="width:260px" @keyup.enter="add"><button class="btn primary" @click="add">添加敏感词</button></div>
      <div>
        <span v-for="w in words" :key="w" class="word" style="display:inline-flex;align-items:center;gap:6px">{{ w }}<button class="word-del" @click="del(w)">✕</button></span>
        <span v-if="!words.length" class="muted">暂无敏感词</span>
      </div>
    </div>
    <div class="card">
      <h3>命中策略</h3>
      <div class="csub">选择敏感词命中后的处理动作（三选一）</div>
      <div v-for="s in strategies" :key="s[0]" class="strategy-item" :class="{ on: strategy === s[0] }" @click="setStrategy(s)">
        <div><b>{{ s[1] }}</b><div class="sr-desc">{{ s[2] }}</div></div>
        <span class="state" :class="strategy === s[0] ? 'ok' : 'off'">{{ strategy === s[0] ? '当前策略' : '选择' }}</span>
      </div>
    </div>
    <div class="card">
      <h3>最近命中记录</h3>
      <table class="tbl">
        <thead><tr><th>时间</th><th>发送人</th><th>会话</th><th>命中词</th><th>处理动作</th><th>状态</th></tr></thead>
        <tbody>
          <tr v-for="(h, i) in hits" :key="i"><td>{{ h.t }}</td><td>{{ h.from }}</td><td>{{ h.conv }}</td><td>{{ h.word }}</td><td><span class="state info">{{ h.action }}</span></td><td><span class="state" :class="h.status === '已处理' ? 'ok' : 'warn'">{{ h.status }}</span></td></tr>
        </tbody>
      </table>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
