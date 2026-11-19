<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))
const data = ref({ list: [], total: 0 })
const depts = ref([])
const roles = ref([])
const f = ref({ kw: '', status: '', dept: '', includeSub: false })
const page = ref(1)
const size = 8
const loading = ref(false)
const modal = ref(null)

const params = () => ({
  kw: f.value.kw.trim(), status: f.value.status, deptId: f.value.dept,
  includeSub: f.value.includeSub ? 1 : '', page: page.value, size
})
async function load() {
  loading.value = true
  try {
    const [u, d, r] = await Promise.all([API.users(params()), API.depts(), API.roles()])
    data.value = u.data || { list: [], total: 0 }
    depts.value = d.data || []
    roles.value = r.data || []
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}
onMounted(load)

const deptName = id => depts.value.find(x => x.id === id)?.name || (id ? '未分配（无效部门）' : '未分配')
const roleText = u => (u.roleNames || []).length ? u.roleNames.join(' / ') : '普通成员'
const pages = computed(() => Math.max(1, Math.ceil((data.value.total || 0) / size)))

function query() { page.value = 1; load() }
function reset() { f.value = { kw: '', status: '', dept: '', includeSub: false }; page.value = 1; load() }
function go(p) { if (p < 1 || p > pages.value) return; page.value = p; load() }

const roleChecks = (u = {}) => roles.value.length
  ? roles.value.map(r => `<label class="role-chk"><input type="checkbox" class="role-id" value="${r.id}" ${(u.roleIds || []).includes(r.id) ? 'checked' : ''}> ${esc(r.name)}</label>`).join('')
  : '<span class="muted">暂无可选角色</span>'
const deptOptions = (cur = 0) => `<option value="0">未分配</option>` + depts.value.map(d => `<option value="${d.id}" ${d.id === +cur ? 'selected' : ''}>${esc(d.name)}</option>`).join('')
const pickedRoles = () => Array.from(document.querySelectorAll('.role-id:checked')).map(x => +x.value)
const val = id => (document.getElementById(id)?.value || '').trim()

function editUser(u) {
  modal.value.open({
    title: '编辑用户',
    body: `<div class="field"><label>姓名 / 昵称</label><input id="eu-nick" value="${esc(u.nickname)}"></div>
      <div class="field"><label>岗位</label><input id="eu-title" value="${esc(u.title)}" placeholder="如 产品经理"></div>
      <div class="field"><label>手机号</label><input id="eu-phone" value="${esc(u.phone)}" placeholder="选填"></div>
      <div class="field"><label>所属部门</label><select id="eu-dept">${deptOptions(u.deptId)}</select></div>
      <div class="field"><label>绑定角色（可多选，留空为普通成员）</label><div class="chk-grid">${roleChecks(u)}</div></div>`,
    okText: '保存',
    onOk: async () => {
      try {
        await API.updateUser(u.id, {
          nickname: val('eu-nick'), title: val('eu-title'), phone: val('eu-phone'),
          deptId: +document.getElementById('eu-dept').value, roleIds: pickedRoles()
        })
        toast('用户资料与角色已保存'); load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function addUser() {
  modal.value.open({
    title: '新增用户',
    body: `<div class="field"><label>登录账号</label><input id="au-name" placeholder="登录账号（唯一）"></div>
      <div class="field"><label>昵称</label><input id="au-nick" placeholder="显示名称"></div>
      <div class="field"><label>初始密码</label><input id="au-pwd" value="123456"></div>
      <div class="field"><label>岗位</label><input id="au-title" placeholder="选填"></div>
      <div class="field"><label>手机号</label><input id="au-phone" placeholder="选填"></div>
      <div class="field"><label>所属部门</label><select id="au-dept">${deptOptions(0)}</select></div>
      <div class="field"><label>绑定角色</label><div class="chk-grid">${roleChecks()}</div></div>`,
    okText: '创建',
    onOk: async () => {
      const username = val('au-name'), password = val('au-pwd')
      if (!username || !password) { toast('账号和密码不能为空', 'err'); return false }
      try {
        await API.createUser({
          username, password, nickname: val('au-nick') || username,
          title: val('au-title'), phone: val('au-phone'),
          deptId: +document.getElementById('au-dept').value, roleIds: pickedRoles()
        })
        toast('用户创建成功'); page.value = 1; load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function toggleStatus(u) {
  const enable = u.status !== 1
  modal.value.open({
    title: (enable ? '启用' : '禁用') + '账号',
    body: `<p>确认${enable ? '启用' : '禁用'}「${esc(u.nickname)}」？${enable ? '启用后该账号可正常登录。' : '禁用后其令牌立即失效，长连接不可再建立，操作记入审计日志。'}</p>`,
    okText: enable ? '启用' : '禁用', danger: !enable,
    onOk: async () => {
      try {
        if (enable) await API.enableUser(u.id); else await API.deleteUser(u.id)
        toast(enable ? '账号已启用' : '账号已禁用，已记审计'); load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function resetPwd(u) {
  modal.value.open({
    title: '重置密码', body: `<p>确认重置「${esc(u.nickname)}」的密码？</p>`, okText: '重置',
    onOk: async () => {
      try { const r = await API.resetPwd(u.id); toast('密码已重置为 ' + (r.data?.password || '123456')) }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
async function exportUsers() {
  try { await API.exportUsers(params()); toast('用户清单已导出') }
  catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div>
    <div class="proto-hint">管理平台用户、岗位与角色绑定；筛选 / 分页由服务端执行，权限变更即时生效。<span class="code" style="margin-left:8px">FR-ADMIN-02 / FR-RBAC-05</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="姓名 / 账号 / 部门 / 手机号" style="width:210px" @keyup.enter="query">
        <select v-model="f.status"><option value="">全部状态</option><option value="1">正常</option><option value="0">禁用</option></select>
        <select v-model="f.dept"><option value="">全部部门</option><option v-for="d in depts" :key="d.id" :value="d.id">{{ d.name }}</option></select>
        <label class="inline-chk"><input type="checkbox" v-model="f.includeSub"> 含下级部门</label>
        <button class="btn" @click="query">查询</button>
        <button class="btn" @click="reset">重置</button>
        <button class="btn" @click="exportUsers">导出 CSV</button>
        <button class="btn primary" @click="addUser">＋ 新增用户</button>
      </div>
      <table class="tbl">
        <thead><tr><th>用户</th><th>部门</th><th>岗位</th><th>角色</th><th>手机号</th><th>状态</th><th>最后登录</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="u in data.list" :key="u.id">
            <td><div class="u-cell"><span class="avatar sm" :style="{ background: u.avatar || '#4f6ef7' }">{{ u.nickname?.slice(0,1) }}</span><div><b>{{ u.nickname }}</b><div class="sr-desc">@{{ u.username }}</div></div></div></td>
            <td>{{ u.deptName || deptName(u.deptId) }}</td>
            <td>{{ u.title || '—' }}</td>
            <td><span class="tag">{{ roleText(u) }}</span></td>
            <td><span class="code">{{ u.phone || '—' }}</span></td>
            <td><span class="state" :class="u.status === 1 ? 'ok' : 'off'">{{ u.status === 1 ? '正常' : '已禁用' }}</span></td>
            <td>{{ u.lastLoginAt ? new Date(u.lastLoginAt).toLocaleString() : '—' }}</td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="editUser(u)">编辑</button>
              <button class="btn sm" @click="resetPwd(u)">重置密码</button>
              <button class="btn sm" :class="u.status === 1 ? 'danger' : 'ghost'" @click="toggleStatus(u)">{{ u.status === 1 ? '禁用' : '启用' }}</button>
            </td>
          </tr>
          <tr v-if="!data.list.length"><td colspan="8" class="loading">{{ loading ? '加载中…' : '未找到匹配的用户' }}</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ data.total }} 条
        <button :disabled="page === 1" @click="go(page - 1)">‹</button>
        <button v-for="p in pages" :key="p" :class="{ on: page === p }" @click="go(p)">{{ p }}</button>
        <button :disabled="page === pages" @click="go(page + 1)">›</button>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
<style scoped>
.inline-chk { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: var(--ink2); }
.chk-grid { display: flex; flex-wrap: wrap; gap: 8px 14px; }
.chk-grid :deep(.role-chk) { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; }
</style>
