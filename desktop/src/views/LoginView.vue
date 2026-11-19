<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@h5/stores/auth'
import { toast } from '@h5/toast'
import { desk } from '../composables/useDesktop'

const auth = useAuthStore()
const account = ref('linxiao')
const password = ref('123456')
const loading = ref(false)
const showServer = ref(false)
const apiBase = ref(desk?.getConfig ? '' : '')
const wsUrl = ref('')
const version = ref(window.neximConfig?.version || '1.0.0')

if (desk) {
  desk.getConfig().then(c => {
    apiBase.value = c.apiBase
    wsUrl.value = c.wsUrl
  }).catch(() => {})
}

async function submit() {
  if (!account.value.trim() || !password.value) return toast('请输入账号和密码', 'err')
  loading.value = true
  try {
    await auth.login(account.value.trim(), password.value)
    toast('登录成功')
  } catch (e) {
    toast(e.message || '登录失败', 'err')
  } finally {
    loading.value = false
  }
}

async function saveServer() {
  if (!desk) return
  const r = await desk.setConfig({ apiBase: apiBase.value, wsUrl: wsUrl.value })
  if (!r.ok) return toast(r.message || '保存失败', 'err')
  toast('服务地址已更新，正在重新加载')
  setTimeout(() => location.reload(), 500)
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-brand">
      <div class="brand-logo">N</div>
      <h1>NexIM</h1>
      <p>企业即时通讯桌面端</p>
      <ul class="brand-points">
        <li>消息多端同步 · 已读回执</li>
        <li>端到端加密与敏感词防护</li>
        <li>音视频通话 · 屏幕截图速发</li>
      </ul>
      <div class="brand-ver">v{{ version }}</div>
    </div>
    <div class="login-panel">
      <h2>账号登录</h2>
      <p class="login-sub">使用企业账号登录，登录后可同步全部会话</p>
      <label class="field">
        <span>账号</span>
        <input v-model="account" autofocus placeholder="账号 / 用户名" @keyup.enter="submit">
      </label>
      <label class="field">
        <span>密码</span>
        <input v-model="password" type="password" placeholder="请输入密码" @keyup.enter="submit">
      </label>
      <button class="btn primary block" :disabled="loading" @click="submit">{{ loading ? '登录中…' : '登 录' }}</button>
      <div class="login-foot">
        <button class="link" @click="showServer = !showServer">服务器设置</button>
        <span>v{{ version }}</span>
      </div>
      <div v-if="showServer" class="server-box">
        <label class="field"><span>API 地址</span><input v-model="apiBase" placeholder="http://127.0.0.1:8080/api/v1"></label>
        <label class="field"><span>长连接地址</span><input v-model="wsUrl" placeholder="ws://127.0.0.1:8081/ws"></label>
        <button class="btn" @click="saveServer">保存并重新加载</button>
      </div>
      <div class="demo-tip">演示账号：linxiao / 123456（管理员）；liangchen / 123456</div>
    </div>
  </div>
</template>
