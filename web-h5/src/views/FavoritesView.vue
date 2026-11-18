<script setup>
import { useRouter } from 'vue-router'
import { t } from '../i18n'
const router = useRouter()
</script>
<template>
  <div class="chat-head">
    <button class="icon-btn" @click="router.back()">‹</button>
    <div><div class="ch-name">{{ t('favorites') }}</div><div class="ch-sub">已收藏的消息与文件</div></div>
  </div>
  <div class="h5-body">
    <div class="loading">{{ t('empty') }}</div>
  </div>
</template>
