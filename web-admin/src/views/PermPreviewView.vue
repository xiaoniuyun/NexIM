<script setup>
import { ref, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
const rows = [
  { name: '林晓', role: 'SUPER_ADMIN', scope: '全部', stat: '菜单 5 · 接口 40 · 按钮 12', note: '内置角色（跳过校验，仍审计）' },
  { name: '赵琳', role: 'OPS_ADMIN', scope: '全部', stat: '菜单 6 · 接口 18 · 按钮 8', note: '运营管理全量' },
  { name: '王敏', role: 'AUDITOR', scope: '全部', stat: '菜单 3 · 接口 6 · 按钮 2', note: '仅审计只读 + 导出' },
  { name: '陈静', role: 'DEPT_ADMIN', scope: '本部门及下级', stat: '菜单 3 · 接口 8 · 按钮 4', note: '受数据范围约束' }
]
const picked = ref('林晓')
const livePerms = ref([])
async function view() {
  toast('正在加载 ' + picked.value + ' 的最终权限…')
  try {
    if (picked.value === '林晓') { const r = await API.permissions(); livePerms.value = r.data || [] }
    else livePerms.value = []
  } catch (e) { toast(e.message, 'err') }
}
const detail = computed(() => rows.find(r => r.name === picked.value))
</script>
<template>
  <div>
    <div class="proto-hint">选择用户查看其最终权限 = 多角色权限并集，数据范围取最宽集合；冲突项给出来源说明。<span class="code" style="margin-left:8px">FR-RBAC-12</span></div>
    <div class="card" style="margin-bottom:14px">
      <div class="toolbar">
        <span>选择用户：</span>
        <select v-model="picked" style="width:220px"><option v-for="r in rows" :key="r.name">{{ r.name }}</option></select>
        <button class="btn primary" @click="view">🔍 查看权限</button>
        <span class="sp"></span>
        <span class="muted">权限码由 /auth/permissions 实时下发，角色变更 5 秒内生效</span>
      </div>
    </div>
    <div class="card">
      <table class="tbl">
        <thead><tr><th>用户</th><th>绑定角色</th><th>数据范围</th><th>权限统计</th><th>来源说明</th></tr></thead>
        <tbody>
          <tr v-for="r in rows" :key="r.name" :style="{ background: picked === r.name ? 'var(--brand-soft)' : '' }">
            <td><div class="u-cell"><span class="avatar sm" style="background:#4f6ef7">{{ r.name.slice(0,1) }}</span>{{ r.name }}</div></td>
            <td><span class="code">{{ r.role }}</span></td><td>{{ r.scope }}</td><td>{{ r.stat }}</td><td>{{ r.note }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="detail" style="margin-top:14px;padding:12px 14px;background:var(--bg);border-radius:9px;font-size:12px;line-height:1.9;color:var(--ink2)">
        <b>{{ detail.name }}</b> 当前绑定「{{ detail.role }}」，数据范围「{{ detail.scope }}」。
        <template v-if="livePerms.length">实时权限码（{{ livePerms.length }} 项）：<span v-for="p in livePerms" :key="p" class="code" style="margin:2px">{{ p }}</span></template>
        <template v-else>示例：用户同时绑定「部门管理员（本部门及下级）」与「审计员（全部）」两个角色时，数据范围取最宽的「全部」，接口权限取并集；若两个角色对同一资源授权冲突，按「允许优先」处理并在审计中留痕。</template>
      </div>
    </div>
  </div>
</template>
