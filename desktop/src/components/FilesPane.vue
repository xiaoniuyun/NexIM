<script setup>
import { computed, ref, onMounted } from 'vue'
import { API, fileUrl } from '@h5/api'
import LineIcon from '@h5/components/LineIcon.vue'
import { toast } from '@h5/toast'
import { bytes, fullTime } from '../composables/useDesktop'

const emit = defineEmits(['open-conv'])
const files = ref([])
const keyword = ref('')
const kind = ref('all')
const loading = ref(true)
const uploading = ref(false)
const uploadInput = ref(null)

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
function openConv(f) {
  if (f.convId) emit('open-conv', f.convId)
  else toast('该文件未关联会话', 'err')
}
function pickUpload() { uploadInput.value?.click() }
async function onUpload(e) {
  const picked = [...(e.target.files || [])]
  e.target.value = ''
  if (!picked.length) return
  uploading.value = true
  let ok = 0
  for (const file of picked) {
    if (file.size > 100 * 1024 * 1024) { toast(`${file.name} 超过 100MB 上限`, 'err'); continue }
    const fd = new FormData()
    fd.append('file', file)
    fd.append('convId', '')
    try { await API.fileUpload(fd); ok++ } catch (err) { toast(err.message || `${file.name} 上传失败`, 'err') }
  }
  uploading.value = false
  if (ok) { toast(`已上传 ${ok} 个文件`); load() }
}
async function removeFile(f) {
  if (!confirm(`删除文件「${f.name}」？`)) return
  try {
    await API.fileDelete(f.fileId)
    files.value = files.value.filter(x => x.fileId !== f.fileId)
    toast('已删除')
  } catch (e) { toast(e.message, 'err') }
}
function iconOf(f) {
  const m = String(f.mimeType || '')
  if (m.startsWith('image/')) return 'image'
  if (m.startsWith('video/')) return 'video'
  if (m.startsWith('audio/')) return 'mic'
  return 'file'
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
      <button class="btn sm primary" :disabled="uploading" @click="pickUpload">{{ uploading ? '上传中…' : '+ 上传' }}</button>
      <input ref="uploadInput" type="file" multiple hidden @change="onUpload">
    </header>
    <div class="pane-body scroll">
      <table class="tbl">
        <thead><tr><th>文件</th><th>大小</th><th>来源会话</th><th>时间</th><th></th></tr></thead>
        <tbody>
          <tr v-for="f in list" :key="f.fileId">
            <td><span class="file-ico"><LineIcon :name="iconOf(f)" :size="18" /></span>{{ f.name }}</td>
            <td>{{ bytes(f.size) }}</td>
            <td class="muted">{{ f.convId || '—' }}</td>
            <td class="muted">{{ fullTime(f.createdAt) }}</td>
            <td class="row-actions">
              <button class="btn sm" @click="download(f)">下载</button>
              <button v-if="f.convId" class="btn sm" @click="openConv(f)">会话</button>
              <button class="btn sm danger" @click="removeFile(f)">删除</button>
            </td>
          </tr>
          <tr v-if="!list.length"><td colspan="5" class="empty-tip">{{ loading ? '加载中…' : '暂无文件，聊天中发送的文件会自动汇总到这里' }}</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
