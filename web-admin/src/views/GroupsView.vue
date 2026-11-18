<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const groups = ref([])
const f = ref({ kw: '', status: '' })
const modal = ref(null)
onMounted(async () => {
  try { groups.value = (await API.groups()).data || [] } catch (e) { toast(e.message, 'err') }
})
const filtered = computed(() => groups.value.filter(g =>
  (!f.value.kw || (g.name || '').includes(f.value.kw.trim()) || String(g.ownerUid).includes(f.value.kw.trim())) &&
  (!f.value.status || (f.value.status === '正常' ? g.status !== 2 : g.status === 2))
))
const statusText = g => g.status === 2 ? '已冻结' : '正常'
const members = g => [['林晓', '群主'], ['张伟', '管理员'], ['李工', '成员'], ['王敏', '成员'], ['赵琳', '成员'], ['陈静', '成员']]

function detail(g) {
  modal.value.open({
    title: '群详情 · ' + g.name,
    body: `<div class="field"><label>群公告</label><div class="muted">${g.notice || '暂无群公告'}</div></div>
      <h3 style="margin:14px 0 8px">成员（${g.memberCount}）</h3>
      <table class="tbl"><thead><tr><th>成员</th><th>角色</th></tr></thead><tbody>${members(g).map(m => `<tr><td>${m[0]}</td><td><span class="state ${m[1] === '群主' ? 'info' : ''}">${m[1]}</span></td></tr>`).join('')}</tbody></table>
      <div class="grid4" style="margin-top:14px"><div class="kpi"><div class="k-t">今日消息</div><div class="k-v" style="font-size:20px">1,284</div></div><div class="kpi"><div class="k-t">本周消息</div><div class="k-v" style="font-size:20px">8,420</div></div><div class="kpi"><div class="k-t">敏感词命中</div><div class="k-v" style="font-size:20px">3</div></div></div>`,
    okText: '关闭', onOk: () => {}
  })
}
function freeze(g) {
  const unfreeze = g.status === 2
  modal.value.open({
    title: (unfreeze ? '解冻' : '冻结') + '群组',
    body: `<p>确认${unfreeze ? '解冻' : '冻结'}群聊「${g.name}」？${unfreeze ? '解冻后群成员可正常发言。' : '冻结后群内所有成员将无法收发消息，操作记入审计。'}</p>`,
    okText: unfreeze ? '解冻' : '冻结', danger: !unfreeze,
    onOk: () => { g.status = unfreeze ? 1 : 2; toast(`群聊已${unfreeze ? '解冻' : '冻结'}（演示）`) }
  })
}
function dismiss(g) {
  modal.value.open({
    title: '解散群组',
    body: `<p style="color:var(--danger)"><b>此操作不可撤销。</b></p><p>解散后「${g.name}」的所有消息与成员关系将被清除。</p>
      <div class="field"><label>管理员密码（MFA）</label><input id="dg-pwd" type="password" placeholder="请输入管理员密码"></div><div class="muted">演示验证码：123456</div>`,
    okText: '确认解散', danger: true,
    onOk: () => {
      if (document.getElementById('dg-pwd').value.trim() !== '123456') { toast('管理员密码错误', 'err'); return false }
      groups.value = groups.value.filter(x => x !== g); toast('群聊已解散（演示）')
    }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">平台可查看、冻结与解散任意群组，全部写操作留痕审计。<span class="code" style="margin-left:8px">FR-GROUP-08 / FR-SEC-01</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="群名称 / 群主" style="width:210px">
        <select v-model="f.status"><option value="">全部状态</option><option>正常</option><option>已冻结</option></select>
        <button class="btn" @click="toast('已按条件检索')">查询</button>
        <button class="btn" @click="toast('导出任务已提交（演示）')">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>群组</th><th>群主</th><th>成员数</th><th>消息量</th><th>创建时间</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="g in filtered" :key="g.groupId">
            <td><div class="u-cell"><span class="avatar sm" :style="{ background: g.avatar || '#4f6ef7' }">{{ g.name?.slice(0,1) }}</span><b>{{ g.name }}</b></div></td>
            <td>{{ g.ownerUid }}</td><td>{{ g.memberCount }}</td><td>{{ g.msgCount || '—' }}</td>
            <td>{{ g.createdAt ? new Date(g.createdAt).toLocaleDateString() : '—' }}</td>
            <td><span class="state" :class="g.status === 2 ? 'warn' : 'ok'">{{ statusText(g) }}</span></td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="detail(g)">查看</button>
              <button class="btn sm" @click="freeze(g)">{{ g.status === 2 ? '解冻' : '冻结' }}</button>
              <button class="btn sm danger" @click="dismiss(g)">解散</button>
            </td>
          </tr>
          <tr v-if="!filtered.length"><td colspan="7" class="loading">暂无群聊</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ filtered.length }} 条 <button class="on">1</button><button>2</button></div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
