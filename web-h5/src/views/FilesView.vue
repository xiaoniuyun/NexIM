<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import LineIcon from '../components/LineIcon.vue'

const router = useRouter()
const files = ref([])
const fileInput = ref(null)
const token = localStorage.getItem('h5_token') || ''
onMounted(async () => {
  try { const r = await API.files(); files.value = r.data || [] } catch (e) { toast(e.message, 'err') }
})
async function upload(e) {
  const f = e.target.files?.[0]
  if (!f) return
  const fd = new FormData(); fd.append('file', f)
  try {
    await API.fileUpload(fd)
    toast('上传成功')
    const r = await API.files(); files.value = r.data || []
  } catch (ex) { toast(ex.message, 'err') }
}
function fmtSize(s) { return s > 1024 * 1024 ? (s / 1024 / 1024).toFixed(1) + ' MB' : (s / 1024).toFixed(1) + ' KB' }
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('files') }}</div><div class="ch-sub">跨端文件传输</div></div>
    <div class="sp"></div>
    <button class="btn-sm" @click="fileInput?.click()">⬆ 上传</button>
    <input ref="fileInput" type="file" style="display:none" @change="upload">
  </div>
  <div class="h5-body">
    <a v-for="f in files" :key="f.fileId" class="qentry" style="text-decoration:none" :href="'/api/v1/files/' + f.fileId + '/download?token=' + encodeURIComponent(token)" target="_blank">
      <span style="color:#4f6ef7"><LineIcon name="file" :size="21" /></span>
      <div style="flex:1;min-width:0"><div class="qe-t">{{ f.name }}</div><div class="qe-s">{{ fmtSize(f.size) }} · {{ new Date(f.createdAt).toLocaleString() }}</div></div>
    </a>
    <div v-if="!files.length" class="loading">{{ t('empty') }}</div>
  </div>
</template>
