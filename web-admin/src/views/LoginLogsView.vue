<script setup>
import { ref } from 'vue'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'
const kw = ref('')
const result = ref('')
const risk = ref('')
const date = ref('')
const modal = ref(null)
const rows = [
  ['2026-11-15 09:12', '林晓', '10.20.3.18', '上海', 'Windows 客户端', '成功', '正常'],
  ['2026-11-15 08:46', '赵琳', '10.20.3.77', '上海', 'macOS 客户端', '成功', '正常'],
  ['2026-11-15 07:12', '未知', '203.0.113.42', '海外', 'Web', '失败 5 次', '异常登录'],
  ['2026-11-15 06:58', '王敏', '10.20.5.44', '北京', 'H5 网页端', '成功', '正常'],
  ['2026-11-14 22:31', '陈静', '10.20.8.12', '广州', 'Windows 客户端', '成功', '异地登录'],
  ['2026-11-14 20:05', '张伟', '10.20.3.9', '上海', 'Windows 客户端', '成功', '正常']
]
const filtered = () => rows.filter(x =>
  (!kw.value || x[1].toLowerCase().includes(kw.value.trim().toLowerCase()) || x[2].includes(kw.value.trim())) &&
  (!result.value || (result.value === '成功' ? x[5].includes('成功') : !x[5].includes('成功'))) &&
  (!risk.value || x[6] === risk.value) &&
  (!date.value || x[0].startsWith(date.value))
)
const riskCls = r => r === '正常' ? 'info' : (r === '异常登录' ? 'danger' : 'warn')
function ban(r) {
  modal.value.open({
    title: '封禁 IP', body: `<p>确认封禁 IP「${r[2]}」？封禁后该 IP 的所有登录请求将被拒绝。</p><div class="muted">来源：${r[3]} · 风险：${r[6]} · 操作写入审计</div>`,
    okText: '确认封禁', danger: true,
    onOk: () => toast(`IP ${r[2]} 已封禁，操作记入审计`)
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">登录日志全量留存，异地 / 高频失败自动告警，异常 IP 可一键封禁。<span class="code" style="margin-left:8px">FR-ADMIN-09 / FR-AUTH-06</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="kw" placeholder="用户 / IP" style="width:190px">
        <select v-model="result"><option value="">全部结果</option><option>成功</option><option>失败</option></select>
        <select v-model="risk"><option value="">全部风险</option><option>异常登录</option><option>异地登录</option><option>正常</option></select>
        <input type="date" v-model="date">
        <button class="btn" @click="toast('已按条件检索')">查询</button>
        <span class="sp"></span>
        <button class="btn" @click="toast('导出任务已提交（演示）')">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>用户</th><th>来源 IP</th><th>地点</th><th>设备</th><th>结果</th><th>风险</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="(r, i) in filtered()" :key="i">
            <td>{{ r[0] }}</td><td>{{ r[1] }}</td><td><span class="code">{{ r[2] }}</span></td>
            <td>{{ r[3] }}</td><td>{{ r[4] }}</td>
            <td><span class="state" :class="r[5].includes('成功') ? 'ok' : 'danger'">{{ r[5] }}</span></td>
            <td><span class="state" :class="riskCls(r[6])">{{ r[6] }}</span></td>
            <td><button v-if="r[6] !== '正常'" class="btn sm danger" @click="ban(r)">封禁 IP</button><span v-else>—</span></td>
          </tr>
          <tr v-if="!filtered().length"><td colspan="8" class="loading">未找到匹配的登录记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 12,480 条 <button>‹</button><button class="on">1</button><button>2</button><button>›</button></div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
