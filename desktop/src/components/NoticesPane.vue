<script setup>
import { computed, ref, onMounted } from 'vue'
import { API } from '@/shared/api'
import { toast } from '@/shared/toast'
import { fullTime } from '../composables/useDesktop'

const notices = ref([])
const loading = ref(true)
const onlyPinned = ref(false)
const active = ref(null)

const list = computed(() => {
  const arr = onlyPinned.value ? notices.value.filter(n => n.pinned) : notices.value
  return [...arr].sort((a, b) => {
    if ((b.pinned || 0) !== (a.pinned || 0)) return (b.pinned || 0) - (a.pinned || 0)
    return (Number(b.createdAt) || 0) - (Number(a.createdAt) || 0)
  })
})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const r = await API.notices()
    notices.value = r.data || []
  } catch (e) { toast(e.message, 'err') } finally { loading.value = false }
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>通知公告</h2>
      <div class="tabs">
        <button :class="{ on: !onlyPinned }" @click="onlyPinned = false">全部 · {{ notices.length }}</button>
        <button :class="{ on: onlyPinned }" @click="onlyPinned = true">置顶</button>
      </div>
      <span class="sp"></span>
      <button class="btn sm" @click="load">刷新</button>
    </header>

    <div class="pane-body scroll">
      <div v-for="n in list" :key="n.id" class="card notice-card" @click="active = active === n.id ? null : n.id">
        <div class="notice-head">
          <span v-if="n.pinned" class="tag">置顶</span>
          <span class="notice-title">{{ n.title }}</span>
          <span class="sp"></span>
          <span class="muted">{{ fullTime(n.createdAt) }}</span>
        </div>
        <div class="notice-meta">{{ n.author || '系统管理员' }} · 范围 {{ n.scope || '全员' }}</div>
        <div class="notice-body" :class="{ clamp: active !== n.id }">{{ n.content }}</div>
        <div class="notice-more">{{ active === n.id ? '收起' : '展开全文' }}</div>
      </div>
      <div v-if="!list.length" class="empty-tip">{{ loading ? '加载中…' : '暂无公告' }}</div>
    </div>
  </div>
</template>
