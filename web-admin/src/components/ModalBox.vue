<script setup>
import { ref } from 'vue'
const visible = ref(false)
const opts = ref({ title: '', body: '', okText: '确定', danger: false, onOk: null })
function open(o) { opts.value = { title: '', body: '', okText: '确定', danger: false, onOk: null, ...o }; visible.value = true }
function close() { visible.value = false }
async function ok() {
  const fn = opts.value.onOk
  if (fn) { const r = await fn(); if (r === false) return }
  close()
}
defineExpose({ open, close })
</script>
<template>
  <div v-if="visible" class="mask" @click.self="close">
    <div class="modal">
      <div class="mh"><span>{{ opts.title }}</span><button class="icon-btn" @click="close">✕</button></div>
      <div class="mb" v-html="opts.body"></div>
      <div class="mf">
        <button class="btn" @click="close">取消</button>
        <button class="btn" :class="opts.danger ? 'danger' : 'primary'" @click="ok">{{ opts.okText }}</button>
      </div>
    </div>
  </div>
</template>
