import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 桌面端业务层已内聚到 src/shared，不依赖 H5 工程。
export default defineConfig({
  plugins: [vue()],
  base: './',
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
    dedupe: ['vue', 'pinia']
  },
  server: {
    port: 5174
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
