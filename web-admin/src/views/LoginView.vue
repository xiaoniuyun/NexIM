<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
const router = useRouter()
const auth = useAuthStore()
const account = ref('linxiao')
const password = ref('123456')
const err = ref('')
const loading = ref(false)
async function doLogin() {
  err.value = ''
  if (!account.value.trim() || !password.value) { err.value = '请填写账号和密码'; return }
  loading.value = true
  try { await auth.login(account.value.trim(), password.value); router.replace('/dashboard') }
  catch (e) { err.value = e.message } finally { loading.value = false }
}
</script>
<template>
  <div class="auth-card">
    <div class="auth-logo">N</div>
    <h1>NexIM Console</h1>
    <p class="auth-sub">企业级即时通讯 · 管理后台</p>
    <form @submit.prevent="doLogin">
      <label>账号</label>
      <input v-model="account" placeholder="请输入管理员账号" autocomplete="username">
      <label>密码</label>
      <input v-model="password" type="password" placeholder="请输入密码" autocomplete="current-password">
      <div class="auth-error">{{ err }}</div>
      <button class="btn primary" type="submit" :disabled="loading">{{ loading ? '登录中...' : '登录' }}</button>
    </form>
    <div class="auth-demo">管理员演示账号 <b>linxiao / 123456</b></div>
  </div>
</template>
