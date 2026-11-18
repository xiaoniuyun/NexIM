<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useChatStore } from './stores/chat'
import { useCallStore } from './stores/call'
import ToastHost from './components/ToastHost.vue'
import TabBar from './components/TabBar.vue'
import IncomingCall from './components/IncomingCall.vue'

const auth = useAuthStore()
const chat = useChatStore()
const call = useCallStore()
const route = useRoute()
const showChrome = computed(() => auth.isLogin && route.meta.tab && !['login', 'call'].includes(route.name))

if (auth.isLogin) {
  chat.connect()
  chat.loadConversations()
  // 注册通话信令回调（CALL_*），任意页面都能收到来电
  call.init(chat)
}
</script>
<template>
  <div class="phone">
    <div class="app-body">
      <router-view v-slot="{ Component }">
        <component :is="Component" />
      </router-view>
    </div>
    <TabBar v-if="showChrome" />
    <IncomingCall />
    <ToastHost />
  </div>
</template>
