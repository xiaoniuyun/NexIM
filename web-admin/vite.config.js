import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 管理后台以 /admin 子路径部署；开发时代理后端
export default defineConfig({
  base: '/admin/',
  plugins: [vue()],
  server: {
    port: 5175,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/ws': { target: 'ws://localhost:8081', ws: true }
    }
  },
  build: { outDir: 'dist' }
})
