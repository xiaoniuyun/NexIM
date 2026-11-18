<script setup>
import { computed } from 'vue'
const props = defineProps({ name: String, color: String, size: { type: String, default: '' } })
const palette = ['#4f6ef7', '#3ba7d8', '#f2994a', '#3ba58c', '#a05ce6', '#e9834a']
const bg = computed(() => {
  if (props.color) return props.color
  let h = 0
  for (const ch of String(props.name || 'U')) h += ch.charCodeAt(0)
  return palette[h % palette.length]
})
</script>
<template>
  <span class="avatar" :class="size" :style="{ background: bg }">{{ (name || 'U').slice(0, 1) }}</span>
</template>
