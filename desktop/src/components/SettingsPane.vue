<script setup>
import { computed, ref, onMounted } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import LineIcon from '@h5/components/LineIcon.vue'
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
const blocked = ref([])
const showBlock = ref(false)
const blockKw = ref('')
const allUsers = ref([])
const perms = ref([])
// 个人资料编辑
const showEdit = ref(false)
const editForm = ref({ nickname: '', signature: '', gender: 0, avatar: '' })
const avatarPick = ref(null)
const uploading = ref(false)
const savingProfile = ref(false)
const genders = [{ label: '保密', value: 0 }, { label: '男', value: 1 }, { label: '女', value: 2 }]

const hasAdmin = computed(() => perms.value.length > 0)
const adminUrl = computed(() => {
  const base = apiBase.value || window.neximConfig?.apiBase || '/api/v1'
  return base.replace(/\/api\/v\d+\/?$/, '') + '/admin/'
})
const blockCandidates = computed(() => {
  const q = blockKw.value.trim().toLowerCase()
  const ids = new Set(blocked.value.map(b => b.uid))
  return allUsers.value
    .filter(u => u.uid !== auth.user?.uid && !ids.has(u.uid))
    .filter(u => !q || String(u.nickname || '').toLowerCase().includes(q) || String(u.username || '').toLowerCase().includes(q))
    .slice(0, 20)
})

onMounted(async () => {
  loadBlocked()
  loadPerms()
  auth.refreshProfile()
  if (!desk) return
  try {
    const c = await desk.getConfig()
    apiBase.value = c.apiBase
    wsUrl.value = c.wsUrl
    autoLaunch.value = await desk.getAutoLaunch()
  } catch (_) {}
})

async function loadBlocked() {
  try { const r = await API.blacklist(); blocked.value = r.data || [] } catch (_) {}
}
async function loadPerms() {
  try {
    const r = await API.permissions()
    perms.value = r.data?.menus || []
  } catch (_) {}
}
async function openBlock() {
  showBlock.value = true
  blockKw.value = ''
  try { const r = await API.users(); allUsers.value = r.data || [] } catch (_) {}
}
async function doBlock(u) {
  try {
    await API.blockUser(u.uid)
    toast(`已将 ${u.nickname || u.uid} 加入黑名单`)
    await loadBlocked()
  } catch (e) { toast(e.message, 'err') }
}
async function unblock(b) {
  try {
    await API.unblockUser(b.uid)
    blocked.value = blocked.value.filter(x => x.uid !== b.uid)
    toast('已移出黑名单')
  } catch (e) { toast(e.message, 'err') }
}
function openAdmin() {
  window.open(adminUrl.value, '_blank')
}

