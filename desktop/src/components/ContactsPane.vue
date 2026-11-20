<script setup>
import { ref, computed, onMounted } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import Modal from './Modal.vue'
import { API } from '@h5/api'
import { useAuthStore } from '@h5/stores/auth'
import { useChatStore } from '@h5/stores/chat'
import { toast } from '@h5/toast'

const emit = defineEmits(['open-conv'])
const auth = useAuthStore()
const chat = useChatStore()

const tab = ref('friends')
const keyword = ref('')
const friends = ref([])
const applies = ref([])
const depts = ref([])
const deptId = ref(0)
const deptMembers = ref([])
const groups = ref([])
const users = ref([])

// 添加好友
const addOpen = ref(false)
const addKw = ref('')
const applied = ref([])
// 新建群聊
const createOpen = ref(false)
const groupName = ref('')
const picked = ref([])
const creating = ref(false)
// 群资料
const groupOpen = ref(false)
const groupDetail = ref(null)
const groupMembers = ref([])
const groupTab = ref('info')

const myUid = computed(() => auth.user?.uid)
const isExternal = computed(() => auth.user?.isExternal === true)
const friendIds = computed(() => new Set(friends.value.map(f => f.uid)))
const isOwner = computed(() => groupDetail.value?.ownerUid && groupDetail.value.ownerUid === myUid.value)

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return friends.value
  return friends.value.filter(u =>
    String(u.nickname || '').toLowerCase().includes(q) ||
    String(u.username || '').toLowerCase().includes(q) ||
    String(u.deptName || '').toLowerCase().includes(q)
  )
})

const addCandidates = computed(() => {
  const q = addKw.value.trim().toLowerCase()
  return users.value
    .filter(u => u.uid !== myUid.value && !friendIds.value.has(u.uid))
    .filter(u => !q || String(u.nickname || '').toLowerCase().includes(q) || String(u.username || '').toLowerCase().includes(q))
    .slice(0, 30)
})
const applyUsers = computed(() => new Map(users.value.map(u => [u.uid, u])))

onMounted(loadAll)
async function loadAll() {
  try {
    // 外部联系人默认无法查看组织架构，跳过分部与成员请求
    const [f, a, g, u] = await Promise.all([
      API.friends(), API.friendApplies(), API.groups(), API.users()
    ])
    friends.value = f.data || []
    applies.value = a.data || []
    groups.value = g.data || []
    users.value = u.data || []
    if (!isExternal.value) {
      const d = await API.depts()
      depts.value = d.data || []
      await pickDept(0)
    }
  } catch (e) { toast(e.message, 'err') }
}
async function refreshFriends() {
  try {
    const [f, a] = await Promise.all([API.friends(), API.friendApplies()])
    friends.value = f.data || []
    applies.value = a.data || []
  } catch (_) {}
}

async function pickDept(id) {
  if (isExternal.value) return
  deptId.value = id
  try {
    const r = await API.deptMembers(id)
    deptMembers.value = r.data || []
  } catch (e) { deptMembers.value = []; toast(e.message, 'err') }
}

function chatWith(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  emit('open-conv', convId)
}
function enterGroup(g) {
  const convId = 'conv_' + g.groupId
  if (!chat.convs.find(c => c.convId === convId)) {
    chat.convs.unshift({ convId, convType: 2, peerId: g.groupId, peerName: g.name, unread: 0, lastMsg: '', lastMsgAt: Date.now() })
  }
  emit('open-conv', convId)
}

