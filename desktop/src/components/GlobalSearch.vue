<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import Avatar from '@/shared/components/Avatar.vue'
import { API } from '@/shared/api'
import { useChatStore } from '@/shared/stores/chat'
import { toast } from '@/shared/toast'

const emit = defineEmits(['close', 'open-conv'])
const chat = useChatStore()
const kw = ref('')
const friends = ref([])
const msgs = ref([])
const input = ref(null)
let timer = null

onMounted(async () => {
  nextTick(() => input.value?.focus())
  try { const r = await API.friends(); friends.value = r.data || [] } catch (_) {}
})

const convHits = computed(() => {
  const q = kw.value.trim().toLowerCase()
  if (!q) return []
  return chat.convs.filter(c => String(c.peerName || c.convId).toLowerCase().includes(q)).slice(0, 6)
})
const friendHits = computed(() => {
  const q = kw.value.trim().toLowerCase()
  if (!q) return []
  return friends.value.filter(u =>
    String(u.nickname || '').toLowerCase().includes(q) || String(u.username || '').toLowerCase().includes(q)
  ).slice(0, 6)
})

watch(kw, () => {
  clearTimeout(timer)
  const q = kw.value.trim()
  if (!q) { msgs.value = []; return }
  timer = setTimeout(doSearch, 300)
})

async function doSearch() {
  const q = kw.value.trim()
  if (!q) return
  try {
    const r = await API.searchMessages(q, '')
    msgs.value = (r.data || []).slice(0, 12)
  } catch (_) { msgs.value = [] }
}

function openConv(convId) { emit('open-conv', convId); emit('close') }
function chatWith(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  emit('open-conv', convId)
  emit('close')
}
function openMsg(m) { openConv(m.convId) }
function onKey(e) { if (e.key === 'Escape') emit('close') }
</script>

<template>
  <teleport to="body">
    <div class="search-mask" @click.self="emit('close')" @keydown="onKey">
      <div class="search-modal">
        <div class="search-input-row">
          <span class="search-ico">⌕</span>
          <input ref="input" v-model="kw" placeholder="搜索会话 / 联系人 / 消息…" @keyup.enter="doSearch">
          <span class="esc">ESC</span>
        </div>
        <div class="search-body scroll">
          <template v-if="!kw.trim()">
            <div class="empty-tip">输入关键词开始全局搜索</div>
          </template>
          <template v-else>
            <section v-if="convHits.length">
              <div class="sec-title">会话</div>
              <div v-for="c in convHits" :key="c.convId" class="hit-row" @click="openConv(c.convId)">
                <Avatar :name="c.peerName || c.convId" :src="c.peerAvatar" size="sm" />
                <span class="hit-name">{{ c.peerName || c.convId }}</span>
                <span class="muted">{{ c.lastMsg }}</span>
              </div>
            </section>
            <section v-if="friendHits.length">
              <div class="sec-title">联系人</div>
              <div v-for="u in friendHits" :key="u.uid" class="hit-row" @click="chatWith(u)">
                <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="sm" />
                <span class="hit-name">{{ u.nickname }}</span>
                <span class="muted">@{{ u.username }}</span>
              </div>
            </section>
            <section v-if="msgs.length">
              <div class="sec-title">消息</div>
              <div v-for="m in msgs" :key="m.msgId" class="hit-row" @click="openMsg(m)">
                <span class="hit-name">{{ m.fromName || m.fromUid }}</span>
                <span class="hit-msg">{{ m.content }}</span>
              </div>
            </section>
            <div v-if="!convHits.length && !friendHits.length && !msgs.length" class="empty-tip">没有匹配结果</div>
          </template>
        </div>
      </div>
    </div>
  </teleport>
</template>
