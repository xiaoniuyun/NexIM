<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
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

const navs = [
  { key: 'msg', icon: 'chat', label: '消息' },
  { key: 'contacts', icon: 'contacts', label: '通讯录' },
  { key: 'files', icon: 'folder', label: '文件' },
  { key: 'favorites', icon: 'star', label: '收藏' },
  { key: 'calls', icon: 'phone', label: '通话' },
  { key: 'notices', icon: 'bell', label: '通知公告' },
  { key: 'stats', icon: 'chart', label: '统计' }
]

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
})

const unread = computed(() => chat.unreadTotal)
</script>

<template>
  <div class="app-shell">
    <nav class="rail">
      <div class="rail-logo">N</div>
      <button
        v-for="n in navs" :key="n.key"
        class="rail-btn" :class="{ on: pane === n.key }"
        :title="n.label" @click="go(n.key)"
      >
        <Icon :name="n.icon" :size="21" />
        <span v-if="n.key === 'msg' && unread > 0" class="rail-badge">{{ unread > 99 ? '99+' : unread }}</span>
      </button>
      <div class="rail-sp"></div>
      <button class="rail-btn" :class="{ on: pane === 'settings' }" title="设置" @click="pane = 'settings'">
        <Icon name="settings" :size="21" />
      </button>
      <button class="rail-avatar" :title="auth.user?.nickname" @click="pane = 'settings'">
        {{ (auth.user?.nickname || '我').slice(0, 1) }}
      </button>
    </nav>

    <ConvColumn v-if="pane === 'msg'" class="list-col" @open="openConv" @search="searchOpen = true" />

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
