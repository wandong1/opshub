import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  optimizeDeps: {
    include: ['js-yaml']
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:9876',
        changeOrigin: true,
        ws: true  // 启用 WebSocket 代理
      },
      // Grafana 代理：将 Grafana sub_path 请求转发到后端（后端再代理到 Grafana）
      // 使用正则匹配所有以 /grafana 开头的路径（兼容各种 sub_path 配置）
      '^/grafana': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:9876',
        changeOrigin: true
      }
    }
  }
})
