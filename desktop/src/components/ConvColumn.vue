<script setup>
import { computed, ref, onMounted, onBeforeUnmount, reactive } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import Icon from './Icon.vue'
import { useChatStore } from '@h5/stores/chat'
import { API } from '@h5/api'
import { toast } from '@h5/toast'
import { pinnedSet, mutedSet, togglePin, toggleMute, timeText } from '../composables/useDesktop'

const emit = defineEmits(['open', 'search'])
const chat = useChatStore()
const keyword = ref('')
const pinned = ref(pinnedSet())
const muted = ref(mutedSet())
const menu = reactive({ show: false, x: 0, y: 0, conv: null })

const list = computed(() => {
  let arr = [...chat.convs]
  const kw = keyword.value.trim().toLowerCase()
  if (kw) {
    arr = arr.filter(c =>
      String(c.peerName || c.convId).toLowerCase().includes(kw) ||
      String(c.lastMsg || '').toLowerCase().includes(kw)
    )
  }
  return arr.sort((a, b) => {
    const pa = pinned.value.has(a.convId) ? 1 : 0
    const pb = pinned.value.has(b.convId) ? 1 : 0
    if (pa !== pb) return pb - pa
    return (Number(b.lastMsgAt) || 0) - (Number(a.lastMsgAt) || 0)
  })
})

function displayName(c) { return c.peerName || c.convId }

function openMenu(e, conv) {
  e.preventDefault()
  menu.show = true
  menu.x = Math.min(e.clientX, window.innerWidth - 190)
  menu.y = Math.min(e.clientY, window.innerHeight - 230)
  menu.conv = conv
}
function closeMenu() { menu.show = false }

function doPin() {
  const c = menu.conv
  const on = togglePin(c.convId)
  pinned.value = pinnedSet()
  toast(on ? '已置顶会话' : '已取消置顶')
  closeMenu()
}
function doMute() {
  const c = menu.conv
  const on = toggleMute(c.convId)
  muted.value = mutedSet()
  toast(on ? '已开启消息免打扰' : '已关闭消息免打扰')
  closeMenu()
}
async function doRead() {
  const c = menu.conv
  closeMenu()
  try {
    await API.markRead(c.convId)
    chat.loadConversations()
    toast('已标记为已读')
  } catch (e) { toast(e.message, 'err') }
}
async function doDelete() {
  const c = menu.conv
  closeMenu()
  if (!confirm(`确认删除与「${displayName(c)}」的会话？历史消息不会从服务端删除。`)) return
  try {
    await API.deleteConv(c.convId)
    if (chat.currentConv === c.convId) chat.currentConv = ''
    await chat.loadConversations()
    toast('会话已删除')
  } catch (e) { toast(e.message, 'err') }
}

async function readAll() {
  try {
    await API.readAll()
    chat.loadConversations()
    toast('已全部标记为已读')
  } catch (e) { toast(e.message, 'err') }
}

onMounted(() => window.addEventListener('click', closeMenu))
onBeforeUnmount(() => window.removeEventListener('click', closeMenu))
</script>

<template>
  <div class="conv-col">
    <header class="conv-head">
      <div class="conv-search-wrap">
        <Icon name="search" :size="16" />
        <input v-model="keyword" placeholder="搜索">
      </div>
      <button class="conv-plus" title="发起会话（Ctrl+K）" @click="emit('search')">
        <Icon name="plus" :size="18" />
      </button>
    </header>

    <div class="conv-list scroll">
      <div
        v-for="c in list" :key="c.convId" class="conv-item"
        :class="{ active: c.convId === chat.currentConv }"
        @click="emit('open', c.convId)"
        @contextmenu="openMenu($event, c)"
      >
        <div class="conv-avatar">
          <Avatar :name="displayName(c)" :src="c.peerAvatar" size="md" />
          <span v-if="(c.unread || 0) > 0" class="badge">{{ c.unread > 99 ? '99+' : c.unread }}</span>
        </div>
        <div class="conv-main">
          <div class="conv-row">
            <span class="conv-name">
              <span v-if="pinned.has(c.convId)" class="pin-mark">置顶</span>
              {{ displayName(c) }}
            </span>
            <span class="conv-time">{{ timeText(c.lastMsgAt) }}</span>
          </div>
          <div class="conv-row">
            <span class="conv-msg">
              <Icon v-if="muted.has(c.convId)" name="bellOff" :size="13" class="mute-ic" />
              {{ c.lastMsg || '暂无消息' }}
            </span>
          </div>
        </div>
      </div>
      <div v-if="!list.length" class="empty-tip">没有匹配的会话</div>
    </div>

    <div v-if="menu.show" class="ctx-menu" :style="{ left: menu.x + 'px', top: menu.y + 'px' }" @click.stop>
      <button @click="doPin">{{ pinned.has(menu.conv.convId) ? '取消置顶' : '置顶会话' }}</button>
      <button @click="doMute">{{ muted.has(menu.conv.convId) ? '关闭免打扰' : '消息免打扰' }}</button>
      <button @click="doRead">标记为已读</button>
      <button @click="readAll(); closeMenu()">全部标为已读</button>
      <button class="danger" @click="doDelete">删除会话</button>
    </div>
  </div>
</template>
