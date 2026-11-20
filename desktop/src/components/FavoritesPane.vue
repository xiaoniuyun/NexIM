<script setup>
import { computed, ref, onMounted } from 'vue'
import { API, fileUrl } from '@h5/api'
import LineIcon from '@h5/components/LineIcon.vue'
import { toast } from '@h5/toast'
import { fullTime } from '../composables/useDesktop'

const emit = defineEmits(['open-conv'])
const list = ref([])
const loading = ref(true)
const filter = ref('all')
const keyword = ref('')
const preview = ref(null)

const TYPE = { 1: '文本', 2: '图片', 3: '系统', 4: '语音', 5: '文件', 6: '视频' }

function parseExtra(v) {
  if (!v) return null
  if (typeof v === 'object') return v
  try { return JSON.parse(v) } catch (_) { return null }
}

const shown = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return list.value.filter(x => {
    if (filter.value === 'image' && x.msgType !== 2) return false
    if (filter.value === 'file' && x.msgType !== 5) return false
    if (filter.value === 'text' && x.msgType !== 1) return false
    if (q && !String(x.content || '').toLowerCase().includes(q) && !String(x.fromName || '').toLowerCase().includes(q)) return false
    return true
  })
})
const counts = computed(() => ({
  all: list.value.length,
  text: list.value.filter(x => x.msgType === 1).length,
  image: list.value.filter(x => x.msgType === 2).length,
  file: list.value.filter(x => x.msgType === 5).length
}))

onMounted(load)
async function load() {
  loading.value = true
  try {
    const r = await API.favorites()
    list.value = r.data || []
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}

function extraOf(item) { return parseExtra(item.extra) || {} }
function imageOf(item) {
  const ex = extraOf(item)
  return item.msgType === 2 && ex.fileId ? fileUrl(ex.fileId) : ''
}
function fileName(item) {
  const ex = extraOf(item)
  return ex.name || item.content || '文件'
}
function openFile(item) {
  const ex = extraOf(item)
  if (ex.fileId) window.open(fileUrl(ex.fileId), '_blank')
}
async function copy(item) {
  try { await navigator.clipboard.writeText(item.content || ''); toast('已复制') } catch (_) { toast('复制失败', 'err') }
}
async function remove(item) {
  if (!confirm('取消收藏这条内容？')) return
  try {
    await API.favoriteRemove(item.id)
    list.value = list.value.filter(x => x.id !== item.id)
    toast('已取消收藏')
  } catch (e) { toast(e.message, 'err') }
}
function jump(item) {
  if (item.convId) emit('open-conv', item.convId)
  else toast('原会话已删除，无法跳转', 'err')
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>收藏</h2>
      <div class="tabs">
        <button :class="{ on: filter === 'all' }" @click="filter = 'all'">全部 · {{ counts.all }}</button>
        <button :class="{ on: filter === 'text' }" @click="filter = 'text'">文本 · {{ counts.text }}</button>
        <button :class="{ on: filter === 'image' }" @click="filter = 'image'">图片 · {{ counts.image }}</button>
        <button :class="{ on: filter === 'file' }" @click="filter = 'file'">文件 · {{ counts.file }}</button>
      </div>
      <input v-model="keyword" class="pane-search" placeholder="搜索收藏内容 / 发送人">
    </header>

    <div class="pane-body scroll">
      <div v-for="item in shown" :key="item.id" class="card fav-card">
        <div class="fav-head">
          <span class="fav-type">{{ TYPE[item.msgType] || '消息' }}</span>
          <span class="fav-from">{{ item.fromName || item.fromUid || '—' }}</span>
          <span class="sp"></span>
          <span class="muted">{{ fullTime(item.createdAt) }}</span>
        </div>
        <div v-if="imageOf(item)" class="fav-img" @click="preview = imageOf(item)">
          <img :src="imageOf(item)" alt="收藏图片">
        </div>
        <div v-else-if="item.msgType === 5" class="file-msg" @click="openFile(item)">
          <span class="file-ico"><LineIcon name="file" :size="18" /></span>
          <div>
            <div class="file-name">{{ fileName(item) }}</div>
            <div class="file-size">点击下载原文件</div>
          </div>
        </div>
        <div v-else class="fav-content">{{ item.content || '[非文本消息]' }}</div>
        <div class="fav-foot">
          <span class="muted">来自 {{ item.convName || item.convId || '—' }}</span>
          <span class="sp"></span>
          <button class="btn sm" @click="jump(item)">定位会话</button>
          <button class="btn sm" @click="copy(item)">复制</button>
          <button class="btn sm danger" @click="remove(item)">取消收藏</button>
        </div>
      </div>
      <div v-if="!shown.length" class="empty-tip">{{ loading ? '加载中…' : '暂无收藏，聊天消息菜单中可收藏内容' }}</div>
    </div>

    <div v-if="preview" class="image-mask" @click="preview = null">
      <img :src="preview" alt="预览">
    </div>
  </div>
</template>
