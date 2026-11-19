<script setup>
import { computed } from 'vue'

const props = defineProps({ name: String, size: { type: Number, default: 20 } })

const paths = {
  chat: ['M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v8a2.5 2.5 0 0 1-2.5 2.5H9l-4.2 3.4c-.4.3-.8 0-.8-.5V5.5z'],
  contacts: ['M12 11a3.2 3.2 0 1 0 0-6.4A3.2 3.2 0 0 0 12 11z', 'M5.5 20c0-3.6 2.9-5.6 6.5-5.6s6.5 2 6.5 5.6'],
  folder: ['M3 7.5A1.5 1.5 0 0 1 4.5 6h4.2l2 2.2h8.8A1.5 1.5 0 0 1 21 9.7v8.8a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18.5v-11z'],
  chart: ['M4 20V11', 'M10 20V5', 'M16 20v-6', 'M2.5 20.5h19'],
  settings: ['M12 15.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4z', 'M12 2.5v2.6', 'M12 18.9v2.6', 'M2.5 12h2.6', 'M18.9 12h2.6', 'M5.2 5.2l1.8 1.8', 'M17 17l1.8 1.8', 'M18.8 5.2L17 7', 'M7 17l-1.8 1.8'],
  search: ['M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14z', 'M20 20l-3.6-3.6'],
  phone: ['M6 3h3l1.6 4-2 1.5a12 12 0 0 0 5.9 5.9L16 12.4 20 14v3a2 2 0 0 1-2.2 2A16 16 0 0 1 4 5.2 2 2 0 0 1 6 3z'],
  video: ['M3 7.5A1.5 1.5 0 0 1 4.5 6h9A1.5 1.5 0 0 1 15 7.5v9A1.5 1.5 0 0 1 13.5 18h-9A1.5 1.5 0 0 1 3 16.5v-9z', 'M15 10.5l6-3.5v10l-6-3.5'],
  logout: ['M14 4h4.5A1.5 1.5 0 0 1 20 5.5v13a1.5 1.5 0 0 1-1.5 1.5H14', 'M9 12H3', 'M6.5 8.5L3 12l3.5 3.5'],
  star: ['M12 3.6l2.7 5.4 6 .9-4.3 4.2 1 5.9-5.4-2.8-5.4 2.8 1-5.9-4.3-4.2 6-.9L12 3.6z'],
  bell: ['M18 8.5a6 6 0 1 0-12 0c0 6.5-2.6 6.5-2.6 8.5h17.2c0-2-2.6-2-2.6-8.5z', 'M9.8 20a2.4 2.4 0 0 0 4.4 0'],
  plus: ['M12 5v14', 'M5 12h14']
}
const d = computed(() => paths[props.name] || [])
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
    <path v-for="(p, i) in d" :key="i" :d="p" />
  </svg>
</template>
