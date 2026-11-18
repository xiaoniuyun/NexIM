<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { API } from '../api'
import { toast } from '../toast'

const router = useRouter()
const auth = useAuthStore()
const visible = ref(false)
const q = ref('')
const list = ref([])
const input = ref(null)

const menu = [
  ['dashboard', '工作台', '概览', '▦'], ['users', '用户管理', '用户与组织', '👥'],
  ['depts', '组织架构', '用户与组织', '🏢'], ['groups', '群组管理', '用户与组织', '💬'],
  ['login-logs', '登录日志', '用户与组织', '📋'], ['roles', '角色权限', '权限治理', '🛡️'],
  ['perm-preview', '权限预览', '权限治理', '🔑'], ['logs', '操作日志', '权限治理', '📜'],
  ['audit', '消息审计', '内容与安全', '👁️'], ['sensitive', '内容安全', '内容与安全', '🚩'],
  ['notices', '公告管理', '内容与安全', '🔔'], ['calls', '通话记录', '音视频', '📞'],
  ['rtc-quality', '质量监控', '音视频', '📊'], ['settings', '系统设置', '系统', '⚙️']
]
const groups = { conv: '会话', user: '用户', admin: '后台页面', cmd: '操作' }

function open() {
  visible.value = true; q.value = ''
  nextTick(() => input.value?.focus())
  render()
}
function close() { visible.value = false }
function render() {
  const s = q.value.trim().toLowerCase()
  const out = []
  if (!s || '用户管理'.includes(s)) {
    // 示例会话
    ;[['产品评审群', '群聊 · 32 人'], ['林晓', '好的，收到！'], ['王敏', '[文件] 需求文档.docx']]
      .filter(x => !s || x[0].toLowerCase().includes(s))
      .forEach(x => out.push({ g: groups.conv, icon: '💬', t: x[0], s: x[1], k: '打开', act: 'conv' }))
    ;[['张伟', '@zhangwei · 研发中心'], ['赵琳', '@zhaolin · 产品部'], ['陈静', '@chenjing · 市场部']]
      .filter(x => !s || x[0].toLowerCase().includes(s) || x[1].includes(s))
      .forEach(x => out.push({ g: groups.user, icon: '👤', t: x[0], s: x[1], k: '查看', act: 'user' }))
  }
  menu.filter(m => !s || m[1].toLowerCase().includes(s) || m[2].includes(s))
    .forEach(m => out.push({ g: groups.admin, icon: m[3], t: m[1], s: m[2], k: '打开', act: 'page', id: m[0] }))
  ;[['🌙 切换到深色主题', 'cmd-theme'], ['☀️ 切换到浅色主题', 'cmd-theme'], ['📞 发起语音通话', 'cmd-call'], ['🎬 发起视频通话', 'cmd-call']]
    .filter(x => !s || x[0].includes(s)).forEach(x => out.push({ g: groups.cmd, icon: '⚙️', t: x[0], s: '操作', k: '', act: x[1] }))
  list.value = out
}
function act(item) {
  close()
  if (item.act === 'page') router.push('/' + item.id)
  else if (item.act === 'cmd-theme') toggleTheme()
  else if (item.act === 'cmd-call') toast(item.t + '（演示）')
  else toast('已打开：' + item.t)
}
function toggleTheme() {
  const dark = document.body.classList.toggle('admin-dark')
  toast(dark ? '已切换到深色主题' : '已切换到浅色主题')
}
onMounted(() => {
  document.addEventListener('keydown', e => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); open() }
    if (e.key === 'Escape') close()
  })
})
defineExpose({ open, close, toggleTheme })
</script>
<template>
  <div v-if="visible" class="cmd-mask" @click.self="close">
    <div class="cmd-panel">
      <div class="cmd-input">🔍<input ref="input" v-model="q" placeholder="搜索会话 / 用户 / 后台页面 / 操作" @input="render"><kbd>ESC</kbd></div>
      <div class="cmd-list">
        <template v-for="(item, i) in list" :key="i">
          <div v-if="i === 0 || list[i-1].g !== item.g" class="cmd-group">{{ item.g }}</div>
          <button class="cmd-item" @click="act(item)">
            <span class="ci-icon">{{ item.icon }}</span>
            <div class="ct"><div class="t">{{ item.t }}</div><div class="s">{{ item.s }}</div></div>
            <span class="ck">{{ item.k }}</span>
          </button>
        </template>
        <div v-if="!list.length" class="loading">无匹配结果</div>
      </div>
    </div>
  </div>
</template>
