import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { API } from '../api'

export const useAuthStore = defineStore('adminAuth', () => {
  const token = ref(localStorage.getItem('admin_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('admin_user') || 'null'))
  const cached = (() => { try { return JSON.parse(localStorage.getItem('admin_perms') || 'null') } catch (e) { return null } })()
  const perms = ref(cached?.permissions || [])
  const dataScope = ref(cached?.dataScope || 'self')
  const version = ref(cached?.version || '—')
  const menus = ref(cached?.menus || [])
  const isLogin = computed(() => !!token.value && !!user.value)
  const isAdmin = computed(() => (user.value?.roles || []).includes('admin'))
  const can = code => perms.value.includes('*') || perms.value.includes(code)

  function refreshPerms() {
    return API.permissions().then(r => {
      const d = r.data || {}
      perms.value = d.permissions || []
      dataScope.value = d.dataScope || 'self'
      version.value = d.version || '1'
      menus.value = d.menus || []
      localStorage.setItem('admin_perms', JSON.stringify(d))
    }).catch(() => {})
  }
  function apply(d) {
    token.value = d.accessToken
    user.value = { uid: d.uid, username: d.username, nickname: d.nickname, avatar: d.avatar, roles: d.roles || [] }
    localStorage.setItem('admin_token', token.value)
    localStorage.setItem('admin_user', JSON.stringify(user.value))
    return refreshPerms()
  }
  function clear() {
    token.value = ''; user.value = null; perms.value = []; dataScope.value = 'self'; version.value = '—'; menus.value = []
    localStorage.removeItem('admin_token'); localStorage.removeItem('admin_user'); localStorage.removeItem('admin_perms')
  }
  async function login(account, password) {
    const r = await API.login(account, password)
    if (!(r.data?.roles || []).includes('admin')) throw new Error('该账号无管理后台权限')
    await apply(r.data)
  }
  async function logout() { try { await API.logout() } catch (e) {} clear() }
  return { token, user, perms, dataScope, version, menus, isLogin, isAdmin, can, refreshPerms, login, logout, clear, apply }
})
