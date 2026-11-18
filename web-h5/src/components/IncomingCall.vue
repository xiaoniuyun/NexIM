<script setup>
import { useRouter } from 'vue-router'
import { useCallStore } from '../stores/call'

const router = useRouter()
const call = useCallStore()

function accept() {
  const convId = call.convId
  const type = call.callType
  call.accept()
  router.push('/call/' + encodeURIComponent(convId) + '?type=' + type)
}
function reject() { call.reject() }
</script>

<template>
  <div v-if="call.state === 'ringing'" class="incoming-mask">
    <div class="incoming-card">
      <div class="ic-avatar">{{ (call.peerName || '?').slice(0, 1) }}</div>
      <div class="ic-name">{{ call.peerName }}</div>
      <div class="ic-sub">{{ call.callType === 'video' ? '邀请你视频通话' : '邀请你语音通话' }}</div>
      <div class="ic-actions">
        <button class="ic-btn reject" @click="reject">拒绝</button>
        <button class="ic-btn accept" @click="accept">接听</button>
      </div>
    </div>
  </div>
</template>
