import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { API, readAuth, writeAuth } from '../api'

function deviceId() {
  // 设备标识全浏览器共享，避免同账号多标签互相踢下线
  let id = localStorage.getItem('h5_device')
  if (!id) { id = 'h5-' + Math.random().toString(36).slice(2, 12); localStorage.setItem('h5_device', id) }
  return id
}

function readUser() {
  try { return JSON.parse(readAuth('h5_user') || 'null') } catch (_) { return null }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(readAuth('h5_token'))
  const refresh = ref(readAuth('h5_refresh'))
  const user = ref(readUser())
  const device = ref(deviceId())
  const status = ref(localStorage.getItem('h5_status') || '在线')
  const isLogin = computed(() => !!token.value && !!user.value)

  function applyAuth(data) {
    token.value = data.accessToken
    refresh.value = data.refreshToken
    user.value = {
      uid: data.uid, username: data.username, nickname: data.nickname,
      avatar: data.avatar, roles: data.roles || []
    }
    writeAuth('h5_token', token.value)
    writeAuth('h5_refresh', refresh.value)
    writeAuth('h5_user', JSON.stringify(user.value))
  }
  function clearAuth() {
    token.value = ''; refresh.value = ''; user.value = null
    ;['h5_token', 'h5_refresh', 'h5_user'].forEach(k => writeAuth(k, null))
  }
  async function login(account, password) {
    const r = await API.login(account, password, device.value)
    applyAuth(r.data)
  }
  async function register(username, password, nickname) {
    const r = await API.register(username, password, nickname)
    applyAuth(r.data)
  }
  async function logout() {
    try { await API.logout() } catch (e) {}
    clearAuth()
  }
  function setStatus(s) { status.value = s; localStorage.setItem('h5_status', s) }

  return { token, refresh, user, device, status, isLogin, applyAuth, clearAuth, login, register, logout, setStatus }
})
