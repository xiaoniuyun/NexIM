import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue') },
  { path: '/dashboard', name: 'dashboard', component: () => import('./views/DashboardView.vue'), meta: { title: '工作台', group: '概览', auth: true } },
  { path: '/users', name: 'users', component: () => import('./views/UsersView.vue'), meta: { title: '用户管理', group: '用户与组织', auth: true } },
  { path: '/depts', name: 'depts', component: () => import('./views/DeptsView.vue'), meta: { title: '组织架构', group: '用户与组织', auth: true } },
  { path: '/groups', name: 'groups', component: () => import('./views/GroupsView.vue'), meta: { title: '群组管理', group: '用户与组织', auth: true } },
  { path: '/login-logs', name: 'loginLogs', component: () => import('./views/LoginLogsView.vue'), meta: { title: '登录日志', group: '用户与组织', auth: true } },
  { path: '/roles', name: 'roles', component: () => import('./views/RolesView.vue'), meta: { title: '角色权限', group: '权限治理', auth: true } },
  { path: '/perm-preview', name: 'permPreview', component: () => import('./views/PermPreviewView.vue'), meta: { title: '权限预览', group: '权限治理', auth: true } },
  { path: '/logs', name: 'logs', component: () => import('./views/LogsView.vue'), meta: { title: '操作日志', group: '权限治理', auth: true } },
  { path: '/audit', name: 'audit', component: () => import('./views/MsgAuditView.vue'), meta: { title: '消息审计', group: '内容与安全', auth: true } },
  { path: '/sensitive', name: 'sensitive', component: () => import('./views/SensitiveView.vue'), meta: { title: '内容安全', group: '内容与安全', auth: true } },
  { path: '/notices', name: 'notices', component: () => import('./views/NoticesView.vue'), meta: { title: '公告管理', group: '内容与安全', auth: true } },
  { path: '/calls', name: 'calls', component: () => import('./views/CallsView.vue'), meta: { title: '通话记录', group: '音视频', auth: true } },
  { path: '/rtc-quality', name: 'rtcQuality', component: () => import('./views/RtcQualityView.vue'), meta: { title: '质量监控', group: '音视频', auth: true } },
  { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue'), meta: { title: '系统设置', group: '系统', auth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
]

const router = createRouter({
  history: createWebHashHistory('/admin/'),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

router.beforeEach(to => {
  const auth = useAuthStore()
  if (to.meta.auth && !auth.isLogin) return { name: 'login' }
  if (to.name === 'login' && auth.isLogin) return { name: 'dashboard' }
  return true
})
export default router
