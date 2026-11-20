// Electron 桌面端由 preload 注入服务地址；浏览器端保持同源相对路径
const API_BASE = (typeof window !== 'undefined' && window.neximConfig && window.neximConfig.apiBase) || '/api/v1'

// 标签页优先：同一浏览器可同时登录多个账号（各自标签页独立会话），
// localStorage 仅作为新开标签页的继承来源。
export function readAuth(key) { return sessionStorage.getItem(key) || localStorage.getItem(key) || '' }
export function writeAuth(key, value) {
  if (value === null || value === undefined) {
    sessionStorage.removeItem(key); localStorage.removeItem(key)
  } else {
    sessionStorage.setItem(key, value); localStorage.setItem(key, value)
  }
}

function getToken() { return readAuth('desk_token') }
function getRefresh() { return readAuth('desk_refresh') }

export async function api(path, opts = {}) {
  opts.headers = opts.headers || {}
  const token = getToken()
  if (token) opts.headers['Authorization'] = 'Bearer ' + token
  if (opts.body && !(opts.body instanceof FormData) && typeof opts.body !== 'string') {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(opts.body)
  }
  const res = await fetch(API_BASE + path, opts)
  let data
  try { data = await res.json() } catch (_) { data = {} }
  if (data && data.code === 2001 && getRefresh()) {
    const r = await fetch(API_BASE + '/auth/refresh', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: getRefresh() })
    })
    const rd = await r.json()
    if (rd.code === 0) {
      writeAuth('desk_token', rd.data.accessToken)
      writeAuth('desk_refresh', rd.data.refreshToken)
      opts.headers['Authorization'] = 'Bearer ' + rd.data.accessToken
      const retry = await fetch(API_BASE + path, opts)
      return retry.json()
    }
  }
  if (res.status >= 400 && data && data.code !== 0) throw new Error(data.message || ('请求失败 ' + res.status))
  return data
}

export function wsUrl() {
  const p = new URLSearchParams({
    token: getToken(),
    device: localStorage.getItem('desk_device') || 'desk',
    platform: 'DESKTOP'
  })
  const custom = typeof window !== 'undefined' && window.neximConfig && window.neximConfig.wsUrl
  if (custom) return custom + (custom.includes('?') ? '&' : '?') + p
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws?${p}`
}

// 文件下载/预览地址：<img>/<a> 无法携带 Authorization，改用 query token
export function fileUrl(fileId) {
  return API_BASE + '/files/' + encodeURIComponent(fileId) + '/download?token=' + encodeURIComponent(getToken())
}

// 头像字段：file:<fileId> 表示上传的图片，历史颜色值或无值返回空（由 Avatar 渲染首字母）
export function avatarUrl(avatar) {
  const v = String(avatar || '')
  return v.startsWith('file:') ? fileUrl(v.slice(5)) : ''
}

// 默认头像：未上传头像时统一展示内置人像剪影
export const DEFAULT_AVATAR = API_BASE + '/assets/default-avatar.svg'

export const API = {
  login: (account, password, deviceId) => api('/auth/login', { method: 'POST', body: { account, password, deviceId, platform: 'DESKTOP' } }),
  register: (username, password, nickname) => api('/auth/register', { method: 'POST', body: { username, password, nickname } }),
  logout: () => api('/auth/logout', { method: 'POST' }),
  devices: () => api('/auth/devices'),
  deleteDevice: id => api('/auth/devices/' + encodeURIComponent(id), { method: 'DELETE' }),
  permissions: () => api('/auth/permissions'),
  me: () => api('/users/me'),
  updateMe: body => api('/users/me', { method: 'PUT', body }),
  users: () => api('/users'),
  userSearch: q => api('/users/search?q=' + encodeURIComponent(q)),
  friends: () => api('/friends'),
  friendApply: (toUid, message) => api('/friends/apply', { method: 'POST', body: { toUid, message } }),
  friendApplies: () => api('/friends/applies'),
  friendHandle: (id, accept) => api('/friends/applies/' + id, { method: 'PUT', body: { accept } }),
  depts: () => api('/depts'),
  deptMembers: id => api('/depts/' + id + '/members'),
  conversations: () => api('/conversations'),
  deleteConv: id => api('/conversations/' + encodeURIComponent(id), { method: 'DELETE' }),
  messages: convId => api('/conversations/' + encodeURIComponent(convId) + '/messages'),
  convSettings: (id, body) => api('/conversations/' + encodeURIComponent(id) + '/settings', { method: 'PATCH', body }),
  readAll: () => api('/conversations/read-all', { method: 'POST' }),
  markRead: convId => api('/messages/read', { method: 'POST', body: { convId } }),
  // WebSocket 不可用时的可靠发送通道（clientMsgId 幂等，可与 WS 重复提交）
  sendMessage: body => api('/messages/send', { method: 'POST', body }),
  revoke: msgId => api('/messages/revoke', { method: 'POST', body: { msgId } }),
  searchMessages: (keyword, convId) => api('/messages/search', { method: 'POST', body: { keyword, convId } }),
  forward: (msgId, convIds, note) => api('/messages/forward', { method: 'POST', body: { msgId, convIds, note } }),
  rtcConfig: () => api('/rtc/config'),
  calls: (limit = 50) => api('/calls?limit=' + limit),
  reportQuality: (callId, body) => api('/calls/' + encodeURIComponent(callId) + '/quality', { method: 'POST', body }),
  favorites: () => api('/favorites'),
  favorite: msgId => api('/favorites', { method: 'POST', body: { msgId } }),
  favoriteRemove: id => api('/favorites/' + id, { method: 'DELETE' }),
  blacklist: () => api('/contacts/blacklist'),
  blockUser: uid => api('/contacts/blacklist', { method: 'POST', body: { uid } }),
  unblockUser: uid => api('/contacts/blacklist/' + encodeURIComponent(uid), { method: 'DELETE' }),
  groups: () => api('/groups'),
  groupDetail: id => api('/groups/' + encodeURIComponent(id)),
  groupMembers: id => api('/groups/' + encodeURIComponent(id) + '/members'),
  groupCreate: body => api('/groups', { method: 'POST', body }),
  groupInvite: (id, uids) => api('/groups/' + encodeURIComponent(id) + '/invite', { method: 'POST', body: { members: uids } }),
  groupLeave: (id, uid) => api('/groups/' + encodeURIComponent(id) + '/members/' + encodeURIComponent(uid), { method: 'DELETE' }),
  groupDissolve: id => api('/groups/' + encodeURIComponent(id), { method: 'DELETE' }),
  files: () => api('/files'),
  fileUpload: fd => api('/files/upload', { method: 'POST', body: fd }),
  fileDelete: id => api('/files/' + encodeURIComponent(id), { method: 'DELETE' }),
  notices: () => api('/notices')
}
