<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))
const data = ref({ list: [], total: 0 })
const f = ref({ kw: '', status: '' })
const page = ref(1)
const size = 10
const loading = ref(false)
const modal = ref(null)

// 外部联系人：注册页注册的账号自动归入该角色（userType=1）
const params = () => ({ userType: 1, kw: f.value.kw.trim(), status: f.value.status, page: page.value, size })
async function load() {
  loading.value = true
  try {
    const r = await API.users(params())
    data.value = r.data || { list: [], total: 0 }
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}
onMounted(load)

const pages = computed(() => Math.max(1, Math.ceil((data.value.total || 0) / size)))
function query() { page.value = 1; load() }
function reset() { f.value = { kw: '', status: '' }; page.value = 1; load() }
function go(p) { if (p < 1 || p > pages.value) return; page.value = p; load() }
const fmt = ts => ts ? new Date(ts).toLocaleString() : '—'

function toggleStatus(u) {
  const enable = u.status !== 1
  modal.value.open({
    title: (enable ? '启用' : '禁用') + '外部联系人',
    body: `<p>确认${enable ? '启用' : '禁用'}「${esc(u.nickname || u.username)}」？${enable ? '启用后可正常登录。' : '禁用后该账号立即无法登录与收发消息。'}</p>`,
    okText: enable ? '启用' : '禁用', danger: !enable,
    onOk: async () => {
      try {
        if (enable) await API.enableUser(u.id); else await API.deleteUser(u.id)
        toast(enable ? '账号已启用' : '账号已禁用'); load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function resetPwd(u) {
  modal.value.open({
    title: '重置密码', body: `<p>确认重置「${esc(u.nickname || u.username)}」的密码？</p>`, okText: '重置',
    onOk: async () => {
      try { const r = await API.resetPwd(u.id); toast('密码已重置为 ' + (r.data?.password || '123456')) }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">通过注册页注册的账号均为外部联系人，默认无法查看组织架构与内部人员，仅可与其他外部联系人建立联系。</div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="姓名 / 账号 / 手机号" style="width:210px" @keyup.enter="query">
        <select v-model="f.status"><option value="">全部状态</option><option value="1">正常</option><option value="0">禁用</option></select>
        <button class="btn" @click="query">查询</button>
        <button class="btn" @click="reset">重置</button>
        <span class="muted">共 {{ data.total }} 位外部联系人</span>
      </div>
      <table class="tbl">
        <thead><tr><th>用户</th><th>手机号</th><th>注册时间</th><th>最后登录</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="u in data.list" :key="u.id">
            <td><div class="u-cell"><span class="avatar sm" :style="{ background: u.avatar || '#4f6ef7' }">{{ (u.nickname || u.username || '?').slice(0,1) }}</span><div><b>{{ u.nickname || u.username }}</b> <span class="tag">外部</span><div class="sr-desc">@{{ u.username }}</div></div></div></td>
            <td><span class="code">{{ u.phone || '—' }}</span></td>
            <td>{{ fmt(u.createdAt) }}</td>
            <td>{{ fmt(u.lastLoginAt) }}</td>
            <td><span class="state" :class="u.status === 1 ? 'ok' : 'off'">{{ u.status === 1 ? '正常' : '已禁用' }}</span></td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="resetPwd(u)">重置密码</button>
              <button class="btn sm" :class="u.status === 1 ? 'danger' : 'ghost'" @click="toggleStatus(u)">{{ u.status === 1 ? '禁用' : '启用' }}</button>
            </td>
          </tr>
          <tr v-if="!data.list.length"><td colspan="6" class="loading">{{ loading ? '加载中…' : '暂无外部联系人' }}</td></tr>
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
