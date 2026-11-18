<script setup>
import { ref } from 'vue'
const visible = ref(false)
const opts = ref({ title: '', body: '', okText: '确定', danger: false, onOk: null })
function open(o) {
  opts.value = { title: '', body: '', okText: '确定', danger: false, onOk: null, ...o }
  visible.value = true
}
function close() { visible.value = false }
async function ok() {
  const fn = opts.value.onOk
  if (fn) {
    const res = await fn()
    if (res === false) return
  }
  close()
}
defineExpose({ open, close })
</script>
<template>
  <div v-if="visible" class="mask" @click.self="close">
    <div class="modal">
      <div class="mh">{{ opts.title }}</div>
      <div class="mb" v-html="opts.body"></div>
      <div class="mf">
        <button class="btn-ghost" @click="close">取消</button>
        <button :class="opts.danger ? 'btn-danger' : 'btn-sm'" @click="ok">{{ opts.okText }}</button>
      </div>
    </div>
  </div>
</template>
