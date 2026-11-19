<script setup>
import { ref } from 'vue'
import Modal from './Modal.vue'
import Avatar from '@h5/components/Avatar.vue'
import { useChatStore } from '@h5/stores/chat'
import { API } from '@h5/api'
import { toast } from '@h5/toast'

const props = defineProps({ msg: Object })
const emit = defineEmits(['close'])
const chat = useChatStore()
const ids = ref([])
const note = ref('')
const sending = ref(false)

async function submit() {
  if (!ids.value.length) return toast('请选择转发目标', 'err')
  sending.value = true
  try {
    await API.forward(props.msg.msgId, ids.value, note.value.trim())
    toast(`已转发到 ${ids.value.length} 个会话`)
    emit('close')
  } catch (e) {
    toast(e.message, 'err')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <Modal title="转发消息" width="440px" @close="emit('close')">
    <div class="forward-preview">{{ msg.content || `[${msg.type}]` }}</div>
    <div class="sec-title">选择会话</div>
    <div class="forward-list">
      <label v-for="c in chat.convs" :key="c.convId" class="forward-row">
        <input v-model="ids" type="checkbox" :value="c.convId">
        <Avatar :name="c.peerName || c.convId" :src="c.peerAvatar" size="sm" />
        <span>{{ c.peerName || c.convId }}</span>
        <span v-if="c.convType === 2" class="tag">群</span>
      </label>
    </div>
    <input v-model="note" class="forward-note" placeholder="附加留言（可选）">
    <template #footer>
      <button class="btn" @click="emit('close')">取消</button>
      <button class="btn primary" :disabled="sending" @click="submit">转发</button>
    </template>
  </Modal>
</template>
