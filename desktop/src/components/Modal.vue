<script setup>
defineProps({ title: String, width: { type: String, default: '420px' } })
const emit = defineEmits(['close'])
</script>
<template>
  <teleport to="body">
    <div class="modal-mask" @click.self="emit('close')">
      <div class="modal" :style="{ width }">
        <header class="modal-head">
          <h3>{{ title }}</h3>
          <button class="icon-btn" @click="emit('close')">✕</button>
        </header>
        <div class="modal-body"><slot /></div>
        <footer v-if="$slots.footer" class="modal-foot"><slot name="footer" /></footer>
      </div>
    </div>
  </teleport>
</template>
