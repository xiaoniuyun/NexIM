<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { API } from '../api'
import { toast } from '../toast'
import Avatar from '../components/Avatar.vue'

const router = useRouter()
const auth = useAuthStore()
const form = ref({ nickname: '', signature: '', gender: 0 })
const avatar = ref('')
const uploading = ref(false)
const saving = ref(false)
const filePick = ref(null)
const genders = [{ label: '保密', value: 0 }, { label: '男', value: 1 }, { label: '女', value: 2 }]

onMounted(async () => {
  await auth.refreshProfile()
  const u = auth.user || {}
  form.value = {
    nickname: u.nickname || '',
    signature: u.signature || '',
    gender: Number(u.gender) || 0
  }
  avatar.value = u.avatar || ''
})

function pickAvatar() {
  if (uploading.value || !filePick.value) return
  filePick.value.value = ''
  filePick.value.click()
}

async function onAvatarPick(e) {
  const file = e.target.files && e.target.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) { toast('头像不能超过 5MB', 'err'); return }
  const fd = new FormData()
  fd.append('file', file)
  uploading.value = true
  try {
    const r = await API.fileUpload(fd)
    const fileId = r.data?.fileId
    if (!fileId) throw new Error('上传失败')
    avatar.value = 'file:' + fileId
    toast('头像已上传，记得保存')
  } catch (err) {
    toast(err.message || '头像上传失败', 'err')
  } finally {
    uploading.value = false
  }
}

async function save() {
  const nickname = form.value.nickname.trim()
  if (!nickname) { toast('昵称不能为空', 'err'); return }
  saving.value = true
  try {
    await auth.updateProfile({
      nickname,
      signature: form.value.signature.trim(),
      gender: form.value.gender,
      avatar: avatar.value
    })
    toast('资料已保存')
    router.back()
  } catch (err) {
    toast(err.message || '保存失败', 'err')
  } finally {
    saving.value = false
  }
}
</script>
<template>
  <div class="h5-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <h2>编辑资料</h2>
  </div>
  <div class="h5-body">
    <div class="pe-avatar">
      <Avatar :name="form.nickname || '我'" :src="avatar" size="lg" />
      <div>
        <div class="pe-name">{{ form.nickname || '未设置昵称' }}</div>
        <button class="btn-ghost pe-change" :disabled="uploading" @click="pickAvatar">
          {{ uploading ? '上传中…' : '更换头像' }}
        </button>
      </div>
    </div>

    <div class="group-block">
      <div class="pe-field">
        <label>昵称</label>
        <input v-model="form.nickname" maxlength="32" placeholder="请输入昵称">
      </div>
      <div class="pe-field">
        <label>性别</label>
        <div class="status-opts">
          <button v-for="g in genders" :key="g.value" :class="{ on: form.gender === g.value }"
                  @click="form.gender = g.value">{{ g.label }}</button>
        </div>
      </div>
      <div class="pe-field">
        <label>个性签名</label>
        <textarea v-model="form.signature" rows="3" maxlength="120" placeholder="介绍一下自己吧"></textarea>
      </div>
      <div class="pe-field">
        <label>账号</label>
        <div class="pe-static">@{{ auth.user?.username }} · UID {{ auth.user?.uid }}</div>
      </div>
    </div>

    <button class="btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存资料' }}</button>
  </div>
  <input ref="filePick" type="file" accept="image/*" class="hidden-file" @change="onAvatarPick">
</template>