function openEdit() {
  const u = auth.user || {}
  editForm.value = {
    nickname: u.nickname || '',
    signature: u.signature || '',
    gender: Number(u.gender) || 0,
    avatar: u.avatar || ''
  }
  showEdit.value = true
}
function pickAvatar() {
  if (uploading.value || !avatarPick.value) return
  avatarPick.value.value = ''
  avatarPick.value.click()
}
async function onAvatarPick(e) {
  const file = e.target.files && e.target.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) return toast('头像不能超过 5MB', 'err')
  const fd = new FormData()
  fd.append('file', file)
  uploading.value = true
  try {
    const r = await API.fileUpload(fd)
    const fileId = r.data?.fileId
    if (!fileId) throw new Error('上传失败')
    editForm.value.avatar = 'file:' + fileId
    toast('头像已上传，记得保存')
  } catch (err) {
    toast(err.message || '头像上传失败', 'err')
  } finally {
    uploading.value = false
  }
}
async function saveProfile() {
  const nickname = editForm.value.nickname.trim()
  if (!nickname) return toast('昵称不能为空', 'err')
  savingProfile.value = true
  try {
    await auth.updateProfile({
      nickname,
      signature: editForm.value.signature.trim(),
      gender: editForm.value.gender,
      avatar: editForm.value.avatar
    })
    toast('资料已保存')
    showEdit.value = false
  } catch (err) {
    toast(err.message || '保存失败', 'err')
  } finally {
    savingProfile.value = false
  }
}

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
        <Avatar :name="auth.user?.nickname || '我'" :src="auth.user?.avatar" size="lg" />
        <div class="profile-main">
          <div class="profile-name">{{ auth.user?.nickname }} <span class="muted">@{{ auth.user?.username }}</span></div>
          <div class="muted">{{ auth.user?.signature || '这个人很懒，什么都没留下' }}</div>
        </div>
        <button class="btn" @click="openEdit">编辑资料</button>
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
        <div class="card-head">
          <h3>隐私与安全</h3>
          <button class="btn sm" @click="openBlock">+ 添加黑名单</button>
        </div>
        <div class="member-list">
          <div v-for="b in blocked" :key="b.uid" class="member-row">
            <Avatar :name="b.nickname || b.uid" :src="b.avatar" size="sm" />
            <div style="flex:1">
              <div>{{ b.nickname || b.uid }} <span class="muted">@{{ b.username }}</span></div>
              <div class="sr-desc">已屏蔽对方消息</div>
            </div>
            <button class="btn sm" @click="unblock(b)">移出</button>
          </div>
          <div v-if="!blocked.length" class="empty-tip">暂无黑名单用户</div>
        </div>
      </div>

      <div v-if="hasAdmin" class="card">
        <div class="card-head"><h3>管理后台</h3><span class="muted">当前账号拥有管理权限</span></div>
        <div class="toolbar">
          <button class="btn primary" @click="openAdmin">打开管理后台</button>
          <span class="muted">{{ adminUrl }}</span>
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

    <Modal v-if="showEdit" title="编辑资料" width="420px" @close="showEdit = false">
      <div style="display:flex;align-items:center;gap:14px;margin-bottom:16px">
        <Avatar :name="editForm.nickname || '我'" :src="editForm.avatar" size="lg" />
        <button class="btn" :disabled="uploading" @click="pickAvatar">{{ uploading ? '上传中…' : '更换头像' }}</button>
        <span class="muted">支持 5MB 以内图片</span>
      </div>
      <label class="field" style="margin-bottom:14px"><span>昵称</span><input v-model="editForm.nickname" maxlength="32" placeholder="请输入昵称"></label>
      <div class="field" style="margin-bottom:14px">
        <span>性别</span>
        <div style="display:flex;gap:8px;margin-top:6px">
          <button v-for="g in genders" :key="g.value" class="btn sm" :class="{ primary: editForm.gender === g.value }" @click="editForm.gender = g.value">{{ g.label }}</button>
        </div>
      </div>
      <label class="field" style="margin-bottom:14px"><span>个性签名</span><input v-model="editForm.signature" maxlength="120" placeholder="介绍一下自己"></label>
      <input ref="avatarPick" type="file" accept="image/*" hidden @change="onAvatarPick">
      <template #footer>
        <button class="btn" @click="showEdit = false">取消</button>
        <button class="btn primary" :disabled="savingProfile" @click="saveProfile">{{ savingProfile ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <Modal v-if="showDevices" title="已登录设备" width="460px" @close="showDevices = false">
      <div class="member-list">
        <div v-for="d in devices" :key="d.deviceId" class="member-row">
          <span style="color:#4f6ef7"><LineIcon name="monitor" :size="18" /></span>
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

    <Modal v-if="showBlock" title="添加黑名单" width="440px" @close="showBlock = false">
      <input v-model="blockKw" class="pane-search modal-search" placeholder="搜索姓名 / 账号">
      <div class="member-list">
        <div v-for="u in blockCandidates" :key="u.uid" class="member-row">
          <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="sm" />
          <div style="flex:1">
            <div>{{ u.nickname }} <span class="muted">@{{ u.username }}</span></div>
            <div class="sr-desc">{{ u.deptName || '' }}</div>
          </div>
          <button class="btn sm danger" @click="doBlock(u)">拉黑</button>
        </div>
        <div v-if="!blockCandidates.length" class="empty-tip">没有可选联系人</div>
      </div>
      <template #footer><button class="btn" @click="showBlock = false">关闭</button></template>
    </Modal>
  </div>
</template>
