<script setup>
import { ref, onMounted } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
import { API } from '@h5/api'
import { toast } from '@h5/toast'
import Modal from './Modal.vue'
import { desk, fullTime } from '../composables/useDesktop'

const auth = useAuthStore()
const chat = useChatStore()
const devices = ref([])
const showDevices = ref(false)
const notifyOn = ref(localStorage.getItem('desk_notify') !== '0')
const autoLaunch = ref(false)
const apiBase = ref('')
const wsUrl = ref('')
const updateState = ref('')
const savingServer = ref(false)
const version = window.neximConfig?.version || desk?.version || '1.0.0'

onMounted(async () => {
  if (!desk) return
  try {
    const c = await desk.getConfig()
    apiBase.value = c.apiBase
    wsUrl.value = c.wsUrl
    autoLaunch.value = await desk.getAutoLaunch()
  } catch (_) {}
})

function toggleNotify() {
  notifyOn.value = !notifyOn.value
  localStorage.setItem('desk_notify', notifyOn.value ? '1' : '0')
  toast(notifyOn.value ? '已开启桌面通知' : '已关闭桌面通知')
}
async function toggleAutoLaunch() {
  if (!desk) return
  const v = await desk.setAutoLaunch(!autoLaunch.value)
  autoLaunch.value = v
  toast(v ? '已设置为开机自动启动' : '已取消开机自动启动')
}
async function saveServer() {
  if (!desk) return toast('仅桌面客户端支持修改服务地址', 'err')
  savingServer.value = true
  const r = await desk.setConfig({ apiBase: apiBase.value, wsUrl: wsUrl.value })
  savingServer.value = false
  if (!r.ok) return toast(r.message || '保存失败', 'err')
  toast('服务地址已更新，正在重新加载')
  setTimeout(() => location.reload(), 600)
}
async function checkUpdate() {
  if (!desk) return toast('仅桌面客户端支持检查更新', 'err')
  const r = await desk.checkUpdate()
  updateState.value = r.status === 'dev' ? '开发模式不检查更新' : (r.message || `当前状态：${r.status}`)
  toast(updateState.value)
}
function installUpdate() { desk?.installUpdate() }

async function openDevices() {
  showDevices.value = true
  try { const r = await API.devices(); devices.value = r.data || [] } catch (e) { toast(e.message, 'err') }
}
async function kick(d) {
  if (!confirm(`让设备「${d.deviceName || d.deviceId}」下线？`)) return
  try {
    await API.deleteDevice(d.deviceId)
    devices.value = devices.value.filter(x => x.deviceId !== d.deviceId)
    toast('设备已下线')
  } catch (e) { toast(e.message, 'err') }
}
async function logout() {
  if (!confirm('确认退出登录？')) return
  await auth.logout()
  chat.reset()
  toast('已退出登录')
}
</script>

<template>
  <div class="pane">
    <header class="pane-head"><h2>设置</h2></header>
    <div class="pane-body scroll">
      <div class="card profile-card">
        <Avatar :name="auth.user?.nickname || '我'" size="lg" />
        <div class="profile-main">
          <div class="profile-name">{{ auth.user?.nickname }} <span class="muted">@{{ auth.user?.username }}</span></div>
          <div class="muted">{{ auth.user?.uid }} · {{ auth.status }}</div>
        </div>
        <button class="btn" @click="openDevices">已登录设备</button>
      </div>

      <div class="card">
        <h3>通知与启动</h3>
        <div class="switch-row" @click="toggleNotify">
          <div>桌面通知<div class="sr-desc">窗口在后台时，新消息通过系统通知提醒</div></div>
          <span class="switch" :class="{ on: notifyOn }"></span>
        </div>
        <div class="switch-row" @click="toggleAutoLaunch">
          <div>开机自动启动<div class="sr-desc">登录系统后自动启动 NexIM</div></div>
          <span class="switch" :class="{ on: autoLaunch }"></span>
        </div>
        <div class="switch-row disabled">
          <div>勿扰模式<div class="sr-desc">可在会话右键菜单中单独设置免打扰</div></div>
          <span class="state off">按会话</span>
        </div>
      </div>

      <div class="card">
        <h3>服务端连接</h3>
        <div class="form-grid">
          <label class="field"><span>API 地址</span><input v-model="apiBase" placeholder="http://127.0.0.1:8080/api/v1"></label>
          <label class="field"><span>长连接地址</span><input v-model="wsUrl" placeholder="ws://127.0.0.1:8081/ws"></label>
        </div>
        <div class="row-end"><button class="btn primary" :disabled="savingServer" @click="saveServer">保存并重新加载</button></div>
      </div>

      <div class="card">
        <div class="card-head"><h3>客户端更新</h3><span class="muted">当前版本 v{{ version }}</span></div>
        <div class="toolbar">
          <button class="btn" @click="checkUpdate">检查更新</button>
          <button class="btn primary" @click="installUpdate">重启并安装</button>
          <span class="muted">{{ updateState }}</span>
        </div>
      </div>

      <div class="card">
        <h3>快捷键</h3>
        <div class="kv"><span>全局搜索</span><span class="kbd">Ctrl + K</span></div>
        <div class="kv"><span>截图并发送</span><span class="kbd">Ctrl + Shift + A</span></div>
        <div class="kv"><span>发送消息</span><span class="kbd">Enter</span></div>
        <div class="kv"><span>换行</span><span class="kbd">Shift + Enter</span></div>
      </div>

      <div class="row-end"><button class="btn danger" @click="logout">退出登录</button></div>
    </div>

    <Modal v-if="showDevices" title="已登录设备" width="460px" @close="showDevices = false">
      <div class="member-list">
        <div v-for="d in devices" :key="d.deviceId" class="member-row">
          <span>🖥</span>
          <div style="flex:1">
            <div>{{ d.deviceName || d.deviceId }}</div>
            <div class="sr-desc">{{ d.platform || 'Web' }} · 最近活跃 {{ fullTime(d.lastActive) }}</div>
          </div>
          <span v-if="d.deviceId === auth.device" class="tag off">当前设备</span>
          <button v-else class="btn sm danger" @click="kick(d)">下线</button>
        </div>
        <div v-if="!devices.length" class="empty-tip">暂无设备记录</div>
      </div>
      <template #footer><button class="btn" @click="showDevices = false">关闭</button></template>
    </Modal>
  </div>
</template>
