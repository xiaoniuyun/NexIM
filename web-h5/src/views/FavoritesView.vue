<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API, fileUrl } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'

const router = useRouter()
const list = ref([])
const loading = ref(true)
const filter = ref('all')
const TYPE = { 1: '文本', 2: '图片', 3: '系统', 4: '语音', 5: '文件', 6: '视频' }

const shown = computed(() => {
  if (filter.value === 'all') return list.value
  const map = { text: 1, image: 2, file: 5 }
  return list.value.filter(x => x.msgType === map[filter.value])
})

onMounted(load)
async function load() {
  loading.value = true
  try { const r = await API.favorites(); list.value = r.data || [] } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}
function fmt(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
function open(item) {
  if (item.convId) router.push('/chat/' + encodeURIComponent(item.convId))
}
async function remove(item) {
  if (!window.confirm('取消收藏这条内容？')) return
  try {
    await API.favoriteRemove(item.id)
    list.value = list.value.filter(x => x.id !== item.id)
    toast('已取消收藏')
  } catch (e) { toast(e.message, 'err') }
}
function parseExtra(v) {
  if (!v) return null
  if (typeof v === 'object') return v
  try { return JSON.parse(v) } catch (_) { return null }
}
function fileOf(item) {
  const ex = parseExtra(item.extra)
  return ex?.fileId ? fileUrl(ex.fileId) : ''
}
</script>
<template>
  <div class="h5-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <h2>{{ t('fav') }}</h2>
    <div class="sp"></div>
  </div>
  <div class="h5-body">
    <div class="chip-row">
      <button class="chip" :class="{ on: filter === 'all' }" @click="filter = 'all'">全部</button>
      <button class="chip" :class="{ on: filter === 'text' }" @click="filter = 'text'">文本</button>
      <button class="chip" :class="{ on: filter === 'image' }" @click="filter = 'image'">图片</button>
      <button class="chip" :class="{ on: filter === 'file' }" @click="filter = 'file'">文件</button>
    </div>
    <div v-if="loading" class="loading">{{ t('loading') }}</div>
    <div v-else-if="!shown.length" class="loading">{{ t('favoritesEmpty') }}</div>
    <div v-else>
      <div v-for="item in shown" :key="item.id" class="card fav-item" @click="open(item)">
        <div class="fav-top">
          <span class="fav-type">{{ TYPE[item.msgType] || '消息' }}</span>
          <span class="fav-from">{{ item.fromName || '—' }}</span>
          <span class="fav-time">{{ fmt(item.createdAt) }}</span>
        </div>
        <div v-if="item.msgType === 2 && fileOf(item)" class="fav-img">
          <img :src="fileOf(item)" alt="">
        </div>
        <div v-else class="fav-content">{{ item.content || '[非文本消息]' }}</div>
        <div class="fav-bottom">
          <span>{{ item.convName || item.convId }}</span>
          <button class="mini-btn danger" @click.stop="remove(item)">取消收藏</button>
        </div>
      </div>
    </div>
  </div>
</template>
