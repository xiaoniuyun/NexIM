import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'

const routes = [
  { path: '/', redirect: '/convs' },
  { path: '/login', name: 'login', component: () => import('./views/LoginView.vue') },
  { path: '/convs', name: 'convs', component: () => import('./views/ConvsView.vue'), meta: { tab: 'convs', auth: true } },
  { path: '/chat/:convId', name: 'chat', component: () => import('./views/ChatView.vue'), meta: { tab: 'convs', auth: true } },
  { path: '/call/:convId', name: 'call', component: () => import('./views/CallView.vue'), meta: { tab: 'convs', auth: true } },
  { path: '/create-group', name: 'createGroup', component: () => import('./views/CreateGroupView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/contacts', name: 'contacts', component: () => import('./views/ContactsView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/friends', name: 'friends', component: () => import('./views/FriendsView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/add-friend', name: 'addFriend', component: () => import('./views/AddFriendView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/groups', name: 'groups', component: () => import('./views/GroupsView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/dept', name: 'dept', component: () => import('./views/DeptView.vue'), meta: { tab: 'contacts', auth: true } },
  { path: '/discover', name: 'discover', component: () => import('./views/DiscoverView.vue'), meta: { tab: 'discover', auth: true } },
  { path: '/files', name: 'files', component: () => import('./views/FilesView.vue'), meta: { tab: 'discover', auth: true } },
  { path: '/favorites', name: 'favorites', component: () => import('./views/FavoritesView.vue'), meta: { tab: 'discover', auth: true } },
  { path: '/mine', name: 'mine', component: () => import('./views/MineView.vue'), meta: { tab: 'mine', auth: true } },
  { path: '/profile', name: 'profile', component: () => import('./views/ProfileView.vue'), meta: { tab: 'mine', auth: true } },
  { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue'), meta: { tab: 'mine', auth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/convs' }
]

const router = createRouter({
  history: createWebHashHistory('/h5/'),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.auth && !auth.isLogin) return { name: 'login' }
  if (to.name === 'login' && auth.isLogin) return { name: 'convs' }
  return true
})

export default router
