<script setup>
import { ref, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

const f = ref({ kw: '', module: '', date: '' })
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = 20

const modules = [
  ['user', '用户管理'], ['role', '角色权限'], ['rbac', '角色权限'], ['audit', '消息审计'],
  ['sensitive', '内容安全'], ['setting', '系统设置'], ['group', '群组管理'],
  ['rtc', '音视频'], ['log', '登录/日志'], ['notice', '公告']
]
const actionText = {
  'user:create': '新增用户', 'user:update': '编辑用户', 'user:disable': '禁用用户',
  'user:enable': '启用用户', 'user:reset': '重置密码', 'user:export': '导出用户',
  'role:update': '更新角色权限', 'rbac:role:grant': '分配用户角色',
  'audit:message:export': '导出消息审计', 'sensitive:add': '新增敏感词',
  'sensitive:delete': '删除敏感词', 'sensitive:import': '导入敏感词库',
  'sensitive:strategy': '切换敏感词策略', 'setting:config': '保存系统设置',
  'group:freeze': '冻结群组', 'group:unfreeze': '解冻群组', 'group:dissolve': '解散群组',
  'group:export': '导出群组', 'rtc:record:manage': '强制挂断通话',
  'rtc:record:export': '导出通话记录', 'log:export': '导出操作日志',
  'log:login:export': '导出登录日志', 'log:login:ban': '封禁 IP', 'notice:publish': '发布公告'
}
function moduleOf(action) {
  const hit = modules.find(m => String(action).startsWith(m[0]))
  return hit ? hit[1] : String(action).split(':')[0]
}
function actText(a) { return actionText[a] || a }
function fmt(ts) { return ts ? new Date(ts).toLocaleString() : '—' }

async function load(reset = false) {
  if (reset) page.value = 1
  try {
    const r = await API.audit({ ...params(), size })
    let list = r.data?.list || []
    if (f.value.date) list = list.filter(l => new Date(l.createdAt).toISOString().slice(0, 10) === f.value.date)
    rows.value = list
    total.value = r.data?.total || 0
  } catch (e) { toast(e.message, 'err') }
}
function params() { return { kw: f.value.kw.trim(), module: f.value.module, page: page.value } }
onMounted(() => load())
async function exportCsv() {
  try { await API.exportAudit(params()); toast('导出已完成') } catch (e) { toast(e.message, 'err') }
}
const pages = () => Math.max(1, Math.ceil(total.value / size))
function go(p) { page.value = Math.min(pages(), Math.max(1, p)); load() }
</script>
<template>
  <div>
    <div class="proto-hint">审计日志仅追加写入，应用账号无 UPDATE / DELETE 权限，保留 ≥ 180 天。<span class="code" style="margin-left:8px">FR-SEC-01 / FR-ADMIN-08</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="操作人 / 关键词" style="width:210px" @keyup.enter="load(true)">
        <select v-model="f.module" @change="load(true)"><option value="">全部模块</option><option v-for="m in modules" :key="m[0]" :value="m[0]">{{ m[1] }}</option></select>
        <input type="date" v-model="f.date" @change="load(true)">
        <button class="btn" @click="load(true)">查询</button>
        <span class="sp"></span>
        <button class="btn" @click="exportCsv">导出日志</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>操作人</th><th>模块</th><th>操作</th><th>对象</th><th>详情</th></tr></thead>
        <tbody>
          <tr v-for="l in rows" :key="l.id">
            <td>{{ fmt(l.createdAt) }}</td><td>{{ l.uid }}</td><td><span class="code">{{ moduleOf(l.action) }}</span></td>
            <td>{{ actText(l.action) }}</td><td>{{ l.target || '—' }}</td><td class="muted">{{ l.detail || '—' }}</td>
          </tr>
          <tr v-if="!rows.length"><td colspan="6" class="loading">暂无审计记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ total }} 条
        <button :disabled="page <= 1" @click="go(page - 1)">‹</button><button class="on">{{ page }}</button>
        <button :disabled="page >= pages()" @click="go(page + 1)">›</button>
        <span class="muted" style="margin-left:8px">/ {{ pages() }} 页</span>
      </div>
    </div>
  </div>
</template>
