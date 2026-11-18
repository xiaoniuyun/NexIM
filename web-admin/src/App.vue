<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { toasts } from './toast'
import CmdPanel from './components/CmdPanel.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const cmdRef = ref(null)
const bellOpen = ref(false)

const menu = [
  { group: '概览', items: [['dashboard', '工作台', '▦']] },
  { group: '用户与组织', items: [['users', '用户管理', '👥'], ['depts', '组织架构', '🏢'], ['groups', '群组管理', '💬'], ['login-logs', '登录日志', '📋']] },
  { group: '权限治理', items: [['roles', '角色权限', '🛡️'], ['perm-preview', '权限预览', '🔑'], ['logs', '操作日志', '📜']] },
  { group: '内容与安全', items: [['audit', '消息审计', '👁️'], ['sensitive', '内容安全', '🚩'], ['notices', '公告管理', '🔔']] },
  { group: '音视频', items: [['calls', '通话记录', '📞'], ['rtc-quality', '质量监控', '📊']] },
  { group: '系统', items: [['settings', '系统设置', '⚙️']] }
]
const isLoginPage = computed(() => route.name === 'login')
const title = computed(() => route.meta.title || '工作台')
const alarms = [
  { ico: '🚩', bg: '#ffedf0', color: '#ef4b5b', t: '内容安全：敏感词命中突增', s: '近 1 小时命中 23 次，建议查看命中记录', to: 'sensitive' },
  { ico: '📋', bg: '#fff5e5', color: '#f59e0b', t: '登录日志：检测到异常登录', s: 'IP 203.0.113.42 连续失败 5 次', to: 'login-logs' }
]
function goAlarm(to) { bellOpen.value = false; router.push('/' + to) }
async function logout() { await auth.logout(); router.replace('/login') }
</script>
<template>
  <div v-if="isLoginPage" class="auth-wrap"><router-view /></div>
  <div v-else class="admin-shell">
    <div class="admin">
      <aside class="admin-side">
        <div class="as-brand"><span class="lg">N</span><span>NexIM Console<small style="display:block;font-size:10px;font-weight:400;opacity:.7">运营管理后台</small></span></div>
        <div id="adminMenu">
          <template v-for="g in menu" :key="g.group">
            <div class="as-group">{{ g.group }}</div>
            <button v-for="it in g.items" :key="it[0]" :class="{ on: '/' + it[0] === route.path }" @click="router.push('/' + it[0])">
              <span>{{ it[2] }}</span><span>{{ it[1] }}</span>
            </button>
          </template>
        </div>
        <div class="as-foot">当前登录：{{ auth.user?.nickname || '超级管理员' }}<br>权限版本 v12 · 全部数据范围<br>所有写操作均记录审计日志</div>
      </aside>
      <div class="admin-main">
        <div class="admin-top">
          <div class="crumb">控制台 / <b>{{ title }}</b></div>
          <div class="sp"></div>
          <div class="at-right">
            <div class="at-search" @click="cmdRef?.open()">🔍 全局搜索用户 / 消息 <kbd>Ctrl K</kbd></div>
            <div style="position:relative">
              <button class="icon-btn bell-btn" title="告警" @click="bellOpen = !bellOpen">🔔<i></i></button>
              <div v-if="bellOpen" class="bell-pop">
                <div class="bh"><span>告警中心</span><span class="muted">2 条未读</span></div>
                <div v-for="a in alarms" :key="a.t" class="bi" @click="goAlarm(a.to)">
                  <span class="bi-ico" :style="{ background: a.bg, color: a.color }">{{ a.ico }}</span>
                  <div style="flex:1"><div class="t">{{ a.t }}</div><div class="s">{{ a.s }}</div></div>
                </div>
              </div>
            </div>
            <button class="icon-btn" title="切换主题" @click="cmdRef?.toggleTheme()">🌙</button>
            <div class="admin-user" style="display:flex;align-items:center;gap:7px;font-size:12px">
              <span class="avatar sm" :style="{ background: auth.user?.avatar || 'linear-gradient(135deg,#3c4b8f,#7b61ff)' }">{{ (auth.user?.nickname || '超').slice(0,1) }}</span>
              {{ auth.user?.nickname || '超级管理员' }}
            </div>
            <button class="btn sm" @click="logout">退出</button>
          </div>
        </div>
        <div class="admin-content">
          <router-view v-slot="{ Component }"><component :is="Component" /></router-view>
        </div>
      </div>
    </div>
  </div>
  <CmdPanel ref="cmdRef" />
  <div class="toast-box">
    <div v-for="t in toasts" :key="t.id" class="toast" :class="t.type">{{ t.msg }}</div>
  </div>
</template>
