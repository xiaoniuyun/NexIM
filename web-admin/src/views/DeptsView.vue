<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const depts = ref([])
const users = ref([])
const selected = ref(null)
const modal = ref(null)
onMounted(async () => {
  try {
    const [d, u] = await Promise.all([API.depts(), API.users()])
    depts.value = d.data || []
    users.value = (u.data || []).list ? (u.data.list) : (u.data || [])
  } catch (e) { toast(e.message, 'err') }
})
const selectedDept = computed(() => depts.value.find(d => d.id === selected.value))
const members = computed(() => {
  if (!selected.value) return users.value
  return users.value.filter(u => u.deptId === selected.value)
})
const countOf = id => users.value.filter(u => u.deptId === id).length
function addDept() {
  modal.value.open({
    title: '新增部门',
    body: `<div class="field"><label>部门名称</label><input id="nd-name" placeholder="输入部门名称"></div>
      <div class="field"><label>上级部门</label><select id="nd-parent"><option value="0">顶级部门</option>${depts.value.map(d => `<option value="${d.id}">${d.name}</option>`).join('')}</select></div>`,
    okText: '创建',
    onOk: () => {
      const name = document.getElementById('nd-name').value.trim()
      if (!name) { toast('请输入部门名称', 'err'); return false }
      toast(`部门「${name}」已创建（演示）`)
    }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">部门为树形结构，支持按物化路径查询子孙层级。<span class="code" style="margin-left:8px">FR-CONTACT-01 / FR-ADMIN-03</span></div>
    <div class="split">
      <div class="card">
        <div class="card-head"><h3>组织架构树</h3><button class="btn sm primary" @click="addDept">＋ 新增部门</button></div>
        <div class="csub">部门层级支持拖拽调整（演示）</div>
        <div class="tree-row" :class="{ on: selected === null }" @click="selected = null">🏢 全部部门 <span class="tr-c">{{ users.length }}</span></div>
        <div v-for="d in depts" :key="d.id" class="tree-row" :class="{ on: selected === d.id }" @click="selected = d.id" :style="{ paddingLeft: 12 + (String(d.path||'').split('/').length-2)*16 + 'px' }">
          📁 {{ d.name }} <span class="tr-c">{{ countOf(d.id) }}</span>
        </div>
      </div>
      <div>
        <div class="card">
          <div class="card-head"><h3>{{ selectedDept ? selectedDept.name : '全部' }} · 成员</h3><span class="muted">编制 {{ members.length + 2 }} · 列表 {{ members.length }} 人</span></div>
          <table class="tbl">
            <thead><tr><th>姓名</th><th>岗位</th><th>手机号</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="u in members" :key="u.id">
                <td><div class="u-cell"><span class="avatar sm" :style="{ background: u.avatar || '#4f6ef7' }">{{ u.nickname?.slice(0,1) }}</span>{{ u.nickname }}</div></td>
                <td>{{ u.deptId ? '成员' : '管理员' }}</td>
                <td><span class="code">138****{{ String(u.id).padStart(4, '0').slice(-4) }}</span></td>
                <td><span class="state" :class="u.status === 1 ? 'ok' : 'off'">{{ u.status === 1 ? '在职' : '禁用' }}</span></td>
              </tr>
              <tr v-if="!members.length"><td colspan="4" class="loading">该部门暂无成员</td></tr>
            </tbody>
          </table>
        </div>
        <div class="card">
          <h3>数据范围示例</h3>
          <div class="csub">按部门树查询时，ORM 自动注入物化路径条件，无需手写 LIKE</div>
          <div class="field"><label>当前部门物化路径</label><div><span class="code">{{ selectedDept?.path || '/1/' }}</span></div></div>
          <div class="field"><label>自动注入的查询条件（示意）</label><div><span class="code">dept.path LIKE '{{ selectedDept?.path || '/1' }}%'</span></div></div>
          <div class="muted">选择「本部门及下级」时，查询范围自动扩展至所有子孙部门。</div>
        </div>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
