<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { lang, toggleLang, t } from '../i18n'
import { toast } from '../toast'

const router = useRouter()
const tab = ref('notify')

const notifyOpts = ref([
  { label: '新消息通知', on: true },
  { label: '显示消息详情', on: true },
  { label: '声音', on: true },
  { label: '振动', on: false },
  { label: '勿扰模式', on: false }
])
const privacyOpts = ref([
  { label: '加我需要验证', on: true },
  { label: '允许通过手机号搜索', on: true },
  { label: '展示在线状态', on: true }
])
const dark = ref(false)
function toggle(o) { o.on = !o.on }
function switchLang() { toggleLang(); toast(lang.value === 'en' ? 'Switched to English' : '已切换为中文') }
function toggleDark() { dark.value = !dark.value; document.body.classList.toggle('h5-dark', dark.value) }
const tabs = computed(() => [['notify', '通知'], ['privacy', '隐私'], ['general', '通用']])
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
      <div v-for="o in notifyOpts" :key="o.label" class="switch-row" @click="toggle(o)">
        <span>{{ o.label }}</span>
        <span class="switch" :class="{ on: o.on }"></span>
      </div>
    </template>

    <template v-else-if="tab === 'privacy'">
      <div v-for="o in privacyOpts" :key="o.label" class="switch-row" @click="toggle(o)">
        <span>{{ o.label }}</span>
        <span class="switch" :class="{ on: o.on }"></span>
      </div>
      <button class="gb-row" @click="toast('已登录设备（演示）')">🖥️ 已登录设备<span class="gb-r">3 台 ›</span></button>
      <button class="gb-row" @click="toast('黑名单（演示）')">🛡️ 黑名单<span class="gb-r">2 ›</span></button>
    </template>

    <template v-else>
      <div class="switch-row" @click="toggleDark">
        <span>深色模式</span><span class="switch" :class="{ on: dark }"></span>
      </div>
      <button class="gb-row" @click="switchLang">🌐 语言 / Language<span class="gb-r">{{ lang === 'zh' ? '简体中文' : 'English' }} ›</span></button>
      <div class="switch-row"><span>字体大小<div class="sr-desc">标准</div></span><span class="state info">标准</span></div>
      <button class="gb-row" @click="toast('缓存已清理（演示）')">🗑️ 清理缓存<span class="gb-r">1.2 GB ›</span></button>
      <div class="switch-row"><span>关于<div class="sr-desc">原型 v2.0 · BRD v2.0 / 架构 v1.12</div></span><span class="state info">v1.0</span></div>
    </template>
  </div>
</template>
