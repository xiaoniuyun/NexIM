<script setup>
import { ref } from 'vue'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'

const defaults = {
  retention: 180, fileLimit: 100, groupLimit: 2000, revokeWindow: 2, heartbeat: 30, signUrl: 15
}
const params = ref({ ...defaults })
const policies = ref([
  ['mfa', '管理员登录 MFA', '开启后管理员登录需二次验证', true],
  ['lock', '登录失败锁定', '连续 5 次失败锁定账号 15 分钟', true],
  ['remoteAlert', '异地登录告警', '检测到异地登录时推送告警', true],
  ['exportMfa', '导出消息需 MFA', '导出审计消息需二次验证', true],
  ['externalAudit', '外部联系人消息审计', '外部联系人会话全量留痕', false]
])
const modal = ref(null)

const fields = [
  ['retention', '云端消息保留（天）', '超过保留期的消息将从云端归档，本地仍可查看'],
  ['fileLimit', '单文件上传上限（MB）', '超过上限的文件将拒绝上传'],
  ['groupLimit', '群成员上限（人）', '单个群聊可容纳的最大成员数'],
  ['revokeWindow', '消息撤回窗口（分钟）', '发送后在此时间内可撤回消息'],
  ['heartbeat', '心跳间隔（秒）', '客户端与网关的心跳保活周期'],
  ['signUrl', '文件签名 URL 有效期（分钟）', '下载链接的预签名有效期']
]
function toggle(p) { p[3] = !p[3] }
function restore() {
  modal.value.open({
    title: '恢复默认配置', body: '<p>确认将所有参数与策略恢复为系统默认值？</p>', okText: '恢复默认', danger: true,
    onOk: () => { params.value = { ...defaults }; policies.value.forEach(p => { p[3] = p[0] !== 'externalAudit' }); toast('已恢复默认配置') }
  })
}
function save() {
  modal.value.open({
    title: '保存配置', body: '<p>确认保存当前配置？部分参数将在 5 秒内热更新至所有节点。</p>', okText: '保存并生效',
    onOk: () => toast('配置已保存并生效 · 热更新已下发至所有节点')
  })
}
</script>
<template>
  <div>
    <div class="proto-hint">系统级参数与安全策略配置，修改后热更新至所有节点。<span class="code" style="margin-left:8px">FR-ADMIN-10 / NFR-MNT-01</span></div>
    <div class="card">
      <div class="card-head"><h3>消息与文件</h3><span class="muted">核心运行参数</span></div>
      <div class="form-grid">
        <div v-for="fld in fields" :key="fld[0]" class="field">
          <label>{{ fld[1] }}</label>
          <input type="number" v-model.number="params[fld[0]]">
          <div class="fh">{{ fld[2] }}</div>
        </div>
      </div>
    </div>
    <div class="card">
      <div class="card-head"><h3>安全策略</h3><span class="muted">全局安全开关</span></div>
      <div v-for="p in policies" :key="p[0]" class="switch-row" @click="toggle(p)">
        <div>{{ p[1] }}<div class="sr-desc">{{ p[2] }}</div></div>
        <span class="switch" :class="{ on: p[3] }"></span>
      </div>
    </div>
    <div style="display:flex;justify-content:flex-end;gap:9px">
      <button class="btn" @click="restore">恢复默认</button>
      <button class="btn primary" @click="save">💾 保存配置</button>
    </div>
    <ModalBox ref="modal" />
  </div>
</template>