async function openAdd() {
  addOpen.value = true
  addKw.value = ''
  applied.value = []
  try { const r = await API.users(); users.value = r.data || [] } catch (_) {}
}
async function applyFriend(u) {
  try {
    await API.friendApply(u.uid, `你好，我是 ${auth.user?.nickname || ''}`)
    applied.value = [...applied.value, u.uid]
    toast('好友申请已发送')
  } catch (e) { toast(e.message || '申请失败', 'err') }
}
function fromName(uid) {
  return applyUsers.value.get(uid)?.nickname || friends.value.find(f => f.uid === uid)?.nickname || uid
}
async function handleApply(a, accept) {
  try {
    await API.friendHandle(a.id, accept)
    toast(accept ? '已添加为好友' : '已拒绝该申请')
    await refreshFriends()
  } catch (e) { toast(e.message, 'err') }
}

function openCreate() {
  createOpen.value = true
  groupName.value = ''
  picked.value = []
}
function togglePick(uid) {
  const i = picked.value.indexOf(uid)
  if (i >= 0) picked.value.splice(i, 1)
  else picked.value.push(uid)
}
async function createGroup() {
  if (!groupName.value.trim()) return toast('请输入群名称', 'err')
  if (!picked.value.length) return toast('请选择至少一位成员', 'err')
  creating.value = true
  try {
    const r = await API.groupCreate({ name: groupName.value.trim(), members: picked.value })
    const g = r.data || {}
    toast('群聊创建成功')
    createOpen.value = false
    try { const gr = await API.groups(); groups.value = gr.data || [] } catch (_) {}
    if (g.groupId) {
      const convId = 'conv_' + g.groupId
      if (!chat.convs.find(c => c.convId === convId)) {
        chat.convs.unshift({ convId, convType: 2, peerId: g.groupId, peerName: g.name, unread: 0, lastMsg: '', lastMsgAt: Date.now() })
      }
      await chat.loadConversations()
      emit('open-conv', convId)
    }
  } catch (e) { toast(e.message, 'err') } finally { creating.value = false }
}

