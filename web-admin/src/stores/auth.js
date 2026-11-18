import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { API } from '../api'

export const useAuthStore = defineStore('adminAuth', () => {
  const token = ref(localStorage.getItem('admin_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('admin_user') || 'null'))
  const perms = ref([])
  const isLogin = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => (user.value?.roles || []).includes('admin'))

  function apply(d) {
    token.value = d.accessToken
    user.value = { uid: d.uid, username: d.username, nickname: d.nickname, avatar: d.avatar, roles: d.roles || [] }
    localStorage.setItem('admin_token', token.value)
    localStorage.setItem('admin_user', JSON.stringify(user.value))
    API.permissions().then(r => { perms.value = r.data || [] }).catch(() => {})
  }
  function clear() {
    token.value = ''; user.value = null; perms.value = []
    localStorage.removeItem('admin_token'); localStorage.removeItem('admin_user')
  }
  async function login(account, password) {
    const r = await API.login(account, password)
    if (!(r.data?.roles || []).includes('admin')) throw new Error('该账号无管理后台权限')
    apply(r.data)
  }
  async function logout() { try { await API.logout() } catch (e) {} clear() }
  return { token, user, perms, isLogin, isAdmin, login, logout, clear, apply }
})
