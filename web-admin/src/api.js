const API_BASE = '/api/v1'
function token() { return localStorage.getItem('admin_token') || '' }

export function qs(obj = {}) {
  return new URLSearchParams(Object.entries(obj).filter(([, v]) => v !== '' && v !== null && v !== undefined)).toString()
}

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

// 带鉴权的文件下载（CSV 导出等）
export async function downloadFile(path, filename) {
  const res = await fetch(API_BASE + path, { headers: { Authorization: 'Bearer ' + token() } })
  if (!res.ok) throw new Error('下载失败 ' + res.status)
  const url = URL.createObjectURL(await res.blob())
  const a = document.createElement('a')
  a.href = url; a.download = filename; a.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export const API = {
  login: (account, password) => api('/auth/login', { method: 'POST', body: { account, password, deviceId: 'admin-web', platform: 'Web' } }),
  logout: () => api('/auth/logout', { method: 'POST' }),
  permissions: () => api('/auth/permissions'),
  dashboard: () => api('/admin/dashboard'),

  // 用户管理：服务端筛选 / 分页 / 导出 / 角色绑定
  users: (params = {}) => api('/admin/users?' + qs(params)),
  createUser: body => api('/admin/users', { method: 'POST', body }),
  updateUser: (id, body) => api(`/admin/users/${id}`, { method: 'PUT', body }),
  deleteUser: id => api(`/admin/users/${id}`, { method: 'DELETE' }),
  resetPwd: id => api(`/admin/users/${id}/reset-password`, { method: 'POST' }),
  enableUser: id => api(`/admin/users/${id}`, { method: 'PUT', body: { status: 1 } }),
  setUserRoles: (id, roleIds) => api(`/admin/users/${id}/roles`, { method: 'PUT', body: { roleIds } }),
  userPermissions: id => api(`/admin/user-permissions/${id}`),
  exportUsers: params => downloadFile('/admin/users-export?' + qs(params), 'nexim-users.csv'),

  // 角色权限
  permCatalog: () => api('/admin/permissions'),
  roles: () => api('/admin/roles'),
  createRole: body => api('/admin/roles', { method: 'POST', body }),
  updateRole: (id, body) => api(`/admin/roles/${id}`, { method: 'PUT', body }),
  deleteRole: id => api(`/admin/roles/${id}`, { method: 'DELETE' }),
  grantRole: body => api('/admin/roles/grant', { method: 'POST', body }),

  // 组织架构
  depts: () => api('/depts'),
  deptTree: () => api('/depts/tree'),
  deptMembers: id => api(`/depts/${id}/members`),
  createDept: body => api('/admin/depts', { method: 'POST', body }),
  updateDept: (id, body) => api(`/admin/depts/${id}`, { method: 'PUT', body }),
  deleteDept: id => api(`/admin/depts/${id}`, { method: 'DELETE' }),

  groups: () => api('/groups'),
  audit: () => api('/admin/audit'),
  messageAudit: () => api('/admin/audit/messages'),
  notices: () => api('/admin/notices'),
  createNotice: body => api('/admin/notices', { method: 'POST', body }),
  sensitive: () => api('/admin/sensitive-words'),
  addSensitive: word => api('/admin/sensitive-words', { method: 'POST', body: { word } })
}
