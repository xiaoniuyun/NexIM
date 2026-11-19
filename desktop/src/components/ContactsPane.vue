<script setup>
import { ref, computed, onMounted } from 'vue'
import Avatar from '@h5/components/Avatar.vue'
import { API } from '@h5/api'
import { useChatStore } from '@h5/stores/chat'
import { toast } from '@h5/toast'

const emit = defineEmits(['open-conv'])
const chat = useChatStore()
const tab = ref('friends')
const keyword = ref('')
const friends = ref([])
const depts = ref([])
const deptId = ref(0)
const deptMembers = ref([])
const groups = ref([])

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return friends.value
  return friends.value.filter(u =>
    String(u.nickname || '').toLowerCase().includes(q) ||
    String(u.username || '').toLowerCase().includes(q) ||
    String(u.deptName || '').toLowerCase().includes(q)
  )
})

onMounted(async () => {
  try {
    const [f, d, g] = await Promise.all([API.friends(), API.depts(), API.groups()])
    friends.value = f.data || []
    depts.value = d.data || []
    groups.value = g.data || []
    pickDept(0)
  } catch (e) { toast(e.message, 'err') }
})

async function pickDept(id) {
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
</script>

<template>
  <div class="pane">
    <header class="pane-head">
      <h2>通讯录</h2>
      <div class="tabs">
        <button :class="{ on: tab === 'friends' }" @click="tab = 'friends'">好友 · {{ friends.length }}</button>
        <button :class="{ on: tab === 'dept' }" @click="tab = 'dept'">组织架构</button>
        <button :class="{ on: tab === 'groups' }" @click="tab = 'groups'">群聊 · {{ groups.length }}</button>
      </div>
      <input v-if="tab === 'friends'" v-model="keyword" class="pane-search" placeholder="搜索姓名 / 账号 / 部门">
    </header>

    <div class="pane-body scroll">
      <template v-if="tab === 'friends'">
        <div v-for="u in filtered" :key="u.uid" class="row-card" @click="chatWith(u)">
          <Avatar :name="u.nickname || u.uid" size="md" />
          <div class="row-main">
            <div class="row-name">{{ u.nickname }} <span class="muted">@{{ u.username }}</span></div>
            <div class="row-sub">{{ u.deptName || '未分配部门' }}</div>
          </div>
          <button class="btn sm">发消息</button>
        </div>
        <div v-if="!filtered.length" class="empty-tip">暂无好友</div>
      </template>

      <template v-else-if="tab === 'dept'">
        <div class="dept-chips">
          <button class="chip" :class="{ on: deptId === 0 }" @click="pickDept(0)">全部</button>
          <button v-for="d in depts" :key="d.id" class="chip" :class="{ on: deptId === d.id }" @click="pickDept(d.id)">{{ d.name }}</button>
        </div>
        <div class="sec-title">成员 · {{ deptMembers.length }} 人</div>
        <div v-for="u in deptMembers" :key="u.uid" class="row-card" @click="chatWith(u)">
          <Avatar :name="u.nickname || u.uid" size="md" />
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
          <button class="btn sm">进入</button>
        </div>
        <div v-if="!groups.length" class="empty-tip">暂未加入任何群聊</div>
      </template>
    </div>
  </div>
</template>
