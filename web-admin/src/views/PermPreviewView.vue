<script setup>
import { ref, computed, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

const users = ref([])
const scopes = ref([])
const pickedId = ref(null)
const effect = ref(null)
const loading = ref(false)
const scopeLabel = v => scopes.value.find(s => s.value === v)?.label || v || '仅本人'

async function load() {
  try {
    const [u, c] = await Promise.all([API.users({ size: 1000 }), API.permCatalog()])
    users.value = u.data?.list || []
    scopes.value = c.data?.scopes || []
    if (!users.value.some(x => x.id === pickedId.value)) pickedId.value = users.value[0]?.id || null
    if (pickedId.value) await view()
  } catch (e) { toast(e.message, 'err') }
}
onMounted(load)

const picked = computed(() => users.value.find(u => u.id === pickedId.value) || null)
async function view() {
  if (!pickedId.value) return
  loading.value = true
  try { const r = await API.userPermissions(pickedId.value); effect.value = r.data || null }
  catch (e) { toast(e.message, 'err'); effect.value = null }
  finally { loading.value = false }
}
const menuPerms = computed(() => (effect.value?.permissions || []).filter(p => p.startsWith('menu:') || p === 'dashboard:view'))
</script>
<template>
  <div>
    <div class="proto-hint">选择用户查看其最终权限 = 多角色权限并集，数据范围取最宽集合；权限码由服务端实时计算并缓存。<span class="code" style="margin-left:8px">FR-RBAC-12</span></div>
    <div class="card" style="margin-bottom:14px">
      <div class="toolbar">
        <span>选择用户：</span>
        <select v-model="pickedId" style="width:220px" @change="view">
          <option v-for="u in users" :key="u.id" :value="u.id">{{ u.nickname }} (@{{ u.username }})</option>
        </select>
        <button class="btn primary" @click="view">🔍 查看权限</button>
        <button class="btn" @click="load">刷新</button>
        <span class="sp"></span>
        <span class="muted">权限版本 {{ effect?.version || '—' }} · 角色变更后缓存即时失效</span>
      </div>
    </div>
    <div class="card">
      <table class="tbl">
        <thead><tr><th>用户</th><th>绑定角色</th><th>数据范围</th><th>部门</th><th>状态</th></tr></thead>
        <tbody>
          <tr v-for="u in users" :key="u.id" :style="{ background: pickedId === u.id ? 'var(--brand-soft)' : '' }" @click="pickedId = u.id; view()">
            <td><div class="u-cell"><span class="avatar sm" style="background:#4f6ef7">{{ u.nickname?.slice(0,1) }}</span>{{ u.nickname }}</div></td>
            <td><span class="code">{{ (u.roleCodes || []).join(' / ') || 'USER' }}</span></td>
            <td>{{ u.roleNames?.length ? '按角色取最宽' : '仅本人' }}</td>
            <td>{{ u.deptName || '未分配' }}</td>
            <td><span class="state" :class="u.status === 1 ? 'ok' : 'off'">{{ u.status === 1 ? '正常' : '禁用' }}</span></td>
          </tr>
          <tr v-if="!users.length"><td colspan="5" class="loading">暂无用户</td></tr>
        </tbody>
      </table>
      <div v-if="picked" style="margin-top:14px;padding:12px 14px;background:var(--bg);border-radius:9px;font-size:12px;line-height:1.9;color:var(--ink2)">
        <div v-if="loading">加载中…</div>
        <template v-else-if="effect">
          <b>{{ picked.nickname }}</b> 绑定 {{ (effect.roleNames || []).join('、') || '无角色' }}；
          数据范围「{{ scopeLabel(effect.dataScope) }}」，共 {{ (effect.permissions || []).length }} 项权限码，其中菜单 {{ menuPerms.length }} 项。
          <div style="margin-top:8px">
            <span v-for="p in effect.permissions" :key="p" class="code" style="margin:2px;display:inline-block">{{ p }}</span>
            <span v-if="!effect.permissions?.length" class="muted">该用户暂无任何权限</span>
          </div>
        </template>
        <div v-else>点击「查看权限」加载该用户的最终权限。</div>
      </div>
    </div>
  </div>
</template>
