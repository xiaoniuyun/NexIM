<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useCallStore } from '../stores/call'
import { t } from '../i18n'
import { toast } from '../toast'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const call = useCallStore()
const mode = ref('login')
const account = ref('linxiao')
const password = ref('123456')
const regUser = ref('')
const regNick = ref('')
const regPwd = ref('')
const err = ref('')
const loading = ref(false)

async function doLogin() {
  err.value = ''
  if (!account.value.trim() || !password.value) { err.value = '请填写账号和密码'; return }
  loading.value = true
  try {
    await auth.login(account.value.trim(), password.value)
    chat.reset(); chat.connect(); call.init(chat); await chat.loadConversations()
    router.replace('/convs')
  } catch (e) { err.value = e.message } finally { loading.value = false }
}
async function doRegister() {
  err.value = ''
  if (regUser.value.trim().length < 3) { err.value = '账号至少 3 位'; return }
  if (regPwd.value.length < 6) { err.value = '密码至少 6 位'; return }
  loading.value = true
  try {
    await auth.register(regUser.value.trim(), regPwd.value, regNick.value.trim())
    chat.reset(); chat.connect(); call.init(chat); await chat.loadConversations()
    router.replace('/convs')
  } catch (e) { err.value = e.message } finally { loading.value = false }
}
</script>
<template>
  <div class="auth-view">
    <div class="auth-logo">N</div>
    <h1>NexIM</h1>
    <p class="auth-sub">企业级即时通讯平台<br>消息 / 文件 / 通话多端同步</p>
    <div class="auth-tabs">
      <button :class="{ on: mode === 'login' }" @click="mode = 'login'">{{ t('login') }}</button>
      <button :class="{ on: mode === 'register' }" @click="mode = 'register'">{{ t('register') }}</button>
    </div>
    <form v-if="mode === 'login'" @submit.prevent="doLogin">
      <label>{{ t('account') }}</label>
      <input v-model="account" :placeholder="t('loginHint')" autocomplete="username">
      <label>{{ t('password') }}</label>
      <input v-model="password" type="password" :placeholder="t('pwdHint')" autocomplete="current-password">
      <div class="auth-error">{{ err }}</div>
      <button class="btn-primary" type="submit" :disabled="loading">{{ loading ? '登录中...' : t('loginBtn') }}</button>
    </form>
    <form v-else @submit.prevent="doRegister">
      <label>{{ t('account') }}</label>
      <input v-model="regUser" placeholder="至少3位">
      <label>{{ t('nickname') }}</label>
      <input v-model="regNick" placeholder="选填">
      <label>{{ t('password') }}</label>
      <input v-model="regPwd" type="password" placeholder="至少6位">
      <div class="auth-error">{{ err }}</div>
      <button class="btn-primary" type="submit" :disabled="loading">{{ loading ? '注册中...' : t('regBtn') }}</button>
    </form>
    <div class="auth-demo">{{ t('demoAccount') }} <b>linxiao / 123456</b>（管理员）<br><b>zhangwei / 123456</b></div>
  </div>
</template>
