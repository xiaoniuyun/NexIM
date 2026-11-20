<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import { useAuthStore } from '../stores/auth'
import ModalBox from '../components/ModalBox.vue'

const auth = useAuthStore()
const canEdit = computed(() => auth.can('dept:edit'))

const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))
const tree = ref([])
const allDepts = ref([])
const allUsers = ref([])
const members = ref([])
const selected = ref(null)
const modal = ref(null)

function flatten(nodes, depth = 0, out = []) {
  for (const n of nodes || []) {
    out.push({ ...n, depth })
    if (n.children?.length) flatten(n.children, depth + 1, out)
  }
  return out
}
const rows = computed(() => flatten(tree.value))
const selectedDept = computed(() => allDepts.value.find(d => d.id === selected.value) || null)
const countOf = id => rows.value.find(r => r.id === id)?.memberCount || 0

async function load() {
  try {
    const [t, d, u] = await Promise.all([API.deptTree(), API.depts(), API.users({ size: 1000 })])
    tree.value = t.data || []
    allDepts.value = d.data || []
    allUsers.value = u.data?.list || []
    await pick(selected.value)
  } catch (e) { toast(e.message, 'err') }
}
onMounted(load)
async function pick(id) {
  selected.value = id
  try { const r = await API.deptMembers(id || 0); members.value = r.data || [] }
  catch (e) { toast(e.message, 'err') }
}

const parentOptions = (selfId = 0, rootLabel = '顶级部门') => `<option value="0">${rootLabel}</option>` +
  allDepts.value.filter(d => d.id !== selfId).map(d => `<option value="${d.id}">${esc(d.name)}</option>`).join('')
const leaderOptions = (cur = '') => `<option value="">未指定</option>` +
  allUsers.value.map(u => `<option value="${u.uid}" ${u.uid === cur ? 'selected' : ''}>${esc(u.nickname)} (@${esc(u.username)})</option>`).join('')
const val = id => (document.getElementById(id)?.value || '').trim()

