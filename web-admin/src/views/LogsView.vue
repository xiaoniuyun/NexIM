<script setup>
import { ref, onMounted, computed } from 'vue'
import { API } from '../api'
import { toast } from '../toast'

const f = ref({ kw: '', module: '', date: '' })
const rows = ref([])
onMounted(async () => {
  const demo = [
    { u: '林晓', m: '角色权限', a: '更新角色 OPS_ADMIN 权限集合', ip: '10.20.3.18', t: '2026-11-15 09:38', r: '成功' },
    { u: '王敏', m: '消息审计', a: '导出审计结果（320 条）', ip: '10.20.5.44', t: '2026-11-15 09:21', r: '成功' },
    { u: '赵琳', m: '用户管理', a: '禁用账号 chenjing', ip: '10.20.3.77', t: '2026-11-15 08:46', r: '成功' },
    { u: '未知', m: '认证', a: '登录失败 5 次触发锁定', ip: '203.0.113.42', t: '2026-11-15 07:12', r: '失败' },
    { u: '林晓', m: '系统设置', a: '修改消息保留天数 180 → 180', ip: '10.20.3.18', t: '2026-11-14 18:02', r: '成功' },
    { u: '周涛', m: '内容安全', a: '新增敏感词 3 条', ip: '10.20.6.12', t: '2026-11-14 16:33', r: '成功' }
  ]
  try {
    const d = (await API.audit()).data || {}
    const real = (d.list || []).map(l => ({ u: String(l.uid), m: String(l.action || '').split(':')[0], a: l.action + (l.target ? ' · ' + l.target : ''), ip: l.detail || '127.0.0.1', t: new Date(l.createdAt).toLocaleString(), r: '成功' }))
    rows.value = real.length ? real : demo
  } catch (e) { rows.value = demo }
})
const filtered = computed(() => rows.value.filter(l =>
  (!f.value.kw || [l.u, l.a, l.ip].some(x => String(x).toLowerCase().includes(f.value.kw.trim().toLowerCase()))) &&
  (!f.value.module || l.m === f.value.module) &&
  (!f.value.date || l.t.startsWith(f.value.date))
))
</script>
<template>
  <div>
    <div class="proto-hint">审计日志仅追加写入，应用账号无 UPDATE / DELETE 权限，保留 ≥ 180 天。<span class="code" style="margin-left:8px">FR-SEC-01 / FR-ADMIN-08</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="f.kw" placeholder="操作人 / IP / 关键词" style="width:210px">
        <select v-model="f.module"><option value="">全部模块</option><option>用户管理</option><option>角色权限</option><option>消息审计</option><option>内容安全</option><option>系统设置</option></select>
        <input type="date" v-model="f.date">
        <button class="btn" @click="toast('已按条件检索')">查询</button>
        <span class="sp"></span>
        <button class="btn" @click="toast('导出任务已提交（演示）')">导出日志</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>操作人</th><th>模块</th><th>操作内容</th><th>来源 IP</th><th>结果</th></tr></thead>
        <tbody>
          <tr v-for="(l, i) in filtered" :key="i">
            <td>{{ l.t }}</td><td>{{ l.u }}</td><td><span class="code">{{ l.m }}</span></td>
            <td>{{ l.a }}</td><td>{{ l.ip }}</td>
            <td><span class="state" :class="l.r === '成功' ? 'ok' : 'danger'">{{ l.r }}</span></td>
          </tr>
          <tr v-if="!filtered.length"><td colspan="6" class="loading">暂无审计记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 1,284 条 <button>‹</button><button class="on">1</button><button>2</button><button>3</button><button>›</button></div>
    </div>
  </div>
</template>
