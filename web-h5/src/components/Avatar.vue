<script setup>
import { computed } from 'vue'
import { avatarUrl } from '../api'
const props = defineProps({ name: String, color: String, src: String, size: { type: String, default: '' } })
const palette = ['#4f6ef7', '#3ba7d8', '#f2994a', '#3ba58c', '#a05ce6', '#e9834a']
const bg = computed(() => {
  if (props.color) return props.color
  let h = 0
  for (const ch of String(props.name || 'U')) h += ch.charCodeAt(0)
  return palette[h % palette.length]
})
// src 支持直接传 URL 或后端头像字段（file:<fileId>）
const img = computed(() => {
  const v = String(props.src || '')
  if (!v) return ''
  return v.startsWith('file:') ? avatarUrl(v) : v
})
</script>
<template>
  <img v-if="img" class="avatar" :class="size" :src="img" :alt="name">
  <span v-else class="avatar" :class="size" :style="{ background: bg }">{{ (name || 'U').slice(0, 1) }}</span>
</template>
