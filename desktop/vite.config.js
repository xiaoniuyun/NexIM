import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 桌面端复用 web-h5 的 API / Pinia stores，保证 IM 协议与消息状态机只维护一份。
export default defineConfig({
  plugins: [vue()],
  base: './',
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@h5': fileURLToPath(new URL('../web-h5/src', import.meta.url))
    },
    // 避免 web-h5 与 desktop 各自安装依赖造成两份 vue/pinia 实例
    dedupe: ['vue', 'pinia']
  },
  server: {
    port: 5174,
    fs: { allow: ['..'] }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
