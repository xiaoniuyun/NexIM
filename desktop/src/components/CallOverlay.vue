<script setup>
import { ref, watch, nextTick, computed } from 'vue'
import Avatar from '@/shared/components/Avatar.vue'
import { useCallStore } from '@/shared/stores/call'

const call = useCallStore()
const localVideo = ref(null)
const remoteVideo = ref(null)

watch([() => call.state, () => call.callType], async () => {
  await nextTick()
  call.bindElements(localVideo.value, remoteVideo.value)
})

const timeText = computed(() => {
  const s = call.duration || 0
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
})
const visible = computed(() => call.state !== 'idle')
</script>

<template>
  <div v-if="visible" class="call-overlay">
    <div class="call-card">
      <div class="call-top">
        <Avatar :name="call.peerName || '通话'" size="lg" />
        <h3>{{ call.peerName || '未知联系人' }}</h3>
        <p class="call-status">
          <template v-if="call.state === 'ringing'">邀请你{{ call.callType === 'video' ? '视频' : '语音' }}通话…</template>
          <template v-else-if="call.state === 'calling'">正在等待对方接听…</template>
          <template v-else-if="call.state === 'connecting'">正在建立连接…</template>
          <template v-else-if="call.state === 'active'">{{ timeText }}</template>
          <template v-else>{{ call.notice || '通话已结束' }}</template>
        </p>
      </div>

      <div class="call-stage" :class="{ audio: call.callType === 'audio' }">
        <video ref="remoteVideo" autoplay playsinline class="remote-video"></video>
        <video ref="localVideo" autoplay playsinline muted class="local-video" v-show="call.callType === 'video'"></video>
      </div>

      <div class="call-actions">
        <template v-if="call.state === 'ringing'">
          <button class="call-btn ok" @click="call.accept()">接听</button>
          <button class="call-btn no" @click="call.reject()">拒绝</button>
        </template>
        <template v-else-if="call.state === 'calling' || call.state === 'connecting'">
          <button class="call-btn no" @click="call.hangup()">取消</button>
        </template>
        <template v-else-if="call.state === 'active'">
          <button class="call-btn" :class="{ active: call.muted }" @click="call.toggleMute()">{{ call.muted ? '取消静音' : '静音' }}</button>
          <button v-if="call.callType === 'video'" class="call-btn" :class="{ active: !call.camOn }" @click="call.toggleCam()">{{ call.camOn ? '关闭摄像头' : '开启摄像头' }}</button>
          <button class="call-btn no" @click="call.hangup()">挂断</button>
        </template>
        <template v-else>
          <button class="call-btn" @click="call.reset()">关闭</button>
        </template>
      </div>
    </div>
  </div>
</template>
