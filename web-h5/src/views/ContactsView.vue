<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API } from '../api'
import { t } from '../i18n'
import Avatar from '../components/Avatar.vue'
import LineIcon from '../components/LineIcon.vue'
import { toast } from '../toast'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const chat = useChatStore()
const auth = useAuthStore()
const keyword = ref('')
const friends = ref([])
const depts = ref([])
const applies = ref([])
const isExternal = computed(() => auth.user?.isExternal === true)
onMounted(async () => {
  try {
    const [f, a] = await Promise.all([API.friends(), API.friendApplies()])
    friends.value = f.data || []
    applies.value = (a.data || []).filter(x => x.status === 0)
    // 外部联系人默认无法查看组织架构，不再请求部门数据
    if (!isExternal.value) {
      const d = await API.depts()
      depts.value = d.data || []
    }
  } catch (e) { toast(e.message, 'err') }
})
const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return friends.value
  return friends.value.filter(u =>
    (u.nickname || '').toLowerCase().includes(q) ||
    (u.username || '').toLowerCase().includes(q) ||
    (u.deptName || '').toLowerCase().includes(q)
  )
})
function subOf(u) {
  if (u.external) return `企业外部用户 · @${u.username}`
  return u.deptName ? `${u.deptName} · @${u.username}` : `· @${u.username}`
}
function chatWith(u) {
  const convId = chat.ensureConv(u.uid, u.nickname)
  router.push('/chat/' + encodeURIComponent(convId))
}
</script>
<template>
  <div class="qq-head">
    <div>
      <div class="qq-name">{{ t('contacts') }}</div>
      <div class="qq-status"><i class="dot"></i>{{ isExternal ? `外部联系人 · ${friends.length} 位` : `${depts.length} 个部门 · ${friends.length} 位外部联系人` }}</div>
    </div>
    <button class="qq-plus" title="发起群聊" @click="router.push('/create-group')"><LineIcon name="users" :size="22" /></button>
    <button class="qq-plus" title="添加朋友" @click="router.push('/add-friend')">＋</button>
  </div>
  <div class="qq-body">
    <div class="qq-search">
      <LineIcon name="search" :size="16" />
      <input v-model="keyword" placeholder="搜索姓名 / 拼音 / 部门">
    </div>
    <div class="qq-list">
      <button class="qq-row" @click="router.push('/friends')">
        <span class="qq-ava"><i class="qq-ico ic-new"><LineIcon name="users" :size="22" /></i><i v-if="applies.length" class="qq-badge">{{ applies.length }}</i></span>
        <div class="qq-main">
          <div class="qq-title"><span class="t">{{ t('newFriends') }}</span></div>
          <div class="qq-sub">{{ applies.length }} 条待处理的好友申请</div>
        </div>
        <span class="qq-arrow">›</span>
      </button>
      <button class="qq-row" @click="router.push('/groups')">
        <span class="qq-ava"><i class="qq-ico ic-group"><LineIcon name="chat" :size="22" /></i></span>
        <div class="qq-main">
          <div class="qq-title"><span class="t">{{ t('groupList') }}</span></div>
          <div class="qq-sub">我加入的群聊</div>
        </div>
        <span class="qq-arrow">›</span>
      </button>
      <button v-if="!isExternal" class="qq-row" @click="router.push('/dept')">
        <span class="qq-ava"><i class="qq-ico ic-dept"><LineIcon name="building" :size="22" /></i></span>
        <div class="qq-main">
          <div class="qq-title"><span class="t">{{ t('org') }}</span></div>
          <div class="qq-sub">{{ depts.length }} 个部门</div>
        </div>
        <span class="qq-arrow">›</span>
      </button>
    </div>
    <div class="qq-sec">企业外部联系人 · {{ filtered.length }} 人</div>
    <div class="qq-list">
      <button v-for="u in filtered" :key="u.uid" class="qq-row" @click="chatWith(u)">
        <span class="qq-ava"><Avatar :name="u.nickname" /></span>
        <div class="qq-main">
          <div class="qq-title">
            <span class="t">{{ u.nickname }}</span>
            <span v-if="u.external" class="qq-tag">外部</span>
          </div>
          <div class="qq-sub">{{ subOf(u) }}</div>
        </div>
        <span class="qq-arrow">›</span>
      </button>
    </div>
    <div v-if="!filtered.length" class="qq-empty">{{ t('noFriend') }}</div>
  </div>
</template>
<style scoped>
.qq-head .qq-plus { margin-left: 0; font-size: 19px; }
.qq-head .qq-plus:first-of-type { margin-left: auto; }
.qq-ico { display: grid; place-items: center; width: 48px; height: 48px; border-radius: 14px; font-style: normal; font-size: 22px; }
.ic-new { background: #eaf2ff; color: #3b82f6; }
.ic-group { background: #e9f8f1; color: #14b184; }
.ic-dept { background: #fff2e8; color: #f0803c; }
.qq-arrow { color: #c3cad9; font-size: 17px; flex: none; }
.qq-sec { font-size: 12px; font-weight: 600; color: #8a95aa; padding: 14px 16px 8px; background: var(--card); }
</style>
