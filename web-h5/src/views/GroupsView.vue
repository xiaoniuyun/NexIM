<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { useChatStore } from '../stores/chat'
import { t } from '../i18n'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'

const router = useRouter()
const chat = useChatStore()
const groups = ref([])
onMounted(async () => {
  try { const r = await API.groups(); groups.value = r.data || [] } catch (e) { toast(e.message, 'err') }
})
function enter(g) {
  const convId = 'conv_' + g.groupId
  if (!chat.convs.find(c => c.convId === convId)) {
    chat.convs.unshift({ convId, convType: 2, peerId: g.groupId, peerName: g.name, unread: 0, lastMsg: '', lastMsgAt: Date.now() })
  }
  router.push('/chat/' + encodeURIComponent(convId))
}
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('groupList') }}</div><div class="ch-sub">{{ groups.length }} 个群聊</div></div>
    <div class="sp"></div>
    <button class="btn-sm" @click="router.push('/create-group')">＋ 发起群聊</button>
  </div>
  <div class="h5-body">
    <button v-for="g in groups" :key="g.groupId" class="qentry" @click="enter(g)">
      <Avatar :name="g.name" />
      <div><div class="qe-t">{{ g.name }}</div><div class="qe-s">{{ g.memberCount }} 位成员 · {{ g.notice || '' }}</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <div v-if="!groups.length" class="loading">{{ t('empty') }}</div>
  </div>
</template>
