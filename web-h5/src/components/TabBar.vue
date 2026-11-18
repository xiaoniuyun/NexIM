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
  { key: 'convs', icon: '💬', label: 'msg', to: '/convs' },
  { key: 'contacts', icon: '👥', label: 'contacts', to: '/contacts' },
  { key: 'discover', icon: '🧭', label: 'discover', to: '/discover' },
  { key: 'mine', icon: '👤', label: 'mine', to: '/mine' }
]
const active = computed(() => route.meta.tab || 'convs')
</script>
<template>
  <nav class="tabbar">
    <button v-for="tab in tabs" :key="tab.key" :class="{ on: active === tab.key }" @click="router.push(tab.to)">
      <span class="ti">{{ tab.icon }}</span>
      <span v-if="tab.key === 'convs' && unread" class="tb-badge">{{ unread > 99 ? '99+' : unread }}</span>
      <span>{{ t(tab.label) }}</span>
    </button>
  </nav>
</template>