function deptForm(d = {}) {
  return `<div class="field"><label>部门名称</label><input id="df-name" value="${esc(d.name)}" placeholder="输入部门名称"></div>
    <div class="field"><label>上级部门（可调整层级）</label><select id="df-parent">${parentOptions(d.id || 0)}</select></div>
    <div class="field"><label>负责人</label><select id="df-leader">${leaderOptions(d.leaderUid)}</select></div>
    <div class="field"><label>排序值</label><input id="df-sort" type="number" value="${d.sort || 0}"></div>
    <div class="field"><label>状态</label><select id="df-status"><option value="1" ${d.status !== 2 ? 'selected' : ''}>启用</option><option value="2" ${d.status === 2 ? 'selected' : ''}>停用</option></select></div>
    <div class="sr-desc">调整上级后，本部及所有下级的物化路径会自动重算。</div>`
}
function formBody() {
  return {
    name: val('df-name'), parentId: +document.getElementById('df-parent').value,
    leaderUid: val('df-leader'), sort: +document.getElementById('df-sort').value || 0,
    status: +document.getElementById('df-status').value
  }
}
function addDept(parentId = 0) {
  modal.value.open({
    title: parentId ? '新增子部门' : '新增部门', body: deptForm(), okText: '创建',
    onOk: async () => {
      const body = formBody()
      if (!body.name) { toast('请输入部门名称', 'err'); return false }
      try { await API.createDept(body); toast(`部门「${body.name}」已创建`); await load() }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
  setTimeout(() => { const el = document.getElementById('df-parent'); if (el) el.value = String(parentId) }, 0)
}
function editDept(d) {
  const html = deptForm(d)
  modal.value.open({
    title: '编辑部门', body: html, okText: '保存',
    onOk: async () => {
      const body = formBody()
      if (!body.name) { toast('请输入部门名称', 'err'); return false }
      try {
        await API.updateDept(d.id, body)
        toast('部门信息已保存'); await load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
  // 回填当前上级部门（表单按「排除自身」生成，需显式选中）
  setTimeout(() => { const el = document.getElementById('df-parent'); if (el) el.value = String(d.parentId || 0) }, 0)
}
function deleteDept(d) {
  modal.value.open({
    title: '删除部门', danger: true, okText: '删除',
    body: `<p>确认删除「${esc(d.name)}」？存在子部门或成员时无法删除。</p>`,
    onOk: async () => {
      try { await API.deleteDept(d.id); if (selected.value === d.id) selected.value = null; toast('部门已删除'); await load() }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function moveMember(u) {
  modal.value.open({
    title: '调整成员部门',
    body: `<p>将「${esc(u.nickname)}」调整到：</p><div class="field"><select id="mm-dept">${parentOptions(0, '未分配部门')}</select></div>`,
    okText: '保存',
    onOk: async () => {
      const deptId = +document.getElementById('mm-dept').value
      try { await API.updateUser(u.id, { deptId }); toast('成员部门已更新'); await load() }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
  setTimeout(() => { const el = document.getElementById('mm-dept'); if (el) el.value = String(u.deptId || 0) }, 0)
}
function addMembers() {
  if (!selected.value) { toast('请先在左侧选择部门', 'err'); return }
  const candidates = allUsers.value.filter(u => u.deptId !== selected.value)
  if (!candidates.length) { toast('暂无可加入的用户', 'err'); return }
  const items = candidates.map(u => `<label class="role-chk"><input type="checkbox" class="pick-uid" value="${esc(u.uid)}"> ${esc(u.nickname)} <span class="muted">${esc(u.deptName || '未分配')}</span></label>`).join('')
  modal.value.open({
    title: `添加成员到「${selectedDept.value?.name || ''}」`,
    body: `<div class="field"><label>选择用户（可多选，加入后将调整其所属部门）</label><div class="chk-grid">${items}</div></div>`,
    okText: '加入部门',
    onOk: async () => {
      const uids = Array.from(document.querySelectorAll('.pick-uid:checked')).map(x => x.value)
      if (!uids.length) { toast('请选择要加入的成员', 'err'); return false }
      try {
        const r = await API.addDeptMembers(selected.value, uids)
        toast(`已加入 ${r.data?.added ?? uids.length} 名成员`); await load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function removeMember(u) {
  modal.value.open({
    title: '移出部门', danger: true, okText: '移出',
    body: `<p>确认将「${esc(u.nickname)}」移出「${esc(u.deptName || '本部门')}」？移出后其部门变为未分配。</p>`,
    onOk: async () => {
      try { await API.removeDeptMember(u.deptId, u.uid); toast('已移出部门'); await load() }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
const scopeHint = computed(() => selectedDept.value ? `dept.path LIKE '${selectedDept.value.path}%'` : '不过滤部门（全部数据）')
</script>
<template>
  <div>
    <div class="proto-hint">部门为树形结构，支持新增 / 编辑 / 停用 / 删除 / 调整层级；物化路径自动维护，成员查询可含所有下级。<span class="code" style="margin-left:8px">FR-CONTACT-01 / FR-ADMIN-03</span></div>
    <div class="split">
      <div class="card">
        <div class="card-head"><h3>组织架构树</h3><button class="btn sm primary" @click="addDept">＋ 新增部门</button></div>
        <div class="csub">共 {{ rows.length }} 个部门 · 点击部门查看成员</div>
        <div class="tree-row" :class="{ on: selected === null }" @click="pick(null)">🏢 全部部门 <span class="tr-c">{{ allUsers.length }}</span></div>
        <div v-for="d in rows" :key="d.id" class="tree-row" :class="{ on: selected === d.id, off: d.status === 2 }"
          :style="{ paddingLeft: 12 + d.depth * 16 + 'px' }" @click="pick(d.id)">
          📁 {{ d.name }} <span class="tr-c">{{ countOf(d.id) }}</span>
          <span v-if="d.status === 2" class="state off">停用</span>
          <span class="tree-ops">
            <button class="btn sm" @click.stop="addDept(d.id)">＋子部门</button>
            <button class="btn sm" @click.stop="editDept(d)">编辑</button>
            <button class="btn sm danger" @click.stop="deleteDept(d)">删除</button>
          </span>
        </div>
        <div v-if="!rows.length" class="loading">暂无部门，请先新增</div>
      </div>
      <div>
        <div class="card">
          <div class="card-head">
            <h3>{{ selectedDept ? selectedDept.name : '全部部门' }} · 成员</h3>
            <div class="head-actions">
              <span class="muted">直属 {{ selectedDept ? countOf(selectedDept.id) : allUsers.length }} 人 · 当前列表 {{ members.length }} 人（含下级）</span>
              <button class="btn sm primary" :disabled="!selectedDept || !canEdit" @click="addMembers">＋ 添加成员</button>
            </div>
          </div>
          <table class="tbl">
            <thead><tr><th>姓名</th><th>部门</th><th>岗位</th><th>手机号</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="u in members" :key="u.uid">
                <td><div class="u-cell"><span class="avatar sm" :style="{ background: u.avatar || '#4f6ef7' }">{{ u.nickname?.slice(0,1) }}</span>{{ u.nickname }}</div></td>
                <td>{{ u.deptName || '未分配' }}</td>
                <td>{{ u.title || '—' }}</td>
                <td><span class="code">{{ u.phone || '—' }}</span></td>
                <td><span class="state" :class="u.online ? 'ok' : 'off'">{{ u.online ? '在线' : '离线' }}</span></td>
                <td>
                  <button class="btn sm" @click="moveMember(u)">调整部门</button>
                  <button class="btn sm danger" :disabled="!canEdit" @click="removeMember(u)">移出</button>
                </td>
              </tr>
              <tr v-if="!members.length"><td colspan="6" class="loading">该范围暂无成员</td></tr>
            </tbody>
          </table>
        </div>
        <div class="card">
          <h3>数据范围示例</h3>
          <div class="csub">选择「本部门及下级」时，按物化路径自动扩展至所有子孙部门</div>
          <div class="field"><label>当前部门物化路径</label><div><span class="code">{{ selectedDept?.path || '全部部门' }}</span></div></div>
          <div class="field"><label>自动注入的查询条件</label><div><span class="code">{{ scopeHint }}</span></div></div>
          <div class="muted">负责人：{{ allUsers.find(u => u.uid === selectedDept?.leaderUid)?.nickname || '未指定' }}</div>
        </div>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
<style scoped>
.tree-row { display: flex; align-items: center; gap: 6px; }
.tree-row.off { opacity: .6; }
.tree-ops { margin-left: auto; display: none; gap: 4px; }
.tree-row:hover .tree-ops { display: flex; }
.head-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.chk-grid { display: flex; flex-wrap: wrap; gap: 8px 14px; max-height: 260px; overflow: auto; }
.chk-grid :deep(.role-chk) { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; }
</style>
