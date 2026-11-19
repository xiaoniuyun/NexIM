<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))
const roles = ref([])
const catalog = ref([])
const scopes = ref([])
const selectedId = ref(null)
const checked = ref(new Set())
const modal = ref(null)
const saving = ref(false)

const selected = computed(() => roles.value.find(r => r.id === selectedId.value) || null)
const locked = computed(() => selected.value?.code === 'SUPER_ADMIN')
const versionText = computed(() => selected.value ? 'v' + (selected.value.version || 1) : '')

async function load(keep = true) {
  try {
    const [r, c] = await Promise.all([API.roles(), API.permCatalog()])
    roles.value = r.data || []
    catalog.value = c.data?.catalog || []
    scopes.value = c.data?.scopes || []
    if (!keep || !roles.value.some(x => x.id === selectedId.value)) {
      selectedId.value = roles.value.length ? roles.value[0].id : null
    }
  } catch (e) { toast(e.message, 'err') }
}
onMounted(() => load(false))

watch(selected, r => {
  checked.value = new Set(r ? (r.permissions || []) : [])
}, { immediate: true })

const isAll = g => g.nodes.length > 0 && (locked.value || g.nodes.every(n => checked.value.has(n.code)))
const isSome = g => g.nodes.some(n => checked.value.has(n.code))
function toggleNode(code) {
  if (locked.value) return
  const s = new Set(checked.value)
  s.has(code) ? s.delete(code) : s.add(code)
  checked.value = s
}
function toggleGroup(g) {
  if (locked.value) return
  const s = new Set(checked.value)
  if (isAll(g)) g.nodes.forEach(n => s.delete(n.code))
  else g.nodes.forEach(n => s.add(n.code))
  checked.value = s
}
function selectAll() {
  if (locked.value) return
  checked.value = new Set(catalog.value.flatMap(g => g.nodes.map(n => n.code)))
}
function clearAll() {
  if (locked.value) return
  checked.value = new Set()
}
async function save() {
  if (!selected.value || locked.value) { toast('超级管理员为内置角色，不可修改', 'err'); return }
  saving.value = true
  try {
    const r = await API.updateRole(selected.value.id, {
      permissions: Array.from(checked.value), dataScope: selected.value.dataScope || 'self'
    })
    await load()
    toast(`权限已保存并生效 · 版本 v${r.data?.version || 1} · 缓存已失效`)
  } catch (e) { toast(e.message, 'err') } finally { saving.value = false }
}
async function setScope(v) {
  if (locked.value) return
  const r = roles.value.find(x => x.id === selectedId.value)
  if (!r) return
  r.dataScope = v
  try { await API.updateRole(r.id, { dataScope: v }); toast('数据范围已保存'); await load() }
  catch (e) { toast(e.message, 'err'); await load() }
}
function newRole() {
  modal.value.open({
    title: '新建角色',
    body: `<div class="field"><label>角色名称</label><input id="nr-name" placeholder="输入角色名称"></div>
      <div class="field"><label>角色编码</label><input id="nr-code" placeholder="如 OPS_MANAGER"></div>
      <div class="field"><label>数据范围</label><select id="nr-scope">${scopes.value.map(s => `<option value="${esc(s.value)}">${esc(s.label)}</option>`).join('')}</select></div>
      <div class="sr-desc">创建后可在右侧勾选权限并保存；用户通过「用户管理 → 编辑」绑定角色。</div>`,
    okText: '创建',
    onOk: async () => {
      const name = (document.getElementById('nr-name')?.value || '').trim()
      const code = (document.getElementById('nr-code')?.value || '').trim()
      const dataScope = document.getElementById('nr-scope')?.value || 'self'
      if (!name || !code) { toast('名称和编码不能为空', 'err'); return false }
      try {
        const r = await API.createRole({ name, code, dataScope, permissions: [] })
        await load(false)
        selectedId.value = r.data?.id || null
        toast(`角色「${name}」已创建`)
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function renameRole() {
  if (!selected.value || locked.value) return
  const r = selected.value
  modal.value.open({
    title: '重命名角色',
    body: `<div class="field"><label>角色名称</label><input id="rr-name" value="${esc(r.name)}"></div>`,
    okText: '保存',
    onOk: async () => {
      const name = (document.getElementById('rr-name')?.value || '').trim()
      if (!name) { toast('名称不能为空', 'err'); return false }
      try { await API.updateRole(r.id, { name }); await load(); toast('角色名称已更新') }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function deleteRole() {
  if (!selected.value || locked.value) return
  const r = selected.value
  modal.value.open({
    title: '删除角色', danger: true, okText: '删除',
    body: `<p>确认删除角色「${esc(r.name)}」？该角色当前绑定 ${r.memberCount || 0} 人，存在绑定时不可删除。</p>`,
    onOk: async () => {
      try { await API.deleteRole(r.id); selectedId.value = null; await load(false); toast('角色已删除') }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">RBAC：角色为权限集合，用户可绑定多角色；保存后权限版本自增，在线用户缓存即时失效，服务端按同一份权限码拦截。<span class="code" style="margin-left:8px">FR-RBAC-01~11</span></div>
    <div class="role-split">
      <div class="card">
        <div class="card-head"><h3>角色列表</h3><button class="btn sm primary" @click="newRole">＋ 新建角色</button></div>
        <div v-for="r in roles" :key="r.id" class="role-item" :class="{ on: selectedId === r.id }" @click="selectedId = r.id">
          <span>{{ r.builtIn ? '🔒' : '👤' }}</span><span>{{ r.name }}</span>
          <span class="ri-c">{{ r.memberCount || 0 }} 人</span>
        </div>
        <div v-if="!roles.length" class="loading">暂无角色</div>
      </div>
      <div class="card">
        <template v-if="selected">
          <div class="card-head">
            <div><h3>{{ selected.name }} · 权限配置</h3><div class="sr-desc">{{ selected.code }} · 权限版本 {{ versionText }} · {{ selected.memberCount || 0 }} 名成员{{ locked ? ' · 内置角色只读' : '' }}</div></div>
            <div class="head-actions">
              <button class="btn sm" :disabled="locked" @click="renameRole">重命名</button>
              <button class="btn sm" :disabled="locked" @click="deleteRole">删除</button>
              <button class="btn sm" :disabled="locked" @click="selectAll">全选</button>
              <button class="btn sm" :disabled="locked" @click="clearAll">清空</button>
              <button class="btn sm primary" :disabled="locked || saving" @click="save">{{ saving ? '保存中…' : '保存并生效' }}</button>
            </div>
          </div>
          <div v-for="g in catalog" :key="g.title" class="perm-group">
            <div class="pg-title">
              <input type="checkbox" :disabled="locked" :checked="isAll(g)" :indeterminate.prop="isSome(g) && !isAll(g)" @change="toggleGroup(g)">
              {{ g.title }}<span class="pg-txt">{{ g.hint }}</span>
            </div>
            <div class="pg-body">
              <div class="perm-node" v-for="n in g.nodes" :key="n.code">
                <label><input type="checkbox" :disabled="locked" :checked="checked.has(n.code) || locked" @change="toggleNode(n.code)">{{ n.label }}</label>
                <span class="pn-type">{{ n.hint }}</span><span class="pn-code">{{ n.code }}</span>
              </div>
            </div>
          </div>
          <h3 style="margin:16px 0 10px">数据范围</h3>
          <div class="scope-list">
            <label v-for="s in scopes" :key="s.value" :class="{ on: selected.dataScope === s.value }" @click.prevent="setScope(s.value)">
              <input type="radio" :disabled="locked" :checked="selected.dataScope === s.value" readonly>{{ s.label }}
            </label>
          </div>
          <div class="sr-desc" style="margin-top:12px">多角色时取最宽数据范围、权限取并集；权限码目录与服务端中间件同源，变更无需重启。</div>
        </template>
        <div v-else class="loading">请选择角色</div>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
<style scoped>
.head-actions { display: flex; gap: 6px; flex-wrap: wrap; }
</style>
