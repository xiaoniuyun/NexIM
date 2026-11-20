<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useAuthStore } from '@/shared/stores/auth'
import { avatarUrl, DEFAULT_AVATAR } from '@/shared/api'
import { useChatStore } from '@/shared/stores/chat'
import Icon from '../components/Icon.vue'
import ConvColumn from '../components/ConvColumn.vue'
import ChatPane from '../components/ChatPane.vue'
import ContactsPane from '../components/ContactsPane.vue'
import FilesPane from '../components/FilesPane.vue'
import FavoritesPane from '../components/FavoritesPane.vue'
import CallsPane from '../components/CallsPane.vue'
import NoticesPane from '../components/NoticesPane.vue'
import StatsPane from '../components/StatsPane.vue'
import SettingsPane from '../components/SettingsPane.vue'
import GlobalSearch from '../components/GlobalSearch.vue'
import { readCache } from '../composables/useDesktop'

const auth = useAuthStore()
const chat = useChatStore()
const pane = ref('msg')
const searchOpen = ref(false)
const myAvatar = computed(() => avatarUrl(auth.user?.avatar) || DEFAULT_AVATAR)

const navs = [
  { key: 'msg', icon: 'chat', label: '消息' },
  { key: 'contacts', icon: 'contacts', label: '通讯录' },
  { key: 'favorites', icon: 'star', label: '收藏' },
  { key: 'files', icon: 'folder', label: '文件' },
  { key: 'calls', icon: 'phone', label: '通话' },
  { key: 'stats', icon: 'chart', label: '统计' }
]

const listWidth = ref(Number(localStorage.getItem('nexim_list_width')) || 278)
const dragging = ref(false)
let dragStartX = 0
let dragStartW = 0

function startDrag(e) {
  dragging.value = true
  dragStartX = e.clientX
  dragStartW = listWidth.value
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', stopDrag)
  document.body.style.userSelect = 'none'
}
function onDrag(e) {
  // 限制 240-480px，避免把对话区挤没
  listWidth.value = Math.min(480, Math.max(240, dragStartW + e.clientX - dragStartX))
}
function stopDrag() {
  dragging.value = false
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', stopDrag)
  document.body.style.userSelect = ''
  localStorage.setItem('nexim_list_width', String(listWidth.value))
}

async function openConv(convId) {
  if (!convId) return
  pane.value = 'msg'
  chat.messages = readCache(convId)
  await chat.openConv(convId)
}

function go(key) {
  const prev = pane.value
  pane.value = key
  // 从其他页面切回消息时，重新打开当前会话并补发已读
  if (key === 'msg' && prev !== 'msg' && chat.currentConv) chat.openConv(chat.currentConv)
}

function onSearchEvent() { searchOpen.value = true }
function onOpenEvent(e) { openConv(e.detail) }

onMounted(async () => {
  window.addEventListener('nexim:search', onSearchEvent)
  window.addEventListener('nexim:open-conv', onOpenEvent)
  await chat.loadConversations()
  if (!chat.currentConv && chat.convs.length) openConv(chat.convs[0].convId)
})
onBeforeUnmount(() => {
  window.removeEventListener('nexim:search', onSearchEvent)
  window.removeEventListener('nexim:open-conv', onOpenEvent)
  stopDrag()
})

const unread = computed(() => chat.unreadTotal)
</script>

<template>
  <div class="app-shell">
    <nav class="rail">
      <button class="rail-avatar" :title="auth.user?.nickname" @click="pane = 'settings'">
        <img v-if="myAvatar" :src="myAvatar" alt="头像">
        <template v-else>{{ (auth.user?.nickname || '我').slice(0, 1) }}</template>
        <i class="rail-dot" :class="{ off: !chat.connected }"></i>
      </button>
      <button
        v-for="n in navs" :key="n.key"
        class="rail-btn" :class="{ on: pane === n.key }"
        :title="n.label" @click="go(n.key)"
      >
        <Icon :name="n.icon" :size="22" />
        <span v-if="n.key === 'msg' && unread > 0" class="rail-badge">{{ unread > 99 ? '99+' : unread }}</span>
      </button>
      <div class="rail-sp"></div>
      <button class="rail-btn" :class="{ on: pane === 'notices' }" title="通知公告" @click="go('notices')">
        <Icon name="bell" :size="22" />
        <i v-if="unread > 0" class="rail-dot-badge"></i>
      </button>
      <button class="rail-btn" :class="{ on: pane === 'settings' }" title="设置" @click="pane = 'settings'">
        <Icon name="menu" :size="22" />
      </button>
    </nav>

    <ConvColumn
      v-if="pane === 'msg'" class="list-col"
      :style="{ width: listWidth + 'px' }"
      @open="openConv" @search="searchOpen = true"
    />
    <div v-if="pane === 'msg'" class="splitter" :class="{ dragging }" title="拖拽调整宽度" @mousedown.prevent="startDrag"></div>

    <main class="main-col">
      <ChatPane v-if="pane === 'msg'" />
      <ContactsPane v-else-if="pane === 'contacts'" @open-conv="openConv" />
      <FilesPane v-else-if="pane === 'files'" @open-conv="openConv" />
      <FavoritesPane v-else-if="pane === 'favorites'" @open-conv="openConv" />
      <CallsPane v-else-if="pane === 'calls'" />
      <NoticesPane v-else-if="pane === 'notices'" />
      <StatsPane v-else-if="pane === 'stats'" />
      <SettingsPane v-else-if="pane === 'settings'" />
    </main>

    <GlobalSearch v-if="searchOpen" @close="searchOpen = false" @open-conv="openConv" />
  </div>
</template>
