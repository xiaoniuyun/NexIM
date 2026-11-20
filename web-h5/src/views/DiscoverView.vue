<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import LineIcon from '../components/LineIcon.vue'

const router = useRouter()
const notices = ref([])
onMounted(async () => {
  try { const r = await API.notices(); notices.value = r.data || [] } catch (e) {}
})
const entries = [
  { name: t('notification'), icon: 'bell', color: '#4f6ef7', to: '/files' },
  { name: t('files'), icon: 'file', color: '#12a594', to: '/files' },
  { name: t('favorites'), icon: 'star', color: '#a05ce6', to: '/favorites' }
]
</script>
<template>
  <div class="h5-head"><h2>{{ t('discover') }}</h2><div class="sp"></div></div>
  <div class="h5-body">
    <button v-for="e in entries" :key="e.name" class="qentry" @click="router.push(e.to)">
      <span :style="{ color: e.color }"><LineIcon :name="e.icon" :size="21" /></span>
      <div><div class="qe-t">{{ e.name }}</div></div>
      <span class="qe-arrow">›</span>
    </button>
    <div class="sec-title">{{ t('discoverHint') }}</div>
    <div v-for="n in notices" :key="n.id" class="qentry">
      <span style="color:#f2994a"><LineIcon name="pin" :size="20" /></span>
      <div style="min-width:0"><div class="qe-t">{{ n.title }}</div><div class="qe-s">{{ n.content }}</div></div>
    </div>
    <div v-if="!notices.length" class="loading">{{ t('empty') }}</div>
  </div>
</template>
