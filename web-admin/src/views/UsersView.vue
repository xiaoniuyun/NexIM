<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const data = ref({ list: [], total: 0 })
const depts = ref([])
const roles = ref([])
const f = ref({ kw: '', status: '', dept: '' })
const page = ref(1)
const modal = ref(null)

async function load() {
  try {
    const [u, d, r] = await Promise.all([API.users(), API.depts(), API.roles()])
    data.value = u.data || { list: [], total: 0 }
    depts.value = d.data || []
    roles.value = r.data || []
  } catch (e) { toast(e.message, 'err') }
}
onMounted(load)

const roleName = uid => {
  const r = roles.value.find(x => x.uid === uid || x.code === uid)
  return r ? r.name : (uid ? uid : '普通成员')
}
const deptName = id => depts.value.find(x => x.id === id)?.name || (id ? '部门' + id : '未分配')

const filtered = computed(() => data.value.list.filter(u => {
  const q = f.value.kw.trim().toLowerCase()
  const okKw = !q || (u.nickname || '').toLowerCase().includes(q) || (u.username || '').toLowerCase().includes(q) || deptName(u.deptId).includes(q)
  const okStatus = !f.value.status || (f.value.status === '正常' ? u.status === 1 : u.status !== 1)
  const okDept = !f.value.dept || u.deptId === +f.value.dept
  return okKw && okStatus && okDept
}))
const paged = computed(() => filtered.value.slice((page.value - 1) * 8, page.value * 8))
const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / 8)))

function query() { page.value = 1; toast('已按条件检索用户') }
function reset() { f.value = { kw: '', status: '', dept: '' }; page.value = 1 }

function editUser(u) {
  modal.value.open({
    title: '编辑用户',
    body: `<div class="field"><label>姓名 / 昵称</label><input id="eu-nick" value="${u.nickname || ''}"></div>
      <div class="field"><label>所属部门</label><select id="eu-dept">${depts.value.map(d => `<option value="${d.id}" ${d.id === u.deptId ? 'selected' : ''}>${d.name}</option>`).join('')}</select></div>
      <div class="field"><label>绑定角色</label><select id="eu-role"><option value="">普通成员</option>${roles.value.map(r => `<option value="${r.id}">${r.name}</option>`).join('')}</select></div>`,
    okText: '保存',
    onOk: async () => {
      const nickname = document.getElementById('eu-nick').value.trim()
      const deptId = +document.getElementById('eu-dept').value || 0
      const roleId = document.getElementById('eu-role').value
      try {
        await API.updateUser(u.id, { nickname, dept_id: deptId })
        if (roleId) await API.grantRole({ uid: u.uid, roleId: +roleId })
        toast('用户资料已保存'); load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function addUser() {
  modal.value.open({
    title: '新增用户',
    body: '<div class="field"><label>姓名 / 账号名</label><input id="au-name" placeholder="登录账号"></div><div class="field"><label>昵称</label><input id="au-nick" placeholder="显示名称"></div><div class="field"><label>初始角色</label><select id="au-role"><option value="">普通成员</option>' + roles.value.map(r => `<option value="${r.id}">${r.name}</option>`).join('') + '</select></div><div class="field"><label>初始密码</label><input id="au-pwd" value="123456"></div>',
    okText: '创建',
    onOk: async () => {
      const username = document.getElementById('au-name').value.trim()
      const nickname = document.getElementById('au-nick').value.trim()
      const password = document.getElementById('au-pwd').value.trim()
      const roleId = document.getElementById('au-role').value
      if (!username || !password) { toast('账号和密码不能为空', 'err'); return false }
      try {
        const r = await API.createUser({ username, nickname, password })
        if (roleId) await API.grantRole({ uid: r.data?.uid || username, roleId: +roleId })
        toast('用户创建成功'); load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function toggleStatus(u) {
  const enable = u.status !== 1
  modal.value.open({
    title: (enable ? '启用' : '禁用') + '账号',
    body: `<p>确认${enable ? '启用' : '禁用'}「${u.nickname}」？${enable ? '启用后该账号可正常登录。' : '禁用后长连接将断开，操作记入审计日志。'}</p>`,
    okText: enable ? '启用' : '禁用', danger: !enable,
    onOk: async () => {
      try {
        if (enable) await API.enableUser(u.id); else await API.deleteUser(u.id)
        toast(enable ? '账号已启用，权限即时生效' : '账号已禁用，长连接断开，已记审计'); load()
      } catch (e) { toast(e.message, 'err') }
    }
  })
}
function resetPwd(u) {
  modal.value.open({
    title: '重置密码', body: `<p>确认重置「${u.nickname}」的密码？</p>`, okText: '重置',
    onOk: async () => { try { const r = await API.resetPwd(u.id); toast('密码已重置为 ' + (r.data?.password || '123456')) } catch (e) { toast(e.message, 'err') } }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">管理平台用户与角色绑定，影响登录与数据访问范围。<span class="code" style="margin-left:8px">FR-ADMIN-02 / FR-RBAC-05</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="姓名 / 账号 / 部门" style="width:200px" @keyup.enter="query">
        <select v-model="f.status"><option value="">全部状态</option><option>正常</option><option>禁用</option></select>
        <select v-model="f.dept"><option value="">全部部门</option><option v-for="d in depts" :key="d.id" :value="d.id">{{ d.name }}</option></select>
        <button class="btn" @click="query">查询</button>
        <button class="btn" @click="reset">重置</button>
        <button class="btn" @click="toast('导出任务已提交（演示）')">导出</button>
        <button class="btn primary" @click="addUser">＋ 新增用户</button>
      </div>
      <table class="tbl">
        <thead><tr><th>用户</th><th>部门</th><th>角色</th><th>状态</th><th>最后登录</th><th>认证</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="u in paged" :key="u.id">
            <td><div class="u-cell"><span class="avatar sm" :style="{ background: u.avatar || '#4f6ef7' }">{{ u.nickname?.slice(0,1) }}</span><div><b>{{ u.nickname }}</b><div class="sr-desc">@{{ u.username }}</div></div></div></td>
            <td>{{ deptName(u.deptId) }}</td>
            <td><span class="tag">{{ roleName(u.roles || u.role) }}</span></td>
            <td><span class="state" :class="u.status === 1 ? 'ok' : 'off'">{{ u.status === 1 ? '正常' : u.status === 3 ? '已禁用' : '禁用' }}</span></td>
            <td>{{ u.lastLoginAt ? new Date(u.lastLoginAt).toLocaleString() : '—' }}</td>
            <td><span class="state ok">已认证</span></td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="editUser(u)">编辑</button>
              <button class="btn sm" @click="resetPwd(u)">重置密码</button>
              <button class="btn sm" :class="u.status === 1 ? 'danger' : 'ghost'" @click="toggleStatus(u)">{{ u.status === 1 ? '禁用' : '启用' }}</button>
            </td>
          </tr>
          <tr v-if="!paged.length"><td colspan="7" class="loading">未找到匹配的用户</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ filtered.length }} 条 <button :disabled="page === 1" @click="page--">‹</button><button v-for="p in pages" :key="p" :class="{ on: page === p }" @click="page = p">{{ p }}</button><button :disabled="page === pages" @click="page++">›</button></div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
