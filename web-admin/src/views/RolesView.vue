<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const permGroups = [
  { t: '菜单与目录权限', hint: '控制动态路由与侧边栏可见性', nodes: [['工作台','dashboard:view','菜单'],['用户管理','menu:users','菜单'],['组织架构','menu:depts','菜单'],['角色权限','menu:roles','菜单'],['消息审计','menu:audit','菜单'],['系统设置','menu:settings','菜单']] },
  { t: '接口权限', hint: '服务端网关中间件强制校验（URL + Method）', nodes: [['用户列表','user:list','GET /api/v1/users'],['编辑用户','user:edit','PUT /api/v1/users/{id}'],['启停账号','user:disable','PATCH /api/v1/users/{id}/status'],['组织编辑','dept:edit','POST /api/v1/depts'],['群组冻结','group:freeze','PATCH /api/v1/groups/{id}'],['群组解散','group:dissolve','DELETE /api/v1/groups/{id}'],['授权变更','rbac:role:grant','PUT /api/v1/admin/roles/{id}/permissions'],['通话记录查看','rtc:record:list','GET /api/v1/rtc/records'],['录制回放','rtc:record:playback','GET /api/v1/rtc/records/{id}/playback'],['质量监控','rtc:quality:view','GET /api/v1/rtc/quality'],['强制挂断','rtc:force-end','POST /api/v1/rtc/force-end']] },
  { t: '按钮 / 操作点权限', hint: 'v-permission 指令控制显隐，服务端仍二次校验', nodes: [['新增用户','user:create','按钮'],['重置密码','user:reset-pwd','按钮'],['导出用户','user:export','按钮'],['导出审计消息','audit:message:export','按钮'],['导出操作日志','log:export','按钮'],['配置敏感词','sensitive:config','按钮'],['处理举报','report:handle','按钮']] }
]
const scopes = [['1','全部数据'],['2','本部门及下级'],['3','本部门'],['4','仅本人'],['5','自定义部门']]

const roles = ref([])
const selected = ref(null)
const checked = ref(new Set())
const scope = ref('1')
const version = ref('v12')
const modal = ref(null)

onMounted(async () => {
  try { roles.value = (await API.roles()).data || []; if (roles.value.length) selected.value = roles.value[0] } catch (e) { toast(e.message, 'err') }
})
watch(selected, r => {
  if (!r) return
  let perms = []
  try { perms = typeof r.permissions === 'string' ? JSON.parse(r.permissions || '[]') : (r.permissions || []) } catch (e) { perms = [] }
  checked.value = new Set(perms)
  scope.value = String(r.dataScope || '1')
}, { immediate: true })

const isAll = g => g.nodes.every(n => checked.value.has(n[1]))
const isSome = g => g.nodes.some(n => checked.value.has(n[1]))
function toggleNode(code) {
  const s = new Set(checked.value)
  s.has(code) ? s.delete(code) : s.add(code)
  checked.value = s
}
function toggleGroup(g) {
  const s = new Set(checked.value)
  if (isAll(g)) g.nodes.forEach(n => s.delete(n[1]))
  else g.nodes.forEach(n => s.add(n[1]))
  checked.value = s
}
function selectAll() { checked.value = new Set(permGroups.flatMap(g => g.nodes.map(n => n[1]))); toast('已全选全部权限') }
function clearAll() { checked.value = new Set(); toast('已清空全部权限') }
function save() {
  version.value = 'v13'
  toast('权限已保存并生效 · 版本 v13 · Redis 缓存已失效')
}
function newRole() {
  modal.value.open({
    title: '新建角色',
    body: '<div class="field"><label>角色名称</label><input id="nr-name" placeholder="输入角色名称"></div><div class="field"><label>角色编码</label><input id="nr-code" placeholder="如 OPS_MANAGER"></div><div class="field"><label>继承父角色</label><select id="nr-parent"><option value="">不继承</option>' + roles.value.map(r => `<option value="${r.id}">${r.name}</option>`).join('') + '</select></div><div class="sr-desc">新角色继承父角色权限后仍可单独调整。</div>',
    okText: '创建',
    onOk: () => {
      const name = document.getElementById('nr-name').value.trim()
      const code = document.getElementById('nr-code').value.trim()
      if (!name || !code) { toast('名称和编码不能为空', 'err'); return false }
      toast(`角色「${name}」已创建（演示）`)
    }
  })
}
const builtIn = r => ['SUPER_ADMIN','OPS_ADMIN','AUDITOR','DEPT_ADMIN'].includes(r.code)
</script>
<template>
  <div>
    <div class="proto-hint">RBAC 权限模型：角色为权限集合，用户可绑定多角色；权限变更即时失效 Redis 缓存。<span class="code" style="margin-left:8px">FR-RBAC-01~11</span></div>
    <div class="role-split">
      <div class="card">
        <div class="card-head"><h3>角色列表</h3><button class="btn sm primary" @click="newRole">＋ 新建角色</button></div>
        <div v-for="r in roles" :key="r.id" class="role-item" :class="{ on: selected?.id === r.id }" @click="selected = r">
          <span>{{ builtIn(r) ? '🔒' : '👤' }}</span><span>{{ r.name }}</span>
          <span class="ri-c">{{ r.memberCount || 0 }} 人</span>
        </div>
        <div v-if="!roles.length" class="loading">暂无角色</div>
      </div>
      <div class="card">
        <template v-if="selected">
          <div class="card-head">
            <div><h3>{{ selected.name }} · 权限配置</h3><div class="sr-desc">{{ selected.code }} · 权限版本 {{ version }}</div></div>
            <div style="display:flex;gap:6px"><button class="btn sm" @click="selectAll">全选</button><button class="btn sm" @click="clearAll">清空</button><button class="btn sm primary" @click="save">保存并生效</button></div>
          </div>
          <div v-for="g in permGroups" :key="g.t" class="perm-group">
            <div class="pg-title">
              <input type="checkbox" :checked="isAll(g)" :indeterminate.prop="isSome(g) && !isAll(g)" @change="toggleGroup(g)">
              {{ g.t }}<span class="pg-txt">{{ g.hint }}</span>
            </div>
            <div class="pg-body">
              <div class="perm-node" v-for="n in g.nodes" :key="n[1]">
                <label><input type="checkbox" :checked="checked.has(n[1])" @change="toggleNode(n[1])">{{ n[0] }}</label>
                <span class="pn-type">{{ n[2] }}</span><span class="pn-code">{{ n[1] }}</span>
              </div>
            </div>
          </div>
          <h3 style="margin:16px 0 10px">数据范围</h3>
          <div class="scope-list">
            <label v-for="s in scopes" :key="s[0]" :class="{ on: scope === s[0] }" @click.prevent="scope = s[0]"><input type="radio" :checked="scope === s[0]" readonly>{{ s[1] }}</label>
          </div>
          <div class="sr-desc" style="margin-top:12px">保存后权限版本升级至 v13，所有在线用户令牌中的权限声明在 5 秒内刷新，Redis 缓存同步失效。</div>
        </template>
        <div v-else class="loading">请选择角色</div>
      </div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
