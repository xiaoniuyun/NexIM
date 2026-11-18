<script setup>
import { toasts } from '../toast'
</script>
<template>
  <div class="toast-box">
    <div v-for="i in toasts" :key="i.id" class="toast" :class="i.type">{{ i.msg }}</div>
  </div>
</template>
