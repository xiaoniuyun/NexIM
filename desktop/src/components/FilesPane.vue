<script setup>
import { computed, ref, onMounted } from 'vue'
import { API, fileUrl } from '@h5/api'
import { toast } from '@h5/toast'
import { bytes, fullTime } from '../composables/useDesktop'

const files = ref([])
const keyword = ref('')
const kind = ref('all')
const loading = ref(true)

const list = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return files.value.filter(f => {
    if (q && !String(f.name || '').toLowerCase().includes(q)) return false
    if (kind.value === 'image' && !String(f.mimeType).startsWith('image/')) return false
    if (kind.value === 'video' && !String(f.mimeType).startsWith('video/')) return false
    if (kind.value === 'doc' && !/pdf|word|excel|ppt|text|zip|rar/.test(String(f.mimeType))) return false
    return true
  })
})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const r = await API.files()
    files.value = r.data || []
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}

function download(f) { window.open(fileUrl(f.fileId), '_blank') }
function iconOf(f) {
  const m = String(f.mimeType || '')
  if (m.startsWith('image/')) return '🖼'
  if (m.startsWith('video/')) return '🎬'
  if (m.startsWith('audio/')) return '🎵'
  if (/pdf/.test(m)) return '📕'
  if (/zip|rar|7z/.test(m)) return '🗜'
  return '📄'
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>文件</h2>
      <div class="tabs">
        <button :class="{ on: kind === 'all' }" @click="kind = 'all'">全部</button>
        <button :class="{ on: kind === 'image' }" @click="kind = 'image'">图片</button>
        <button :class="{ on: kind === 'video' }" @click="kind = 'video'">视频</button>
        <button :class="{ on: kind === 'doc' }" @click="kind = 'doc'">文档</button>
      </div>
      <input v-model="keyword" class="pane-search" placeholder="搜索文件名">
    </header>
    <div class="pane-body scroll">
      <table class="tbl">
        <thead><tr><th>文件</th><th>大小</th><th>来源会话</th><th>时间</th><th></th></tr></thead>
        <tbody>
          <tr v-for="f in list" :key="f.fileId">
            <td><span class="file-ico">{{ iconOf(f) }}</span>{{ f.name }}</td>
            <td>{{ bytes(f.size) }}</td>
            <td class="muted">{{ f.convId || '—' }}</td>
            <td class="muted">{{ fullTime(f.createdAt) }}</td>
            <td><button class="btn sm" @click="download(f)">下载</button></td>
          </tr>
          <tr v-if="!list.length"><td colspan="5" class="empty-tip">{{ loading ? '加载中…' : '暂无文件，聊天中发送的文件会自动汇总到这里' }}</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
