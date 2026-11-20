<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '../stores/chat'
import { t } from '../i18n'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const unread = computed(() => chat.unreadTotal)
const tabs = [
  {
    key: 'convs', label: 'msg', to: '/convs', solid: true,
    icon: 'M4 6.6A2.6 2.6 0 0 1 6.6 4h10.8A2.6 2.6 0 0 1 20 6.6v7.2a2.6 2.6 0 0 1-2.6 2.6H9.2L4.9 19.5V6.6z'
  },
  {
    key: 'channels', label: 'channels', to: '/groups',
    icon: 'M9.4 4.2 7.3 19.8M16.7 4.2l-2.1 15.6M4.3 9.2h15.4M3.9 14.8h15.4'
  },
  {
    key: 'contacts', label: 'contacts', to: '/contacts',
    icon: 'M12 11.2a3.4 3.4 0 1 0 0-6.8 3.4 3.4 0 0 0 0 6.8zM5.4 20c0-3.7 2.9-5.8 6.6-5.8s6.6 2.1 6.6 5.8'
  },
  {
    key: 'moments', label: 'moments', to: '/discover',
    icon: 'M20.4 12a8.4 8.4 0 1 1-2.5-6M20.4 4.4v4.4h-4.4'
  }
]
const active = computed(() => route.meta.tab || 'convs')
</script>
<template>
  <nav class="tabbar">
    <button v-for="tab in tabs" :key="tab.key" :class="{ on: active === tab.key }" @click="router.push(tab.to)">
      <span class="ti">
        <svg width="24" height="24" viewBox="0 0 24 24"
          :fill="tab.solid && active === tab.key ? 'currentColor' : 'none'"
          :stroke="tab.solid && active === tab.key ? 'none' : 'currentColor'"
          :stroke-width="active === tab.key ? 2 : 1.7" stroke-linecap="round" stroke-linejoin="round">
          <path :d="tab.icon" />
        </svg>
      </span>
      <span v-if="tab.key === 'convs' && unread" class="tb-badge">{{ unread > 99 ? '99+' : unread }}</span>
      <span>{{ t(tab.label) }}</span>
    </button>
  </nav>
</template>
