<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { lang, toggleLang, t } from '../i18n'
import { toast } from '../toast'
import { API } from '../api'
import { useAuthStore } from '../stores/auth'
import Avatar from '../components/Avatar.vue'

const router = useRouter()
const auth = useAuthStore()
const tab = ref('notify')

const KEY = 'h5_settings'
const defaults = {
  notify: true, detail: true, sound: true, vibrate: false, dnd: false,
  verify: true, phoneSearch: true, online: true, dark: false, font: 'normal'
}
function loadPrefs() {
  try { return { ...defaults, ...(JSON.parse(localStorage.getItem(KEY) || '{}')) } } catch (_) { return { ...defaults } }
}
const prefs = ref(loadPrefs())
const dark = computed(() => prefs.value.dark)
const tabs = computed(() => [['notify', '通知'], ['privacy', '隐私'], ['general', '通用']])

watch(prefs, v => {
  localStorage.setItem(KEY, JSON.stringify(v))
  document.body.classList.toggle('h5-dark', !!v.dark)
  document.body.classList.toggle('h5-font-large', v.font === 'large')
  document.body.classList.toggle('h5-font-small', v.font === 'small')
}, { deep: true })

onMounted(() => {
  document.body.classList.toggle('h5-dark', !!prefs.value.dark)
  document.body.classList.toggle('h5-font-large', prefs.value.font === 'large')
  document.body.classList.toggle('h5-font-small', prefs.value.font === 'small')
})

const devices = ref([])
const showDevices = ref(false)
const showBlack = ref(false)
const blocked = ref([])
const blockKeyword = ref('')
const blockResults = ref([])

function set(k, v) { prefs.value[k] = v }
function switchLang() { toggleLang(); toast(lang.value === 'en' ? 'Switched to English' : '已切换为中文'); prefs.value.lang = lang.value }
function toggleDark() { prefs.value.dark = !prefs.value.dark }
function setFont(f) { prefs.value.font = f }

