<script setup>
import { ref, onMounted } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const groups = ref([])
const f = ref({ kw: '', status: '' })
const modal = ref(null)

async function load() {
  try {
    const r = await API.adminGroups({ kw: f.value.kw.trim(), status: f.value.status })
    groups.value = r.data || []
  } catch (e) { toast(e.message, 'err') }
}
onMounted(load)

const statusText = g => g.status === 3 ? '已解散' : g.status === 2 ? '已冻结' : '正常'
const stateCls = g => g.status === 3 ? 'off' : g.status === 2 ? 'warn' : 'ok'
function fmtDate(ts) { return ts ? new Date(ts).toLocaleDateString() : '—' }
function esc(s) { return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])) }

async function detail(g) {
  let rows = []
  let notice = g.notice || ''
  try {
    const [dr, mr] = await Promise.all([API.groupDetail(g.groupId), API.groupMembers(g.groupId)])
    notice = dr.data?.notice || notice
    rows = mr.data || []
  } catch (e) { toast(e.message, 'err') }
  const memberHtml = rows.length
    ? `<h3 style="margin:14px 0 8px">成员（${rows.length}）</h3>
      <table class="tbl"><thead><tr><th>成员</th><th>角色</th><th>加入时间</th></tr></thead><tbody>${rows.map(m => `<tr><td>${esc(m.nickname || m.uid)}</td><td><span class="state ${m.role === 'owner' ? 'info' : ''}">${m.role === 'owner' ? '群主' : m.role === 'admin' ? '管理员' : '成员'}</span></td><td>${m.joinedAt ? new Date(m.joinedAt).toLocaleDateString() : '—'}</td></tr>`).join('')}</tbody></table>`
    : '<div class="muted" style="margin-top:12px">暂无成员数据</div>'
  modal.value.open({
    title: '群详情 · ' + g.name,
    body: `<div class="field"><label>群公告</label><div class="muted">${esc(notice || '暂无群公告')}</div></div>
      <div class="grid4" style="margin-top:14px">
        <div class="kpi"><div class="k-t">成员</div><div class="k-v" style="font-size:20px">${g.memberCount || 0}</div></div>
        <div class="kpi"><div class="k-t">消息量</div><div class="k-v" style="font-size:20px">${g.msgCount || 0}</div></div>
        <div class="kpi"><div class="k-t">状态</div><div class="k-v" style="font-size:20px">${statusText(g)}</div></div>
      </div>
      <div class="muted" style="margin-top:12px">群主：${esc(g.ownerName || g.ownerUid)} · 群ID ${esc(g.groupId)}</div>
      ${memberHtml}`,
    okText: '关闭', onOk: () => {}
  })
}
function freeze(g) {
  const unfreeze = g.status === 2
  modal.value.open({
    title: (unfreeze ? '解冻' : '冻结') + '群组',
    body: `<p>确认${unfreeze ? '解冻' : '冻结'}群聊「${esc(g.name)}」？${unfreeze ? '解冻后群成员可正常发言。' : '冻结后群内所有成员将无法收发消息，操作记入审计。'}</p>`,
    okText: unfreeze ? '解冻' : '冻结', danger: !unfreeze,
    onOk: async () => {
      try {
        await API.setGroupStatus(g.groupId, unfreeze ? 1 : 2)
        toast(`群聊已${unfreeze ? '解冻' : '冻结'}`)
        load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function dismiss(g) {
  modal.value.open({
    title: '解散群组',
    body: `<p style="color:var(--danger)"><b>此操作不可撤销。</b></p><p>解散后「${esc(g.name)}」将不可再使用，历史消息与成员关系保留供审计。</p>
      <div class="field"><label>管理员密码（MFA）</label><input id="dg-pwd" type="password" placeholder="请输入管理员密码"></div><div class="muted">演示验证码：123456</div>`,
    okText: '确认解散', danger: true,
    onOk: async () => {
      const el = document.getElementById('dg-pwd')
      if (!el || el.value.trim() !== '123456') { toast('管理员密码错误', 'err'); return false }
      try {
        await API.dissolveGroup(g.groupId)
        toast('群聊已解散')
        load()
      } catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
async function exportCsv() {
  try { await API.exportGroups({ kw: f.value.kw.trim(), status: f.value.status }); toast('导出已完成') } catch (e) { toast(e.message, 'err') }
}
</script>
<template>
  <div>
    <div class="proto-hint">平台可查看、冻结与解散任意群组，全部写操作留痕审计。<span class="code" style="margin-left:8px">FR-GROUP-08 / FR-SEC-01</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="群名称 / 群主" style="width:210px" @keyup.enter="load">
        <select v-model="f.status" @change="load"><option value="">全部状态</option><option value="1">正常</option><option value="2">已冻结</option><option value="3">已解散</option></select>
        <button class="btn" @click="load">查询</button>
        <button class="btn" @click="exportCsv">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>群组</th><th>群主</th><th>成员数</th><th>消息量</th><th>创建时间</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="g in groups" :key="g.groupId">
            <td><div class="u-cell"><span class="avatar sm" :style="{ background: g.avatar || '#4f6ef7' }">{{ g.name?.slice(0,1) }}</span><b>{{ g.name }}</b></div></td>
            <td>{{ g.ownerName || g.ownerUid }}</td><td>{{ g.memberCount }}</td><td>{{ g.msgCount || 0 }}</td>
            <td>{{ fmtDate(g.createdAt) }}</td>
            <td><span class="state" :class="stateCls(g)">{{ statusText(g) }}</span></td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="detail(g)">查看</button>
              <button v-if="g.status !== 3" class="btn sm" @click="freeze(g)">{{ g.status === 2 ? '解冻' : '冻结' }}</button>
              <button v-if="g.status !== 3" class="btn sm danger" @click="dismiss(g)">解散</button>
              <span v-else class="muted">已归档</span>
            </td>
          </tr>
          <tr v-if="!groups.length"><td colspan="7" class="loading">暂无群聊</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ groups.length }} 条</div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