async function openGroupDetail(g) {
  groupTab.value = 'info'
  groupDetail.value = g
  groupMembers.value = []
  groupOpen.value = true
  try {
    const [d, m] = await Promise.all([API.groupDetail(g.groupId), API.groupMembers(g.groupId)])
    groupDetail.value = d.data || g
    groupMembers.value = m.data || []
  } catch (e) { toast(e.message, 'err') }
}
async function kickMember(m) {
  if (!confirm(`将「${m.nickname || m.uid}」移出群聊？`)) return
  try {
    await API.groupLeave(groupDetail.value.groupId, m.uid)
    toast('已移出成员')
    const [d, ms] = await Promise.all([API.groupDetail(groupDetail.value.groupId), API.groupMembers(groupDetail.value.groupId)])
    groupDetail.value = d.data || groupDetail.value
    groupMembers.value = ms.data || []
    const gr = await API.groups(); groups.value = gr.data || []
  } catch (e) { toast(e.message, 'err') }
}
async function dissolveGroup() {
  if (!confirm('解散后群成员将无法继续在群内发言，确认解散？')) return
  try {
    await API.groupDissolve(groupDetail.value.groupId)
    const convId = 'conv_' + groupDetail.value.groupId
    if (chat.currentConv === convId) chat.currentConv = ''
    chat.convs = chat.convs.filter(c => c.convId !== convId)
    toast('群聊已解散')
    groupOpen.value = false
    const gr = await API.groups(); groups.value = gr.data || []
  } catch (e) { toast(e.message, 'err') }
}
async function leaveGroup(g) {
  if (!confirm(`确认退出「${g.name}」？`)) return
  try {
    await API.groupLeave(g.groupId, myUid.value)
    toast('已退出群聊')
    groupOpen.value = false
    const convId = 'conv_' + g.groupId
    if (chat.currentConv === convId) chat.currentConv = ''
    chat.convs = chat.convs.filter(c => c.convId !== convId)
    const gr = await API.groups(); groups.value = gr.data || []
  } catch (e) { toast(e.message, 'err') }
}
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>通讯录</h2>
      <div class="tabs">
        <button :class="{ on: tab === 'friends' }" @click="tab = 'friends'">外部联系人 · {{ friends.length }}</button>
        <button :class="{ on: tab === 'applies' }" @click="tab = 'applies'; refreshFriends()">
          新的朋友<span v-if="applies.length" class="count-dot">{{ applies.length }}</span>
        </button>
        <button :class="{ on: tab === 'dept' }" v-if="!isExternal" @click="tab = 'dept'">组织架构</button>
        <button :class="{ on: tab === 'groups' }" @click="tab = 'groups'">群聊 · {{ groups.length }}</button>
      </div>
      <input v-if="tab === 'friends'" v-model="keyword" class="pane-search" placeholder="搜索姓名 / 账号 / 部门">
      <button v-if="tab === 'friends' || tab === 'applies'" class="btn sm" @click="openAdd">+ 添加好友</button>
      <button v-if="tab === 'groups'" class="btn sm primary" @click="openCreate">+ 新建群聊</button>
    </header>

    <div class="pane-body scroll">
      <template v-if="tab === 'friends'">
        <div v-for="u in filtered" :key="u.uid" class="row-card" @click="chatWith(u)">
          <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="md" />
          <div class="row-main">
            <div class="row-name">{{ u.nickname }} <span class="muted">@{{ u.username }}</span><span v-if="u.external" class="tag ext">外部</span></div>
            <div class="row-sub">{{ u.external ? '企业外部用户' : (u.deptName || '未分配部门') }}<span v-if="u.title"> · {{ u.title }}</span></div>
          </div>
          <span v-if="u.online" class="state ok">在线</span>
          <button class="btn sm">发消息</button>
        </div>
        <div v-if="!filtered.length" class="empty-tip">暂无好友，可点击右上角「添加好友」</div>
      </template>

      <template v-else-if="tab === 'applies'">
        <div v-for="a in applies" :key="a.id" class="row-card">
          <Avatar :name="fromName(a.fromUid)" size="md" />
          <div class="row-main">
            <div class="row-name">{{ fromName(a.fromUid) }}</div>
            <div class="row-sub">{{ a.message || '请求添加你为好友' }}</div>
          </div>
          <button class="btn sm primary" @click="handleApply(a, true)">同意</button>
          <button class="btn sm" @click="handleApply(a, false)">拒绝</button>
        </div>
        <div v-if="!applies.length" class="empty-tip">暂无新的好友申请</div>
      </template>

      <template v-else-if="tab === 'dept'">
        <div class="dept-chips">
          <button class="chip" :class="{ on: deptId === 0 }" @click="pickDept(0)">全部</button>
          <button v-for="d in depts" :key="d.id" class="chip" :class="{ on: deptId === d.id }" @click="pickDept(d.id)">{{ d.name }}</button>
        </div>
        <div class="sec-title">成员 · {{ deptMembers.length }} 人</div>
        <div v-for="u in deptMembers" :key="u.uid" class="row-card" @click="chatWith(u)">
          <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="md" />
          <div class="row-main">
            <div class="row-name">{{ u.nickname }} <span class="muted">@{{ u.username }}</span></div>
            <div class="row-sub">{{ u.deptName || '' }}</div>
          </div>
          <button class="btn sm">发消息</button>
        </div>
        <div v-if="!deptMembers.length" class="empty-tip">该部门暂无成员</div>
      </template>

      <template v-else>
        <div v-for="g in groups" :key="g.groupId" class="row-card" @click="enterGroup(g)">
          <Avatar :name="g.name" size="md" />
          <div class="row-main">
            <div class="row-name">{{ g.name }}</div>
            <div class="row-sub">{{ g.memberCount }} 位成员 · {{ g.notice || '暂无群公告' }}</div>
          </div>
          <button class="btn sm" @click.stop="openGroupDetail(g)">群资料</button>
          <button class="btn sm primary">进入</button>
        </div>
        <div v-if="!groups.length" class="empty-tip">暂未加入任何群聊，可点击右上角「新建群聊」</div>
      </template>
    </div>

    <Modal v-if="addOpen" title="添加好友" width="460px" @close="addOpen = false">
      <input v-model="addKw" class="pane-search modal-search" placeholder="搜索姓名 / 账号">
      <div class="member-list">
        <div v-for="u in addCandidates" :key="u.uid" class="member-row">
          <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="sm" />
          <div style="flex:1">
            <div>{{ u.nickname }} <span class="muted">@{{ u.username }}</span><span v-if="u.external" class="tag ext">外部</span></div>
            <div class="sr-desc">{{ u.external ? '企业外部用户' : (u.deptName || '企业内部用户') }}</div>
          </div>
          <button class="btn sm primary" :disabled="applied.includes(u.uid)" @click="applyFriend(u)">
            {{ applied.includes(u.uid) ? '已申请' : '添加' }}
          </button>
        </div>
        <div v-if="!addCandidates.length" class="empty-tip">没有可添加的联系人</div>
      </div>
      <template #footer><button class="btn" @click="addOpen = false">关闭</button></template>
    </Modal>

    <Modal v-if="createOpen" title="新建群聊" width="480px" @close="createOpen = false">
      <label class="field"><span>群名称</span><input v-model="groupName" placeholder="请输入群名称" maxlength="32"></label>
      <div class="sec-title">选择成员 · 已选 {{ picked.length }} 人</div>
      <div class="member-list">
        <label v-for="u in friends" :key="u.uid" class="member-row check">
          <input type="checkbox" :checked="picked.includes(u.uid)" @change="togglePick(u.uid)">
          <Avatar :name="u.nickname || u.uid" :src="u.avatar" size="sm" />
          <span>{{ u.nickname }} <span class="muted">@{{ u.username }}</span></span>
        </label>
        <div v-if="!friends.length" class="empty-tip">暂无好友可邀请</div>
      </div>
      <template #footer>
        <button class="btn" @click="createOpen = false">取消</button>
        <button class="btn primary" :disabled="creating" @click="createGroup">{{ creating ? '创建中…' : '创建并进入' }}</button>
      </template>
    </Modal>

    <Modal v-if="groupOpen && groupDetail" :title="groupDetail.name" width="520px" @close="groupOpen = false">
      <div class="tabs group-tabs">
        <button :class="{ on: groupTab === 'info' }" @click="groupTab = 'info'">群资料</button>
        <button :class="{ on: groupTab === 'members' }" @click="groupTab = 'members'">成员 · {{ groupMembers.length }}</button>
      </div>
      <template v-if="groupTab === 'info'">
        <div class="kv"><span>群 ID</span><span>{{ groupDetail.groupId }}</span></div>
        <div class="kv"><span>群主</span><span>{{ groupDetail.ownerUid === myUid ? '我' : groupDetail.ownerUid }}</span></div>
        <div class="kv"><span>成员数</span><span>{{ groupDetail.memberCount }} 人</span></div>
        <div class="sec-title">群公告</div>
        <div class="group-notice">{{ groupDetail.notice || '暂无群公告' }}</div>
      </template>
      <template v-else>
        <div class="member-list">
          <div v-for="m in groupMembers" :key="m.uid" class="member-row">
            <Avatar :name="m.nickname || m.uid" size="sm" />
            <span style="flex:1">{{ m.nickname || m.uid }}</span>
            <span v-if="m.role === 'owner'" class="tag">群主</span>
            <button
              v-else-if="isOwner" class="btn sm danger"
              @click="kickMember(m)"
            >移出</button>
          </div>
        </div>
      </template>
      <template #footer>
        <button v-if="isOwner" class="btn danger" @click="dissolveGroup">解散群聊</button>
        <button v-else class="btn" @click="leaveGroup(groupDetail)">退出群聊</button>
        <button class="btn" @click="groupOpen = false">关闭</button>
      </template>
    </Modal>
  </div>
</template>
