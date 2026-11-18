import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// H5 以 /h5 子路径部署；本地开发时 /api、/ws 代理到后端
export default defineConfig({
  base: '/h5/',
  plugins: [vue()],
  server: {
    port: 5174,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/ws': { target: 'ws://localhost:8081', ws: true }
    }
  },
  build: { outDir: 'dist' }
})