async function openDevices() {
  showDevices.value = true
  try { const r = await API.devices(); devices.value = r.data || [] } catch (e) { toast(e.message, 'err') }
}
async function kick(d) {
  if (!window.confirm(`让设备「${d.deviceName || d.deviceId}」下线？`)) return
  try {
    await API.deleteDevice(d.deviceId)
    devices.value = devices.value.filter(x => x.deviceId !== d.deviceId)
    toast(t('deviceKicked'))
  } catch (e) { toast(e.message, 'err') }
}
function fmtTime(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  return `${d.getMonth() + 1}-${d.getDate()} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

async function openBlack() {
  showBlack.value = true
  blockResults.value = []
  blockKeyword.value = ''
  await loadBlack()
}
async function loadBlack() {
  try { const r = await API.blacklist(); blocked.value = r.data || [] } catch (e) { toast(e.message, 'err') }
}
async function doBlockSearch() {
  const q = blockKeyword.value.trim()
  if (!q) return
  try { const r = await API.userSearch(q); blockResults.value = (r.data || []).filter(u => u.uid !== auth.user?.uid) } catch (e) { toast(e.message, 'err') }
}
async function addBlock(u) {
  try {
    await API.blockUser(u.uid)
    toast(`已拉黑 ${u.nickname || u.uid}`)
    blockResults.value = []
    blockKeyword.value = ''
    await loadBlack()
  } catch (e) { toast(e.message, 'err') }
}
async function unblock(u) {
  try { await API.unblockUser(u.uid); blocked.value = blocked.value.filter(x => x.uid !== u.uid); toast('已移出黑名单') } catch (e) { toast(e.message, 'err') }
}

function clearCache() {
  const keep = ['h5_token', 'h5_refresh', 'h5_user', 'h5_device', 'h5_status', 'h5_settings']
  Object.keys(localStorage).filter(k => !keep.includes(k)).forEach(k => localStorage.removeItem(k))
  sessionStorage.clear()
  if (window.caches) caches.keys().then(keys => keys.forEach(k => caches.delete(k))).catch(() => {})
  toast(t('cacheCleared'))
}
</script>
<template>
  <div class="h5-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <h2>设置</h2>
  </div>
  <div class="h5-body" style="padding:12px">
    <div style="display:flex;gap:6px;margin-bottom:12px">
      <button v-for="tb in tabs" :key="tb[0]" class="btn sm" :class="tab === tb[0] ? 'primary' : ''" @click="tab = tb[0]">{{ tb[1] }}</button>
    </div>

    <template v-if="tab === 'notify'">
      <div class="switch-row" @click="set('notify', !prefs.notify)"><span>新消息通知</span><span class="switch" :class="{ on: prefs.notify }"></span></div>
      <div class="switch-row" @click="set('detail', !prefs.detail)"><span>显示消息详情</span><span class="switch" :class="{ on: prefs.detail }"></span></div>
      <div class="switch-row" @click="set('sound', !prefs.sound)"><span>声音提醒</span><span class="switch" :class="{ on: prefs.sound }"></span></div>
      <div class="switch-row" @click="set('vibrate', !prefs.vibrate)"><span>震动提醒</span><span class="switch" :class="{ on: prefs.vibrate }"></span></div>
      <div class="switch-row" @click="set('dnd', !prefs.dnd)"><span>勿扰模式<div class="sr-desc">22:00 - 08:00</div></span><span class="switch" :class="{ on: prefs.dnd }"></span></div>
      <div class="sr-desc" style="padding:8px 2px">{{ t('privacyTip') }}</div>
    </template>

    <template v-else-if="tab === 'privacy'">
      <div class="switch-row" @click="set('verify', !prefs.verify)"><span>加我需要验证</span><span class="switch" :class="{ on: prefs.verify }"></span></div>
      <div class="switch-row" @click="set('phoneSearch', !prefs.phoneSearch)"><span>允许通过手机号搜索</span><span class="switch" :class="{ on: prefs.phoneSearch }"></span></div>
      <div class="switch-row" @click="set('online', !prefs.online)"><span>展示在线状态</span><span class="switch" :class="{ on: prefs.online }"></span></div>
      <button class="gb-row" @click="openDevices">🖥️ 已登录设备<span class="gb-r">管理 ›</span></button>
      <button class="gb-row" @click="openBlack">🛡️ 黑名单<span class="gb-r">{{ blocked.length }} ›</span></button>
    </template>

    <template v-else>
      <div class="switch-row" @click="toggleDark">
        <span>{{ t('theme') }}</span><span class="switch" :class="{ on: dark }"></span>
      </div>
      <button class="gb-row" @click="switchLang">🌐 语言 / Language<span class="gb-r">{{ lang === 'zh' ? '简体中文' : 'English' }} ›</span></button>
      <div class="switch-row">
        <span>字体大小</span>
        <span style="display:flex;gap:5px">
          <button class="btn sm" :class="prefs.font === 'small' ? 'primary' : ''" @click="setFont('small')">{{ t('fontSmall') }}</button>
          <button class="btn sm" :class="prefs.font === 'normal' ? 'primary' : ''" @click="setFont('normal')">{{ t('fontNormal') }}</button>
          <button class="btn sm" :class="prefs.font === 'large' ? 'primary' : ''" @click="setFont('large')">{{ t('fontLarge') }}</button>
        </span>
      </div>
      <button class="gb-row" @click="clearCache">🗑️ 清理缓存<span class="gb-r">{{ t('cacheCleared') }} ›</span></button>
      <div class="switch-row"><span>关于<div class="sr-desc">NexIM H5 · 企业即时通讯</div></span><span class="state info">v1.0</span></div>
    </template>
  </div>

  <!-- 已登录设备 -->
  <div v-if="showDevices" class="mask" @click.self="showDevices = false">
    <div class="modal">
      <div class="mh">已登录设备</div>
      <div class="mb">
        <div v-for="d in devices" :key="d.deviceId" class="member-row">
          <span style="font-size:20px">🖥️</span>
          <div style="flex:1">
            <div>{{ d.deviceName || d.deviceId }}</div>
            <div class="sr-desc">{{ d.platform || 'Web' }} · 最近活跃 {{ fmtTime(d.lastActive) }}</div>
          </div>
          <span v-if="d.deviceId === auth.device" class="tag-pin">当前设备</span>
          <button v-else class="mini-btn danger" @click="kick(d)">{{ t('remoteLogout') }}</button>
        </div>
        <div v-if="!devices.length" class="loading">暂无设备记录</div>
      </div>
      <div class="mf"><button class="btn-ghost" @click="showDevices = false">{{ t('confirm') }}</button></div>
    </div>
  </div>

  <!-- 黑名单 -->
  <div v-if="showBlack" class="mask" @click.self="showBlack = false">
    <div class="modal">
      <div class="mh">{{ t('blacklist') }}</div>
      <div class="mb">
        <div class="h5-search" style="margin-bottom:8px">🔍<input v-model="blockKeyword" placeholder="搜索用户加入黑名单" @keyup.enter="doBlockSearch"></div>
        <button v-for="u in blockResults" :key="u.uid" class="member-row" style="width:100%" @click="addBlock(u)">
          <Avatar :name="u.nickname" size="sm" />
          <span style="flex:1;text-align:left">{{ u.nickname }} <span class="sr-desc">@{{ u.username }}</span></span>
          <span class="mini-btn">{{ t('addBlacklist') }}</span>
        </button>
        <div class="sec-title" style="margin-top:8px">已拉黑</div>
        <div v-for="u in blocked" :key="u.uid" class="member-row">
          <Avatar :name="u.nickname || u.uid" size="sm" />
          <span style="flex:1">{{ u.nickname || u.uid }}</span>
          <button class="mini-btn" @click="unblock(u)">{{ t('unblock') }}</button>
        </div>
        <div v-if="!blocked.length" class="loading">{{ t('blacklistEmpty') }}</div>
      </div>
      <div class="mf"><button class="btn-ghost" @click="showBlack = false">{{ t('confirm') }}</button></div>
    </div>
  </div>
</template>
