<script setup>
import { computed, ref } from 'vue'
import { avatarUrl, DEFAULT_AVATAR } from '../api'
const props = defineProps({ name: String, color: String, src: String, size: { type: String, default: '' } })
const palette = ['#4f6ef7', '#3ba7d8', '#f2994a', '#3ba58c', '#a05ce6', '#e9834a']
const broken = ref(false)
const raw = computed(() => String(props.src || '').trim())
// 历史头像值为 #rrggbb 颜色，用首字母渲染；无头像时使用内置默认头像
const colorValue = computed(() => (raw.value.startsWith('#') ? raw.value : ''))
const bg = computed(() => {
  if (props.color) return props.color
  if (colorValue.value) return colorValue.value
  let h = 0
  for (const ch of String(props.name || 'U')) h += ch.charCodeAt(0)
  return palette[h % palette.length]
})
// src 支持直接传 URL、颜色值或后端头像字段（file:<fileId>）
const img = computed(() => {
  if (broken.value) return ''
  const v = raw.value
  if (!v) return DEFAULT_AVATAR
  if (v.startsWith('file:')) return avatarUrl(v)
  if (v.startsWith('#')) return ''
  return v
})
</script>
<template>
  <img v-if="img" class="avatar" :class="size" :src="img" :alt="name" @error="broken = true">
  <span v-else class="avatar" :class="size" :style="{ background: bg }">{{ (name || 'U').slice(0, 1) }}</span>
</template>
