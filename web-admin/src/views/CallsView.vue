<script setup>
import { ref, computed } from 'vue'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const kw = ref('')
const typeF = ref('')
const resultF = ref('')
const modal = ref(null)
const rows = ref([
  { t: '今天 09:41', type: '1v1 视频', from: '林晓', peer: '张伟', room: 'RM88213', members: 2, dur: '12:36', mode: 'P2P', q: '良好', rec: '已录制' },
  { t: '今天 09:12', type: '1v1 语音', from: '张伟', peer: '林晓', room: 'RM88109', members: 2, dur: '03:08', mode: 'P2P', q: '良好', rec: '—' },
  { t: '今天 08:47', type: '群组视频', from: '赵琳', peer: '前端研发组', room: 'RM88042', members: 7, dur: '28:15', mode: 'SFU', q: '一般', rec: '已录制' },
  { t: '昨天 20:03', type: '1v1 视频', from: '陈静', peer: '林晓', room: 'RM87120', members: 2, dur: '00:00', mode: 'P2P', q: '未接', rec: '—' },
  { t: '昨天 17:32', type: '群组语音', from: '李工', peer: '运维值班群', room: 'RM86901', members: 5, dur: '16:44', mode: 'SFU', q: '良好', rec: '—' },
  { t: '昨天 15:20', type: '1v1 语音', from: '周涛', peer: '外部客户', room: 'RM86654', members: 2, dur: '05:51', mode: 'TURN', q: '强制挂断', rec: '已录制' }
])
const filtered = computed(() => rows.value.filter(r =>
  (!kw.value || [r.from, r.peer, r.room].some(x => x.includes(kw.value.trim()))) &&
  (!typeF.value || r.type === typeF.value) &&
  (!resultF.value || (resultF.value === '已接' ? r.q !== '未接' && r.q !== '强制挂断' : resultF.value === '强制挂断' ? r.q === '强制挂断' : r.q === '未接'))
))
const qCls = q => q === '良好' ? 'ok' : (q === '未接' ? 'off' : q === '强制挂断' ? 'danger' : 'warn')

function playback(r) {
  modal.value.open({
    title: '录制回放 · MFA 二次验证',
    body: `<p>回放通话录制属于高敏操作，请输入管理员 MFA 验证码。</p><div class="mfa-code">123456</div>
      <div class="field"><label>验证码</label><input id="pb-code" placeholder="输入 6 位验证码"></div>
      <div class="muted">通话：${r.from} → ${r.peer} · ${r.t} · 房间 ${r.room}</div>`,
    okText: '验证并回放',
    onOk: () => {
      if (document.getElementById('pb-code').value.trim() !== '123456') { toast('验证码错误', 'err'); return false }
      toast(`正在加载录制文件 MP4/H.264 · ${r.room}（演示）`)
    }
  })
}
function forceEnd(r) {
  modal.value.open({
    title: '强制挂断',
    body: `<p>确认强制结束通话「${r.from} → ${r.peer}」？</p>
      <div class="muted">双方将收到 CALL_FORCE_END 信令，该记录标记为「强制挂断」，操作写入审计日志。</div>`,
    okText: '强制挂断', danger: true,
    onOk: () => { r.q = '强制挂断'; r.dur = '已结束'; toast('通话已强制挂断，已写审计（演示）') }
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">通话记录来自 rtc 服务落库，回放受 rtc:record:playback 权限与 MFA 保护，强制挂断写入审计。<span class="code" style="margin-left:8px">FR-RTC-09/10/14</span></div>
    <div class="card">
      <div class="toolbar">
        <input v-model="kw" placeholder="发起人 / 对端 / 房间号" style="width:210px">
        <select v-model="typeF"><option value="">全部类型</option><option>1v1 语音</option><option>1v1 视频</option><option>群组语音</option><option>群组视频</option></select>
        <select v-model="resultF"><option value="">全部结果</option><option>已接</option><option>未接</option><option>强制挂断</option></select>
        <button class="btn primary" @click="toast('已按条件检索')">检索</button>
        <span class="sp"></span>
        <button class="btn" @click="toast('导出任务已提交（演示）')">导出</button>
      </div>
      <table class="tbl">
        <thead><tr><th>时间</th><th>类型</th><th>发起人</th><th>对端 / 群</th><th>房间号</th><th>成员数</th><th>时长</th><th>模式</th><th>质量</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="(r, i) in filtered" :key="i">
            <td>{{ r.t }}</td><td><span class="state info">{{ r.type }}</span></td>
            <td>{{ r.from }}</td><td>{{ r.peer }}</td><td><span class="code">{{ r.room }}</span></td>
            <td>{{ r.members }}</td><td>{{ r.dur }}</td><td><span class="code">{{ r.mode }}</span></td>
            <td><span class="state" :class="qCls(r.q)">{{ r.q }}</span></td>
            <td style="white-space:nowrap">
              <button v-if="r.rec === '已录制'" class="btn sm" @click="playback(r)">回放</button>
              <button class="btn sm danger" @click="forceEnd(r)">强制挂断</button>
            </td>
          </tr>
          <tr v-if="!filtered.length"><td colspan="10" class="loading">未找到匹配的通话记录</td></tr>
        </tbody>
      </table>
      <div class="pager">共 3,286 条 <button>‹</button><button class="on">1</button><button>2</button><button>›</button></div>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
