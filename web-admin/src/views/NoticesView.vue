<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const notices = ref([])
const f = ref({ kw: '', status: '' })
const modal = ref(null)
const scopes = ['全员', '研发中心', '市场部', '管理层', '外部协作']
async function load() { try { notices.value = (await API.notices()).data || [] } catch (e) { toast(e.message, 'err') } }
onMounted(load)
const filtered = computed(() => notices.value.filter(n =>
  (!f.value.kw || (n.title || '').includes(f.value.kw.trim()) || (n.scope || '').includes(f.value.kw.trim())) &&
  (!f.value.status || (f.value.status === '已发布' ? n.status !== 2 : f.value.status === '已下线' ? n.status === 2 : n.status === 0))
))
const statusText = n => n.status === 2 ? '已下线' : n.status === 0 ? '定时中' : '已发布'
const fmt = ts => ts ? new Date(ts).toLocaleString() : '—'

function formBody(n) {
  return `<div class="field"><label>公告标题</label><input id="no-title" value="${n?.title || ''}"></div>
    <div class="field"><label>定向范围</label><select id="no-scope">${scopes.map(s => `<option ${n?.scope === s ? 'selected' : ''}>${s}</option>`).join('')}</select></div>
    ${n ? '' : '<div class="field"><label>定时发布（可选）</label><input type="datetime-local" id="no-time"><div class="fh">留空则立即发布</div></div>'}
    <div class="field"><label>正文（支持富文本与附件）</label><textarea id="no-content" rows="5">${n?.content || ''}</textarea></div>`
}
function create() {
  modal.value.open({
    title: '新建公告', body: formBody(null), okText: '发布',
    onOk: async () => {
      const title = document.getElementById('no-title').value.trim()
      const content = document.getElementById('no-content').value.trim()
      const scope = document.getElementById('no-scope').value
      const time = document.getElementById('no-time')?.value
      if (!title || !content) { toast('标题和正文不能为空', 'err'); return false }
      try { await API.createNotice({ title, content, scope }); toast(time ? '公告已设为定时发布' : '公告发布成功，已推送至目标用户'); load() }
      catch (e) { toast(e.message, 'err'); return false }
    }
  })
}
function edit(n) {
  modal.value.open({
    title: '编辑公告', body: formBody(n), okText: '保存',
    onOk: () => {
      n.title = document.getElementById('no-title').value.trim()
      n.scope = document.getElementById('no-scope').value
      n.content = document.getElementById('no-content').value.trim()
      toast('公告已保存并重新推送，保留原有已读状态')
    }
  })
}
function offline(n) {
  modal.value.open({
    title: '下线公告', body: `<p>确认下线公告「${n.title}」？下线后用户端将不再展示。</p>`, okText: '下线', danger: true,
    onOk: () => { n.status = 2; toast('公告已下线') }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">公告支持定向发布与定时发布，编辑后保留已读状态重新推送。<span class="code" style="margin-left:8px">FR-ADMIN-11</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="标题 / 定向范围" style="width:220px">
        <select v-model="f.status"><option value="">全部状态</option><option>已发布</option><option>定时中</option><option>已下线</option></select>
        <button class="btn" @click="toast('已按条件检索')">查询</button>
        <button class="btn primary" @click="create">＋ 新建公告</button>
      </div>
      <table class="tbl">
        <thead><tr><th>标题</th><th>定向范围</th><th>状态</th><th>发布时间</th><th>已读率</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="n in filtered" :key="n.id">
            <td><div class="u-cell">{{ n.pinned ? '📌' : '' }}<b>{{ n.title }}</b><span v-if="n.pinned" class="tag" style="margin-left:4px">置顶</span></div></td>
            <td>{{ n.scope || '全员' }}</td>
            <td><span class="state" :class="n.status === 2 ? 'off' : n.status === 0 ? 'warn' : 'ok'">{{ statusText(n) }}</span></td>
            <td>{{ fmt(n.createdAt) }}</td><td>{{ n.readRate || '86%' }}</td>
            <td style="white-space:nowrap">
              <button class="btn sm" @click="edit(n)">编辑</button>
              <button v-if="n.status !== 2" class="btn sm danger" @click="offline(n)">下线</button>
            </td>
          </tr>
          <tr v-if="!filtered.length"><td colspan="6" class="loading">暂无公告</td></tr>
        </tbody>
      </table>
      <div class="pager">共 {{ filtered.length }} 条 <button class="on">1</button><button>2</button></div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
