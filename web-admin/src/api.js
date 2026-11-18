const API_BASE = '/api/v1'
function token() { return localStorage.getItem('admin_token') || '' }

export async function api(path, opts = {}) {
  opts.headers = opts.headers || {}
  if (token()) opts.headers['Authorization'] = 'Bearer ' + token()
  if (opts.body && !(opts.body instanceof FormData) && typeof opts.body !== 'string') {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(opts.body)
  }
  const res = await fetch(API_BASE + path, opts)
  let data; try { data = await res.json() } catch (_) { data = {} }
  if (res.status >= 400 && data && data.code !== 0) throw new Error(data.message || ('请求失败 ' + res.status))
  return data
}

export const API = {
  login: (account, password) => api('/auth/login', { method: 'POST', body: { account, password, deviceId: 'admin-web', platform: 'Web' } }),
  logout: () => api('/auth/logout', { method: 'POST' }),
  permissions: () => api('/auth/permissions'),
  dashboard: () => api('/admin/dashboard'),
  users: (page = 1, size = 20) => api(`/admin/users?page=${page}&size=${size}`),
  createUser: body => api('/admin/users', { method: 'POST', body }),
  updateUser: (id, body) => api(`/admin/users/${id}`, { method: 'PUT', body }),
  deleteUser: id => api(`/admin/users/${id}`, { method: 'DELETE' }),
  resetPwd: id => api(`/admin/users/${id}/reset-password`, { method: 'POST' }),
  enableUser: id => api(`/admin/users/${id}`, { method: 'PUT', body: { status: 1 } }),
  roles: () => api('/admin/roles'),
  grantRole: body => api('/admin/roles/grant', { method: 'POST', body }),
  audit: () => api('/admin/audit'),
  messageAudit: () => api('/admin/audit/messages'),
  notices: () => api('/admin/notices'),
  createNotice: body => api('/admin/notices', { method: 'POST', body }),
  sensitive: () => api('/admin/sensitive-words'),
  addSensitive: word => api('/admin/sensitive-words', { method: 'POST', body: { word } }),
  depts: () => api('/depts'),
  groups: () => api('/groups')
}
