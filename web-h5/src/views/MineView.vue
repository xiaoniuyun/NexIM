<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { API } from '../api'
import { t } from '../i18n'
import { toast } from '../toast'
import ModalBox from '../components/ModalBox.vue'
import Avatar from '../components/Avatar.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const devices = ref([])
const modal = ref(null)
onMounted(async () => {
  try { const r = await API.devices(); devices.value = r.data || [] } catch (e) {}
})
const statusList = [
  { label: t('online'), value: '在线' },
  { label: t('busy'), value: '忙碌' },
  { label: t('invisible'), value: '隐身' }
]
function setStatus(s) { auth.setStatus(s); toast('状态已更新为 ' + s) }
function logout() {
  modal.value?.open({
    title: '退出登录', body: '<p>确认退出当前账号？</p>', okText: '退出', danger: true,
    onOk: async () => {
      await auth.logout(); chat.reset(); router.replace('/login')
    }
  })
}
</script>
<template>
  <div class="h5-head">
    <h2>{{ t('mine') }}</h2><div class="sp"></div>
    <button class="icon-btn" @click="router.push('/settings')">⚙️</button>
  </div>
  <div class="h5-body">
    <div class="profile-card" @click="router.push('/profile')">
      <Avatar :name="auth.user?.nickname || 'U'" :src="auth.user?.avatar" size="lg" color="rgba(255,255,255,.25)" />
      <div>
        <div class="pc-name">{{ auth.user?.nickname || '用户' }}</div>
        <div class="pc-sub">{{ auth.user?.signature || ('@' + auth.user?.username + ' · UID ' + auth.user?.uid) }}</div>
      </div>
    </div>
    <div class="group-block">
      <button class="gb-row" @click="router.push('/profile')">✏️ 编辑资料<span class="gb-r">昵称 / 头像 / 签名 ›</span></button>
    </div>
    <div class="group-block">
      <div class="gb-row" style="color:var(--muted);font-size:11px">{{ t('status') }}</div>
      <div class="status-opts">
        <button v-for="s in statusList" :key="s.value" :class="{ on: auth.status === s.value }" @click="setStatus(s.value)">{{ s.label }}</button>
      </div>
    </div>
    <div class="group-block">
      <button class="gb-row" @click="router.push('/settings')">🖥️ {{ t('devices') }}<span class="gb-r">{{ devices.length }} 台 ›</span></button>
    </div>
    <div class="group-block">
      <div v-for="d in devices" :key="d.deviceId" class="gb-row">📱 {{ d.deviceName || d.platform }}<span class="gb-r">{{ d.pushChannel || d.platform }}</span></div>
      <div v-if="!devices.length" class="gb-row" style="color:var(--muted)">暂无其他设备</div>
    </div>
    <div class="group-block">
      <button class="gb-row" @click="router.push('/settings')">🔔 {{ t('notification') }}<span class="gb-r">已开启 ›</span></button>
      <button class="gb-row" @click="router.push('/settings')">🔒 {{ t('privacy') }}<span class="gb-r">›</span></button>
      <button class="gb-row" @click="router.push('/settings')">⚙️ {{ t('general') }}<span class="gb-r">›</span></button>
    </div>
    <button class="btn-primary" style="background:var(--danger-soft);color:var(--danger);box-shadow:none" @click="logout">{{ t('logout') }}</button>
  </div>
  <ModalBox ref="modal" />
</template>
